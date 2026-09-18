// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import "testing"

func TestCloseRequestHandlersReturnBool(t *testing.T) {
	if err := validateHandler("AllowClose", "return true", true); err != nil {
		t.Fatalf("valid close-request handler was rejected: %v", err)
	}
	if err := validateHandler("AllowClose", "return true", false); err == nil {
		t.Fatal("boolean handler body was accepted for a void event")
	}
}

func TestHandlerCannotMixVoidAndBooleanEvents(t *testing.T) {
	project := newProject()
	project.mainForm().Events = map[string]string{eventCloseRequest: "SharedHandler"}
	project.mainForm().Root.Children[2].Events[eventClick] = "SharedHandler"
	project.Handlers["SharedHandler"] = "return true"
	if err := project.validate(); err == nil {
		t.Fatal("handler with incompatible event signatures was accepted")
	}
}

func TestTypedHandlerBodyCanUseEventParameters(t *testing.T) {
	signature := eventSignatureFor(kindList, eventSelect)
	body := `if index >= 0 { println(value) }`
	if err := validateHandlerSignature("ItemSelected", body, signature); err != nil {
		t.Fatalf("typed event body was rejected: %v", err)
	}
	if got := signature.parameterDeclaration(); got != "index int, value string" {
		t.Fatalf("parameter declaration = %q", got)
	}
	if got := signature.arguments(); got != "index, value" {
		t.Fatalf("argument list = %q", got)
	}
}

func TestHandlerCannotMixDifferentParameterSignatures(t *testing.T) {
	project := newProject()
	textBox := project.mainForm().Root.Children[1]
	textBox.Events = map[string]string{eventChange: "SharedChange"}
	check, err := project.addNear(project.mainForm().Root.ID, kindCheckBox)
	if err != nil {
		t.Fatal(err)
	}
	check.Events = map[string]string{eventChange: "SharedChange"}
	project.Handlers["SharedChange"] = "// Respond to the change."
	if err := project.validate(); err == nil {
		t.Fatal("handler with string and bool parameters was accepted")
	}
}

func TestHandlerCanShareIdenticalParameterSignatures(t *testing.T) {
	project := newProject()
	textBox := project.mainForm().Root.Children[1]
	textBox.Events = map[string]string{eventChange: "TextEntered", eventSubmit: "TextEntered"}
	project.Handlers["TextEntered"] = `println(value)`
	if err := project.validate(); err != nil {
		t.Fatalf("matching string event signatures should share a handler: %v", err)
	}
}

func TestEveryValueEventHasFriendlyTypedParameters(t *testing.T) {
	want := map[widgetKind]map[string]string{
		kindCanvas: {
			eventDraw:      "canvas *rosaline.DrawingCanvas",
			eventMouseDown: "event rosaline.MouseEvent",
			eventKeyDown:   "event rosaline.KeyEvent",
		},
		kindTextBox:    {eventChange: "value string", eventSubmit: "value string"},
		kindCheckBox:   {eventChange: "checked bool"},
		kindSlider:     {eventChange: "value float64"},
		kindList:       {eventSelect: "index int, value string"},
		kindTable:      {eventActivate: "index int, row []string"},
		kindTree:       {eventExpand: "node *rosaline.TreeNode, expanded bool"},
		kindTabs:       {eventChange: "index int, title string"},
		kindRadioGroup: {eventChange: "value string"},
	}
	for kind, events := range want {
		for name, declaration := range events {
			signature := eventSignatureFor(kind, name)
			if got := signature.parameterDeclaration(); got != declaration {
				t.Errorf("%s.%s parameters = %q, want %q", kind, name, got, declaration)
			}
		}
	}
}

func TestCanvasExposesDrawingPointerAndKeyboardEvents(t *testing.T) {
	want := []string{eventDraw, eventMouseDown, eventDoubleClick, eventMouseMove, eventMouseUp, eventKeyDown, eventKeyUp}
	for _, event := range want {
		if !supportsEvent(kindCanvas, event) {
			t.Fatalf("Canvas does not expose %s", event)
		}
	}
	if got := eventSpecsFor(kindCanvas)[0].Name; got != eventDraw {
		t.Fatalf("Canvas default event = %s, want %s", got, eventDraw)
	}
}

func TestEventCodeHeaderShowsCompleteGeneratedSignature(t *testing.T) {
	studio := newStudio()
	studio.codeHandler = "ProjectExpanded"
	studio.codeParameters = parameters("node", "*rosaline.TreeNode", "expanded", "bool")
	if got := studio.codeHeader(); got != "func (app *Application) ProjectExpanded(node *rosaline.TreeNode, expanded bool)" {
		t.Fatalf("code header = %q", got)
	}
}

func TestUnusedBooleanHandlerRemainsValid(t *testing.T) {
	project := newProject()
	project.Handlers["FormerCloseRequest"] = "return true"
	if err := project.validate(); err != nil {
		t.Fatalf("unused boolean handler should remain editable: %v", err)
	}
}

func TestDataControlsExposeTheirNativeEvents(t *testing.T) {
	tests := map[widgetKind][]string{
		kindList:       {eventSelect, eventActivate},
		kindTable:      {eventSelect, eventActivate},
		kindTree:       {eventSelect, eventActivate, eventExpand},
		kindRadioGroup: {eventChange},
	}
	for kind, events := range tests {
		for _, event := range events {
			if !supportsEvent(kind, event) {
				t.Fatalf("%s does not expose %s", kind, event)
			}
		}
	}
}
