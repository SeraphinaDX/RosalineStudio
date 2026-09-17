// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"strings"
)

type menuKind string

const (
	menuKindMenu      menuKind = "Menu"
	menuKindItem      menuKind = "Item"
	menuKindSeparator menuKind = "Separator"
)

type designMenu struct {
	ID       string        `json:"id"`
	Kind     menuKind      `json:"kind"`
	Action   string        `json:"action,omitempty"`
	Text     string        `json:"text,omitempty"`
	Handler  string        `json:"handler,omitempty"`
	Shortcut string        `json:"shortcut,omitempty"`
	Children []*designMenu `json:"children,omitempty"`
}

func cloneDesignMenus(source []*designMenu) []*designMenu {
	result := make([]*designMenu, 0, len(source))
	for _, menu := range source {
		if menu == nil {
			result = append(result, nil)
			continue
		}
		clone := *menu
		clone.Children = cloneDesignMenus(menu.Children)
		result = append(result, &clone)
	}
	return result
}

func (project *designProject) nextMenuID() string {
	used := make(map[string]bool)
	if project != nil {
		for _, form := range project.Forms {
			if form != nil {
				visitDesignMenus(form.Menus, func(menu *designMenu) { used[menu.ID] = true })
			}
		}
	}
	for number := 1; ; number++ {
		id := fmt.Sprintf("menu-%d", number)
		if !used[id] {
			return id
		}
	}
}

func (project *designProject) prepareCopiedMenus(menus []*designMenu) {
	used := make(map[string]bool)
	for _, form := range project.Forms {
		if form != nil {
			visitDesignMenus(form.Menus, func(menu *designMenu) { used[menu.ID] = true })
		}
	}
	next := 1
	visitDesignMenus(menus, func(menu *designMenu) {
		for used[fmt.Sprintf("menu-%d", next)] {
			next++
		}
		menu.ID = fmt.Sprintf("menu-%d", next)
		used[menu.ID] = true
		next++
	})
}

func visitDesignMenus(menus []*designMenu, visit func(*designMenu)) {
	for _, menu := range menus {
		if menu == nil {
			continue
		}
		visit(menu)
		visitDesignMenus(menu.Children, visit)
	}
}

func findDesignMenu(menus []*designMenu, id string) *designMenu {
	for _, menu := range menus {
		if menu == nil {
			continue
		}
		if menu.ID == id {
			return menu
		}
		if result := findDesignMenu(menu.Children, id); result != nil {
			return result
		}
	}
	return nil
}

func menuLocation(menus []*designMenu, id string) (*designMenu, []*designMenu, int) {
	for index, menu := range menus {
		if menu == nil {
			continue
		}
		if menu.ID == id {
			return nil, menus, index
		}
		if parent, siblings, childIndex := menuChildLocation(menu, id); childIndex >= 0 {
			return parent, siblings, childIndex
		}
	}
	return nil, nil, -1
}

func menuChildLocation(parent *designMenu, id string) (*designMenu, []*designMenu, int) {
	for index, child := range parent.Children {
		if child == nil {
			continue
		}
		if child.ID == id {
			return parent, parent.Children, index
		}
		if foundParent, siblings, childIndex := menuChildLocation(child, id); childIndex >= 0 {
			return foundParent, siblings, childIndex
		}
	}
	return nil, nil, -1
}

func (project *designProject) addTopMenu(form *designForm) *designMenu {
	menu := &designMenu{ID: project.nextMenuID(), Kind: menuKindMenu, Text: "Menu"}
	form.Menus = append(form.Menus, menu)
	return menu
}

func (project *designProject) addMenuChild(form *designForm, selectedID string, kind menuKind) (*designMenu, error) {
	if form == nil {
		return nil, errors.New("there is no active form")
	}
	parent := findDesignMenu(form.Menus, selectedID)
	if parent == nil || parent.Kind != menuKindMenu {
		candidate, _, _ := menuLocation(form.Menus, selectedID)
		parent = candidate
	}
	if parent == nil || parent.Kind != menuKindMenu {
		return nil, errors.New("select a menu before adding an entry")
	}
	entry := &designMenu{ID: project.nextMenuID(), Kind: kind}
	switch kind {
	case menuKindMenu:
		entry.Text = "Submenu"
	case menuKindItem:
		entry.Text = "Menu Item"
	case menuKindSeparator:
	default:
		return nil, fmt.Errorf("unknown menu entry kind %q", kind)
	}
	parent.Children = append(parent.Children, entry)
	return entry, nil
}

func moveDesignMenu(form *designForm, id string, difference int) error {
	if form == nil {
		return errors.New("there is no active form")
	}
	parent, siblings, index := menuLocation(form.Menus, id)
	if index < 0 {
		return errors.New("menu entry was not found")
	}
	target := index + difference
	if target < 0 || target >= len(siblings) {
		return errors.New("menu entry is already at that edge")
	}
	siblings[index], siblings[target] = siblings[target], siblings[index]
	if parent == nil {
		form.Menus = siblings
	} else {
		parent.Children = siblings
	}
	return nil
}

func removeDesignMenu(form *designForm, id string) error {
	if form == nil {
		return errors.New("there is no active form")
	}
	parent, siblings, index := menuLocation(form.Menus, id)
	if index < 0 {
		return errors.New("menu entry was not found")
	}
	siblings = append(siblings[:index], siblings[index+1:]...)
	if parent == nil {
		form.Menus = siblings
	} else {
		parent.Children = siblings
	}
	return nil
}

func validateDesignMenus(menus []*designMenu, actions map[string]*designAction, seen map[string]bool) error {
	var validate func([]*designMenu, bool) error
	validate = func(entries []*designMenu, topLevel bool) error {
		for _, entry := range entries {
			if entry == nil {
				return errors.New("contains an empty entry")
			}
			if strings.TrimSpace(entry.ID) == "" || seen[entry.ID] {
				return fmt.Errorf("entry ID %q is empty or repeated", entry.ID)
			}
			seen[entry.ID] = true
			if topLevel && entry.Kind != menuKindMenu {
				return errors.New("top-level entries must be menus")
			}
			switch entry.Kind {
			case menuKindMenu:
				if strings.TrimSpace(entry.Text) == "" {
					return errors.New("a menu has an empty caption")
				}
				if entry.Action != "" || entry.Handler != "" || entry.Shortcut != "" {
					return fmt.Errorf("menu %q cannot have an action, handler, or shortcut", entry.Text)
				}
				if err := validate(entry.Children, false); err != nil {
					return err
				}
			case menuKindItem:
				if strings.TrimSpace(entry.Text) == "" {
					return errors.New("a menu item has an empty caption")
				}
				if len(entry.Children) != 0 {
					return fmt.Errorf("menu item %q cannot contain entries", entry.Text)
				}
				if entry.Action != "" && actions[entry.Action] == nil {
					return fmt.Errorf("menu item %q refers to missing action %q", entry.Text, entry.Action)
				}
				if entry.Handler != "" && !validHandlerName(entry.Handler) {
					return fmt.Errorf("menu item %q has invalid handler name %q", entry.Text, entry.Handler)
				}
				if err := validateMenuShortcut(entry.Shortcut); err != nil {
					return fmt.Errorf("menu item %q: %w", entry.Text, err)
				}
			case menuKindSeparator:
				if entry.Action != "" || entry.Text != "" || entry.Handler != "" || entry.Shortcut != "" || len(entry.Children) != 0 {
					return errors.New("a separator cannot have properties or children")
				}
			default:
				return fmt.Errorf("unknown entry kind %q", entry.Kind)
			}
		}
		return nil
	}
	return validate(menus, true)
}

func validateMenuShortcut(shortcut string) error {
	shortcut = strings.TrimSpace(shortcut)
	if shortcut == "" {
		return nil
	}
	parts := strings.Split(shortcut, "+")
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return fmt.Errorf("shortcut %q contains an empty key", shortcut)
		}
	}
	return nil
}

func defaultMenuHandlerName(menu *designMenu) string {
	if menu == nil {
		return "MenuItemClick"
	}
	return exportedIdentifier(menu.Text) + "Click"
}

func defaultMenuHandlerBody(menu *designMenu) string {
	label := "Menu item"
	if menu != nil && strings.TrimSpace(menu.Text) != "" {
		label = menu.Text
	}
	return fmt.Sprintf("rosaline.Message(%q, %q)", "Menu", label+" selected.")
}
