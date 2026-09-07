// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	rosaline "github.com/SeraphinaDX/Rosaline"
)

type inspectorState struct {
	Text     string
	Name     string
	Action   string
	Options  string
	Gap      string
	Padding  string
	Columns  string
	Width    string
	Height   string
	Minimum  string
	Maximum  string
	Step     string
	Expand   bool
	Primary  bool
	Bold     bool
	Password bool
	Vertical bool
}

type projectInspectorState struct {
	Title   string
	Module  string
	Width   string
	Height  string
	Padding string
	Theme   string
}

type studio struct {
	project    *designProject
	path       string
	selectedID string
	dirty      bool
	status     string

	inspector inspectorState
	settings  projectInspectorState

	palette  *rosaline.ListWidget
	tree     *rosaline.TreeWidget
	canvas   *rosaline.CanvasWidget
	treeByID map[string]*rosaline.TreeNode
	syncTree bool
	dragID   string

	undo [][]byte
	redo [][]byte

	runTask      *rosaline.Task
	runMu        sync.Mutex
	runDirectory string
	runOutput    string
}

func newStudio() *studio {
	project := newProject()
	result := &studio{
		project:    project,
		selectedID: project.Root.ID,
		status:     "Ready - double-click a palette item to add it",
	}
	result.loadInspector()
	result.loadProjectInspector()
	return result
}

func (studio *studio) run() {
	paletteNames := make([]string, len(paletteKinds))
	for index, kind := range paletteKinds {
		paletteNames[index] = string(kind)
	}
	studio.palette = rosaline.List(paletteNames...).Size(21, 10)
	studio.palette.OnActivate(func(index int, _ string) {
		if index >= 0 && index < len(paletteKinds) {
			studio.addWidget(paletteKinds[index])
		}
	})

	studio.tree = rosaline.Tree().Width(220).Height(14).Expand()
	studio.tree.OnSelect(func(node *rosaline.TreeNode) {
		if studio.syncTree || node == nil {
			return
		}
		studio.selectNode(node.Value())
	})

	studio.canvas = rosaline.Canvas(func(canvas *rosaline.DrawingCanvas) {
		drawPreview(canvas, studio.project, layoutPreview(studio.project), studio.selectedID)
	}).Size(previewWidth, previewHeight).Focus()
	studio.canvas.OnMouseDown(func(event rosaline.MouseEvent) {
		if event.Button != rosaline.MouseLeft {
			return
		}
		if node := previewBoxAt(layoutPreview(studio.project), event.X, event.Y); node != nil {
			studio.dragID = node.ID
			studio.selectNode(node.ID)
		}
	})
	studio.canvas.OnMouseUp(func(event rosaline.MouseEvent) {
		if event.Button != rosaline.MouseLeft || studio.dragID == "" {
			return
		}
		dragged := studio.dragID
		studio.dragID = ""
		target := previewBoxAt(layoutPreview(studio.project), event.X, event.Y)
		if target == nil || target.ID == dragged {
			return
		}
		before := designSnapshot(studio.project)
		if err := studio.project.moveTo(dragged, target.ID); err != nil {
			studio.status = "Could not move widget: " + err.Error()
			return
		}
		studio.commitChange(before, "Moved "+dragged)
	})

	studio.runTask = rosaline.Background(func(ctx context.Context, report *rosaline.TaskReporter) error {
		studio.runMu.Lock()
		directory := studio.runDirectory
		studio.runMu.Unlock()
		if _, err := exec.LookPath("go"); err != nil {
			return fmt.Errorf("find Go command: %w", err)
		}
		studio.runMu.Lock()
		studio.runOutput = ""
		studio.runMu.Unlock()
		var output bytes.Buffer
		environment := generatedCommandEnvironment()

		if !report.Report(5, "Downloading Rosaline and its dependencies...") {
			return ctx.Err()
		}
		download := exec.CommandContext(ctx, "go", "mod", "tidy")
		download.Dir = directory
		download.Env = environment
		download.Stdout = &output
		download.Stderr = &output
		if err := download.Run(); err != nil {
			studio.rememberRunOutput(output.String())
			return fmt.Errorf("download generated application dependencies: %w", err)
		}

		if !report.Report(35, "Building generated application...") {
			return ctx.Err()
		}
		output.Reset()
		command := exec.CommandContext(ctx, "go", "run", ".")
		command.Dir = directory
		command.Env = environment
		command.Stdout = &output
		command.Stderr = &output
		if err := command.Start(); err != nil {
			return fmt.Errorf("start preview: %w", err)
		}
		report.Post(func() { studio.status = "Preview running - close its window to finish" })
		err := command.Wait()
		studio.rememberRunOutput(output.String())
		if err != nil {
			return fmt.Errorf("preview exited: %w", err)
		}
		report.Post(func() { studio.status = "Preview closed successfully" })
		return nil
	}).OnProgress(func(progress rosaline.TaskProgress) {
		studio.status = progress.Message
	}).OnDone(func(err error) {
		if err == nil || errors.Is(err, context.Canceled) {
			return
		}
		studio.status = "Preview failed"
		studio.runMu.Lock()
		output := studio.runOutput
		studio.runMu.Unlock()
		message := err.Error()
		if output != "" {
			message += "\n\n" + output
		}
		rosaline.Error("Could not run generated application", message)
	})

	studio.rebuildTree()

	menu := rosaline.MenuBar(
		rosaline.Menu("File",
			rosaline.MenuItem("New Design", studio.newDesign).Shortcut("Primary+N"),
			rosaline.MenuItem("Open Design...", studio.openDesign).Shortcut("Primary+O"),
			rosaline.MenuSeparator(),
			rosaline.MenuItem("Save", func() { studio.save() }).Shortcut("Primary+S"),
			rosaline.MenuItem("Save As...", func() { studio.saveAs() }).Shortcut("Primary+Shift+S"),
			rosaline.MenuSeparator(),
			rosaline.MenuItem("Generate Go Project", func() { studio.generate() }).Shortcut("Primary+G"),
			rosaline.MenuItem("Build and Run", studio.runGenerated).Shortcut("F5"),
			rosaline.MenuSeparator(),
			rosaline.MenuItem("Quit", rosaline.Quit).Shortcut("Primary+Q"),
		),
		rosaline.Menu("Edit",
			rosaline.MenuItem("Undo", studio.undoChange).Shortcut("Primary+Z"),
			rosaline.MenuItem("Redo", studio.redoChange).Shortcut("Primary+Shift+Z"),
			rosaline.MenuSeparator(),
			rosaline.MenuItem("Move Up", func() { studio.moveSelected(-1) }).Shortcut("Alt+Up"),
			rosaline.MenuItem("Move Down", func() { studio.moveSelected(1) }).Shortcut("Alt+Down"),
			rosaline.MenuItem("Delete Widget", studio.deleteSelected).Shortcut("Delete"),
		),
		rosaline.Menu("Help",
			rosaline.MenuItem("Quick Help", studio.showHelp).Shortcut("F1"),
			rosaline.MenuItem("About", studio.showAbout),
		),
	)

	theme := rosaline.DefaultTheme
	theme.Background = rosaline.Hex("#f1e5ed")
	theme.Surface = rosaline.Hex("#fffafd")
	theme.Primary = rosaline.Hex("#bd3d79")
	theme.Border = rosaline.Hex("#d2aec2")

	rosaline.RunApp(rosaline.App{
		Title:          "Rosaline Studio - Untitled",
		Width:          1280,
		Height:         760,
		Padding:        12,
		Theme:          theme,
		Menu:           menu,
		Tasks:          []*rosaline.Task{studio.runTask},
		OnCloseRequest: func() bool { return studio.confirmChanges("closing") },
		Content: rosaline.Column(
			rosaline.Row(
				rosaline.Label("ROSALINE STUDIO").Bold().FontSize(19).Color(theme.Primary),
				rosaline.Label("Visual Go application designer").Color(theme.Muted),
				rosaline.Spring(),
				rosaline.LabelFunc(studio.documentName).Bold(),
			).Gap(12),
			rosaline.Row(
				rosaline.Size(studio.buildPalettePanel(), 220, 610),
				rosaline.Center(rosaline.Card(studio.canvas).Padding(5)),
				rosaline.Size(studio.buildInspectorPanel(), 330, 610),
			).Gap(10).Expand(),
			rosaline.Row(
				rosaline.LabelFunc(func() string { return studio.status }).Color(theme.Muted),
				rosaline.Spring(),
				rosaline.Label("F5 Run - Primary+G Generate - Primary+S Save").Color(theme.Muted),
			).Gap(8),
		).Gap(9).Expand(),
	})
}

func generatedCommandEnvironment() []string {
	environment := make([]string, 0, len(os.Environ())+2)
	for _, value := range os.Environ() {
		if strings.HasPrefix(value, "CGO_ENABLED=") || strings.HasPrefix(value, "GOWORK=") {
			continue
		}
		environment = append(environment, value)
	}
	return append(environment, "CGO_ENABLED=0", "GOWORK=off")
}

func (studio *studio) rememberRunOutput(output string) {
	studio.runMu.Lock()
	studio.runOutput = tailOutput(output, 2400)
	studio.runMu.Unlock()
}

func (studio *studio) buildPalettePanel() rosaline.Widget {
	return rosaline.Column(
		rosaline.Label("Widget Palette").Bold().Color(rosaline.Rose),
		rosaline.Label("Double-click or select and Add").Color(rosaline.DefaultTheme.Muted),
		studio.palette,
		rosaline.Button("Add Widget", func() {
			index, _, ok := studio.palette.Selected()
			if ok && index < len(paletteKinds) {
				studio.addWidget(paletteKinds[index])
			}
		}).Primary(),
		rosaline.Separator(),
		rosaline.Label("Hierarchy").Bold().Color(rosaline.Rose),
		studio.tree,
		rosaline.Row(
			rosaline.Button("Up", func() { studio.moveSelected(-1) }),
			rosaline.Button("Down", func() { studio.moveSelected(1) }),
			rosaline.Button("Delete", studio.deleteSelected),
		).Gap(5),
	).Gap(7).Expand()
}

func (studio *studio) buildInspectorPanel() rosaline.Widget {
	contentProperties := rosaline.Column(
		inspectorField("Text or placeholder", rosaline.TextBox(&studio.inspector.Text).Width(24)),
		inspectorField("State field name", rosaline.TextBox(&studio.inspector.Name).Width(24)),
		inspectorField("Button action name", rosaline.TextBox(&studio.inspector.Action).Width(24)),
		inspectorField("Comma-separated options", rosaline.TextBox(&studio.inspector.Options).Width(24)),
	).Gap(8)

	layoutProperties := rosaline.Column(
		rosaline.Grid(2,
			inspectorField("Gap", rosaline.TextBox(&studio.inspector.Gap).Width(8)),
			inspectorField("Padding", rosaline.TextBox(&studio.inspector.Padding).Width(8)),
			inspectorField("Grid columns", rosaline.TextBox(&studio.inspector.Columns).Width(8)),
			inspectorField("Width", rosaline.TextBox(&studio.inspector.Width).Width(8)),
			inspectorField("Height", rosaline.TextBox(&studio.inspector.Height).Width(8)),
			inspectorField("Minimum", rosaline.TextBox(&studio.inspector.Minimum).Width(8)),
			inspectorField("Maximum", rosaline.TextBox(&studio.inspector.Maximum).Width(8)),
			inspectorField("Step", rosaline.TextBox(&studio.inspector.Step).Width(8)),
		).Gap(8),
		rosaline.CheckBox("Use available space", &studio.inspector.Expand),
	).Gap(10)

	styleProperties := rosaline.Column(
		rosaline.CheckBox("Primary button", &studio.inspector.Primary),
		rosaline.CheckBox("Bold label", &studio.inspector.Bold),
		rosaline.CheckBox("Password input", &studio.inspector.Password),
		rosaline.CheckBox("Vertical slider or progress", &studio.inspector.Vertical),
	).Gap(8)

	widgetPanel := rosaline.Column(
		rosaline.LabelFunc(func() string { return previewDescription(studio.project.find(studio.selectedID)) }).Bold(),
		rosaline.Tabs(
			rosaline.Tab("Content", contentProperties),
			rosaline.Tab("Layout", layoutProperties),
			rosaline.Tab("Style", styleProperties),
		).Expand(),
		compactInspectorWidget(rosaline.Button("Apply Properties", studio.applyInspector).Primary()),
		rosaline.Label("Unsupported properties are ignored.").Color(rosaline.DefaultTheme.Muted),
	).Gap(8).Expand()

	projectPanel := rosaline.Column(
		rosaline.Label("Application Settings").Bold(),
		inspectorField("Window title", rosaline.TextBox(&studio.settings.Title).Width(24)),
		inspectorField("Go module path", rosaline.TextBox(&studio.settings.Module).Width(24)),
		rosaline.Grid(2,
			inspectorField("Width", rosaline.TextBox(&studio.settings.Width).Width(8)),
			inspectorField("Height", rosaline.TextBox(&studio.settings.Height).Width(8)),
			inspectorField("Window padding", rosaline.TextBox(&studio.settings.Padding).Width(8)),
		).Gap(8),
		rosaline.Label("Theme"),
		compactInspectorWidget(rosaline.ComboBox(&studio.settings.Theme, "Rosaline", "Lavender", "Midnight").Width(22)),
		compactInspectorWidget(rosaline.Button("Apply Application Settings", studio.applyProjectInspector).Primary()),
		rosaline.Separator(),
		rosaline.Label("Generated-file safety").Bold(),
		rosaline.Label("Studio regenerates only:"),
		rosaline.Label("ui_generated.go and state_generated.go"),
		rosaline.Label("Your other files are preserved.").Color(rosaline.DefaultTheme.Muted),
	).Gap(8)

	return rosaline.Tabs(
		rosaline.Tab("Widget", widgetPanel),
		rosaline.Tab("Application", projectPanel),
	).Expand()
}

func inspectorField(label string, field rosaline.Widget) rosaline.Widget {
	return rosaline.Column(
		rosaline.Label(label),
		compactInspectorWidget(field),
	).Gap(3)
}

func compactInspectorWidget(widget rosaline.Widget) rosaline.Widget {
	return rosaline.Align(widget, rosaline.AlignStart, rosaline.AlignStart)
}

func (studio *studio) selectNode(id string) {
	if studio.project.find(id) == nil {
		return
	}
	studio.selectedID = id
	studio.loadInspector()
	studio.syncTreeSelection()
	if studio.canvas != nil {
		studio.canvas.Redraw()
	}
}

func (studio *studio) addWidget(kind widgetKind) {
	before := designSnapshot(studio.project)
	node, err := studio.project.addNear(studio.selectedID, kind)
	if err != nil {
		studio.status = "Could not add widget: " + err.Error()
		return
	}
	studio.selectedID = node.ID
	studio.commitChange(before, "Added "+string(kind))
}

func (studio *studio) deleteSelected() {
	before := designSnapshot(studio.project)
	parent := studio.project.parentOf(studio.selectedID)
	if err := studio.project.remove(studio.selectedID); err != nil {
		studio.status = "Could not delete widget: " + err.Error()
		return
	}
	studio.selectedID = studio.project.Root.ID
	if parent != nil {
		studio.selectedID = parent.ID
	}
	studio.commitChange(before, "Deleted widget")
}

func (studio *studio) moveSelected(difference int) {
	before := designSnapshot(studio.project)
	if err := studio.project.moveBy(studio.selectedID, difference); err != nil {
		studio.status = "Could not move widget: " + err.Error()
		return
	}
	studio.commitChange(before, "Moved widget")
}

func (studio *studio) applyInspector() {
	node := studio.project.find(studio.selectedID)
	if node == nil {
		return
	}
	before := designSnapshot(studio.project)
	node.Text = studio.inspector.Text
	node.Name = studio.inspector.Name
	node.Action = studio.inspector.Action
	node.Options = splitOptions(studio.inspector.Options)
	node.Gap = parseInteger(studio.inspector.Gap, node.Gap)
	node.Padding = parseInteger(studio.inspector.Padding, node.Padding)
	node.Columns = max(1, parseInteger(studio.inspector.Columns, node.Columns))
	node.Width = max(0, parseInteger(studio.inspector.Width, node.Width))
	node.Height = max(0, parseInteger(studio.inspector.Height, node.Height))
	node.Minimum = parseNumber(studio.inspector.Minimum, node.Minimum)
	node.Maximum = parseNumber(studio.inspector.Maximum, node.Maximum)
	node.Step = max(0, parseNumber(studio.inspector.Step, node.Step))
	node.Expand = studio.inspector.Expand
	node.Primary = studio.inspector.Primary
	node.Bold = studio.inspector.Bold
	node.Password = studio.inspector.Password
	node.Vertical = studio.inspector.Vertical
	studio.commitChange(before, "Updated "+string(node.Kind))
}

func (studio *studio) applyProjectInspector() {
	if strings.TrimSpace(studio.settings.Module) == "" || strings.ContainsAny(studio.settings.Module, " \t\r\n") {
		studio.status = "The Go module path cannot be empty or contain spaces"
		return
	}
	before := designSnapshot(studio.project)
	studio.project.Title = defaultText(studio.settings.Title, "My Rosaline App")
	studio.project.Module = strings.TrimSpace(studio.settings.Module)
	studio.project.Width = max(320, parseInteger(studio.settings.Width, studio.project.Width))
	studio.project.Height = max(240, parseInteger(studio.settings.Height, studio.project.Height))
	studio.project.Padding = max(0, parseInteger(studio.settings.Padding, studio.project.Padding))
	studio.project.Theme = studio.settings.Theme
	studio.commitChange(before, "Updated application settings")
}

func (studio *studio) commitChange(before []byte, message string) {
	studio.undo = append(studio.undo, before)
	if len(studio.undo) > 100 {
		studio.undo = studio.undo[len(studio.undo)-100:]
	}
	studio.redo = nil
	studio.dirty = true
	studio.status = message
	studio.refreshDesign()
}

func (studio *studio) undoChange() {
	if len(studio.undo) == 0 {
		studio.status = "Nothing to undo"
		return
	}
	current := designSnapshot(studio.project)
	last := len(studio.undo) - 1
	restored, err := restoreSnapshot(studio.undo[last])
	if err != nil {
		studio.status = "Could not undo: " + err.Error()
		return
	}
	studio.undo = studio.undo[:last]
	studio.redo = append(studio.redo, current)
	studio.project = restored
	studio.ensureSelection()
	studio.dirty = true
	studio.status = "Undid last change"
	studio.loadProjectInspector()
	studio.refreshDesign()
}

func (studio *studio) redoChange() {
	if len(studio.redo) == 0 {
		studio.status = "Nothing to redo"
		return
	}
	current := designSnapshot(studio.project)
	last := len(studio.redo) - 1
	restored, err := restoreSnapshot(studio.redo[last])
	if err != nil {
		studio.status = "Could not redo: " + err.Error()
		return
	}
	studio.redo = studio.redo[:last]
	studio.undo = append(studio.undo, current)
	studio.project = restored
	studio.ensureSelection()
	studio.dirty = true
	studio.status = "Redid change"
	studio.loadProjectInspector()
	studio.refreshDesign()
}

func (studio *studio) refreshDesign() {
	studio.ensureSelection()
	studio.loadInspector()
	studio.rebuildTree()
	if studio.canvas != nil {
		studio.canvas.Redraw()
	}
	studio.updateWindowTitle()
}

func (studio *studio) ensureSelection() {
	if studio.project.find(studio.selectedID) == nil {
		studio.selectedID = studio.project.Root.ID
	}
}

func (studio *studio) rebuildTree() {
	if studio.tree == nil || studio.project == nil || studio.project.Root == nil {
		return
	}
	studio.treeByID = make(map[string]*rosaline.TreeNode)
	var build func(*designNode) *rosaline.TreeNode
	build = func(node *designNode) *rosaline.TreeNode {
		children := make([]*rosaline.TreeNode, 0, len(node.Children))
		for _, child := range node.Children {
			children = append(children, build(child))
		}
		label := string(node.Kind)
		if node.Text != "" && !node.Kind.container() {
			label += " - " + defaultText(node.Text, node.Name)
		} else if node.Name != "" {
			label += " - " + node.Name
		}
		result := rosaline.Node(label, children...).WithValue(node.ID).Expanded()
		studio.treeByID[node.ID] = result
		return result
	}
	studio.syncTree = true
	studio.tree.SetNodes(build(studio.project.Root))
	studio.tree.Select(studio.treeByID[studio.selectedID])
	studio.syncTree = false
}

func (studio *studio) syncTreeSelection() {
	if studio.tree == nil || studio.treeByID == nil {
		return
	}
	studio.syncTree = true
	studio.tree.Select(studio.treeByID[studio.selectedID])
	studio.syncTree = false
}

func (studio *studio) loadInspector() {
	node := studio.project.find(studio.selectedID)
	if node == nil {
		studio.inspector = inspectorState{}
		return
	}
	studio.inspector = inspectorState{
		Text: node.Text, Name: node.Name, Action: node.Action,
		Options: strings.Join(node.Options, ", "),
		Gap:     strconv.Itoa(node.Gap), Padding: strconv.Itoa(node.Padding),
		Columns: strconv.Itoa(max(1, node.Columns)), Width: strconv.Itoa(node.Width), Height: strconv.Itoa(node.Height),
		Minimum: numberLiteral(node.Minimum), Maximum: numberLiteral(node.Maximum), Step: numberLiteral(node.Step),
		Expand: node.Expand, Primary: node.Primary, Bold: node.Bold,
		Password: node.Password, Vertical: node.Vertical,
	}
}

func (studio *studio) loadProjectInspector() {
	studio.settings = projectInspectorState{
		Title: studio.project.Title, Module: studio.project.Module,
		Width: strconv.Itoa(studio.project.Width), Height: strconv.Itoa(studio.project.Height),
		Padding: strconv.Itoa(studio.project.Padding), Theme: studio.project.Theme,
	}
}

func (studio *studio) newDesign() {
	if !studio.confirmChanges("creating a new design") {
		return
	}
	studio.project = newProject()
	studio.path = ""
	studio.selectedID = studio.project.Root.ID
	studio.undo, studio.redo = nil, nil
	studio.dirty = false
	studio.status = "Created a new design"
	studio.loadProjectInspector()
	studio.refreshDesign()
}

func (studio *studio) openDesign() {
	if !studio.confirmChanges("opening another design") {
		return
	}
	path, ok := rosaline.OpenFileDialog(rosaline.FileDialogOptions{
		Title: "Open Rosaline Design",
		Filters: []rosaline.FileFilter{
			{Name: "Rosaline designs", Extensions: []string{".rosaline"}},
			{Name: "All files", Extensions: []string{"*"}},
		},
	})
	if !ok {
		return
	}
	project, err := loadDesign(path)
	if err != nil {
		rosaline.Error("Could not open design", err.Error())
		return
	}
	studio.project = project
	studio.path = path
	studio.selectedID = project.Root.ID
	studio.undo, studio.redo = nil, nil
	studio.dirty = false
	studio.status = "Opened " + filepath.Base(path)
	studio.loadProjectInspector()
	studio.refreshDesign()
}

func (studio *studio) save() bool {
	if studio.path == "" {
		return studio.saveAs()
	}
	if err := saveDesign(studio.path, studio.project); err != nil {
		rosaline.Error("Could not save design", err.Error())
		studio.status = "Save failed"
		return false
	}
	studio.dirty = false
	studio.status = "Saved " + filepath.Base(studio.path)
	studio.updateWindowTitle()
	return true
}

func (studio *studio) saveAs() bool {
	path, ok := rosaline.SaveFileDialog(rosaline.FileDialogOptions{
		Title:            "Save Rosaline Design",
		InitialFile:      "app.rosaline",
		DefaultExtension: ".rosaline",
		Filters: []rosaline.FileFilter{
			{Name: "Rosaline designs", Extensions: []string{".rosaline"}},
		},
	})
	if !ok {
		return false
	}
	oldPath := studio.path
	studio.path = path
	if !studio.save() {
		studio.path = oldPath
		return false
	}
	return true
}

func (studio *studio) confirmChanges(action string) bool {
	if !studio.dirty {
		return true
	}
	switch rosaline.AskSaveChanges("Unsaved design", "Save changes before "+action+"?") {
	case rosaline.SaveChanges:
		return studio.save()
	case rosaline.DiscardChanges:
		return true
	default:
		return false
	}
}

func (studio *studio) generate() bool {
	if !studio.save() {
		return false
	}
	directory := generatedApplicationDirectory(studio.path)
	report, err := generateProject(studio.project, directory)
	if err != nil {
		rosaline.Error("Could not generate project", err.Error())
		studio.status = "Generation failed"
		return false
	}
	studio.runMu.Lock()
	studio.runDirectory = directory
	studio.runMu.Unlock()
	studio.status = fmt.Sprintf("Generated %d files in %s; preserved %d developer files", len(report.Updated)+len(report.Created), filepath.Base(directory), len(report.Kept))
	return true
}

func (studio *studio) runGenerated() {
	if studio.runTask.Running() {
		studio.status = "A preview is already running"
		return
	}
	if !studio.save() {
		return
	}
	directory := generatedApplicationDirectory(studio.path)
	if generatedApplicationNeedsSetup(directory) {
		message := "This generated application has not been set up yet.\n\n" +
			"Studio will:\n" +
			"- create or update " + directory + "\n" +
			"- generate readable Rosaline Go files\n" +
			"- download Rosaline and its dependencies\n" +
			"- build and run the application\n\n" +
			"Set it up now?"
		if !rosaline.Confirm("Set up generated application", message) {
			studio.status = "Application setup canceled"
			return
		}
	}
	if !studio.generate() {
		return
	}
	studio.status = "Starting generated application..."
	studio.runTask.Start()
}

func (studio *studio) updateWindowTitle() {
	window := rosaline.MainWindow()
	if window == nil {
		return
	}
	mark := ""
	if studio.dirty {
		mark = " *"
	}
	window.SetTitle("Rosaline Studio - " + studio.documentName() + mark)
}

func (studio *studio) documentName() string {
	if studio.path == "" {
		return "Untitled"
	}
	return filepath.Base(studio.path)
}

func (studio *studio) showHelp() {
	rosaline.Message(
		"Rosaline Studio Quick Help",
		"1. Select a container in the hierarchy.\n2. Double-click a palette item to add it.\n3. Select widgets in the preview or hierarchy.\n4. Edit properties and choose Apply.\n5. Drag a preview widget onto a container or sibling to move it.\n6. Save, then press F5 to generate and run.\n\nStudio never overwrites handlers.go, main.go, go.mod, or README.md.",
	)
	studio.canvas.Focus()
}

func (studio *studio) showAbout() {
	rosaline.Message(
		"About Rosaline Studio",
		"Rosaline Studio v0.1.5\n\nA pure-Go visual application designer built with Rosaline.\n\nGenerated code remains normal, readable Rosaline Go.",
	)
	studio.canvas.Focus()
}

func splitOptions(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	seen := make(map[string]bool)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		result = append(result, part)
	}
	return result
}

func parseInteger(value string, fallback int) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}

func parseNumber(value string, fallback float64) float64 {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func tailOutput(value string, limit int) string {
	value = strings.TrimSpace(value)
	if len(value) <= limit || limit <= 0 {
		return value
	}
	return "...\n" + value[len(value)-limit:]
}
