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

	handwritten := []byte("package main\n\n// Mine stays mine.\n")
	handlers := filepath.Join(directory, "handlers.go")
	if err := os.WriteFile(handlers, handwritten, 0o644); err != nil {
		t.Fatal(err)
	}
	project.Title = "A Changed Title"
	project.Width = 901
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
	project.Root.Children = []*designNode{
		{ID: "name", Kind: kindTextBox, Name: "Name", Events: map[string]string{eventSubmit: "SubmitName"}},
	}
	project.Handlers = map[string]string{
		"SubmitName": `rosaline.Message("Hello", app.State.Name)`,
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
	if !strings.Contains(string(ui), ".OnSubmit(func(string) { app.SubmitName() })") {
		t.Fatalf("generated UI did not bind OnSubmit:\n%s", ui)
	}
	if !strings.Contains(string(events), "func (app *Application) SubmitName()") || !strings.Contains(string(events), "app.State.Name") {
		t.Fatalf("generated event method is incomplete:\n%s", events)
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
	project.Root.Children = []*designNode{{
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
	project.Root = &designNode{ID: "root", Kind: kindGrid, Columns: 3}
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
	project.Root.Children = []*designNode{
		{ID: "a", Kind: kindTextBox, Name: "display name"},
		{ID: "b", Kind: kindTextArea, Name: "display-name"},
		{ID: "c", Kind: kindCheckBox, Name: "123"},
	}
	context := makeGenerationContext(project)
	want := []generatedField{
		{Name: "DisplayName", Type: "string"},
		{Name: "DisplayName2", Type: "string"},
		{Name: "Value", Type: "bool"},
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
	if _, err := generateProject(newProject(), directory); err != nil {
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
