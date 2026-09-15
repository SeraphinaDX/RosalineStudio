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
	Asset    string
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

	palette    *rosaline.ListWidget
	tree       *rosaline.TreeWidget
	canvas     *rosaline.CanvasWidget
	workspace  *rosaline.TabsWidget
	codeEditor *rosaline.TextAreaWidget
	eventList  *rosaline.ListWidget
	treeByID   map[string]*rosaline.TreeNode
	syncTree   bool
	dragID     string

	availableEvents []eventSpec
	selectedEvent   int
	eventHandler    string
	codeHandler     string
	codeBody        string
	previewPictures map[string]*rosaline.Picture

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
		project:         project,
		selectedID:      project.Root.ID,
		status:          "Ready - double-click a palette item to add it",
		selectedEvent:   -1,
		previewPictures: make(map[string]*rosaline.Picture),
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
	studio.tree.OnKeyDown(studio.handleDesignerKey)
	studio.tree.ContextMenu(studio.widgetContextEntries()...)

	studio.canvas = rosaline.Canvas(func(canvas *rosaline.DrawingCanvas) {
		drawPreview(canvas, studio.project, layoutPreview(studio.project), studio.selectedID, studio.previewPicture)
	}).Size(previewWidth, previewHeight).Focus()
	studio.canvas.OnMouseDown(func(event rosaline.MouseEvent) {
		if event.Button != rosaline.MouseLeft && event.Button != rosaline.MouseRight {
			return
		}
		if node := previewBoxAt(layoutPreview(studio.project), event.X, event.Y); node != nil {
			if event.Button == rosaline.MouseLeft {
				studio.dragID = node.ID
			}
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
	studio.canvas.OnDoubleClick(func(event rosaline.MouseEvent) {
		studio.dragID = ""
		node := previewBoxAt(layoutPreview(studio.project), event.X, event.Y)
		if node == nil {
			return
		}
		studio.selectNode(node.ID)
		studio.editDefaultEvent()
	})
	studio.canvas.OnKeyDown(studio.handleDesignerKey)
	studio.canvas.ContextMenu(studio.widgetContextEntries()...)

	studio.eventList = rosaline.List().Size(28, 6)
	studio.eventList.OnSelect(func(index int, _ string) {
		studio.selectEvent(index)
	}).OnActivate(func(index int, _ string) {
		studio.selectEvent(index)
		studio.editSelectedEvent()
	})
	studio.codeEditor = rosaline.TextArea(&studio.codeBody).Size(72, 28).Expand()
	studio.loadEvents()

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
			rosaline.MenuItem("Delete Selected Widget", studio.deleteSelected),
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
				rosaline.Label("Visual Go RAD environment").Color(theme.Muted),
				rosaline.Spring(),
				rosaline.LabelFunc(studio.documentName).Bold(),
			).Gap(12),
			rosaline.Row(
				rosaline.Size(studio.buildPalettePanel(), 220, 610),
				studio.buildWorkspace(),
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

func (studio *studio) buildWorkspace() rosaline.Widget {
	form := rosaline.Column(
		rosaline.Row(
			rosaline.LabelFunc(studio.selectedWidgetLabel).Bold(),
			rosaline.Spring(),
			rosaline.Button("Move Up", func() { studio.moveSelected(-1) }),
			rosaline.Button("Move Down", func() { studio.moveSelected(1) }),
			rosaline.Button("Delete Selected", studio.deleteSelected),
		).Gap(6),
		rosaline.Center(rosaline.Card(studio.canvas).Padding(5)),
	).Gap(7).Expand()
	code := rosaline.Column(
		rosaline.LabelFunc(studio.codeHeader).Bold().Color(rosaline.Rose),
		rosaline.Label("Write the body of this Go event method. The app and rosaline names are ready to use.").Color(rosaline.DefaultTheme.Muted),
		studio.codeEditor,
		rosaline.Row(
			rosaline.Button("Save Event Code", func() { studio.saveOpenHandler() }).Primary(),
			rosaline.Button("Back to Form", studio.showDesigner),
		).Gap(8),
	).Gap(8).Expand()
	studio.workspace = rosaline.Tabs(
		rosaline.Tab("Form", form),
		rosaline.Tab("Code", code),
	).Expand()
	return rosaline.Card(studio.workspace).Padding(5).Expand()
}

func (studio *studio) buildInspectorPanel() rosaline.Widget {
	contentProperties := rosaline.Column(
		inspectorField("Text or placeholder", rosaline.TextBox(&studio.inspector.Text).Width(24)),
		inspectorField("State field name", rosaline.TextBox(&studio.inspector.Name).Width(24)),
		inspectorField("Image asset", rosaline.LabelFunc(func() string { return defaultText(studio.inspector.Asset, "No image selected") }).Color(rosaline.DefaultTheme.Muted)),
		rosaline.Row(
			rosaline.Button("Choose Image...", studio.chooseImage),
			rosaline.Button("Clear", studio.clearImage),
		).Gap(6),
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
		rosaline.Label("ui, state, and events _generated.go files"),
		rosaline.Label("Your other files are preserved.").Color(rosaline.DefaultTheme.Muted),
	).Gap(8)

	eventPanel := rosaline.Column(
		rosaline.LabelFunc(func() string { return previewDescription(studio.project.find(studio.selectedID)) }).Bold(),
		rosaline.Label("Available events").Color(rosaline.DefaultTheme.Muted),
		studio.eventList,
		rosaline.LabelFunc(studio.selectedEventDescription).Color(rosaline.DefaultTheme.Muted),
		inspectorField("Handler method", rosaline.TextBox(&studio.eventHandler).Width(24)),
		rosaline.Row(
			rosaline.Button("Assign and Edit", studio.editSelectedEvent).Primary(),
			rosaline.Button("Clear", studio.clearSelectedEvent),
		).Gap(6),
		rosaline.Label("Double-click a form control to edit its default event.").Color(rosaline.DefaultTheme.Muted),
	).Gap(8)

	return rosaline.Tabs(
		rosaline.Tab("Properties", widgetPanel),
		rosaline.Tab("Events", eventPanel),
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
	studio.loadEvents()
	studio.syncTreeSelection()
	if studio.canvas != nil {
		studio.canvas.Redraw()
	}
}

func (studio *studio) loadEvents() {
	node := studio.project.find(studio.selectedID)
	studio.availableEvents = nil
	studio.selectedEvent = -1
	studio.eventHandler = ""
	if node != nil {
		studio.availableEvents = eventSpecsFor(node.Kind)
	}
	items := make([]string, 0, len(studio.availableEvents))
	for _, event := range studio.availableEvents {
		label := event.Name
		if handler := eventHandler(node, event.Name); handler != "" {
			label += "  -  " + handler
		}
		items = append(items, label)
	}
	if len(studio.availableEvents) != 0 {
		studio.selectedEvent = 0
		studio.eventHandler = eventHandler(node, studio.availableEvents[0].Name)
	}
	if studio.eventList != nil {
		studio.eventList.SetItems(items...)
		if len(items) != 0 {
			studio.eventList.Select(0)
		}
	}
}

func (studio *studio) selectEvent(index int) {
	if index < 0 || index >= len(studio.availableEvents) {
		studio.selectedEvent = -1
		studio.eventHandler = ""
		return
	}
	studio.selectedEvent = index
	node := studio.project.find(studio.selectedID)
	studio.eventHandler = eventHandler(node, studio.availableEvents[index].Name)
}

func (studio *studio) selectedEventDescription() string {
	if studio.selectedEvent < 0 || studio.selectedEvent >= len(studio.availableEvents) {
		return "This control has no editable events yet."
	}
	return studio.availableEvents[studio.selectedEvent].Description
}

func (studio *studio) editDefaultEvent() {
	if len(studio.availableEvents) == 0 {
		studio.status = "The selected control has no editable events"
		return
	}
	studio.selectedEvent = 0
	studio.eventHandler = eventHandler(studio.project.find(studio.selectedID), studio.availableEvents[0].Name)
	if studio.eventList != nil {
		studio.eventList.Select(0)
	}
	studio.editSelectedEvent()
}

func (studio *studio) editSelectedEvent() {
	if studio.selectedEvent < 0 || studio.selectedEvent >= len(studio.availableEvents) {
		studio.status = "Select an event first"
		return
	}
	node := studio.project.find(studio.selectedID)
	if node == nil {
		return
	}
	event := studio.availableEvents[studio.selectedEvent].Name
	handler := strings.TrimSpace(studio.eventHandler)
	if handler == "" {
		handler = defaultHandlerName(node, event)
	}
	if !validHandlerName(handler) {
		studio.status = "Handler names must be valid Go identifiers"
		return
	}
	if !studio.confirmOpenHandler(handler) {
		return
	}
	if handler == studio.codeHandler && eventHandler(node, event) == handler && studio.codeEditor.Modified() {
		if studio.workspace != nil {
			studio.workspace.Select(1)
		}
		studio.codeEditor.Focus()
		return
	}
	before := designSnapshot(studio.project)
	changed := eventHandler(node, event) != handler
	setEventHandler(node, event, handler)
	if studio.project.Handlers == nil {
		studio.project.Handlers = make(map[string]string)
	}
	if _, exists := studio.project.Handlers[handler]; !exists {
		studio.project.Handlers[handler] = defaultHandlerBody(node, event)
		changed = true
	}
	if changed {
		studio.commitChange(before, "Assigned "+event+" to "+handler)
	}
	studio.codeHandler = handler
	studio.codeBody = studio.project.Handlers[handler]
	studio.eventHandler = handler
	studio.codeEditor.SetText(studio.codeBody)
	studio.codeEditor.MarkSaved()
	if studio.workspace != nil {
		studio.workspace.Select(1)
	}
	studio.codeEditor.Focus()
	studio.status = "Editing " + handler
}

func (studio *studio) clearSelectedEvent() {
	if studio.selectedEvent < 0 || studio.selectedEvent >= len(studio.availableEvents) {
		studio.status = "Select an event first"
		return
	}
	node := studio.project.find(studio.selectedID)
	event := studio.availableEvents[studio.selectedEvent].Name
	if eventHandler(node, event) == "" {
		studio.status = event + " is already empty"
		return
	}
	before := designSnapshot(studio.project)
	setEventHandler(node, event, "")
	studio.commitChange(before, "Cleared "+event)
}

func (studio *studio) saveOpenHandler() bool {
	if studio.codeHandler == "" || studio.codeEditor == nil {
		return true
	}
	if err := validateHandler(studio.codeHandler, studio.codeBody); err != nil {
		rosaline.Error("Invalid event code", err.Error())
		studio.status = "Event code has a Go syntax error"
		return false
	}
	if studio.project.Handlers[studio.codeHandler] != studio.codeBody {
		before := designSnapshot(studio.project)
		studio.project.Handlers[studio.codeHandler] = studio.codeBody
		studio.commitChange(before, "Saved event "+studio.codeHandler)
	} else {
		studio.status = "Event code is already saved"
	}
	studio.codeEditor.MarkSaved()
	return true
}

func (studio *studio) confirmOpenHandler(next string) bool {
	if studio.codeEditor == nil || !studio.codeEditor.Modified() || studio.codeHandler == "" || studio.codeHandler == next {
		return true
	}
	switch rosaline.AskSaveChanges("Unsaved event code", "Save changes to "+studio.codeHandler+"?") {
	case rosaline.SaveChanges:
		return studio.saveOpenHandler()
	case rosaline.DiscardChanges:
		studio.codeBody = studio.project.Handlers[studio.codeHandler]
		studio.codeEditor.SetText(studio.codeBody)
		studio.codeEditor.MarkSaved()
		return true
	default:
		return false
	}
}

func (studio *studio) showDesigner() {
	if !studio.confirmOpenHandler("") {
		return
	}
	if studio.workspace != nil {
		studio.workspace.Select(0)
	}
	studio.canvas.Focus()
}

func (studio *studio) codeHeader() string {
	if studio.codeHandler == "" {
		return "No event handler selected"
	}
	return "func (app *Application) " + studio.codeHandler + "()"
}

func (studio *studio) chooseImage() {
	node := studio.project.find(studio.selectedID)
	if node == nil || node.Kind != kindImage {
		studio.status = "Select an Image control first"
		return
	}
	if studio.path == "" && !studio.saveAs() {
		return
	}
	path, ok := rosaline.OpenFileDialog(rosaline.FileDialogOptions{
		Title: "Choose Image",
		Filters: []rosaline.FileFilter{
			{Name: "Images", Extensions: []string{".png", ".jpg", ".jpeg", ".gif", ".bmp", ".tif", ".tiff", ".webp", ".avif"}},
			{Name: "All files", Extensions: []string{"*"}},
		},
	})
	if !ok {
		return
	}
	asset, picture, err := importImageAsset(path, studio.path)
	if err != nil {
		rosaline.Error("Could not import image", err.Error())
		studio.status = "Image import failed"
		return
	}
	before := designSnapshot(studio.project)
	node.Asset = asset
	if node.Width <= 0 || node.Height <= 0 {
		node.Width = min(480, max(80, picture.Width()))
		node.Height = min(320, max(60, picture.Height()))
	}
	studio.previewPictures[asset] = picture
	studio.commitChange(before, "Imported "+filepath.Base(path))
}

func (studio *studio) clearImage() {
	node := studio.project.find(studio.selectedID)
	if node == nil || node.Kind != kindImage {
		studio.status = "Select an Image control first"
		return
	}
	if node.Asset == "" {
		studio.status = "The image is already empty"
		return
	}
	before := designSnapshot(studio.project)
	delete(studio.previewPictures, node.Asset)
	node.Asset = ""
	studio.commitChange(before, "Cleared image")
}

func (studio *studio) previewPicture(asset string) *rosaline.Picture {
	if asset == "" || studio.path == "" {
		return nil
	}
	if picture := studio.previewPictures[asset]; picture != nil {
		return picture
	}
	picture, err := rosaline.LoadImage(filepath.Join(designAssetDirectory(studio.path), asset))
	if err != nil {
		return nil
	}
	studio.previewPictures[asset] = picture
	return picture
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
	studio.deleteSelectedWithConfirm(rosaline.Confirm)
}

func (studio *studio) deleteSelectedWithConfirm(confirm func(title, text string) bool) {
	node := studio.project.find(studio.selectedID)
	if node == nil {
		studio.status = "Select a widget before deleting"
		return
	}
	if node == studio.project.Root {
		studio.status = "The root layout cannot be deleted; select one of its children"
		return
	}
	contained := containedWidgetCount(node)
	if contained > 0 {
		message := fmt.Sprintf("Delete this %s and its %d contained widget", node.Kind, contained)
		if contained != 1 {
			message += "s"
		}
		message += "? You can undo this change."
		if confirm == nil || !confirm("Delete layout?", message) {
			studio.status = "Deletion cancelled"
			return
		}
	}
	before := designSnapshot(studio.project)
	nextSelection := studio.project.selectionAfterRemoval(studio.selectedID)
	deleted := string(node.Kind)
	if err := studio.project.remove(studio.selectedID); err != nil {
		studio.status = "Could not delete widget: " + err.Error()
		return
	}
	studio.selectedID = nextSelection
	studio.commitChange(before, "Deleted "+deleted+" - use Primary+Z to undo")
}

func (studio *studio) handleDesignerKey(event rosaline.KeyEvent) {
	if event.Is(rosaline.KeyDelete) && !event.Control && !event.Alt && !event.Primary {
		studio.deleteSelected()
	}
}

func (studio *studio) widgetContextEntries() []rosaline.MenuEntry {
	return []rosaline.MenuEntry{
		rosaline.MenuItem("Edit Default Event", studio.editDefaultEvent),
		rosaline.MenuSeparator(),
		rosaline.MenuItem("Move Up", func() { studio.moveSelected(-1) }),
		rosaline.MenuItem("Move Down", func() { studio.moveSelected(1) }),
		rosaline.MenuSeparator(),
		rosaline.MenuItem("Delete Selected Widget", studio.deleteSelected),
	}
}

func (studio *studio) selectedWidgetLabel() string {
	node := studio.project.find(studio.selectedID)
	if node == nil {
		return "No widget selected"
	}
	if node == studio.project.Root {
		return string(node.Kind) + " (root layout)"
	}
	label := string(node.Kind)
	if text := strings.TrimSpace(defaultText(node.Text, node.Name)); text != "" {
		label += " - " + text
	}
	return label
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
	studio.loadEvents()
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
		Text: node.Text, Name: node.Name, Asset: node.Asset,
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
	studio.codeHandler, studio.codeBody = "", ""
	studio.previewPictures = make(map[string]*rosaline.Picture)
	if studio.codeEditor != nil {
		studio.codeEditor.SetText("")
		studio.codeEditor.MarkSaved()
	}
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
	studio.codeHandler, studio.codeBody = "", ""
	studio.previewPictures = make(map[string]*rosaline.Picture)
	if studio.codeEditor != nil {
		studio.codeEditor.SetText("")
		studio.codeEditor.MarkSaved()
	}
	studio.status = "Opened " + filepath.Base(path)
	studio.loadProjectInspector()
	studio.refreshDesign()
}

func (studio *studio) save() bool {
	if studio.codeEditor != nil && studio.codeEditor.Modified() && !studio.saveOpenHandler() {
		return false
	}
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
	if err := copyDesignAssets(studio.project, oldPath, path); err != nil {
		rosaline.Error("Could not copy design assets", err.Error())
		return false
	}
	studio.path = path
	studio.previewPictures = make(map[string]*rosaline.Picture)
	if !studio.save() {
		studio.path = oldPath
		return false
	}
	return true
}

func (studio *studio) confirmChanges(action string) bool {
	if studio.codeEditor != nil && studio.codeEditor.Modified() && !studio.confirmOpenHandler("") {
		return false
	}
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
		"1. Select a container in the hierarchy.\n2. Double-click a palette item to add it.\n3. Select controls in the Form or hierarchy.\n4. Right-click or use Delete Selected to remove mistakes; Primary+Z restores them.\n5. Edit values in Properties and choose Apply.\n6. Use Events to assign a handler, or double-click a form control.\n7. Write the handler body in Code and save it.\n8. Add Image controls and choose picture files in Properties.\n9. Press F5 to generate and run.\n\nStudio never overwrites handlers.go, main.go, go.mod, or README.md.",
	)
	studio.canvas.Focus()
}

func (studio *studio) showAbout() {
	rosaline.Message(
		"About Rosaline Studio",
		"Rosaline Studio v0.2.0\n\nA pure-Go Lazarus-style RAD environment built with Rosaline.\n\nGenerated code remains normal, readable Rosaline Go.",
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
