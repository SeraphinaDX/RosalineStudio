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
	Parameters  []eventParameter
	ReturnsBool bool
}

type eventParameter struct {
	Name string
	Type string
}

func parameters(values ...string) []eventParameter {
	result := make([]eventParameter, 0, len(values)/2)
	for index := 0; index+1 < len(values); index += 2 {
		result = append(result, eventParameter{Name: values[index], Type: values[index+1]})
	}
	return result
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

func eventSpecsFor(kind widgetKind) []eventSpec {
	switch kind {
	case kindButton, kindImage:
		return []eventSpec{{Name: eventClick, Description: "Runs when the control is clicked."}}
	case kindTextBox:
		return []eventSpec{
			{Name: eventChange, Description: "Runs after the text changes.", Parameters: parameters("value", "string")},
			{Name: eventSubmit, Description: "Runs when Enter is pressed.", Parameters: parameters("value", "string")},
		}
	case kindTextArea, kindComboBox:
		return []eventSpec{{Name: eventChange, Description: "Runs after the value changes.", Parameters: parameters("value", "string")}}
	case kindCheckBox:
		return []eventSpec{{Name: eventChange, Description: "Runs after the checked state changes.", Parameters: parameters("checked", "bool")}}
	case kindSlider:
		return []eventSpec{{Name: eventChange, Description: "Runs after the numeric value changes.", Parameters: parameters("value", "float64")}}
	case kindRadioGroup:
		return []eventSpec{{Name: eventChange, Description: "Runs after the selected radio choice changes.", Parameters: parameters("value", "string")}}
	case kindList:
		return []eventSpec{
			{Name: eventSelect, Description: "Runs after the selected list item changes.", Parameters: parameters("index", "int", "value", "string")},
			{Name: eventActivate, Description: "Runs when a list item is double-clicked or activated with Enter.", Parameters: parameters("index", "int", "value", "string")},
		}
	case kindTable:
		return []eventSpec{
			{Name: eventSelect, Description: "Runs after the selected table row changes.", Parameters: parameters("index", "int", "row", "[]string")},
			{Name: eventActivate, Description: "Runs when a table row is double-clicked or activated with Enter.", Parameters: parameters("index", "int", "row", "[]string")},
		}
	case kindTree:
		return []eventSpec{
			{Name: eventSelect, Description: "Runs after the selected tree node changes.", Parameters: parameters("node", "*rosaline.TreeNode")},
			{Name: eventActivate, Description: "Runs when a tree node is double-clicked or activated with Enter.", Parameters: parameters("node", "*rosaline.TreeNode")},
			{Name: eventExpand, Description: "Runs after a tree node is opened or closed.", Parameters: parameters("node", "*rosaline.TreeNode", "expanded", "bool")},
		}
	case kindTabs:
		return []eventSpec{{Name: eventChange, Description: "Runs after the selected tab page changes.", Parameters: parameters("index", "int", "title", "string")}}
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

func eventSpecFor(kind widgetKind, name string) (eventSpec, bool) {
	for _, event := range eventSpecsFor(kind) {
		if event.Name == name {
			return event, true
		}
	}
	return eventSpec{}, false
}

func validHandlerName(name string) bool {
	name = strings.TrimSpace(name)
	return token.IsIdentifier(name) && !token.Lookup(name).IsKeyword()
}

func validateHandler(name, body string, returnsBool bool) error {
	return validateHandlerSignature(name, body, handlerSignature{ReturnsBool: returnsBool, Seen: true})
}

func validateHandlerSignature(name, body string, signature handlerSignature) error {
	if !validHandlerName(name) {
		return fmt.Errorf("invalid event handler name %q", name)
	}
	result := ""
	if signature.ReturnsBool {
		result = " bool"
	}
	source := "package main\nimport rosaline \"github.com/SeraphinaDX/Rosaline\"\nvar _ = rosaline.Message\nfunc " + name + "(" + signature.parameterDeclaration() + ")" + result + " {\n" + body + "\n}\n"
	file, err := parser.ParseFile(token.NewFileSet(), "event.go", source, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("event handler %s contains invalid Go: %w", name, err)
	}
	var declaration *ast.FuncDecl
	for _, item := range file.Decls {
		if function, ok := item.(*ast.FuncDecl); ok && function.Name.Name == name {
			declaration = function
			break
		}
	}
	if declaration == nil || declaration.Body == nil {
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
		if signature.ReturnsBool {
			hasValueReturn = hasValueReturn || len(statement.Results) != 0
			invalidReturn = invalidReturn || len(statement.Results) == 0
		} else {
			invalidReturn = invalidReturn || len(statement.Results) != 0
		}
		return true
	})
	if invalidReturn || (signature.ReturnsBool && !hasValueReturn) {
		return fmt.Errorf("event handler %s has a return statement that does not match its event", name)
	}
	return nil
}

type handlerSignature struct {
	Parameters  []eventParameter
	ReturnsBool bool
	Seen        bool
}

func (signature handlerSignature) parameterDeclaration() string {
	parts := make([]string, 0, len(signature.Parameters))
	for _, parameter := range signature.Parameters {
		parts = append(parts, parameter.Name+" "+parameter.Type)
	}
	return strings.Join(parts, ", ")
}

func (signature handlerSignature) arguments() string {
	parts := make([]string, 0, len(signature.Parameters))
	for _, parameter := range signature.Parameters {
		parts = append(parts, parameter.Name)
	}
	return strings.Join(parts, ", ")
}

func (signature handlerSignature) display() string {
	result := "func(" + signature.parameterDeclaration() + ")"
	if signature.ReturnsBool {
		result += " bool"
	}
	return result
}

func (signature handlerSignature) compatible(other handlerSignature) bool {
	if signature.ReturnsBool != other.ReturnsBool || len(signature.Parameters) != len(other.Parameters) {
		return false
	}
	for index, parameter := range signature.Parameters {
		if parameter != other.Parameters[index] {
			return false
		}
	}
	return true
}

func (event eventSpec) signature() handlerSignature {
	return handlerSignature{Parameters: append([]eventParameter(nil), event.Parameters...), ReturnsBool: event.ReturnsBool, Seen: true}
}

func eventSignatureFor(kind widgetKind, name string) handlerSignature {
	if event, ok := eventSpecFor(kind, name); ok {
		return event.signature()
	}
	return handlerSignature{Seen: true}
}

func formEventSignature(name string) handlerSignature {
	for _, event := range formEventSpecs() {
		if event.Name == name {
			return event.signature()
		}
	}
	return handlerSignature{Seen: true}
}

func projectHandlerSignatures(project *designProject) (map[string]handlerSignature, error) {
	result := make(map[string]handlerSignature)
	register := func(name string, signature handlerSignature) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil
		}
		if existing := result[name]; existing.Seen && !existing.compatible(signature) {
			return fmt.Errorf("event handler %s is assigned to incompatible signatures %s and %s", name, existing.display(), signature.display())
		}
		signature.Seen = true
		result[name] = signature
		return nil
	}
	var visit func(*designNode) error
	visit = func(node *designNode) error {
		if node == nil {
			return nil
		}
		for event, handler := range node.Events {
			if err := register(handler, eventSignatureFor(node.Kind, event)); err != nil {
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
				if err := register(action.Handler, handlerSignature{Seen: true}); err != nil {
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
					if err := register(component.Handler, handlerSignature{Seen: true}); err != nil {
						return nil, err
					}
				}
			}
			for event, handler := range form.Events {
				if err := register(handler, formEventSignature(event)); err != nil {
					return nil, err
				}
			}
			var menuErr error
			visitDesignMenus(form.Menus, func(menu *designMenu) {
				if menuErr == nil && menu.Kind == menuKindItem && menu.Action == "" {
					menuErr = register(menu.Handler, handlerSignature{Seen: true})
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
			if err := validateHandlerSignature(name, body, signature); err != nil {
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

func handlerAcceptsSignature(project *designProject, name string, signature handlerSignature) error {
	signatures, err := projectHandlerSignatures(project)
	if err != nil {
		return err
	}
	if existing := signatures[name]; existing.Seen && !existing.compatible(signature) {
		return fmt.Errorf("handler %s already uses %s; this event needs %s", name, existing.display(), signature.display())
	}
	if project != nil {
		if body, exists := project.Handlers[name]; exists {
			return validateHandlerSignature(name, body, signature)
		}
	}
	return nil
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
			return "// index is the selected position; value is the selected item.\n// Add your response here."
		case kindTable:
			return "// index is the selected position; row contains the selected cells.\n// Add your response here."
		case kindTree:
			if event == eventExpand {
				return "// node is the changed item; expanded reports whether it is open.\n// Add your response here."
			}
			return "// node is the selected tree item.\n// Add your response here."
		case kindTabs:
			return "// index is the selected page; title is its visible title.\n// Add your response here."
		case kindCheckBox:
			return "// checked contains the control's new state.\n// Add your response here."
		case kindTextBox, kindTextArea, kindComboBox, kindRadioGroup, kindSlider:
			return "// value contains the control's new value.\n// Add your response here."
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
