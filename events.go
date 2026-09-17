// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

const (
	eventClick        = "OnClick"
	eventChange       = "OnChange"
	eventSubmit       = "OnSubmit"
	eventSelect       = "OnSelect"
	eventActivate     = "OnActivate"
	eventExpand       = "OnExpand"
	eventOpen         = "OnOpen"
	eventCloseRequest = "OnCloseRequest"
	eventClose        = "OnClose"
)

type eventSpec struct {
	Name        string
	Description string
	ReturnsBool bool
}

func formEventSpecs() []eventSpec {
	return []eventSpec{
		{Name: eventOpen, Description: "Runs after the form and its controls are mounted."},
		{Name: eventCloseRequest, Description: "Runs before a close request. Return true to close or false to keep the form open.", ReturnsBool: true},
		{Name: eventClose, Description: "Runs after the form has closed."},
	}
}

func supportsFormEvent(name string) bool {
	for _, event := range formEventSpecs() {
		if event.Name == name {
			return true
		}
	}
	return false
}

func eventReturnsBool(name string) bool { return name == eventCloseRequest }

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
	case kindRadioGroup:
		return []eventSpec{{Name: eventChange, Description: "Runs after the selected radio choice changes."}}
	case kindList:
		return []eventSpec{
			{Name: eventSelect, Description: "Runs after the selected list item changes."},
			{Name: eventActivate, Description: "Runs when a list item is double-clicked or activated with Enter."},
		}
	case kindTable:
		return []eventSpec{
			{Name: eventSelect, Description: "Runs after the selected table row changes."},
			{Name: eventActivate, Description: "Runs when a table row is double-clicked or activated with Enter."},
		}
	case kindTree:
		return []eventSpec{
			{Name: eventSelect, Description: "Runs after the selected tree node changes."},
			{Name: eventActivate, Description: "Runs when a tree node is double-clicked or activated with Enter."},
			{Name: eventExpand, Description: "Runs after a tree node is opened or closed."},
		}
	case kindTabs:
		return []eventSpec{{Name: eventChange, Description: "Runs after the selected tab page changes."}}
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

func validateHandler(name, body string, returnsBool bool) error {
	if !validHandlerName(name) {
		return fmt.Errorf("invalid event handler name %q", name)
	}
	result := ""
	if returnsBool {
		result = " bool"
	}
	source := "package main\nfunc " + name + "()" + result + " {\n" + body + "\n}\n"
	file, err := parser.ParseFile(token.NewFileSet(), "event.go", source, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("event handler %s contains invalid Go: %w", name, err)
	}
	declaration, ok := file.Decls[0].(*ast.FuncDecl)
	if !ok || declaration.Body == nil {
		return fmt.Errorf("event handler %s could not be parsed", name)
	}
	hasValueReturn := false
	invalidReturn := false
	ast.Inspect(declaration.Body, func(node ast.Node) bool {
		if _, nested := node.(*ast.FuncLit); nested {
			return false
		}
		statement, ok := node.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		if returnsBool {
			hasValueReturn = hasValueReturn || len(statement.Results) != 0
			invalidReturn = invalidReturn || len(statement.Results) == 0
		} else {
			invalidReturn = invalidReturn || len(statement.Results) != 0
		}
		return true
	})
	if invalidReturn || (returnsBool && !hasValueReturn) {
		return fmt.Errorf("event handler %s has a return statement that does not match its event", name)
	}
	return nil
}

type handlerSignature struct {
	ReturnsBool bool
	Seen        bool
}

func projectHandlerSignatures(project *designProject) (map[string]handlerSignature, error) {
	result := make(map[string]handlerSignature)
	register := func(name string, returnsBool bool) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil
		}
		if existing := result[name]; existing.Seen && existing.ReturnsBool != returnsBool {
			return fmt.Errorf("event handler %s is assigned to events with incompatible return values", name)
		}
		result[name] = handlerSignature{ReturnsBool: returnsBool, Seen: true}
		return nil
	}
	var visit func(*designNode) error
	visit = func(node *designNode) error {
		if node == nil {
			return nil
		}
		for _, handler := range node.Events {
			if err := register(handler, false); err != nil {
				return err
			}
		}
		for _, child := range node.Children {
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	if project != nil {
		for _, action := range project.Actions {
			if action != nil {
				if err := register(action.Handler, false); err != nil {
					return nil, err
				}
			}
		}
		for _, form := range project.Forms {
			if form == nil {
				continue
			}
			for _, component := range form.Components {
				if component != nil && component.Kind == componentTimer {
					if err := register(component.Handler, false); err != nil {
						return nil, err
					}
				}
			}
			for event, handler := range form.Events {
				if err := register(handler, eventReturnsBool(event)); err != nil {
					return nil, err
				}
			}
			var menuErr error
			visitDesignMenus(form.Menus, func(menu *designMenu) {
				if menuErr == nil && menu.Kind == menuKindItem && menu.Action == "" {
					menuErr = register(menu.Handler, false)
				}
			})
			if menuErr != nil {
				return nil, menuErr
			}
			if err := visit(form.Root); err != nil {
				return nil, err
			}
		}
	}
	return result, nil
}

func validateProjectHandlers(project *designProject) error {
	signatures, err := projectHandlerSignatures(project)
	if err != nil {
		return err
	}
	for name, body := range project.Handlers {
		signature := signatures[name]
		if signature.Seen {
			if err := validateHandler(name, body, signature.ReturnsBool); err != nil {
				return err
			}
			continue
		}
		if err := validateHandler(name, body, false); err != nil {
			if boolErr := validateHandler(name, body, true); boolErr != nil {
				return err
			}
		}
	}
	return nil
}

func handlerReturnsBool(project *designProject, name string) bool {
	signatures, _ := projectHandlerSignatures(project)
	return signatures[name].ReturnsBool
}

func defaultHandlerName(node *designNode, event string) string {
	base := "Widget"
	if node != nil {
		base = exportedIdentifier(node.Component)
		if base == "Value" {
			base = exportedIdentifier(defaultText(node.Name, node.Text))
		}
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
	if node != nil {
		switch node.Kind {
		case kindList:
			return fmt.Sprintf("// Read the current item with app.Widgets().%s.Selected().\n// Add your response here.", node.Component)
		case kindTable:
			return fmt.Sprintf("// Read the current row with app.Widgets().%s.Selected().\n// Add your response here.", node.Component)
		case kindTree:
			return fmt.Sprintf("// Read the current node with app.Widgets().%s.Selected().\n// Add your response here.", node.Component)
		}
	}
	if node != nil && node.Name != "" {
		return fmt.Sprintf("// app.State.%s already contains the new value.\n// Use app.Widgets().%s to update this control.\n// Add your response here.", exportedIdentifier(node.Name), node.Component)
	}
	return "// Add your event code here."
}

func defaultFormHandlerName(form *designForm, event string) string {
	base := "Form"
	if form != nil && validComponentName(form.Name) {
		base = form.Name
	}
	return base + strings.TrimPrefix(event, "On")
}

func defaultFormHandlerBody(form *designForm, event string) string {
	if event == eventCloseRequest {
		return "// Return false here when the form must remain open.\nreturn true"
	}
	name := "form"
	if form != nil {
		name = form.Name
	}
	return fmt.Sprintf("// %s %s.\n// Add your response here.", name, strings.ToLower(strings.TrimPrefix(event, "On")))
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

func formEventHandler(form *designForm, event string) string {
	if form == nil || form.Events == nil {
		return ""
	}
	return strings.TrimSpace(form.Events[event])
}

func setFormEventHandler(form *designForm, event, handler string) {
	if form == nil {
		return
	}
	if handler = strings.TrimSpace(handler); handler == "" {
		delete(form.Events, event)
		return
	}
	if form.Events == nil {
		form.Events = make(map[string]string)
	}
	form.Events[event] = handler
}
