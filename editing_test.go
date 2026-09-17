// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestSelectionAfterRemovalPrefersNearbyWidget(t *testing.T) {
	project := newProject()
	root := project.mainForm().Root
	if got := project.selectionAfterRemoval("node-2"); got != "node-3" {
		t.Fatalf("middle widget should select its next sibling, got %q", got)
	}
	if got := project.selectionAfterRemoval("node-3"); got != "node-2" {
		t.Fatalf("last widget should select its previous sibling, got %q", got)
	}
	root.Children = root.Children[:1]
	if got := project.selectionAfterRemoval("node-1"); got != root.ID {
		t.Fatalf("only child should select its parent, got %q", got)
	}
}

func TestDeleteSelectedPreservesHandlerAndCanBeUndone(t *testing.T) {
	studio := newStudio()
	studio.selectedID = "node-3"
	wantBody := studio.project.Handlers["ContinueClick"]

	studio.deleteSelectedWithConfirm(func(string, string) bool {
		t.Fatal("a leaf widget should not require confirmation")
		return false
	})
	if studio.project.find("node-3") != nil {
		t.Fatal("selected button was not deleted")
	}
	if studio.selectedID != "node-2" {
		t.Fatalf("expected nearby selection node-2, got %q", studio.selectedID)
	}
	if studio.project.Handlers["ContinueClick"] != wantBody {
		t.Fatal("deleting a widget destroyed its event method")
	}
	if !strings.Contains(studio.status, "Primary+Z") {
		t.Fatalf("delete status did not explain undo: %q", studio.status)
	}

	studio.undoChange()
	restored := studio.project.find("node-3")
	if restored == nil || restored.Events[eventClick] != "ContinueClick" {
		t.Fatalf("undo did not restore the button and event binding: %#v", restored)
	}
}

func TestDeletePopulatedLayoutRequiresConfirmation(t *testing.T) {
	studio := newStudio()
	layout, err := studio.project.addNear(studio.project.mainForm().Root.ID, kindColumn)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := studio.project.addNear(layout.ID, kindButton); err != nil {
		t.Fatal(err)
	}
	studio.selectedID = layout.ID

	called := false
	studio.deleteSelectedWithConfirm(func(title, message string) bool {
		called = true
		if title != "Delete layout?" || !strings.Contains(message, "1 contained widget") {
			t.Fatalf("unexpected confirmation: %q %q", title, message)
		}
		return false
	})
	if !called || studio.project.find(layout.ID) == nil {
		t.Fatal("cancelled deletion removed the layout")
	}

	studio.deleteSelectedWithConfirm(func(string, string) bool { return true })
	if studio.project.find(layout.ID) != nil {
		t.Fatal("confirmed deletion kept the layout")
	}
	studio.undoChange()
	if restored := studio.project.find(layout.ID); restored == nil || containedWidgetCount(restored) != 1 {
		t.Fatal("undo did not restore the complete layout subtree")
	}
}

func TestRootLayoutCannotBeDeleted(t *testing.T) {
	studio := newStudio()
	studio.deleteSelectedWithConfirm(func(string, string) bool {
		t.Fatal("root deletion should be rejected before confirmation")
		return true
	})
	if studio.project.mainForm().Root == nil || !strings.Contains(studio.status, "root layout") {
		t.Fatalf("root deletion was not safely rejected: %q", studio.status)
	}
}

func TestCopyPasteAndDuplicateAreUndoable(t *testing.T) {
	studio := newStudio()
	studio.selectedID = "node-3"
	studio.copySelected()
	studio.pasteClipboard()
	pastedID := studio.selectedID
	pasted := studio.project.find(pastedID)
	if pasted == nil || pasted.Component == "ContinueButton" || pasted.Events[eventClick] != "ContinueClick" {
		t.Fatalf("unexpected pasted widget: %#v", pasted)
	}
	studio.undoChange()
	if studio.project.find(pastedID) != nil {
		t.Fatal("undo did not remove the pasted widget")
	}

	studio.selectedID = "node-3"
	studio.duplicateSelected()
	duplicateID := studio.selectedID
	if studio.project.find(duplicateID) == nil {
		t.Fatal("duplicate command did not insert a widget")
	}
	studio.undoChange()
	if studio.project.find(duplicateID) != nil {
		t.Fatal("undo did not remove the duplicated widget")
	}
}

func TestPasteIntoTabsUsesTheActivePage(t *testing.T) {
	studio := newStudio()
	tabs, err := studio.project.addNear(studio.project.mainForm().Root.ID, kindTabs)
	if err != nil {
		t.Fatal(err)
	}
	active := tabs.Children[1]
	studio.tabPages[tabs.ID] = active.ID
	studio.selectedID = "node-1"
	studio.copySelected()
	studio.selectedID = tabs.ID
	studio.pasteClipboard()
	pasted := studio.project.find(studio.selectedID)
	if pasted == nil || studio.project.parentOf(pasted.ID) != active {
		t.Fatalf("pasted widget did not enter active page: %#v", pasted)
	}
}

func TestInspectorAppliesMultilineData(t *testing.T) {
	studio := newStudio()
	list, err := studio.project.addNear(studio.project.mainForm().Root.ID, kindList)
	if err != nil {
		t.Fatal(err)
	}
	studio.selectedID = list.ID
	studio.loadInspector()
	studio.inspector.Data = "Alpha\n\nBeta\n Gamma "
	studio.applyInspector()
	if !reflect.DeepEqual(list.Data, []string{"Alpha", "Beta", "Gamma"}) {
		t.Fatalf("unexpected list data: %v", list.Data)
	}
}
