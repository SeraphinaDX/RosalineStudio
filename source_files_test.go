// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestScanProjectFilesClassifiesStudioProject(t *testing.T) {
	root := t.TempDir()
	writeTestProjectFile(t, root, "main.go", "package main\n")
	writeTestProjectFile(t, root, "events_generated.go", "package main\n")
	writeTestProjectFile(t, root, "go.mod", "module example.test/app\n")
	writeTestProjectFile(t, root, "assets/logo.png", "not really a png")
	writeTestProjectFile(t, root, ".git/config", "ignored")

	files, err := scanProjectFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	kinds := make(map[string]projectFileKind)
	for _, file := range files {
		kinds[file.Path] = file.Kind
	}
	want := map[string]projectFileKind{
		"main.go":             projectFileSource,
		"events_generated.go": projectFileGenerated,
		"go.mod":              projectFileProject,
		"assets/logo.png":     projectFileAsset,
	}
	for path, kind := range want {
		if kinds[path] != kind {
			t.Errorf("%s kind = %d, want %d", path, kinds[path], kind)
		}
	}
	if _, found := kinds[".git/config"]; found {
		t.Fatal("hidden directory should not be scanned")
	}
}

func TestValidateGoFilename(t *testing.T) {
	for _, name := range []string{"notes.go", "window_2.go", "A.go"} {
		if err := validateGoFilename(name); err != nil {
			t.Errorf("validateGoFilename(%q): %v", name, err)
		}
	}
	for _, name := range []string{"", "notes.txt", "folder/notes.go", "ui_generated.go", "spaces here.go"} {
		if err := validateGoFilename(name); err == nil {
			t.Errorf("validateGoFilename(%q) unexpectedly succeeded", name)
		}
	}
}

func TestSafeProjectPathRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	path, err := safeProjectPath(root, "handlers.go")
	if err != nil || path != filepath.Join(root, "handlers.go") {
		t.Fatalf("safe path = %q, %v", path, err)
	}
	for _, value := range []string{"../secret.go", "folder/../../secret.go"} {
		if _, err := safeProjectPath(root, value); err == nil {
			t.Errorf("safeProjectPath(%q) unexpectedly succeeded", value)
		}
	}
}

func TestParseBuildDiagnostics(t *testing.T) {
	root := t.TempDir()
	abs := filepath.Join(root, "handlers.go")
	separator := "/"
	if runtime.GOOS == "windows" {
		separator = `\`
	}
	output := strings.Join([]string{
		"# example.test/app",
		"./main.go:12:5: undefined: welcome",
		abs + ":7: missing return",
		"./main.go:12:5: undefined: welcome",
	}, "\n")
	diagnostics := parseBuildDiagnostics(output, root)
	if len(diagnostics) != 2 {
		t.Fatalf("got %d diagnostics: %#v", len(diagnostics), diagnostics)
	}
	if diagnostics[0].File != "main.go" || diagnostics[0].Line != 12 || diagnostics[0].Column != 5 {
		t.Errorf("first diagnostic = %#v", diagnostics[0])
	}
	if diagnostics[1].File != "handlers.go" || diagnostics[1].Line != 7 {
		t.Errorf("second diagnostic = %#v (separator %q)", diagnostics[1], separator)
	}
}

func TestSourceDocumentDirtyOnlyForEditableFiles(t *testing.T) {
	editable := &sourceDocument{File: projectFile{Path: "handlers.go", Kind: projectFileSource}, Text: "new", Saved: "old"}
	if !editable.dirty() {
		t.Fatal("editable changed document should be dirty")
	}
	generated := &sourceDocument{File: projectFile{Path: "ui_generated.go", Kind: projectFileGenerated}, Text: "new", Saved: "old"}
	if generated.dirty() {
		t.Fatal("generated document must never become saveable")
	}
}

func writeTestProjectFile(t *testing.T, root, relative, contents string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}
