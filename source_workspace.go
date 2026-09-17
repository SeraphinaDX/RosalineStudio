// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	rosaline "github.com/SeraphinaDX/Rosaline"
)

const maximumSourceFileSize = 2 << 20

type sourceDocument struct {
	File  projectFile
	Text  string
	Saved string
}

func (document *sourceDocument) dirty() bool {
	return document != nil && document.File.editable() && document.Text != document.Saved
}

func (studio *studio) buildSourceWorkspace() rosaline.Widget {
	files := rosaline.Column(
		rosaline.Label("Project Files").Bold().Color(rosaline.Rose),
		rosaline.Label("Developer files are editable; generated files are protected.").Color(rosaline.DefaultTheme.Muted),
		studio.sourceTree,
		rosaline.Label("New Go filename"),
		rosaline.TextBox(&studio.sourceFileName).Width(23),
		rosaline.Row(
			rosaline.Button("New", studio.newSourceFile).Primary(),
			rosaline.Button("Rename", studio.renameSourceFile),
			rosaline.Button("Delete", studio.deleteSourceFile),
		).Gap(5),
		rosaline.Button("Refresh Files", studio.refreshProjectFiles),
	).Gap(6).Expand()

	editor := rosaline.Column(
		studio.sourceTabs,
		rosaline.Row(
			rosaline.LabelFunc(studio.sourceHeader).Bold(),
			rosaline.Spring(),
			rosaline.Button("Save", studio.saveActiveSource).Primary(),
			rosaline.Button("Save All", func() { studio.saveAllSourceDocuments() }),
			rosaline.Button("Format", studio.formatActiveSource),
			rosaline.Button("Close", studio.closeActiveSource),
		).Gap(6),
		studio.sourceEditor,
	).Gap(7).Expand()

	source := rosaline.Row(
		rosaline.Size(files, 235, 500),
		editor,
	).Gap(9).Expand()
	output := rosaline.Column(
		rosaline.Row(
			rosaline.Label("Build Output").Bold().Color(rosaline.Rose),
			rosaline.Spring(),
			rosaline.Button("Build", studio.buildGenerated).Primary(),
			rosaline.Button("Build and Run", studio.runGenerated),
		).Gap(7),
		studio.buildOutputEditor,
		rosaline.Label("Errors and warnings (double-click to open source)").Bold(),
		studio.buildDiagnostics,
	).Gap(7).Expand()
	studio.sourceWorkspace = rosaline.Tabs(
		rosaline.Tab("Source", source),
		rosaline.Tab("Build Output", output),
	).Expand()
	return studio.sourceWorkspace
}

func (studio *studio) showSourceWorkspace() {
	if !studio.prepareSourceWorkspace() {
		return
	}
	studio.workspace.Select(workspaceSourceTab)
	studio.sourceWorkspace.Select(0)
}

func (studio *studio) prepareSourceWorkspace() bool {
	if !studio.confirmOpenHandler("") {
		return false
	}
	if studio.path == "" {
		if !studio.saveAs() {
			return false
		}
	}
	directory := generatedApplicationDirectory(studio.path)
	if generatedApplicationNeedsSetup(directory) {
		if !studio.generate() {
			return false
		}
	}
	studio.refreshProjectFiles()
	if studio.activeSource == "" {
		if _, ok := studio.sourceFiles["handlers.go"]; ok {
			studio.openSourceFile("handlers.go")
		}
	}
	return true
}

func (studio *studio) sourceRoot() string {
	if studio.path == "" {
		return ""
	}
	return generatedApplicationDirectory(studio.path)
}

func (studio *studio) refreshProjectFiles() {
	root := studio.sourceRoot()
	files, err := scanProjectFiles(root)
	if err != nil {
		studio.status = "Could not refresh project files: " + err.Error()
		return
	}
	studio.sourceFiles = make(map[string]projectFile, len(files))
	studio.sourceNodes = make(map[string]*rosaline.TreeNode, len(files))
	groups := map[projectFileKind][]*rosaline.TreeNode{}
	for _, file := range files {
		studio.sourceFiles[file.Path] = file
		node := rosaline.Node(file.Path).WithValue(file.Path)
		studio.sourceNodes[file.Path] = node
		groups[file.Kind] = append(groups[file.Kind], node)
	}
	roots := []*rosaline.TreeNode{
		rosaline.Node("Developer Source", groups[projectFileSource]...).WithValue("category:source").Expanded(),
		rosaline.Node("Generated (Read Only)", groups[projectFileGenerated]...).WithValue("category:generated").Expanded(),
		rosaline.Node("Project Files", groups[projectFileProject]...).WithValue("category:project").Expanded(),
		rosaline.Node("Assets", groups[projectFileAsset]...).WithValue("category:assets"),
	}
	studio.sourceTree.SetNodes(roots...)
	studio.rebuildSourceTabs()
	studio.status = fmt.Sprintf("Found %d project files", len(files))
}

func (studio *studio) openSourceFile(relative string) {
	file, ok := studio.sourceFiles[filepath.ToSlash(relative)]
	if !ok || !file.textFile() {
		if ok {
			studio.status = "Binary asset: " + file.Path
		}
		return
	}
	if document := studio.sourceDocuments[file.Path]; document != nil {
		studio.activateSourceDocument(file.Path)
		return
	}
	full, err := safeProjectPath(studio.sourceRoot(), file.Path)
	if err != nil {
		studio.status = err.Error()
		return
	}
	info, err := os.Stat(full)
	if err != nil {
		studio.status = "Could not open " + file.Path + ": " + err.Error()
		return
	}
	if info.Size() > maximumSourceFileSize {
		studio.status = "File is too large for Studio's source editor"
		return
	}
	data, err := os.ReadFile(full)
	if err != nil {
		studio.status = "Could not open " + file.Path + ": " + err.Error()
		return
	}
	text := string(data)
	studio.sourceDocuments[file.Path] = &sourceDocument{File: file, Text: text, Saved: text}
	studio.sourceOrder = append(studio.sourceOrder, file.Path)
	studio.rebuildSourceTabs()
	studio.activateSourceDocument(file.Path)
}

func (studio *studio) activateSourceDocument(path string) {
	document := studio.sourceDocuments[path]
	if document == nil {
		return
	}
	studio.activeSource = path
	studio.syncSource = true
	studio.sourceTabValue = path
	studio.sourceText = document.Text
	studio.sourceTabs.Select(path)
	studio.sourceEditor.SetGoSyntax(strings.EqualFold(filepath.Ext(path), ".go"))
	studio.sourceEditor.SetReadOnly(!document.File.editable())
	studio.sourceEditor.SetText(document.Text)
	studio.sourceEditor.MarkSaved()
	studio.syncSource = false
	if node := studio.sourceNodes[path]; node != nil {
		studio.sourceTree.Select(node)
	}
	studio.status = "Opened " + path
}

func (studio *studio) switchSourceFile(path string) {
	if studio.syncSource || path == "" || path == studio.activeSource {
		return
	}
	studio.activateSourceDocument(path)
}

func (studio *studio) sourceChanged(text string) {
	if studio.syncSource || studio.activeSource == "" {
		return
	}
	document := studio.sourceDocuments[studio.activeSource]
	if document == nil || !document.File.editable() {
		return
	}
	document.Text = text
	studio.sourceText = text
	studio.rebuildSourceTabs()
}

func (studio *studio) rebuildSourceTabs() {
	choices := make([]rosaline.RadioChoice, 0, len(studio.sourceOrder))
	for _, path := range studio.sourceOrder {
		document := studio.sourceDocuments[path]
		if document == nil {
			continue
		}
		label := filepath.Base(path)
		if document.dirty() {
			label += " *"
		}
		choices = append(choices, rosaline.Choice(label, path))
	}
	studio.syncSource = true
	studio.sourceTabs.SetChoices(choices...)
	if studio.activeSource != "" {
		studio.sourceTabs.Select(studio.activeSource)
	}
	studio.syncSource = false
}

func (studio *studio) sourceHeader() string {
	if studio.activeSource == "" {
		return "Open a file from Project Files"
	}
	document := studio.sourceDocuments[studio.activeSource]
	if document == nil {
		return studio.activeSource
	}
	mode := "editable"
	if !document.File.editable() {
		mode = "read only"
	}
	return document.File.Path + " — " + mode
}

func (studio *studio) saveActiveSource() {
	if studio.activeSource == "" {
		return
	}
	if err := studio.saveSourceDocument(studio.activeSource, true); err != nil {
		rosaline.Error("Could not save source", err.Error())
	}
}

func (studio *studio) saveSourceDocument(path string, formatGo bool) error {
	document := studio.sourceDocuments[path]
	if document == nil || !document.File.editable() {
		return nil
	}
	text := document.Text
	formatted := false
	if formatGo && strings.EqualFold(filepath.Ext(path), ".go") {
		if data, err := format.Source([]byte(text)); err == nil {
			text = string(data)
			formatted = text != document.Text
		}
	}
	full, err := safeProjectPath(studio.sourceRoot(), path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	document.Text = text
	document.Saved = text
	if path == studio.activeSource {
		studio.syncSource = true
		studio.sourceText = text
		studio.sourceEditor.SetText(text)
		studio.sourceEditor.MarkSaved()
		studio.syncSource = false
	}
	studio.rebuildSourceTabs()
	if formatted {
		studio.status = "Formatted and saved " + path
	} else {
		studio.status = "Saved " + path
	}
	return nil
}

func (studio *studio) saveAllSourceDocuments() bool {
	for _, path := range studio.sourceOrder {
		if document := studio.sourceDocuments[path]; document != nil && document.dirty() {
			if err := studio.saveSourceDocument(path, true); err != nil {
				rosaline.Error("Could not save source", err.Error())
				return false
			}
		}
	}
	return true
}

func (studio *studio) formatActiveSource() {
	document := studio.sourceDocuments[studio.activeSource]
	if document == nil || !document.File.editable() || !strings.EqualFold(filepath.Ext(document.File.Path), ".go") {
		studio.status = "Only editable Go files can be formatted"
		return
	}
	formatted, err := format.Source([]byte(document.Text))
	if err != nil {
		studio.status = "Could not format: " + err.Error()
		return
	}
	document.Text = string(formatted)
	studio.activateSourceDocument(document.File.Path)
	studio.rebuildSourceTabs()
	studio.status = "Formatted " + document.File.Path + " (save to write it)"
}

func (studio *studio) closeActiveSource() {
	path := studio.activeSource
	if path == "" || !studio.confirmSourceDocument(path, "closing it") {
		return
	}
	delete(studio.sourceDocuments, path)
	index := -1
	for candidate, open := range studio.sourceOrder {
		if open == path {
			index = candidate
			break
		}
	}
	if index >= 0 {
		studio.sourceOrder = append(studio.sourceOrder[:index], studio.sourceOrder[index+1:]...)
	}
	studio.activeSource = ""
	studio.sourceText = ""
	studio.sourceEditor.SetText("")
	studio.rebuildSourceTabs()
	if len(studio.sourceOrder) != 0 {
		if index < 0 {
			index = 0
		}
		studio.activateSourceDocument(studio.sourceOrder[min(index, len(studio.sourceOrder)-1)])
	}
}

func (studio *studio) confirmSourceDocument(path, action string) bool {
	document := studio.sourceDocuments[path]
	if document == nil || !document.dirty() {
		return true
	}
	switch rosaline.AskSaveChanges("Unsaved source", "Save "+path+" before "+action+"?") {
	case rosaline.SaveChanges:
		return studio.saveSourceDocument(path, true) == nil
	case rosaline.DiscardChanges:
		return true
	default:
		return false
	}
}

func (studio *studio) confirmSourceChanges(action string) bool {
	for _, path := range append([]string(nil), studio.sourceOrder...) {
		if !studio.confirmSourceDocument(path, action) {
			return false
		}
	}
	return true
}

func (studio *studio) resetSourceWorkspace() {
	studio.sourceFiles = make(map[string]projectFile)
	studio.sourceNodes = make(map[string]*rosaline.TreeNode)
	studio.sourceDocuments = make(map[string]*sourceDocument)
	studio.sourceOrder = nil
	studio.activeSource = ""
	studio.sourceTabValue = ""
	studio.sourceText = ""
	studio.sourceFileName = ""
	studio.buildOutput = ""
	studio.diagnostics = nil
	if studio.sourceTree != nil {
		studio.sourceTree.SetNodes()
		studio.sourceEditor.SetText("")
		studio.sourceTabs.SetChoices()
		studio.buildOutputEditor.SetText("")
		studio.buildDiagnostics.SetItems()
	}
}

func (studio *studio) newSourceFile() {
	if !studio.prepareSourceWorkspace() {
		return
	}
	name := strings.TrimSpace(studio.sourceFileName)
	if err := validateGoFilename(name); err != nil {
		rosaline.Error("Invalid Go filename", err.Error())
		return
	}
	full, err := safeProjectPath(studio.sourceRoot(), name)
	if err != nil {
		rosaline.Error("Could not create source file", err.Error())
		return
	}
	file, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		rosaline.Error("Could not create source file", err.Error())
		return
	}
	_, writeErr := file.WriteString("package main\n")
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		rosaline.Error("Could not create source file", err.Error())
		return
	}
	studio.sourceFileName = ""
	studio.refreshProjectFiles()
	studio.openSourceFile(name)
}

func (studio *studio) renameSourceFile() {
	document := studio.sourceDocuments[studio.activeSource]
	if document == nil || !document.File.editable() {
		studio.status = "Open an editable source file before renaming"
		return
	}
	if protectedDeveloperFile(document.File.Path) {
		studio.status = document.File.Path + " is required by generated applications"
		return
	}
	name := strings.TrimSpace(studio.sourceFileName)
	if err := validateGoFilename(name); err != nil {
		rosaline.Error("Invalid Go filename", err.Error())
		return
	}
	if !studio.confirmSourceDocument(document.File.Path, "renaming it") {
		return
	}
	oldPath := document.File.Path
	oldFull, _ := safeProjectPath(studio.sourceRoot(), oldPath)
	newFull, err := safeProjectPath(studio.sourceRoot(), name)
	if err != nil {
		rosaline.Error("Could not rename source file", err.Error())
		return
	}
	if err := os.Rename(oldFull, newFull); err != nil {
		rosaline.Error("Could not rename source file", err.Error())
		return
	}
	delete(studio.sourceDocuments, oldPath)
	studio.sourceOrder = removeString(studio.sourceOrder, oldPath)
	studio.activeSource = ""
	studio.sourceFileName = ""
	studio.refreshProjectFiles()
	studio.openSourceFile(name)
}

func (studio *studio) deleteSourceFile() {
	document := studio.sourceDocuments[studio.activeSource]
	if document == nil || !document.File.editable() {
		studio.status = "Open an editable source file before deleting"
		return
	}
	if protectedDeveloperFile(document.File.Path) {
		studio.status = document.File.Path + " is required by generated applications"
		return
	}
	if !rosaline.Confirm("Delete source file", "Permanently delete "+document.File.Path+"?") {
		return
	}
	full, _ := safeProjectPath(studio.sourceRoot(), document.File.Path)
	if err := os.Remove(full); err != nil {
		rosaline.Error("Could not delete source file", err.Error())
		return
	}
	path := document.File.Path
	delete(studio.sourceDocuments, path)
	studio.sourceOrder = removeString(studio.sourceOrder, path)
	studio.activeSource = ""
	studio.refreshProjectFiles()
	if len(studio.sourceOrder) != 0 {
		studio.activateSourceDocument(studio.sourceOrder[0])
	}
	studio.status = "Deleted " + path
}

func removeString(values []string, remove string) []string {
	result := values[:0]
	for _, value := range values {
		if value != remove {
			result = append(result, value)
		}
	}
	return result
}

func buildExecutablePath(directory string) string {
	extension := ""
	if runtime.GOOS == "windows" {
		extension = ".exe"
	}
	return filepath.Join(directory, ".rosaline-studio-preview"+extension)
}

func (studio *studio) buildGenerated() {
	studio.startBuild(false)
}

func (studio *studio) startBuild(run bool) {
	if studio.runTask.Running() {
		studio.status = "A build or preview is already running"
		return
	}
	if !studio.save() {
		return
	}
	directory := generatedApplicationDirectory(studio.path)
	if generatedApplicationNeedsSetup(directory) {
		message := "Studio will create the Go project, download Rosaline and its dependencies, and build the application. Continue?"
		if !rosaline.Confirm("Set up generated application", message) {
			studio.status = "Application setup canceled"
			return
		}
	}
	if !studio.generate() {
		return
	}
	studio.runMu.Lock()
	studio.runDirectory = directory
	studio.runAfterBuild = run
	studio.runOutput = ""
	studio.runMu.Unlock()
	studio.buildOutput = "Building..."
	studio.buildOutputEditor.SetText(studio.buildOutput)
	studio.workspace.Select(workspaceSourceTab)
	studio.sourceWorkspace.Select(1)
	studio.status = "Starting build..."
	studio.runTask.Start()
}

func (studio *studio) publishBuildOutput(buildErr error) {
	studio.runMu.Lock()
	output := strings.TrimSpace(studio.runOutput)
	studio.runMu.Unlock()
	if output == "" {
		if buildErr == nil {
			output = "Build succeeded."
		} else {
			output = buildErr.Error()
		}
	}
	studio.buildOutput = output
	studio.buildOutputEditor.SetText(output)
	studio.diagnostics = parseBuildDiagnostics(output, studio.sourceRoot())
	labels := make([]string, len(studio.diagnostics))
	for index, diagnostic := range studio.diagnostics {
		labels[index] = diagnostic.label()
	}
	studio.buildDiagnostics.SetItems(labels...)
	if buildErr != nil {
		studio.workspace.Select(workspaceSourceTab)
		studio.sourceWorkspace.Select(1)
	}
}

func (studio *studio) openDiagnostic(index int) {
	if index < 0 || index >= len(studio.diagnostics) {
		return
	}
	diagnostic := studio.diagnostics[index]
	studio.workspace.Select(workspaceSourceTab)
	studio.sourceWorkspace.Select(0)
	studio.openSourceFile(diagnostic.File)
	studio.sourceEditor.GoTo(diagnostic.Line, max(0, diagnostic.Column-1))
	studio.status = diagnostic.Message
}

func (studio *studio) openGeneratedHandler() {
	if studio.codeHandler == "" || !studio.saveOpenHandler() || !studio.generate() {
		return
	}
	studio.showSourceWorkspace()
	path := "events_generated.go"
	studio.openSourceFile(path)
	document := studio.sourceDocuments[path]
	if document == nil {
		return
	}
	line := 1
	needle := "func (app *Application) " + studio.codeHandler + "("
	if offset := strings.Index(document.Text, needle); offset >= 0 {
		line += strings.Count(document.Text[:offset], "\n")
	}
	studio.sourceEditor.GoTo(line, 0)
}
