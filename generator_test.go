// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"go/parser"
	"go/token"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestGenerateProjectCreatesParseableGoAndPreservesDeveloperFiles(t *testing.T) {
	directory := t.TempDir()
	project := newProject()
	report, err := generateProject(project, directory)
	if err != nil {
		t.Fatalf("first generation: %v", err)
	}
	if len(report.Created) != 7 || len(report.Updated) != 0 {
		t.Fatalf("unexpected first report: %#v", report)
	}
	parseGeneratedGo(t, directory)
	goMod, err := os.ReadFile(filepath.Join(directory, "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goMod), "require "+rosalineModule+" "+rosalineVersion) {
		t.Fatalf("generated go.mod does not require %s: %s", rosalineVersion, goMod)
	}

	handwritten := []byte("package main\n\n// Mine stays mine.\n")
	handlers := filepath.Join(directory, "handlers.go")
	if err := os.WriteFile(handlers, handwritten, 0o644); err != nil {
		t.Fatal(err)
	}
	project.mainForm().Title = "A Changed Title"
	project.mainForm().Width = 901
	report, err = generateProject(project, directory)
	if err != nil {
		t.Fatalf("second generation: %v", err)
	}
	if len(report.Kept) != 4 {
		t.Fatalf("developer files were not preserved: %#v", report)
	}
	got, err := os.ReadFile(handlers)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(handwritten) {
		t.Fatal("handlers.go was overwritten")
	}
	ui, err := os.ReadFile(filepath.Join(directory, "ui_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ui), `Title:   "A Changed Title"`) || !strings.Contains(string(ui), "Width:   901") {
		t.Fatal("regenerated application settings were not written to ui_generated.go")
	}
}

func TestGeneratorCreatesEventMethods(t *testing.T) {
	project := newProject()
	project.mainForm().Root.Children = []*designNode{
		{ID: "name", Kind: kindTextBox, Name: "Name", Events: map[string]string{eventSubmit: "SubmitName"}},
	}
	project.Handlers = map[string]string{
		"SubmitName": `rosaline.Message("Hello", value)`,
	}
	directory := t.TempDir()
	if _, err := generateProject(project, directory); err != nil {
		t.Fatal(err)
	}
	ui, err := os.ReadFile(filepath.Join(directory, "ui_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	events, err := os.ReadFile(filepath.Join(directory, "events_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(ui), ".OnSubmit(func(value string) { app.SubmitName(value) })") {
		t.Fatalf("generated UI did not bind OnSubmit:\n%s", ui)
	}
	if !strings.Contains(string(events), "func (app *Application) SubmitName(value string)") || !strings.Contains(string(events), `rosaline.Message("Hello", value)`) {
		t.Fatalf("generated event method is incomplete:\n%s", events)
	}
}

func TestGeneratorCreatesNestedMenusAndClickHandlers(t *testing.T) {
	project := newProject()
	project.mainForm().Menus = []*designMenu{{
		ID: "menu-1", Kind: menuKindMenu, Text: "File",
		Children: []*designMenu{
			{ID: "menu-2", Kind: menuKindItem, Text: "Save", Handler: "SaveClick", Shortcut: "Primary+S"},
			{ID: "menu-3", Kind: menuKindSeparator},
			{ID: "menu-4", Kind: menuKindMenu, Text: "Recent", Children: []*designMenu{
				{ID: "menu-5", Kind: menuKindItem, Text: "Notes", Handler: "OpenNotes"},
			}},
		},
	}}
	project.Handlers["SaveClick"] = "// Save."
	project.Handlers["OpenNotes"] = "// Open notes."
	directory := t.TempDir()
	if _, err := generateProject(project, directory); err != nil {
		t.Fatal(err)
	}
	ui, err := os.ReadFile(filepath.Join(directory, "ui_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	events, err := os.ReadFile(filepath.Join(directory, "events_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`Menu:    buildMainFormMenu(app)`,
		`rosaline.Menu("Recent"`,
		`rosaline.MenuItem("Save", func() { app.SaveClick() }).Shortcut("Primary+S")`,
	} {
		if !strings.Contains(string(ui), want) {
			t.Fatalf("generated UI is missing %q:\n%s", want, ui)
		}
	}
	if !strings.Contains(string(events), "func (app *Application) SaveClick()") || !strings.Contains(string(events), "func (app *Application) OpenNotes()") {
		t.Fatalf("generated menu handlers are incomplete:\n%s", events)
	}
}

func TestGeneratorSharesActionsBetweenMenusAndToolbar(t *testing.T) {
	project := newProject()
	action := project.addAction()
	action.Name = "SaveAction"
	action.Text = "Save"
	action.Handler = "SaveDocument"
	action.Shortcut = "Primary+S"
	project.Handlers[action.Handler] = `app.Widgets().WelcomeLabel.SetText("Saved")`
	form := project.mainForm()
	form.Menus = []*designMenu{{
		ID: "menu-1", Kind: menuKindMenu, Text: "File",
		Children: []*designMenu{{ID: "menu-2", Kind: menuKindItem, Action: action.ID, Text: "Fallback"}},
	}}
	form.Toolbar = []*designToolbarItem{
		{ID: "tool-1", Kind: toolbarItemAction, Action: action.ID},
		{ID: "tool-2", Kind: toolbarItemSeparator},
	}

	directory := t.TempDir()
	if _, err := generateProject(project, directory); err != nil {
		t.Fatal(err)
	}
	uiData, err := os.ReadFile(filepath.Join(directory, "ui_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	eventsData, err := os.ReadFile(filepath.Join(directory, "events_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	ui := string(uiData)
	for _, want := range []string{
		`rosaline.MenuItem("Save", func() { app.SaveDocument() }).Shortcut("Primary+S")`,
		`rosaline.Button("Save", func() { app.SaveDocument() })`,
		`rosaline.Separator().Vertical()`,
	} {
		if !strings.Contains(ui, want) {
			t.Fatalf("generated shared action UI is missing %q:\n%s", want, ui)
		}
	}
	if strings.Count(string(eventsData), "func (app *Application) SaveDocument()") != 1 {
		t.Fatalf("shared action handler was not generated exactly once:\n%s", eventsData)
	}
	parseGeneratedGo(t, directory)
}

func TestGeneratorCreatesTabsPagesAndChangeEvent(t *testing.T) {
	project := newProject()
	root := project.mainForm().Root
	root.Children = nil
	tabs, err := project.addNear(root.ID, kindTabs)
	if err != nil {
		t.Fatal(err)
	}
	tabs.Component = "PreferencesTabs"
	tabs.Events = map[string]string{eventChange: "PreferencesPageChanged"}
	project.Handlers["PreferencesPageChanged"] = `app.Widgets().WelcomeLabel.SetText("Page changed")`
	label, err := project.addNear(tabs.Children[0].ID, kindLabel)
	if err != nil {
		t.Fatal(err)
	}
	label.Component = "WelcomeLabel"
	label.Text = "Welcome"

	directory := t.TempDir()
	if _, err := generateProject(project, directory); err != nil {
		t.Fatal(err)
	}
	uiData, err := os.ReadFile(filepath.Join(directory, "ui_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	stateData, err := os.ReadFile(filepath.Join(directory, "state_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	ui := string(uiData)
	state := string(stateData)
	for _, want := range []string{
		"rosaline.Tabs(",
		`rosaline.Tab("General", rosaline.Column(`,
		`rosaline.Tab("Advanced", rosaline.Column().Gap(10).Padding(12).Expand())`,
		`.OnChange(func(index int, title string) { app.PreferencesPageChanged(index, title) })`,
	} {
		if !strings.Contains(ui, want) {
			t.Fatalf("generated tabs are missing %q:\n%s", want, ui)
		}
	}
	if !strings.Contains(state, "PreferencesTabs *rosaline.TabsWidget") || !strings.Contains(state, "WelcomeLabel") || !strings.Contains(state, "*rosaline.LabelWidget") {
		t.Fatalf("generated widget references are incomplete:\n%s", state)
	}
	if strings.Contains(state, "GeneralPage *") || strings.Contains(state, "AdvancedPage *") {
		t.Fatalf("structural TabPage entries leaked into UIWidgets:\n%s", state)
	}
}

func TestGeneratorCreatesDataControlsAndEvents(t *testing.T) {
	project := newProject()
	root := project.mainForm().Root
	root.Children = nil

	list, err := project.addNear(root.ID, kindList)
	if err != nil {
		t.Fatal(err)
	}
	list.Component = "PaletteList"
	list.Data = []string{"Rose", "Lavender"}
	list.Events = map[string]string{eventSelect: "PaletteSelected", eventActivate: "PaletteActivated"}

	table, err := project.addNear(root.ID, kindTable)
	if err != nil {
		t.Fatal(err)
	}
	table.Component = "ProjectTable"
	table.Data = []string{"Name | Type", "Rosaline | Library", "Studio | Application"}
	table.Events = map[string]string{eventSelect: "ProjectSelected"}

	tree, err := project.addNear(root.ID, kindTree)
	if err != nil {
		t.Fatal(err)
	}
	tree.Component = "ProjectTree"
	tree.Data = []string{"Project/Forms/MainForm", "Project/Assets"}
	tree.Events = map[string]string{eventActivate: "TreeActivated", eventExpand: "TreeExpanded"}

	radio, err := project.addNear(root.ID, kindRadioGroup)
	if err != nil {
		t.Fatal(err)
	}
	radio.Component = "ViewModeRadioGroup"
	radio.Name = "ViewMode"
	radio.Data = []string{"Automatic = auto", "Details = details"}
	radio.Horizontal = true
	radio.Events = map[string]string{eventChange: "ViewModeChanged"}

	project.Handlers = map[string]string{
		"PaletteSelected":  "// Selected.",
		"PaletteActivated": "// Activated.",
		"ProjectSelected":  "// Selected.",
		"TreeActivated":    "// Activated.",
		"TreeExpanded":     "// Expanded.",
		"ViewModeChanged":  "// Changed.",
	}

	directory := t.TempDir()
	if _, err := generateProject(project, directory); err != nil {
		t.Fatal(err)
	}
	uiData, err := os.ReadFile(filepath.Join(directory, "ui_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	stateData, err := os.ReadFile(filepath.Join(directory, "state_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	eventsData, err := os.ReadFile(filepath.Join(directory, "events_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	ui := string(uiData)
	state := string(stateData)
	events := string(eventsData)
	for _, want := range []string{
		`rosaline.List("Rose", "Lavender").Expand().OnSelect(func(index int, value string) { app.PaletteSelected(index, value) }).OnActivate(func(index int, value string) { app.PaletteActivated(index, value) })`,
		`rosaline.Table("Name", "Type").SetRows([]string{"Rosaline", "Library"}, []string{"Studio", "Application"}).Expand().OnSelect(func(index int, row []string) { app.ProjectSelected(index, row) })`,
		`rosaline.Node("Forms"`,
		`.WithValue("Project/Forms").Expanded()`,
		`.OnActivate(func(node *rosaline.TreeNode) { app.TreeActivated(node) }).OnExpand(func(node *rosaline.TreeNode, expanded bool) { app.TreeExpanded(node, expanded) })`,
		`rosaline.RadioGroup(&app.State.ViewMode, rosaline.Choice("Automatic", "auto"), rosaline.Choice("Details", "details")).Horizontal().OnChange(func(value string) { app.ViewModeChanged(value) })`,
	} {
		if !strings.Contains(ui, want) {
			t.Fatalf("generated data controls are missing %q:\n%s", want, ui)
		}
	}
	for _, want := range []string{
		"PaletteList",
		"*rosaline.ListWidget",
		"ProjectTable",
		"*rosaline.TableWidget",
		"ProjectTree",
		"*rosaline.TreeWidget",
		"ViewModeRadioGroup",
		"*rosaline.RadioGroupWidget",
		"ViewMode string",
	} {
		if !strings.Contains(state, want) {
			t.Fatalf("generated component state is missing %q:\n%s", want, state)
		}
	}
	for _, want := range []string{
		"func (app *Application) PaletteSelected(index int, value string)",
		"func (app *Application) ProjectSelected(index int, row []string)",
		"func (app *Application) TreeExpanded(node *rosaline.TreeNode, expanded bool)",
		"func (app *Application) ViewModeChanged(value string)",
	} {
		if !strings.Contains(events, want) {
			t.Fatalf("generated typed event methods are missing %q:\n%s", want, events)
		}
	}
}

func TestGeneratorCreatesMultipleFormsAndLifecycleEvents(t *testing.T) {
	project := newProject()
	main := project.mainForm()
	settings := project.addForm()
	settings.Name = "SettingsForm"
	settings.Title = "Settings"
	main.Events = map[string]string{eventOpen: "MainFormOpen"}
	settings.Events = map[string]string{
		eventOpen:         "SettingsOpen",
		eventCloseRequest: "SettingsCloseRequest",
		eventClose:        "SettingsClose",
	}
	main.Root.Children[2].Events[eventClick] = "ShowSettings"
	project.Handlers = map[string]string{
		"MainFormOpen":         "// Main form mounted.",
		"SettingsOpen":         "// Settings mounted.",
		"SettingsCloseRequest": "return true",
		"SettingsClose":        "// Settings closed.",
		"ShowSettings":         "app.Windows().SettingsForm.Show()",
	}
	directory := t.TempDir()
	if _, err := generateProject(project, directory); err != nil {
		t.Fatal(err)
	}
	uiData, err := os.ReadFile(filepath.Join(directory, "ui_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	stateData, err := os.ReadFile(filepath.Join(directory, "state_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	eventsData, err := os.ReadFile(filepath.Join(directory, "events_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	ui := string(uiData)
	state := string(stateData)
	events := string(eventsData)
	for _, want := range []string{
		"generatedWindows.SettingsForm = rosaline.NewWindow",
		"Parent:         generatedWindows.MainForm",
		"app.SettingsOpen()",
		"OnCloseRequest: func() bool { return app.SettingsCloseRequest() }",
		"func (app *Application) Windows() *UIWindows",
	} {
		if !strings.Contains(ui, want) {
			t.Fatalf("generated UI is missing %q:\n%s", want, ui)
		}
	}
	if !strings.Contains(state, "SettingsForm *rosaline.Window") {
		t.Fatalf("generated window references are incomplete:\n%s", state)
	}
	if !strings.Contains(events, "func (app *Application) SettingsCloseRequest() bool") ||
		!strings.Contains(events, "app.Windows().SettingsForm.Show()") {
		t.Fatalf("generated lifecycle handlers are incomplete:\n%s", events)
	}
}

func TestGeneratorExposesNamedComponentsToEvents(t *testing.T) {
	project := newProject()
	project.mainForm().Root.Children[2].Component = "SaveButton"
	project.mainForm().Root.Children[0].Component = "StatusLabel"
	project.Handlers["ContinueClick"] = `app.Widgets().StatusLabel.SetText("Saved")
app.Widgets().SaveButton.SetEnabled(false)`
	directory := t.TempDir()
	if _, err := generateProject(project, directory); err != nil {
		t.Fatal(err)
	}
	state, err := os.ReadFile(filepath.Join(directory, "state_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	ui, err := os.ReadFile(filepath.Join(directory, "ui_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	events, err := os.ReadFile(filepath.Join(directory, "events_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(state), "StatusLabel") || !strings.Contains(string(state), "*rosaline.LabelWidget") ||
		!strings.Contains(string(state), "SaveButton") || !strings.Contains(string(state), "*rosaline.ButtonWidget") {
		t.Fatalf("generated component fields are incomplete:\n%s", state)
	}
	if !strings.Contains(string(ui), "rememberWidget(&generatedWidgets.StatusLabel") || !strings.Contains(string(ui), "func (app *Application) Widgets() *UIWidgets") {
		t.Fatalf("generated UI does not capture component references:\n%s", ui)
	}
	if !strings.Contains(string(events), `app.Widgets().StatusLabel.SetText("Saved")`) || !strings.Contains(string(events), "app.Widgets().SaveButton.SetEnabled(false)") {
		t.Fatalf("generated event cannot use components:\n%s", events)
	}
}

func TestGeneratorCopiesAndEmbedsImageAssets(t *testing.T) {
	parent := t.TempDir()
	directory := filepath.Join(parent, "gallery")
	assetDirectory := directory + ".assets"
	if err := os.MkdirAll(assetDirectory, 0o755); err != nil {
		t.Fatal(err)
	}
	assetName := "rose.png"
	file, err := os.Create(filepath.Join(assetDirectory, assetName))
	if err != nil {
		t.Fatal(err)
	}
	pixels := image.NewRGBA(image.Rect(0, 0, 8, 5))
	pixels.Set(3, 2, color.RGBA{R: 196, G: 63, B: 122, A: 255})
	if err := png.Encode(file, pixels); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	project := newProject()
	project.mainForm().Root.Children = []*designNode{{
		ID: "picture", Kind: kindImage, Text: "A rose", Asset: assetName,
		Width: 320, Height: 180, Events: map[string]string{eventClick: "PictureClick"},
	}}
	project.Handlers = map[string]string{"PictureClick": "// Picture clicked."}
	report, err := generateProject(project, directory)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(directory, "assets", assetName)); err != nil {
		t.Fatalf("generated asset is missing: %v", err)
	}
	ui, err := os.ReadFile(filepath.Join(directory, "ui_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(ui)
	if !strings.Contains(text, "//go:embed assets/*") || !strings.Contains(text, `generatedPicture("rose.png")`) || !strings.Contains(text, ".Fit(320, 180)") {
		t.Fatalf("generated image support is incomplete:\n%s", text)
	}
	found := false
	for _, name := range report.Created {
		found = found || name == "assets/rose.png"
	}
	if !found {
		t.Fatalf("asset was not reported: %#v", report)
	}
}

func TestGeneratedApplicationUsesSeparateDirectory(t *testing.T) {
	tests := []struct {
		designPath string
		want       string
	}{
		{filepath.Join("home", "studio", "greeting.rosaline"), filepath.Join("home", "studio", "greeting")},
		{filepath.Join("projects", "My App.rosaline"), filepath.Join("projects", "My App")},
		{filepath.Join("projects", ".rosaline"), filepath.Join("projects", "rosaline-app")},
	}
	for _, test := range tests {
		if got := generatedApplicationDirectory(test.designPath); got != test.want {
			t.Fatalf("generatedApplicationDirectory(%q): want %q, got %q", test.designPath, test.want, got)
		}
		if got := generatedApplicationDirectory(test.designPath); got == filepath.Dir(test.designPath) {
			t.Fatalf("generated application reused design directory for %q", test.designPath)
		}
	}
}

func TestGeneratedApplicationSetupDetection(t *testing.T) {
	directory := t.TempDir()
	if !generatedApplicationNeedsSetup(directory) {
		t.Fatal("empty directory should require setup")
	}
	for _, name := range []string{"go.mod", "go.sum", "ui_generated.go"} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte("test\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if generatedApplicationNeedsSetup(directory) {
		t.Fatal("prepared generated directory unexpectedly requires setup")
	}
	if err := os.Remove(filepath.Join(directory, "go.sum")); err != nil {
		t.Fatal(err)
	}
	if !generatedApplicationNeedsSetup(directory) {
		t.Fatal("missing dependency checksums should require setup")
	}
}

func TestGeneratorRefusesToReplaceForeignGeneratedFile(t *testing.T) {
	directory := t.TempDir()
	foreign := filepath.Join(directory, "ui_generated.go")
	want := []byte("package main\n\n// This file belongs to somebody else.\n")
	if err := os.WriteFile(foreign, want, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := generateProject(newProject(), directory); err == nil {
		t.Fatal("expected generation to reject a foreign ui_generated.go")
	}
	got, err := os.ReadFile(foreign)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("foreign file changed after rejected generation")
	}
}

func TestGeneratorHandlesEmptyGrid(t *testing.T) {
	project := newProject()
	project.mainForm().Root = &designNode{ID: "root", Kind: kindGrid, Component: "MainGrid", Columns: 3}
	directory := t.TempDir()
	if _, err := generateProject(project, directory); err != nil {
		t.Fatalf("generate empty grid: %v", err)
	}
	parseGeneratedGo(t, directory)
	data, err := os.ReadFile(filepath.Join(directory, "ui_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "rosaline.Grid(3)") {
		t.Fatalf("empty grid did not use a valid call:\n%s", data)
	}
}

func TestGeneratedStateNamesAreSafeAndUnique(t *testing.T) {
	project := newProject()
	project.mainForm().Root.Children = []*designNode{
		{ID: "a", Kind: kindTextBox, Name: "display name"},
		{ID: "b", Kind: kindTextArea, Name: "display-name"},
		{ID: "c", Kind: kindCheckBox, Name: "123"},
		{ID: "d", Kind: kindTextBox, Name: "DisplayName2"},
	}
	context := makeGenerationContext(project)
	want := []generatedField{
		{Name: "DisplayName", Type: "string"},
		{Name: "DisplayName2", Type: "string"},
		{Name: "Value", Type: "bool"},
		{Name: "DisplayName3", Type: "string"},
	}
	if len(context.fields) != len(want) {
		t.Fatalf("want %d fields, got %d", len(want), len(context.fields))
	}
	for index := range want {
		if context.fields[index] != want[index] {
			t.Fatalf("field %d: want %#v, got %#v", index, want[index], context.fields[index])
		}
	}
}

func TestGeneratedApplicationBuilds(t *testing.T) {
	rosalineSource := os.Getenv("ROSALINE_SOURCE")
	if rosalineSource == "" {
		t.Skip("set ROSALINE_SOURCE to run the generated-application build test")
	}
	absSource, err := filepath.Abs(rosalineSource)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	project := newProject()
	secondary := project.addForm()
	secondary.Name = "SettingsForm"
	secondary.Events = map[string]string{
		eventOpen:         "SettingsOpen",
		eventCloseRequest: "AllowSettingsClose",
		eventClose:        "SettingsClose",
	}
	project.Handlers["SettingsOpen"] = "// Opened."
	project.Handlers["AllowSettingsClose"] = "return true"
	project.Handlers["SettingsClose"] = "// Closed."
	project.mainForm().Menus = []*designMenu{{
		ID: "menu-1", Kind: menuKindMenu, Text: "File", Children: []*designMenu{
			{ID: "menu-2", Kind: menuKindItem, Text: "Run", Handler: "RunClick", Shortcut: "F5"},
			{ID: "menu-3", Kind: menuKindMenu, Text: "Recent", Children: []*designMenu{
				{ID: "menu-4", Kind: menuKindItem, Text: "Example", Handler: "RecentClick"},
			}},
		},
	}}
	secondary.Menus = []*designMenu{{ID: "menu-5", Kind: menuKindMenu, Text: "Help"}}
	project.Handlers["RunClick"] = "// Run."
	project.Handlers["RecentClick"] = "// Recent."
	action := project.addAction()
	action.Name, action.Text, action.Handler, action.Shortcut = "RefreshAction", "Refresh", "RefreshClick", "Primary+R"
	project.Handlers[action.Handler] = "// Refresh."
	project.mainForm().Menus[0].Children = append(project.mainForm().Menus[0].Children,
		&designMenu{ID: "menu-6", Kind: menuKindItem, Action: action.ID, Text: "Refresh"})
	project.mainForm().Toolbar = []*designToolbarItem{
		{ID: "tool-1", Kind: toolbarItemAction, Action: action.ID},
		{ID: "tool-2", Kind: toolbarItemSeparator},
	}
	timer, err := project.addComponent(project.mainForm(), componentTimer)
	if err != nil {
		t.Fatal(err)
	}
	project.Handlers[timer.Handler] = "// Timer fired."
	openDialog, err := project.addComponent(project.mainForm(), componentOpenDialog)
	if err != nil {
		t.Fatal(err)
	}
	openDialog.Filters = []string{"Go source | .go", "All files | *"}
	project.Handlers["RunClick"] = "_, _ = app.Components().OpenDialog.Execute()\napp.Components().Timer.Stop()"
	if _, err := project.addComponent(secondary, componentSaveDialog); err != nil {
		t.Fatal(err)
	}
	for _, kind := range paletteKinds {
		node, err := project.addNear(project.mainForm().Root.ID, kind)
		if err != nil {
			t.Fatalf("add %s: %v", kind, err)
		}
		var event, handler, body string
		switch kind {
		case kindTextBox, kindTextArea, kindComboBox, kindRadioGroup, kindSlider:
			event, handler, body = eventChange, exportedIdentifier(string(kind))+"Changed", "_ = value"
		case kindCheckBox:
			event, handler, body = eventChange, "CheckChanged", "_ = checked"
		case kindList:
			event, handler, body = eventSelect, "ListSelected", "_, _ = index, value"
		case kindTable:
			event, handler, body = eventSelect, "TableSelected", "_, _ = index, row"
		case kindTree:
			event, handler, body = eventExpand, "TreeExpanded", "_, _ = node, expanded"
		case kindTabs:
			event, handler, body = eventChange, "TabsChanged", "_, _ = index, title"
		}
		if event != "" {
			node.Events = map[string]string{event: handler}
			project.Handlers[handler] = body
		}
	}
	if _, err := generateProject(project, directory); err != nil {
		t.Fatal(err)
	}
	modPath := filepath.Join(directory, "go.mod")
	modData, err := os.ReadFile(modPath)
	if err != nil {
		t.Fatal(err)
	}
	modData = append(modData, []byte("\nreplace "+rosalineModule+" => "+filepath.ToSlash(absSource)+"\n")...)
	if err := os.WriteFile(modPath, modData, 0o644); err != nil {
		t.Fatal(err)
	}
	goCommand := filepath.Join(runtime.GOROOT(), "bin", "go")
	download := exec.Command(goCommand, "mod", "tidy")
	download.Dir = directory
	download.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off")
	if output, err := download.CombinedOutput(); err != nil {
		t.Fatalf("prepare generated application: %v\n%s", err, output)
	}
	command := exec.Command(goCommand, "test", ".")
	command.Dir = directory
	command.Env = append(os.Environ(), "CGO_ENABLED=0", "GOWORK=off")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("generated application did not build: %v\n%s", err, output)
	}
}

func parseGeneratedGo(t *testing.T, directory string) {
	t.Helper()
	entries, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	files := token.NewFileSet()
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) != ".go" {
			continue
		}
		path := filepath.Join(directory, entry.Name())
		if _, err := parser.ParseFile(files, path, nil, parser.AllErrors); err != nil {
			t.Fatalf("parse %s: %v", entry.Name(), err)
		}
	}
}
