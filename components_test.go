// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNonvisualComponentsRoundTripAndDuplicate(t *testing.T) {
	project := newProject()
	form := project.mainForm()
	timer, err := project.addComponent(form, componentTimer)
	if err != nil {
		t.Fatal(err)
	}
	dialog, err := project.addComponent(form, componentOpenDialog)
	if err != nil {
		t.Fatal(err)
	}
	dialog.Filters = []string{"Images | .png, .jpg", "All files | *"}
	copy, err := project.duplicateForm(form.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(copy.Components) != 2 || copy.Components[0].ID == timer.ID || copy.Components[0].Name == timer.Name || copy.Components[0].Handler == timer.Handler {
		t.Fatalf("duplicated form retained component identity: %#v", copy.Components)
	}
	if len(copy.Components[1].Filters) != 2 || &copy.Components[1].Filters[0] == &dialog.Filters[0] {
		t.Fatal("duplicated dialog filters were not copied independently")
	}
	path := filepath.Join(t.TempDir(), "components.rosaline")
	if err := saveDesign(path, project); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadDesign(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.mainForm().Components) != 2 || loaded.mainForm().Components[1].Filters[0] != "Images | .png, .jpg" {
		t.Fatalf("components did not round trip: %#v", loaded.mainForm().Components)
	}
}

func TestComponentValidationAndFileFilterParsing(t *testing.T) {
	filters, err := parseDesignFileFilters([]string{"Images | .png, .jpg", "All files | *"})
	if err != nil || len(filters) != 2 || len(filters[0].Extensions) != 2 {
		t.Fatalf("valid filters were not parsed: %#v, %v", filters, err)
	}
	if _, err := parseDesignFileFilters([]string{"missing separator"}); err == nil {
		t.Fatal("invalid filter syntax was accepted")
	}
	project := newProject()
	timer, err := project.addComponent(project.mainForm(), componentTimer)
	if err != nil {
		t.Fatal(err)
	}
	timer.Interval = 0
	if err := project.validate(); err == nil || !strings.Contains(err.Error(), "at least 1 millisecond") {
		t.Fatalf("invalid timer interval was accepted: %v", err)
	}
}

func TestVersionSevenDesignMigratesToComponentsSchema(t *testing.T) {
	project := newProject()
	project.Version = 7
	data, err := json.Marshal(project)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "v7.rosaline")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadDesign(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Version != designVersion || loaded.mainForm().Components != nil {
		t.Fatalf("version 7 did not migrate cleanly: %#v", loaded)
	}
}

func TestGeneratorCreatesTimersAndFileDialogs(t *testing.T) {
	project := newProject()
	form := project.mainForm()
	timer, _ := project.addComponent(form, componentTimer)
	timer.Name, timer.Interval, timer.Handler = "RefreshTimer", 750, "RefreshTick"
	project.Handlers[timer.Handler] = `app.Widgets().WelcomeLabel.SetText("Refreshed")`
	open, _ := project.addComponent(form, componentOpenDialog)
	open.Name, open.Title = "OpenDocumentDialog", "Open a document"
	open.Filters = []string{"Text files | .txt, .md"}
	save, _ := project.addComponent(form, componentSaveDialog)
	save.Name, save.DefaultExtension = "SaveDocumentDialog", ".txt"

	directory := t.TempDir()
	if _, err := generateProject(project, directory); err != nil {
		t.Fatal(err)
	}
	parseGeneratedGo(t, directory)
	uiData, _ := os.ReadFile(filepath.Join(directory, "ui_generated.go"))
	stateData, _ := os.ReadFile(filepath.Join(directory, "state_generated.go"))
	eventsData, _ := os.ReadFile(filepath.Join(directory, "events_generated.go"))
	ui, state, events := string(uiData), string(stateData), string(eventsData)
	for _, want := range []string{
		`"time"`,
		`generatedComponents.RefreshTimer = rosaline.Every(time.Duration(750)*time.Millisecond, func() { app.RefreshTick() })`,
		`Timers:  []*rosaline.Timer{generatedComponents.RefreshTimer}`,
		`generatedComponents.OpenDocumentDialog = &UIFileDialog{save: false`,
		`generatedComponents.SaveDocumentDialog = &UIFileDialog{save: true`,
		`func (app *Application) Components() *UIComponents`,
	} {
		if !strings.Contains(ui, want) {
			t.Fatalf("generated UI is missing %q:\n%s", want, ui)
		}
	}
	for _, want := range []string{"type UIComponents struct", "RefreshTimer", "*rosaline.Timer", "OpenDocumentDialog", "*UIFileDialog", "func (dialog *UIFileDialog) Execute() (string, bool)"} {
		if !strings.Contains(state, want) {
			t.Fatalf("generated state is missing %q:\n%s", want, state)
		}
	}
	if strings.Count(events, "func (app *Application) RefreshTick()") != 1 {
		t.Fatalf("timer handler was not generated exactly once:\n%s", events)
	}
}
