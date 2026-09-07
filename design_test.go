// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"path/filepath"
	"reflect"
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
