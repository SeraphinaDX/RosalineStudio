// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestDesignRoundTrip(t *testing.T) {
	project := newProject()
	project.mainForm().Title = "Round trip"
	project.mainForm().Menus = []*designMenu{{
		ID: "menu-1", Kind: menuKindMenu, Text: "File",
		Children: []*designMenu{{ID: "menu-2", Kind: menuKindItem, Text: "Save", Handler: "ContinueClick", Shortcut: "Primary+S"}},
	}}
	path := filepath.Join(t.TempDir(), "nested", "app.rosaline")
	if err := saveDesign(path, project); err != nil {
		t.Fatalf("saveDesign: %v", err)
	}
	loaded, err := loadDesign(path)
	if err != nil {
		t.Fatalf("loadDesign: %v", err)
	}
	if !reflect.DeepEqual(project, loaded) {
		t.Fatalf("loaded design differs\nwant: %#v\n got: %#v", project, loaded)
	}
}

func TestMenuDesignerModelSupportsNestedMenus(t *testing.T) {
	project := newProject()
	form := project.mainForm()
	file := project.addTopMenu(form)
	file.Text = "File"
	recent, err := project.addMenuChild(form, file.ID, menuKindMenu)
	if err != nil {
		t.Fatal(err)
	}
	recent.Text = "Open Recent"
	item, err := project.addMenuChild(form, recent.ID, menuKindItem)
	if err != nil {
		t.Fatal(err)
	}
	item.Text, item.Handler, item.Shortcut = "Notes", "OpenNotes", "Primary+O"
	project.Handlers["OpenNotes"] = "// Open the file."
	if err := project.validate(); err != nil {
		t.Fatalf("nested menu is invalid: %v", err)
	}
	copy, err := project.duplicateForm(form.ID)
	if err != nil {
		t.Fatal(err)
	}
	if copy.Menus[0].ID == file.ID || copy.Menus[0].Children[0].ID == recent.ID || copy.Menus[0].Children[0].Children[0].ID == item.ID {
		t.Fatal("duplicated menus retained source IDs")
	}
	if err := project.validate(); err != nil {
		t.Fatalf("duplicated menus are invalid: %v", err)
	}
}

func TestAddMoveAndRemoveNodes(t *testing.T) {
	project := newProject()
	root := project.mainForm().Root
	row, err := project.addNear(root.ID, kindRow)
	if err != nil {
		t.Fatalf("add row: %v", err)
	}
	button, err := project.addNear(row.ID, kindButton)
	if err != nil {
		t.Fatalf("add button: %v", err)
	}
	if err := project.moveTo(button.ID, root.Children[0].ID); err != nil {
		t.Fatalf("move button: %v", err)
	}
	if project.parentOf(button.ID) != root {
		t.Fatal("button was not moved to the root")
	}
	if err := project.remove(button.ID); err != nil {
		t.Fatalf("remove button: %v", err)
	}
	if project.find(button.ID) != nil {
		t.Fatal("removed button is still present")
	}
}

func TestAddedComponentsReceiveUniqueNames(t *testing.T) {
	project := newProject()
	root := project.mainForm().Root
	first, err := project.addNear(root.ID, kindButton)
	if err != nil {
		t.Fatal(err)
	}
	second, err := project.addNear(root.ID, kindButton)
	if err != nil {
		t.Fatal(err)
	}
	if first.Component == second.Component || !validComponentName(first.Component) || !validComponentName(second.Component) {
		t.Fatalf("component names are not safe and unique: %q, %q", first.Component, second.Component)
	}
}

func TestDuplicateCopiesSubtreeWithIndependentIdentity(t *testing.T) {
	project := newProject()
	layout, err := project.addNear(project.mainForm().Root.ID, kindColumn)
	if err != nil {
		t.Fatal(err)
	}
	input, err := project.addNear(layout.ID, kindTextBox)
	if err != nil {
		t.Fatal(err)
	}
	input.Events = map[string]string{eventSubmit: "SubmitText"}
	project.Handlers["SubmitText"] = "// Submitted."

	duplicate, err := project.duplicate(layout.ID)
	if err != nil {
		t.Fatal(err)
	}
	if duplicate.ID == layout.ID || duplicate.Component == layout.Component {
		t.Fatalf("duplicate retained layout identity: %#v", duplicate)
	}
	if len(duplicate.Children) != 1 {
		t.Fatalf("duplicate lost its child: %#v", duplicate)
	}
	child := duplicate.Children[0]
	if child.ID == input.ID || child.Component == input.Component || child.Name == input.Name {
		t.Fatalf("duplicate child identity was not made unique: %#v", child)
	}
	if child.Events[eventSubmit] != "SubmitText" {
		t.Fatal("duplicate should retain its event-handler assignment")
	}
	child.Text = "Changed copy"
	if input.Text == child.Text {
		t.Fatal("editing the duplicate changed the original")
	}
	if err := project.validate(); err != nil {
		t.Fatalf("duplicated design is invalid: %v", err)
	}
}

func TestVersionTwoDesignWithoutComponentsIsUpgraded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old-v2.rosaline")
	data := []byte(`{"version":2,"module":"example.com/old","title":"Old","width":720,"height":520,"padding":16,"theme":"Rosaline","root":{"id":"root","kind":"Column","children":[{"id":"node-1","kind":"Button","text":"Save"}]}}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	project, err := loadDesign(path)
	if err != nil {
		t.Fatal(err)
	}
	root := project.mainForm().Root
	if project.Version != designVersion || project.mainForm().Name != "MainForm" {
		t.Fatalf("old design did not migrate to a main form: %#v", project)
	}
	if !validComponentName(root.Component) || !validComponentName(root.Children[0].Component) {
		t.Fatalf("old design did not receive component names: %#v", root)
	}
}

func TestVersionThreeDesignIsUpgraded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old-v3.rosaline")
	data := []byte(`{"version":3,"module":"example.com/old","forms":[{"id":"form-1","name":"MainForm","title":"Old","width":720,"height":520,"padding":16,"theme":"Rosaline","root":{"id":"root","kind":"Column","component":"MainLayout"}}]}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	project, err := loadDesign(path)
	if err != nil {
		t.Fatal(err)
	}
	if project.Version != designVersion || len(project.mainForm().Menus) != 0 {
		t.Fatalf("version-3 design was not migrated: %#v", project)
	}
}

func TestVersionFourDesignIsUpgraded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old-v4.rosaline")
	data := []byte(`{"version":4,"module":"example.com/old","forms":[{"id":"form-1","name":"MainForm","title":"Old","width":720,"height":520,"padding":16,"theme":"Rosaline","root":{"id":"root","kind":"Column","component":"MainLayout"}}]}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	project, err := loadDesign(path)
	if err != nil {
		t.Fatal(err)
	}
	if project.Version != designVersion {
		t.Fatalf("version-4 design was not migrated: %#v", project)
	}
}

func TestTabsCreateAndManagePages(t *testing.T) {
	project := newProject()
	tabs, err := project.addNear(project.mainForm().Root.ID, kindTabs)
	if err != nil {
		t.Fatal(err)
	}
	if len(tabs.Children) != 2 || tabs.Children[0].Kind != kindTabPage || tabs.Children[0].Text != "General" || tabs.Children[1].Text != "Advanced" {
		t.Fatalf("new Tabs did not receive friendly starter pages: %#v", tabs.Children)
	}
	button, err := project.addNear(tabs.ID, kindButton)
	if err != nil {
		t.Fatal(err)
	}
	if project.parentOf(button.ID) != tabs.Children[0] {
		t.Fatal("adding a control to Tabs did not target its first page")
	}
	page, err := project.addTabPage(tabs, "Notifications")
	if err != nil {
		t.Fatal(err)
	}
	if page.Text != "Notifications" || project.parentOf(page.ID) != tabs {
		t.Fatalf("unexpected added page: %#v", page)
	}
	copy, err := project.duplicate(page.ID)
	if err != nil {
		t.Fatal(err)
	}
	if copy.Kind != kindTabPage || copy.ID == page.ID || copy.Component == page.Component {
		t.Fatalf("tab page copy did not receive independent identity: %#v", copy)
	}
	if err := project.validate(); err != nil {
		t.Fatalf("tab design is invalid: %v", err)
	}
}

func TestTabsEnforcePageContainmentAndKeepOnePage(t *testing.T) {
	project := newProject()
	tabs, err := project.addNear(project.mainForm().Root.ID, kindTabs)
	if err != nil {
		t.Fatal(err)
	}
	for len(tabs.Children) > 1 {
		if err := project.remove(tabs.Children[len(tabs.Children)-1].ID); err != nil {
			t.Fatal(err)
		}
	}
	if err := project.remove(tabs.Children[0].ID); err == nil || !strings.Contains(err.Error(), "at least one page") {
		t.Fatalf("expected last-page deletion to fail, got %v", err)
	}

	broken := newProject()
	broken.mainForm().Root.Children = []*designNode{{
		ID: "tabs", Kind: kindTabs, Component: "BrokenTabs",
		Children: []*designNode{{ID: "button", Kind: kindButton, Component: "WrongButton", Text: "Wrong"}},
	}}
	if err := broken.validate(); err == nil || !strings.Contains(err.Error(), "only TabPage") {
		t.Fatalf("expected direct control in Tabs to fail validation, got %v", err)
	}

	broken = newProject()
	broken.mainForm().Root.Children = []*designNode{{ID: "page", Kind: kindTabPage, Component: "LoosePage", Text: "Loose"}}
	if err := broken.validate(); err == nil || !strings.Contains(err.Error(), "direct children of Tabs") {
		t.Fatalf("expected loose TabPage to fail validation, got %v", err)
	}
}

func TestMoveRejectsCycles(t *testing.T) {
	project := newProject()
	row, err := project.addNear(project.mainForm().Root.ID, kindRow)
	if err != nil {
		t.Fatal(err)
	}
	column, err := project.addNear(row.ID, kindColumn)
	if err != nil {
		t.Fatal(err)
	}
	if err := project.moveTo(row.ID, column.ID); err == nil {
		t.Fatal("expected moving a parent into its child to fail")
	}
}

func TestSingleChildContainersAreValidated(t *testing.T) {
	project := newProject()
	project.mainForm().Root = &designNode{
		ID: "root", Kind: kindCard, Component: "RootCard",
		Children: []*designNode{
			{ID: "one", Kind: kindLabel, Component: "OneLabel"},
			{ID: "two", Kind: kindLabel, Component: "TwoLabel"},
		},
	}
	if err := project.validate(); err == nil {
		t.Fatal("expected card with two children to fail validation")
	}
}

func TestSplitOptionsTrimsAndDeduplicates(t *testing.T) {
	want := []string{"Rose", "Violet", "Pearl"}
	got := splitOptions(" Rose, Violet, Rose, , Pearl ")
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestWelcomeExampleLoads(t *testing.T) {
	project, err := loadDesign(filepath.Join("examples", "welcome.rosaline"))
	if err != nil {
		t.Fatalf("load welcome example: %v", err)
	}
	if project.mainForm().Title != "Welcome" || project.find("node-8") == nil {
		t.Fatalf("unexpected welcome example: %#v", project)
	}
}

func TestNotepadExampleLoadsWithMenus(t *testing.T) {
	project, err := loadDesign(filepath.Join("examples", "notepad.rosaline"))
	if err != nil {
		t.Fatalf("load notepad example: %v", err)
	}
	if len(project.mainForm().Menus) != 3 || findDesignMenu(project.mainForm().Menus, "menu-15") == nil {
		t.Fatalf("notepad menus are incomplete: %#v", project.mainForm().Menus)
	}
}

func TestPreferencesExampleLoadsWithTabs(t *testing.T) {
	project, err := loadDesign(filepath.Join("examples", "preferences.rosaline"))
	if err != nil {
		t.Fatalf("load preferences example: %v", err)
	}
	tabs := project.find("node-3")
	if tabs == nil || tabs.Kind != kindTabs || len(tabs.Children) != 3 {
		t.Fatalf("preferences tabs are incomplete: %#v", tabs)
	}
	directory := t.TempDir()
	if _, err := generateProject(project, directory); err != nil {
		t.Fatalf("generate preferences example: %v", err)
	}
	parseGeneratedGo(t, directory)
}

func TestNewNestedLayoutsUseNaturalSize(t *testing.T) {
	for _, kind := range []widgetKind{kindColumn, kindRow, kindGrid, kindStack} {
		if node := defaultNode(kind, "test"); node.Expand {
			t.Fatalf("new %s unexpectedly expands by default", kind)
		}
	}
}

func TestVersionOneDesignsAreRejectedClearly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.rosaline")
	data := []byte(`{"version":1,"module":"example.com/old","title":"Old","width":720,"height":520,"padding":16,"theme":"Rosaline","root":{"id":"root","kind":"Column"}}`)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := loadDesign(path); err == nil || !strings.Contains(err.Error(), "unsupported design version 1") {
		t.Fatalf("want an unsupported-version error, got %v", err)
	}
}

func TestEventValidation(t *testing.T) {
	project := newProject()
	project.mainForm().Root.Children[2].Events[eventClick] = "not-valid!"
	if err := project.validate(); err == nil {
		t.Fatal("invalid handler name was accepted")
	}
	project = newProject()
	project.Handlers["ContinueClick"] = "if {"
	if err := project.validate(); err == nil {
		t.Fatal("invalid event code was accepted")
	}
}

func TestFormsCanBeAddedDuplicatedAndRemoved(t *testing.T) {
	project := newProject()
	settings := project.addForm()
	settings.Name = "SettingsForm"
	settings.Title = "Settings"
	if _, err := project.addNear(settings.Root.ID, kindButton); err != nil {
		t.Fatal(err)
	}
	pasted, err := project.insertCopy(settings.Root.ID, project.find("node-3"))
	if err != nil {
		t.Fatal(err)
	}
	if project.formContainingNode(pasted.ID) != settings || pasted.Component == "ContinueButton" {
		t.Fatalf("cross-form paste did not create a unique widget on the destination form: %#v", pasted)
	}
	copy, err := project.duplicateForm(settings.ID)
	if err != nil {
		t.Fatal(err)
	}
	if copy.ID == settings.ID || copy.Name == settings.Name || copy.Root.ID == settings.Root.ID {
		t.Fatalf("duplicated form retained an identity: %#v", copy)
	}
	if project.formContainingNode(copy.Root.ID) != copy {
		t.Fatal("duplicated form widgets are not associated with the copy")
	}
	if err := project.removeForm(settings.ID); err != nil {
		t.Fatal(err)
	}
	if project.form(settings.ID) != nil || len(project.Forms) != 2 {
		t.Fatalf("form was not removed: %#v", project.Forms)
	}
	if err := project.removeForm(project.mainForm().ID); err == nil {
		t.Fatal("main form deletion was accepted")
	}
	if err := project.validate(); err != nil {
		t.Fatalf("multi-form design is invalid: %v", err)
	}
}

func TestWidgetsCannotMoveBetweenForms(t *testing.T) {
	project := newProject()
	form := project.addForm()
	if err := project.moveTo("node-3", form.Root.ID); err == nil || !strings.Contains(err.Error(), "within one form") {
		t.Fatalf("cross-form move should fail clearly, got %v", err)
	}
}
