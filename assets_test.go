// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSafeAssetNameIsStableAndEmbedFriendly(t *testing.T) {
	first := safeAssetName("My Rose Picture.PNG", []byte("same pixels"))
	second := safeAssetName("My Rose Picture.PNG", []byte("same pixels"))
	if first != second {
		t.Fatalf("asset name is not stable: %q and %q", first, second)
	}
	if strings.ContainsAny(first, " \\/") || !strings.HasPrefix(first, "my-rose-picture-") || !strings.HasSuffix(first, ".png") {
		t.Fatalf("asset name is not safe for embedding: %q", first)
	}
}

func TestCopyDesignAssetsFollowsSaveAs(t *testing.T) {
	directory := t.TempDir()
	oldPath := filepath.Join(directory, "old.rosaline")
	newPath := filepath.Join(directory, "new.rosaline")
	project := newProject()
	project.Root.Children = []*designNode{{ID: "picture", Kind: kindImage, Asset: "rose.png"}}
	if err := os.MkdirAll(designAssetDirectory(oldPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(designAssetDirectory(oldPath), "rose.png"), []byte("picture"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyDesignAssets(project, oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(designAssetDirectory(newPath), "rose.png")); err != nil || string(data) != "picture" {
		t.Fatalf("copied asset = %q, %v", data, err)
	}
}
