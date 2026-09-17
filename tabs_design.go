// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"strings"
)

func validateWidgetContainment(parent, child widgetKind) error {
	if parent == kindTabs {
		if child != kindTabPage {
			return errors.New("Tabs can contain only TabPage entries")
		}
		return nil
	}
	if child == kindTabPage {
		return errors.New("TabPage entries must be direct children of Tabs")
	}
	return nil
}

func (project *designProject) initializeTabs(tabs *designNode) {
	if project == nil || tabs == nil || tabs.Kind != kindTabs || len(tabs.Children) != 0 {
		return
	}
	project.addTabPage(tabs, "General")
	project.addTabPage(tabs, "Advanced")
}

func (project *designProject) addTabPage(tabs *designNode, title string) (*designNode, error) {
	if project == nil || tabs == nil || tabs.Kind != kindTabs {
		return nil, errors.New("select a Tabs component first")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = fmt.Sprintf("Page %d", len(tabs.Children)+1)
	}
	page := defaultNode(kindTabPage, project.nextID())
	page.Text = title
	page.Component = project.nextComponentNameFor(title + "Page")
	tabs.Children = append(tabs.Children, page)
	return page, nil
}

func tabPageIndex(tabs *designNode, id string) int {
	if tabs == nil || tabs.Kind != kindTabs {
		return -1
	}
	for index, page := range tabs.Children {
		if page != nil && page.Kind == kindTabPage && page.ID == id {
			return index
		}
	}
	return -1
}

func firstTabPage(tabs *designNode) *designNode {
	if tabs == nil || tabs.Kind != kindTabs {
		return nil
	}
	for _, page := range tabs.Children {
		if page != nil && page.Kind == kindTabPage {
			return page
		}
	}
	return nil
}

func (project *designProject) tabPageContaining(id string) (tabs, page *designNode) {
	if project == nil {
		return nil, nil
	}
	node := project.find(id)
	for node != nil {
		if node.Kind == kindTabPage {
			parent := project.parentOf(node.ID)
			if parent != nil && parent.Kind == kindTabs {
				return parent, node
			}
			return nil, nil
		}
		node = project.parentOf(node.ID)
	}
	return nil, nil
}

func (project *designProject) tabsForSelection(id string) (*designNode, *designNode) {
	if project == nil {
		return nil, nil
	}
	node := project.find(id)
	if node != nil && node.Kind == kindTabs {
		return node, firstTabPage(node)
	}
	return project.tabPageContaining(id)
}
