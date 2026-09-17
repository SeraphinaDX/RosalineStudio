// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"fmt"
	"strings"

	rosaline "github.com/SeraphinaDX/Rosaline"
)

func (studio *studio) selectedMenu() *designMenu {
	form := studio.activeForm()
	if form == nil {
		return nil
	}
	return findDesignMenu(form.Menus, studio.menuID)
}

func (studio *studio) ensureMenuSelection() {
	form := studio.activeForm()
	if form == nil {
		studio.menuID = ""
		return
	}
	if findDesignMenu(form.Menus, studio.menuID) != nil {
		return
	}
	studio.menuID = ""
	if len(form.Menus) != 0 && form.Menus[0] != nil {
		studio.menuID = form.Menus[0].ID
	}
}

func (studio *studio) selectMenu(id string) {
	form := studio.activeForm()
	if form == nil || findDesignMenu(form.Menus, id) == nil {
		return
	}
	studio.menuID = id
	studio.loadMenuInspector()
	studio.syncMenuSelection()
}

func (studio *studio) loadMenuInspector() {
	menu := studio.selectedMenu()
	if menu == nil {
		studio.menu = menuInspectorState{Action: noSharedAction}
		if studio.menuActionChoice != nil {
			studio.menuActionChoice.Select(noSharedAction)
		}
		return
	}
	studio.menu = menuInspectorState{
		Action:   noSharedAction,
		Caption:  menu.Text,
		Shortcut: menu.Shortcut,
		Handler:  menu.Handler,
	}
	if action := studio.project.action(menu.Action); action != nil {
		studio.menu.Action = action.Name
	}
	if studio.menuActionChoice != nil {
		studio.menuActionChoice.Select(studio.menu.Action)
	}
}

func (studio *studio) selectedMenuLabel() string {
	menu := studio.selectedMenu()
	if menu == nil {
		return "No menu entry selected"
	}
	if menu.Kind == menuKindSeparator {
		return "Separator"
	}
	if action := studio.project.action(menu.Action); action != nil {
		return string(menu.Kind) + " - " + action.Text + "  [" + action.Name + "]"
	}
	return string(menu.Kind) + " - " + menu.Text
}

func (studio *studio) rebuildMenuTree() {
	form := studio.activeForm()
	if studio.menuTree == nil || form == nil {
		return
	}
	studio.menuByID = make(map[string]*rosaline.TreeNode)
	var build func(*designMenu) *rosaline.TreeNode
	build = func(menu *designMenu) *rosaline.TreeNode {
		children := make([]*rosaline.TreeNode, 0, len(menu.Children))
		for _, child := range menu.Children {
			if child != nil {
				children = append(children, build(child))
			}
		}
		label := string(menu.Kind)
		switch menu.Kind {
		case menuKindSeparator:
			label = "──────── Separator"
		default:
			text, handler, shortcut := menu.Text, menu.Handler, menu.Shortcut
			if action := studio.project.action(menu.Action); action != nil {
				text, handler, shortcut = action.Text, action.Handler, action.Shortcut
			}
			label += " - " + text
			if shortcut != "" {
				label += "  [" + shortcut + "]"
			}
			if handler != "" {
				label += "  → " + handler
			}
			if menu.Action != "" {
				label += "  {shared}"
			}
		}
		result := rosaline.Node(label, children...).WithValue(menu.ID).Expanded()
		studio.menuByID[menu.ID] = result
		return result
	}
	nodes := make([]*rosaline.TreeNode, 0, len(form.Menus))
	for _, menu := range form.Menus {
		if menu != nil {
			nodes = append(nodes, build(menu))
		}
	}
	studio.syncMenu = true
	studio.menuTree.SetNodes(nodes...)
	studio.syncMenuSelection()
	studio.syncMenu = false
}

func (studio *studio) syncMenuSelection() {
	if studio.menuTree == nil {
		return
	}
	studio.syncMenu = true
	if node := studio.menuByID[studio.menuID]; node != nil {
		studio.menuTree.Select(node)
	}
	studio.syncMenu = false
}

func (studio *studio) showMenuDesigner() {
	if !studio.confirmOpenHandler("") {
		return
	}
	if studio.workspace != nil {
		studio.workspace.Select(workspaceMenuTab)
	}
}

func (studio *studio) addTopMenu() {
	form := studio.activeForm()
	if form == nil {
		return
	}
	before := designSnapshot(studio.project)
	menu := studio.project.addTopMenu(form)
	studio.menuID = menu.ID
	studio.commitChange(before, "Added top-level menu")
	studio.showMenuDesigner()
}

func (studio *studio) addMenuEntry(kind menuKind) {
	form := studio.activeForm()
	if form == nil {
		return
	}
	if len(form.Menus) == 0 {
		studio.addTopMenu()
		form = studio.activeForm()
	}
	before := designSnapshot(studio.project)
	menu, err := studio.project.addMenuChild(form, studio.menuID, kind)
	if err != nil {
		studio.status = "Could not add menu entry: " + err.Error()
		return
	}
	studio.menuID = menu.ID
	studio.commitChange(before, "Added "+strings.ToLower(string(kind)))
	studio.showMenuDesigner()
}

func (studio *studio) moveMenuEntry(difference int) {
	if studio.selectedMenu() == nil {
		studio.status = "Select a menu entry first"
		return
	}
	before := designSnapshot(studio.project)
	if err := moveDesignMenu(studio.activeForm(), studio.menuID, difference); err != nil {
		studio.status = "Could not move menu entry: " + err.Error()
		return
	}
	studio.commitChange(before, "Moved menu entry")
}

func (studio *studio) deleteMenuEntry() {
	menu := studio.selectedMenu()
	if menu == nil {
		studio.status = "Select a menu entry first"
		return
	}
	message := "Delete " + strings.ToLower(string(menu.Kind)) + " " + defaultText(menu.Text, "separator") + "?"
	if len(menu.Children) != 0 {
		message = fmt.Sprintf("Delete %s and its %d contained entries?", menu.Text, len(menu.Children))
	}
	if !rosaline.Confirm("Delete menu entry?", message+" You can undo this change.") {
		studio.status = "Menu deletion cancelled"
		return
	}
	before := designSnapshot(studio.project)
	if err := removeDesignMenu(studio.activeForm(), studio.menuID); err != nil {
		studio.status = "Could not delete menu entry: " + err.Error()
		return
	}
	studio.menuID = ""
	studio.commitChange(before, "Deleted menu entry - use Primary+Z to undo")
}

func (studio *studio) applyMenuInspector() {
	menu := studio.selectedMenu()
	if menu == nil {
		studio.status = "Select a menu entry first"
		return
	}
	before := designSnapshot(studio.project)
	switch menu.Kind {
	case menuKindMenu:
		caption := strings.TrimSpace(studio.menu.Caption)
		if caption == "" {
			studio.status = "A menu caption cannot be empty"
			return
		}
		menu.Text = caption
		menu.Action = ""
		menu.Handler = ""
		menu.Shortcut = ""
	case menuKindItem:
		actionID := studio.menuActionID(studio.menu.Action)
		if studio.menu.Action != "" && studio.menu.Action != noSharedAction && actionID == "" {
			studio.status = "Choose an existing shared action"
			return
		}
		if actionID != "" {
			menu.Action = actionID
			studio.commitChange(before, "Linked menu item to shared action")
			return
		}
		caption := strings.TrimSpace(studio.menu.Caption)
		handler := strings.TrimSpace(studio.menu.Handler)
		shortcut := strings.TrimSpace(studio.menu.Shortcut)
		if caption == "" {
			studio.status = "A menu item caption cannot be empty"
			return
		}
		if handler != "" && !validHandlerName(handler) {
			studio.status = "Handler names must be valid Go identifiers"
			return
		}
		if err := validateMenuShortcut(shortcut); err != nil {
			studio.status = err.Error()
			return
		}
		menu.Action = ""
		menu.Text, menu.Handler, menu.Shortcut = caption, handler, shortcut
	case menuKindSeparator:
		studio.status = "Separators have no editable properties"
		return
	}
	studio.commitChange(before, "Updated menu properties")
}

func (studio *studio) editMenuHandler() {
	menu := studio.selectedMenu()
	if menu == nil || menu.Kind != menuKindItem {
		studio.status = "Select a menu item to edit its click event"
		return
	}
	if action := studio.project.action(menu.Action); action != nil {
		studio.actionID = action.ID
		studio.loadActionInspector()
		studio.editActionHandler()
		return
	}
	handler := strings.TrimSpace(studio.menu.Handler)
	if handler == "" {
		handler = strings.TrimSpace(menu.Handler)
	}
	if handler == "" {
		handler = defaultMenuHandlerName(menu)
	}
	if !validHandlerName(handler) {
		studio.status = "Handler names must be valid Go identifiers"
		return
	}
	if !studio.confirmOpenHandler(handler) {
		return
	}
	if body, exists := studio.project.Handlers[handler]; exists {
		if err := validateHandler(handler, body, false); err != nil {
			studio.status = "That existing handler has an incompatible Go signature"
			return
		}
	}
	before := designSnapshot(studio.project)
	changed := menu.Handler != handler
	menu.Handler = handler
	if studio.project.Handlers == nil {
		studio.project.Handlers = make(map[string]string)
	}
	if _, exists := studio.project.Handlers[handler]; !exists {
		studio.project.Handlers[handler] = defaultMenuHandlerBody(menu)
		changed = true
	}
	if changed {
		studio.commitChange(before, "Assigned menu click to "+handler)
	}
	studio.codeHandler = handler
	studio.codeReturnsBool = false
	studio.codeBody = studio.project.Handlers[handler]
	studio.menu.Handler = handler
	studio.codeEditor.SetText(studio.codeBody)
	studio.codeEditor.MarkSaved()
	if studio.workspace != nil {
		studio.workspace.Select(workspaceCodeTab)
	}
	studio.codeEditor.Focus()
	studio.status = "Editing " + handler
}

func (studio *studio) clearMenuHandler() {
	menu := studio.selectedMenu()
	if menu == nil || menu.Kind != menuKindItem {
		studio.status = "Select a menu item first"
		return
	}
	if menu.Action != "" {
		studio.status = "This item uses a shared action; edit or unlink the action instead"
		return
	}
	if menu.Handler == "" {
		studio.status = "That menu item has no click handler"
		return
	}
	before := designSnapshot(studio.project)
	menu.Handler = ""
	studio.commitChange(before, "Cleared menu click handler")
}
