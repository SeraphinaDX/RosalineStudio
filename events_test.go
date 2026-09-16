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

func TestUnusedBooleanHandlerRemainsValid(t *testing.T) {
	project := newProject()
	project.Handlers["FormerCloseRequest"] = "return true"
	if err := project.validate(); err != nil {
		t.Fatalf("unused boolean handler should remain editable: %v", err)
	}
}
