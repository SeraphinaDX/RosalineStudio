// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"strings"
)

type toolbarItemKind string

const (
	toolbarItemAction    toolbarItemKind = "Action"
	toolbarItemSeparator toolbarItemKind = "Separator"
)

// designAction is one application command that can be reused by menus and
// toolbar buttons. The handler remains an ordinary generated Go method.
type designAction struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Text     string `json:"text"`
	Handler  string `json:"handler,omitempty"`
	Shortcut string `json:"shortcut,omitempty"`
}

// designToolbarItem places either an action button or a separator on a form.
type designToolbarItem struct {
	ID     string          `json:"id"`
	Kind   toolbarItemKind `json:"kind"`
	Action string          `json:"action,omitempty"`
}

func (project *designProject) action(id string) *designAction {
	if project == nil {
		return nil
	}
	for _, action := range project.Actions {
		if action != nil && action.ID == id {
			return action
		}
	}
	return nil
}

func (project *designProject) actionNamed(name string) *designAction {
	if project == nil {
		return nil
	}
	for _, action := range project.Actions {
		if action != nil && action.Name == name {
			return action
		}
	}
	return nil
}

func (project *designProject) nextActionID() string {
	used := make(map[string]bool)
	for _, action := range project.Actions {
		if action != nil {
			used[action.ID] = true
		}
	}
	for number := 1; ; number++ {
		id := fmt.Sprintf("action-%d", number)
		if !used[id] {
			return id
		}
	}
}

func (project *designProject) nextActionName(base string) string {
	used := make(map[string]bool)
	for _, action := range project.Actions {
		if action != nil {
			used[action.Name] = true
		}
	}
	return uniqueGeneratedName(exportedIdentifier(base), used)
}

func (project *designProject) addAction() *designAction {
	name := project.nextActionName("Action")
	handler := project.nextHandlerName(name + "Execute")
	action := &designAction{
		ID:      project.nextActionID(),
		Name:    name,
		Text:    "New Action",
		Handler: handler,
	}
	project.Actions = append(project.Actions, action)
	if project.Handlers == nil {
		project.Handlers = make(map[string]string)
	}
	project.Handlers[handler] = defaultActionHandlerBody(action)
	return action
}

func (project *designProject) duplicateAction(id string) (*designAction, error) {
	source := project.action(id)
	if source == nil {
		return nil, errors.New("action was not found")
	}
	clone := *source
	clone.ID = project.nextActionID()
	clone.Name = project.nextActionName(source.Name)
	clone.Text = source.Text + " Copy"
	clone.Handler = project.nextHandlerName(clone.Name + "Execute")
	project.Actions = append(project.Actions, &clone)
	if project.Handlers == nil {
		project.Handlers = make(map[string]string)
	}
	if body, ok := project.Handlers[source.Handler]; ok {
		project.Handlers[clone.Handler] = body
	} else {
		project.Handlers[clone.Handler] = defaultActionHandlerBody(&clone)
	}
	return &clone, nil
}

func (project *designProject) nextHandlerName(base string) string {
	used := make(map[string]bool, len(project.Handlers))
	for name := range project.Handlers {
		used[name] = true
	}
	return uniqueGeneratedName(exportedIdentifier(base), used)
}

func (project *designProject) removeAction(id string) error {
	index := -1
	for candidate, action := range project.Actions {
		if action != nil && action.ID == id {
			index = candidate
			break
		}
	}
	if index < 0 {
		return errors.New("action was not found")
	}
	project.Actions = append(project.Actions[:index], project.Actions[index+1:]...)
	for _, form := range project.Forms {
		if form == nil {
			continue
		}
		visitDesignMenus(form.Menus, func(menu *designMenu) {
			if menu.Action == id {
				menu.Action = ""
			}
		})
		kept := form.Toolbar[:0]
		for _, item := range form.Toolbar {
			if item == nil || item.Action != id {
				kept = append(kept, item)
			}
		}
		form.Toolbar = kept
	}
	return nil
}

func defaultActionHandlerBody(action *designAction) string {
	label := "Action"
	if action != nil && strings.TrimSpace(action.Text) != "" {
		label = action.Text
	}
	return fmt.Sprintf("rosaline.Message(%q, %q)", "Action", label+" selected.")
}

func cloneDesignToolbar(source []*designToolbarItem) []*designToolbarItem {
	result := make([]*designToolbarItem, 0, len(source))
	for _, item := range source {
		if item == nil {
			result = append(result, nil)
			continue
		}
		clone := *item
		result = append(result, &clone)
	}
	return result
}

func (project *designProject) nextToolbarID() string {
	used := make(map[string]bool)
	for _, form := range project.Forms {
		if form == nil {
			continue
		}
		for _, item := range form.Toolbar {
			if item != nil {
				used[item.ID] = true
			}
		}
	}
	for number := 1; ; number++ {
		id := fmt.Sprintf("tool-%d", number)
		if !used[id] {
			return id
		}
	}
}

func (project *designProject) prepareCopiedToolbar(toolbar []*designToolbarItem) {
	used := make(map[string]bool)
	for _, form := range project.Forms {
		if form == nil {
			continue
		}
		for _, item := range form.Toolbar {
			if item != nil {
				used[item.ID] = true
			}
		}
	}
	next := 1
	for _, item := range toolbar {
		if item != nil {
			for used[fmt.Sprintf("tool-%d", next)] {
				next++
			}
			item.ID = fmt.Sprintf("tool-%d", next)
			used[item.ID] = true
			next++
		}
	}
}

func (project *designProject) addToolbarAction(form *designForm, actionID string) (*designToolbarItem, error) {
	if form == nil {
		return nil, errors.New("there is no active form")
	}
	if project.action(actionID) == nil {
		return nil, errors.New("select a project action first")
	}
	item := &designToolbarItem{ID: project.nextToolbarID(), Kind: toolbarItemAction, Action: actionID}
	form.Toolbar = append(form.Toolbar, item)
	return item, nil
}

func (project *designProject) addToolbarSeparator(form *designForm) (*designToolbarItem, error) {
	if form == nil {
		return nil, errors.New("there is no active form")
	}
	item := &designToolbarItem{ID: project.nextToolbarID(), Kind: toolbarItemSeparator}
	form.Toolbar = append(form.Toolbar, item)
	return item, nil
}

func findToolbarItem(form *designForm, id string) *designToolbarItem {
	if form == nil {
		return nil
	}
	for _, item := range form.Toolbar {
		if item != nil && item.ID == id {
			return item
		}
	}
	return nil
}

func moveToolbarItem(form *designForm, id string, difference int) error {
	if form == nil {
		return errors.New("there is no active form")
	}
	index := -1
	for candidate, item := range form.Toolbar {
		if item != nil && item.ID == id {
			index = candidate
			break
		}
	}
	if index < 0 {
		return errors.New("toolbar item was not found")
	}
	target := index + difference
	if target < 0 || target >= len(form.Toolbar) {
		return errors.New("toolbar item is already at that edge")
	}
	form.Toolbar[index], form.Toolbar[target] = form.Toolbar[target], form.Toolbar[index]
	return nil
}

func removeToolbarItem(form *designForm, id string) error {
	if form == nil {
		return errors.New("there is no active form")
	}
	for index, item := range form.Toolbar {
		if item != nil && item.ID == id {
			form.Toolbar = append(form.Toolbar[:index], form.Toolbar[index+1:]...)
			return nil
		}
	}
	return errors.New("toolbar item was not found")
}

func validateDesignActions(project *designProject, seen map[string]bool) (map[string]*designAction, error) {
	byID := make(map[string]*designAction)
	names := make(map[string]bool)
	for _, action := range project.Actions {
		if action == nil {
			return nil, errors.New("project contains an empty action")
		}
		if strings.TrimSpace(action.ID) == "" || seen[action.ID] {
			return nil, fmt.Errorf("action ID %q is empty or repeated", action.ID)
		}
		seen[action.ID] = true
		byID[action.ID] = action
		if !validComponentName(action.Name) || names[action.Name] {
			return nil, fmt.Errorf("action name %q is invalid or repeated", action.Name)
		}
		names[action.Name] = true
		if strings.TrimSpace(action.Text) == "" {
			return nil, fmt.Errorf("action %s has an empty caption", action.Name)
		}
		if action.Handler != "" && !validHandlerName(action.Handler) {
			return nil, fmt.Errorf("action %s has invalid handler name %q", action.Name, action.Handler)
		}
		if err := validateMenuShortcut(action.Shortcut); err != nil {
			return nil, fmt.Errorf("action %s: %w", action.Name, err)
		}
	}
	return byID, nil
}

func validateDesignToolbar(form *designForm, actions map[string]*designAction, seen map[string]bool) error {
	for _, item := range form.Toolbar {
		if item == nil {
			return errors.New("contains an empty item")
		}
		if strings.TrimSpace(item.ID) == "" || seen[item.ID] {
			return fmt.Errorf("item ID %q is empty or repeated", item.ID)
		}
		seen[item.ID] = true
		switch item.Kind {
		case toolbarItemAction:
			if actions[item.Action] == nil {
				return fmt.Errorf("item %s refers to missing action %q", item.ID, item.Action)
			}
		case toolbarItemSeparator:
			if item.Action != "" {
				return fmt.Errorf("separator %s cannot refer to an action", item.ID)
			}
		default:
			return fmt.Errorf("item %s has unknown kind %q", item.ID, item.Kind)
		}
	}
	return nil
}
