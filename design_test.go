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
	project.Title = "Round trip"
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

func TestAddMoveAndRemoveNodes(t *testing.T) {
	project := newProject()
	row, err := project.addNear(project.Root.ID, kindRow)
	if err != nil {
		t.Fatalf("add row: %v", err)
	}
	button, err := project.addNear(row.ID, kindButton)
	if err != nil {
		t.Fatalf("add button: %v", err)
	}
	if err := project.moveTo(button.ID, project.Root.Children[0].ID); err != nil {
		t.Fatalf("move button: %v", err)
	}
	if project.parentOf(button.ID) != project.Root {
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
	first, err := project.addNear(project.Root.ID, kindButton)
	if err != nil {
		t.Fatal(err)
	}
	second, err := project.addNear(project.Root.ID, kindButton)
	if err != nil {
		t.Fatal(err)
	}
	if first.Component == second.Component || !validComponentName(first.Component) || !validComponentName(second.Component) {
		t.Fatalf("component names are not safe and unique: %q, %q", first.Component, second.Component)
	}
}

func TestDuplicateCopiesSubtreeWithIndependentIdentity(t *testing.T) {
	project := newProject()
	layout, err := project.addNear(project.Root.ID, kindColumn)
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
	if !validComponentName(project.Root.Component) || !validComponentName(project.Root.Children[0].Component) {
		t.Fatalf("old design did not receive component names: %#v", project.Root)
	}
}

func TestMoveRejectsCycles(t *testing.T) {
	project := newProject()
	row, err := project.addNear(project.Root.ID, kindRow)
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
	project.Root = &designNode{
		ID:   "root",
		Kind: kindCard,
		Children: []*designNode{
			{ID: "one", Kind: kindLabel},
			{ID: "two", Kind: kindLabel},
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
	if project.Title != "Welcome" || project.find("node-8") == nil {
		t.Fatalf("unexpected welcome example: %#v", project)
	}
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
	project.Root.Children[2].Events[eventClick] = "not-valid!"
	if err := project.validate(); err == nil {
		t.Fatal("invalid handler name was accepted")
	}
	project = newProject()
	project.Handlers["ContinueClick"] = "if {"
	if err := project.validate(); err == nil {
		t.Fatal("invalid event code was accepted")
	}
}
