// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"strings"
)

const (
	eventClick  = "OnClick"
	eventChange = "OnChange"
	eventSubmit = "OnSubmit"
)

type eventSpec struct {
	Name        string
	Description string
}

func eventSpecsFor(kind widgetKind) []eventSpec {
	switch kind {
	case kindButton, kindImage:
		return []eventSpec{{Name: eventClick, Description: "Runs when the control is clicked."}}
	case kindTextBox:
		return []eventSpec{
			{Name: eventChange, Description: "Runs after the text changes."},
			{Name: eventSubmit, Description: "Runs when Enter is pressed."},
		}
	case kindTextArea, kindCheckBox, kindComboBox, kindSlider:
		return []eventSpec{{Name: eventChange, Description: "Runs after the value changes."}}
	default:
		return nil
	}
}

func supportsEvent(kind widgetKind, name string) bool {
	for _, event := range eventSpecsFor(kind) {
		if event.Name == name {
			return true
		}
	}
	return false
}

func validHandlerName(name string) bool {
	name = strings.TrimSpace(name)
	return token.IsIdentifier(name) && !token.Lookup(name).IsKeyword()
}

func validateHandler(name, body string) error {
	if !validHandlerName(name) {
		return fmt.Errorf("invalid event handler name %q", name)
	}
	source := "package main\nfunc " + name + "() {\n" + body + "\n}\n"
	if _, err := parser.ParseFile(token.NewFileSet(), "event.go", source, parser.AllErrors); err != nil {
		return fmt.Errorf("event handler %s contains invalid Go: %w", name, err)
	}
	return nil
}

func defaultHandlerName(node *designNode, event string) string {
	base := "Widget"
	if node != nil {
		base = exportedIdentifier(defaultText(node.Name, node.Text))
		if base == "Value" {
			base = exportedIdentifier(string(node.Kind))
		}
	}
	suffix := strings.TrimPrefix(event, "On")
	return base + suffix
}

func defaultHandlerBody(node *designNode, event string) string {
	if event == eventClick {
		label := "Control"
		if node != nil {
			label = defaultText(node.Text, string(node.Kind))
		}
		return fmt.Sprintf("rosaline.Message(%q, %q)", "Event", label+" clicked.")
	}
	if node != nil && node.Name != "" {
		return fmt.Sprintf("// app.State.%s already contains the new value.\n// Add your response here.", exportedIdentifier(node.Name))
	}
	return "// Add your event code here."
}

func eventHandler(node *designNode, event string) string {
	if node == nil || node.Events == nil {
		return ""
	}
	return strings.TrimSpace(node.Events[event])
}

func setEventHandler(node *designNode, event, handler string) {
	if node == nil {
		return
	}
	if handler = strings.TrimSpace(handler); handler == "" {
		if node.Events != nil {
			delete(node.Events, event)
		}
		return
	}
	if node.Events == nil {
		node.Events = make(map[string]string)
	}
	node.Events[event] = handler
}
