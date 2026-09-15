// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"strings"
	"testing"
)

func TestSelectionAfterRemovalPrefersNearbyWidget(t *testing.T) {
	project := newProject()
	if got := project.selectionAfterRemoval("node-2"); got != "node-3" {
		t.Fatalf("middle widget should select its next sibling, got %q", got)
	}
	if got := project.selectionAfterRemoval("node-3"); got != "node-2" {
		t.Fatalf("last widget should select its previous sibling, got %q", got)
	}
	project.Root.Children = project.Root.Children[:1]
	if got := project.selectionAfterRemoval("node-1"); got != project.Root.ID {
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
	layout, err := studio.project.addNear(studio.project.Root.ID, kindColumn)
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
	if studio.project.Root == nil || !strings.Contains(studio.status, "root layout") {
		t.Fatalf("root deletion was not safely rejected: %q", studio.status)
	}
}
