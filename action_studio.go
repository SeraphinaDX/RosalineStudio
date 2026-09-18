// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"fmt"
	"strings"

	rosaline "github.com/SeraphinaDX/Rosaline"
)

const noSharedAction = "(no shared action)"

type actionInspectorState struct {
	Name     string
	Caption  string
	Shortcut string
	Handler  string
}

func (studio *studio) selectedAction() *designAction {
	return studio.project.action(studio.actionID)
}

func (studio *studio) selectedToolbarItem() *designToolbarItem {
	return findToolbarItem(studio.activeForm(), studio.toolbarID)
}

func (studio *studio) ensureActionSelection() {
	if studio.selectedAction() != nil {
		return
	}
	studio.actionID = ""
	if len(studio.project.Actions) != 0 && studio.project.Actions[0] != nil {
		studio.actionID = studio.project.Actions[0].ID
	}
}

func (studio *studio) ensureToolbarSelection() {
	if studio.selectedToolbarItem() != nil {
		return
	}
	studio.toolbarID = ""
	form := studio.activeForm()
	if form != nil && len(form.Toolbar) != 0 && form.Toolbar[0] != nil {
		studio.toolbarID = form.Toolbar[0].ID
	}
}

func (studio *studio) selectAction(index int) {
	if studio.syncAction || index < 0 || index >= len(studio.actionIDs) {
		return
	}
	if !studio.confirmOpenHandler("") {
		studio.syncActionSelection()
		return
	}
	studio.actionID = studio.actionIDs[index]
	studio.loadActionInspector()
	studio.status = "Selected action " + studio.selectedActionLabel()
}

func (studio *studio) selectToolbarItem(index int) {
	if studio.syncToolbar || index < 0 || index >= len(studio.toolbarIDs) {
		return
	}
	studio.toolbarID = studio.toolbarIDs[index]
	studio.status = "Selected toolbar item"
}

func (studio *studio) loadActionInspector() {
	action := studio.selectedAction()
	if action == nil {
		studio.action = actionInspectorState{}
		return
	}
	studio.action = actionInspectorState{
		Name: action.Name, Caption: action.Text,
		Shortcut: action.Shortcut, Handler: action.Handler,
	}
}

func (studio *studio) selectedActionLabel() string {
	action := studio.selectedAction()
	if action == nil {
		return "No project action selected"
	}
	return action.Name + " - " + action.Text
}

func (studio *studio) rebuildActions() {
	studio.ensureActionSelection()
	labels := make([]string, 0, len(studio.project.Actions))
	studio.actionIDs = studio.actionIDs[:0]
	selected := -1
	options := []string{noSharedAction}
	for _, action := range studio.project.Actions {
		if action == nil {
			continue
		}
		label := action.Name + " - " + action.Text
		if action.Shortcut != "" {
			label += "  [" + action.Shortcut + "]"
		}
		labels = append(labels, label)
		studio.actionIDs = append(studio.actionIDs, action.ID)
		options = append(options, action.Name)
		if action.ID == studio.actionID {
			selected = len(labels) - 1
		}
	}
	if studio.actionList != nil {
		studio.syncAction = true
		studio.actionList.SetItems(labels...)
		studio.actionList.Select(selected)
		studio.syncAction = false
	}
	if studio.menuActionChoice != nil {
		studio.menuActionChoice.SetOptions(options...)
		studio.menuActionChoice.Select(studio.menu.Action)
	}
	studio.loadActionInspector()
	studio.rebuildToolbar()
}

func (studio *studio) rebuildToolbar() {
	studio.ensureToolbarSelection()
	form := studio.activeForm()
	labels := make([]string, 0)
	studio.toolbarIDs = studio.toolbarIDs[:0]
	selected := -1
	if form != nil {
		for _, item := range form.Toolbar {
			if item == nil {
				continue
			}
			label := "──────── Separator"
			if item.Kind == toolbarItemAction {
				if action := studio.project.action(item.Action); action != nil {
					label = action.Name + " - " + action.Text
				} else {
					label = "Missing action"
				}
			}
			labels = append(labels, label)
			studio.toolbarIDs = append(studio.toolbarIDs, item.ID)
			if item.ID == studio.toolbarID {
				selected = len(labels) - 1
			}
		}
	}
	if studio.toolbarList != nil {
		studio.syncToolbar = true
		studio.toolbarList.SetItems(labels...)
		studio.toolbarList.Select(selected)
		studio.syncToolbar = false
	}
}

func (studio *studio) syncActionSelection() {
	if studio.actionList == nil {
		return
	}
	index := -1
	for candidate, id := range studio.actionIDs {
		if id == studio.actionID {
			index = candidate
			break
		}
	}
	studio.syncAction = true
	studio.actionList.Select(index)
	studio.syncAction = false
}

func (studio *studio) showActionDesigner() {
	if !studio.confirmOpenHandler("") {
		return
	}
	if studio.workspace != nil {
		studio.workspace.Select(workspaceActionTab)
	}
}

func (studio *studio) addAction() {
	before := designSnapshot(studio.project)
	action := studio.project.addAction()
	studio.actionID = action.ID
	studio.commitChange(before, "Added project action "+action.Name)
	studio.showActionDesigner()
}

func (studio *studio) duplicateAction() {
	if studio.selectedAction() == nil {
		studio.status = "Select an action first"
		return
	}
	before := designSnapshot(studio.project)
	action, err := studio.project.duplicateAction(studio.actionID)
	if err != nil {
		studio.status = "Could not duplicate action: " + err.Error()
		return
	}
	studio.actionID = action.ID
	studio.commitChange(before, "Duplicated project action")
}

func (studio *studio) deleteAction() {
	action := studio.selectedAction()
	if action == nil {
		studio.status = "Select an action first"
		return
	}
	if !rosaline.Confirm("Delete project action?", fmt.Sprintf("Delete %s? Menu links will become ordinary items and toolbar buttons using it will be removed. You can undo this change.", action.Name)) {
		studio.status = "Action deletion cancelled"
		return
	}
	before := designSnapshot(studio.project)
	if err := studio.project.removeAction(action.ID); err != nil {
		studio.status = "Could not delete action: " + err.Error()
		return
	}
	studio.actionID = ""
	studio.toolbarID = ""
	studio.commitChange(before, "Deleted project action - use Primary+Z to undo")
}

func (studio *studio) applyActionInspector() {
	action := studio.selectedAction()
	if action == nil {
		studio.status = "Select an action first"
		return
	}
	name := strings.TrimSpace(studio.action.Name)
	caption := strings.TrimSpace(studio.action.Caption)
	shortcut := strings.TrimSpace(studio.action.Shortcut)
	handler := strings.TrimSpace(studio.action.Handler)
	if !validComponentName(name) {
		studio.status = "Action names must be exported Go identifiers such as SaveAction"
		return
	}
	if other := studio.project.actionNamed(name); other != nil && other.ID != action.ID {
		studio.status = "That action name is already in use"
		return
	}
	if caption == "" {
		studio.status = "An action caption cannot be empty"
		return
	}
	if handler != "" && !validHandlerName(handler) {
		studio.status = "Handler names must be valid Go identifiers"
		return
	}
	if handler != "" {
		if err := handlerAcceptsSignature(studio.project, handler, handlerSignature{Seen: true}); err != nil {
			studio.status = err.Error()
			return
		}
	}
	if err := validateMenuShortcut(shortcut); err != nil {
		studio.status = err.Error()
		return
	}
	before := designSnapshot(studio.project)
	action.Name, action.Text = name, caption
	action.Shortcut, action.Handler = shortcut, handler
	if handler != "" {
		if studio.project.Handlers == nil {
			studio.project.Handlers = make(map[string]string)
		}
		if _, exists := studio.project.Handlers[handler]; !exists {
			studio.project.Handlers[handler] = defaultActionHandlerBody(action)
		}
	}
	studio.commitChange(before, "Updated project action "+name)
}

func (studio *studio) editActionHandler() {
	action := studio.selectedAction()
	if action == nil {
		studio.status = "Select an action first"
		return
	}
	handler := strings.TrimSpace(studio.action.Handler)
	if handler == "" {
		handler = strings.TrimSpace(action.Handler)
	}
	if handler == "" {
		handler = studio.project.nextHandlerName(action.Name + "Execute")
	}
	if !validHandlerName(handler) {
		studio.status = "Handler names must be valid Go identifiers"
		return
	}
	if !studio.confirmOpenHandler(handler) {
		return
	}
	signature := handlerSignature{Seen: true}
	if err := handlerAcceptsSignature(studio.project, handler, signature); err != nil {
		studio.status = err.Error()
		return
	}
	before := designSnapshot(studio.project)
	changed := action.Handler != handler
	action.Handler = handler
	if studio.project.Handlers == nil {
		studio.project.Handlers = make(map[string]string)
	}
	if _, exists := studio.project.Handlers[handler]; !exists {
		studio.project.Handlers[handler] = defaultActionHandlerBody(action)
		changed = true
	}
	if changed {
		studio.commitChange(before, "Assigned action execute handler "+handler)
	}
	studio.codeHandler = handler
	studio.codeParameters = nil
	studio.codeReturnsBool = false
	studio.codeBody = studio.project.Handlers[handler]
	studio.action.Handler = handler
	studio.codeEditor.SetText(studio.codeBody)
	studio.codeEditor.MarkSaved()
	if studio.workspace != nil {
		studio.workspace.Select(workspaceCodeTab)
	}
	studio.codeEditor.Focus()
	studio.status = "Editing " + handler
}

func (studio *studio) addSelectedActionToToolbar() {
	if studio.selectedAction() == nil {
		studio.status = "Select or create a project action first"
		return
	}
	before := designSnapshot(studio.project)
	item, err := studio.project.addToolbarAction(studio.activeForm(), studio.actionID)
	if err != nil {
		studio.status = "Could not add toolbar button: " + err.Error()
		return
	}
	studio.toolbarID = item.ID
	studio.commitChange(before, "Added action to "+studio.activeForm().Name+" toolbar")
}

func (studio *studio) addToolbarSeparator() {
	before := designSnapshot(studio.project)
	item, err := studio.project.addToolbarSeparator(studio.activeForm())
	if err != nil {
		studio.status = "Could not add toolbar separator: " + err.Error()
		return
	}
	studio.toolbarID = item.ID
	studio.commitChange(before, "Added toolbar separator")
}

func (studio *studio) moveToolbarItem(difference int) {
	if studio.selectedToolbarItem() == nil {
		studio.status = "Select a toolbar item first"
		return
	}
	before := designSnapshot(studio.project)
	if err := moveToolbarItem(studio.activeForm(), studio.toolbarID, difference); err != nil {
		studio.status = "Could not move toolbar item: " + err.Error()
		return
	}
	studio.commitChange(before, "Moved toolbar item")
}

func (studio *studio) removeToolbarItem() {
	if studio.selectedToolbarItem() == nil {
		studio.status = "Select a toolbar item first"
		return
	}
	before := designSnapshot(studio.project)
	if err := removeToolbarItem(studio.activeForm(), studio.toolbarID); err != nil {
		studio.status = "Could not remove toolbar item: " + err.Error()
		return
	}
	studio.toolbarID = ""
	studio.commitChange(before, "Removed toolbar item")
}

func (studio *studio) menuActionID(name string) string {
	if name == "" || name == noSharedAction {
		return ""
	}
	if action := studio.project.actionNamed(name); action != nil {
		return action.ID
	}
	return ""
}
