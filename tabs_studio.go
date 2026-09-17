// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"fmt"
	"strings"

	rosaline "github.com/SeraphinaDX/Rosaline"
)

func (studio *studio) activeTabPage(tabs *designNode) *designNode {
	if tabs == nil || tabs.Kind != kindTabs {
		return nil
	}
	if studio.tabPages == nil {
		studio.tabPages = make(map[string]string)
	}
	if id := studio.tabPages[tabs.ID]; id != "" {
		if index := tabPageIndex(tabs, id); index >= 0 {
			return tabs.Children[index]
		}
	}
	page := firstTabPage(tabs)
	if page != nil {
		studio.tabPages[tabs.ID] = page.ID
	}
	return page
}

func (studio *studio) selectedTabsAndPage() (*designNode, *designNode) {
	if studio == nil || studio.project == nil {
		return nil, nil
	}
	node := studio.project.find(studio.selectedID)
	if node != nil && node.Kind == kindTabs {
		return node, studio.activeTabPage(node)
	}
	tabs, page := studio.project.tabPageContaining(studio.selectedID)
	if tabs != nil && page != nil {
		studio.tabPages[tabs.ID] = page.ID
	}
	return tabs, page
}

func (studio *studio) activateTabPageFor(id string) {
	if studio == nil || studio.project == nil {
		return
	}
	tabs, page := studio.project.tabPageContaining(id)
	if tabs != nil && page != nil {
		if studio.tabPages == nil {
			studio.tabPages = make(map[string]string)
		}
		studio.tabPages[tabs.ID] = page.ID
	}
}

func (studio *studio) loadPageInspector() {
	_, page := studio.selectedTabsAndPage()
	studio.pageTitle = ""
	if page != nil {
		studio.pageTitle = page.Text
	}
}

func (studio *studio) selectedPageLabel() string {
	tabs, page := studio.selectedTabsAndPage()
	if tabs == nil {
		return "No tab page selected"
	}
	if page == nil {
		return tabs.Component + " has no pages"
	}
	return fmt.Sprintf("%s - %s (%d of %d)", tabs.Component, defaultText(page.Text, "Page"), tabPageIndex(tabs, page.ID)+1, len(tabs.Children))
}

func (studio *studio) addTabPage() {
	tabs, _ := studio.selectedTabsAndPage()
	if tabs == nil {
		studio.status = "Select a Tabs component or something inside one first"
		return
	}
	before := designSnapshot(studio.project)
	page, err := studio.project.addTabPage(tabs, "")
	if err != nil {
		studio.status = "Could not add page: " + err.Error()
		return
	}
	studio.tabPages[tabs.ID] = page.ID
	studio.selectedID = page.ID
	studio.commitChange(before, "Added tab page")
}

func (studio *studio) duplicateTabPage() {
	tabs, page := studio.selectedTabsAndPage()
	if tabs == nil || page == nil {
		studio.status = "Select a tab page first"
		return
	}
	before := designSnapshot(studio.project)
	clone, err := studio.project.duplicate(page.ID)
	if err != nil {
		studio.status = "Could not duplicate page: " + err.Error()
		return
	}
	clone.Text = defaultText(page.Text, "Page") + " Copy"
	studio.tabPages[tabs.ID] = clone.ID
	studio.selectedID = clone.ID
	studio.commitChange(before, "Duplicated tab page")
}

func (studio *studio) moveTabPage(difference int) {
	tabs, page := studio.selectedTabsAndPage()
	if tabs == nil || page == nil {
		studio.status = "Select a tab page first"
		return
	}
	before := designSnapshot(studio.project)
	if err := studio.project.moveBy(page.ID, difference); err != nil {
		studio.status = "Could not move page: " + err.Error()
		return
	}
	studio.tabPages[tabs.ID] = page.ID
	studio.selectedID = page.ID
	studio.commitChange(before, "Moved tab page")
}

func (studio *studio) deleteTabPage() {
	tabs, page := studio.selectedTabsAndPage()
	if tabs == nil || page == nil {
		studio.status = "Select a tab page first"
		return
	}
	if len(tabs.Children) <= 1 {
		studio.status = "Tabs must keep at least one page"
		return
	}
	message := "Delete " + defaultText(page.Text, "this page") + "?"
	if count := containedWidgetCount(page); count != 0 {
		message = fmt.Sprintf("Delete %s and its %d contained widget", defaultText(page.Text, "this page"), count)
		if count != 1 {
			message += "s"
		}
		message += "?"
	}
	if !rosaline.Confirm("Delete tab page?", message+" You can undo this change.") {
		studio.status = "Page deletion cancelled"
		return
	}
	before := designSnapshot(studio.project)
	index := tabPageIndex(tabs, page.ID)
	if err := studio.project.remove(page.ID); err != nil {
		studio.status = "Could not delete page: " + err.Error()
		return
	}
	index = min(index, len(tabs.Children)-1)
	next := tabs.Children[index]
	studio.tabPages[tabs.ID] = next.ID
	studio.selectedID = next.ID
	studio.commitChange(before, "Deleted tab page - use Primary+Z to undo")
}

func (studio *studio) applyPageTitle() {
	_, page := studio.selectedTabsAndPage()
	if page == nil {
		studio.status = "Select a tab page first"
		return
	}
	title := strings.TrimSpace(studio.pageTitle)
	if title == "" {
		studio.status = "A tab page title cannot be empty"
		return
	}
	before := designSnapshot(studio.project)
	page.Text = title
	studio.commitChange(before, "Renamed tab page")
}
