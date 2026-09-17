// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"strings"
	"testing"
)

func TestPreviewContainsEveryDesignNode(t *testing.T) {
	project := newProject()
	boxes := layoutPreview(project.mainForm())
	if len(boxes) != 4 {
		t.Fatalf("want 4 preview boxes, got %d", len(boxes))
	}
	for _, box := range boxes {
		if box.Rect.Width <= 0 || box.Rect.Height <= 0 {
			t.Fatalf("invalid bounds for %s: %#v", box.Node.ID, box.Rect)
		}
	}
}

func TestPreviewUsesNaturalControlHeights(t *testing.T) {
	project := newProject()
	boxes := layoutPreview(project.mainForm())
	if boxes[1].Rect.Height > 32 {
		t.Fatalf("label was stretched to %.1f pixels", boxes[1].Rect.Height)
	}
	if boxes[2].Rect.Height > 42 {
		t.Fatalf("text box was stretched to %.1f pixels", boxes[2].Rect.Height)
	}
	if boxes[3].Rect.Height > 42 {
		t.Fatalf("button was stretched to %.1f pixels", boxes[3].Rect.Height)
	}
	usedBottom := boxes[3].Rect.Y + boxes[3].Rect.Height
	rootBottom := boxes[0].Rect.Y + boxes[0].Rect.Height
	if rootBottom-usedBottom < 80 {
		t.Fatal("natural controls unexpectedly consumed all vertical space")
	}
}

func TestExpandedControlAbsorbsRemainingHeight(t *testing.T) {
	project := newProject()
	project.mainForm().Root.Children = []*designNode{
		{ID: "label", Kind: kindLabel, Component: "NotesLabel", Text: "Notes"},
		{ID: "notes", Kind: kindTextArea, Component: "NotesTextArea", Name: "Notes", Expand: true},
		{ID: "save", Kind: kindButton, Component: "SaveButton", Text: "Save"},
	}
	boxes := layoutPreview(project.mainForm())
	if boxes[2].Rect.Height <= boxes[3].Rect.Height*2 {
		t.Fatalf("expanded text area did not receive remaining height: %.1f versus %.1f", boxes[2].Rect.Height, boxes[3].Rect.Height)
	}
}

func TestPreviewReflectsApplicationResolution(t *testing.T) {
	project := newProject()
	form := project.mainForm()
	form.Width, form.Height = 1200, 400
	wide := previewWindowBounds(form)
	form.Width, form.Height = 400, 1000
	tall := previewWindowBounds(form)
	if wide.Width <= wide.Height {
		t.Fatalf("wide resolution produced non-wide preview: %#v", wide)
	}
	if tall.Height <= tall.Width {
		t.Fatalf("tall resolution produced non-tall preview: %#v", tall)
	}
	if wide == tall {
		t.Fatal("changing resolution did not change the preview window")
	}
}

func TestPreviewContentBoundsReservesToolbar(t *testing.T) {
	form := newProject().mainForm()
	without := previewContentBounds(form)
	form.Toolbar = []*designToolbarItem{{ID: "tool-1", Kind: toolbarItemSeparator}}
	with := previewContentBounds(form)
	if with.Y <= without.Y || with.Height >= without.Height {
		t.Fatalf("toolbar did not reserve preview space: without=%#v with=%#v", without, with)
	}
}

func TestPreviewHitTestingPrefersDeepestWidget(t *testing.T) {
	project := newProject()
	boxes := layoutPreview(project.mainForm())
	child := boxes[1]
	x := child.Rect.X + child.Rect.Width/2
	y := child.Rect.Y + child.Rect.Height/2
	got := previewBoxAt(boxes, x, y)
	if got == nil || got.ID != child.Node.ID {
		t.Fatalf("want %s, got %#v", child.Node.ID, got)
	}
}

func TestPreviewShowsOnlyTheActiveTabPage(t *testing.T) {
	project := newProject()
	root := project.mainForm().Root
	root.Children = nil
	tabs, err := project.addNear(root.ID, kindTabs)
	if err != nil {
		t.Fatal(err)
	}
	first, second := tabs.Children[0], tabs.Children[1]
	firstControl, err := project.addNear(first.ID, kindLabel)
	if err != nil {
		t.Fatal(err)
	}
	secondControl, err := project.addNear(second.ID, kindButton)
	if err != nil {
		t.Fatal(err)
	}

	boxes := layoutPreview(project.mainForm(), map[string]string{tabs.ID: second.ID})
	seenFirst, seenSecond, activeSecond := false, false, false
	for _, box := range boxes {
		seenFirst = seenFirst || box.Node.ID == firstControl.ID
		seenSecond = seenSecond || box.Node.ID == secondControl.ID
		activeSecond = activeSecond || box.Node.ID == second.ID && box.Active
	}
	if seenFirst || !seenSecond || !activeSecond {
		t.Fatalf("active page preview is wrong: first=%v second=%v active=%v boxes=%#v", seenFirst, seenSecond, activeSecond, boxes)
	}
}

func TestPreviewLaysOutDataControls(t *testing.T) {
	project := newProject()
	root := project.mainForm().Root
	root.Children = nil
	for _, kind := range []widgetKind{kindList, kindTable, kindTree, kindRadioGroup} {
		if _, err := project.addNear(root.ID, kind); err != nil {
			t.Fatal(err)
		}
	}
	boxes := layoutPreview(project.mainForm())
	seen := make(map[widgetKind]bool)
	for _, box := range boxes {
		if box.Rect.Width <= 0 || box.Rect.Height <= 0 {
			t.Fatalf("invalid preview box: %#v", box)
		}
		seen[box.Node.Kind] = true
	}
	for _, kind := range []widgetKind{kindList, kindTable, kindTree, kindRadioGroup} {
		if !seen[kind] {
			t.Fatalf("preview is missing %s", kind)
		}
	}
}

func TestTailOutput(t *testing.T) {
	if got := tailOutput("  short output  ", 50); got != "short output" {
		t.Fatalf("unexpected short output: %q", got)
	}
	if got := tailOutput("0123456789", 4); got != "...\n6789" {
		t.Fatalf("unexpected truncated output: %q", got)
	}
}

func TestGeneratedCommandEnvironmentForcesStandalonePureGoBuild(t *testing.T) {
	t.Setenv("CGO_ENABLED", "1")
	t.Setenv("GOWORK", "/tmp/parent.work")
	environment := generatedCommandEnvironment()
	joined := "\n" + strings.Join(environment, "\n") + "\n"
	if !strings.Contains(joined, "\nCGO_ENABLED=0\n") {
		t.Fatal("generated command does not disable CGo")
	}
	if !strings.Contains(joined, "\nGOWORK=off\n") {
		t.Fatal("generated command does not disable an enclosing workspace")
	}
	if strings.Count(joined, "\nCGO_ENABLED=") != 1 || strings.Count(joined, "\nGOWORK=") != 1 {
		t.Fatalf("generated command retained conflicting environment entries: %q", joined)
	}
}
