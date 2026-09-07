// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import "testing"

func TestPreviewContainsEveryDesignNode(t *testing.T) {
	project := newProject()
	boxes := layoutPreview(project)
	if len(boxes) != 4 {
		t.Fatalf("want 4 preview boxes, got %d", len(boxes))
	}
	for _, box := range boxes {
		if box.Rect.Width <= 0 || box.Rect.Height <= 0 {
			t.Fatalf("invalid bounds for %s: %#v", box.Node.ID, box.Rect)
		}
	}
}

func TestPreviewHitTestingPrefersDeepestWidget(t *testing.T) {
	project := newProject()
	boxes := layoutPreview(project)
	child := boxes[1]
	x := child.Rect.X + child.Rect.Width/2
	y := child.Rect.Y + child.Rect.Height/2
	got := previewBoxAt(boxes, x, y)
	if got == nil || got.ID != child.Node.ID {
		t.Fatalf("want %s, got %#v", child.Node.ID, got)
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
