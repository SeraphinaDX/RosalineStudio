// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

const designVersion = 6

type widgetKind string

const (
	kindColumn      widgetKind = "Column"
	kindRow         widgetKind = "Row"
	kindGrid        widgetKind = "Grid"
	kindStack       widgetKind = "Stack"
	kindCard        widgetKind = "Card"
	kindScroll      widgetKind = "Scroll"
	kindTabs        widgetKind = "Tabs"
	kindTabPage     widgetKind = "TabPage"
	kindLabel       widgetKind = "Label"
	kindImage       widgetKind = "Image"
	kindButton      widgetKind = "Button"
	kindTextBox     widgetKind = "TextBox"
	kindTextArea    widgetKind = "TextArea"
	kindCheckBox    widgetKind = "CheckBox"
	kindComboBox    widgetKind = "ComboBox"
	kindRadioGroup  widgetKind = "RadioGroup"
	kindList        widgetKind = "List"
	kindTable       widgetKind = "Table"
	kindTree        widgetKind = "Tree"
	kindSlider      widgetKind = "Slider"
	kindProgressBar widgetKind = "ProgressBar"
	kindSpacer      widgetKind = "Spacer"
)

var paletteKinds = []widgetKind{
	kindLabel,
	kindImage,
	kindButton,
	kindTextBox,
	kindTextArea,
	kindCheckBox,
	kindComboBox,
	kindRadioGroup,
	kindList,
	kindTable,
	kindTree,
	kindSlider,
	kindProgressBar,
	kindSpacer,
	kindColumn,
	kindRow,
	kindGrid,
	kindStack,
	kindCard,
	kindScroll,
	kindTabs,
}

type designNode struct {
	ID         string            `json:"id"`
	Kind       widgetKind        `json:"kind"`
	Component  string            `json:"component,omitempty"`
	Text       string            `json:"text,omitempty"`
	Name       string            `json:"name,omitempty"`
	Asset      string            `json:"asset,omitempty"`
	Events     map[string]string `json:"events,omitempty"`
	Options    []string          `json:"options,omitempty"`
	Data       []string          `json:"data,omitempty"`
	Children   []*designNode     `json:"children,omitempty"`
	Gap        int               `json:"gap,omitempty"`
	Padding    int               `json:"padding,omitempty"`
	Columns    int               `json:"columns,omitempty"`
	Width      int               `json:"width,omitempty"`
	Height     int               `json:"height,omitempty"`
	Minimum    float64           `json:"minimum,omitempty"`
	Maximum    float64           `json:"maximum,omitempty"`
	Step       float64           `json:"step,omitempty"`
	Expand     bool              `json:"expand,omitempty"`
	Primary    bool              `json:"primary,omitempty"`
	Bold       bool              `json:"bold,omitempty"`
	Password   bool              `json:"password,omitempty"`
	Vertical   bool              `json:"vertical,omitempty"`
	Horizontal bool              `json:"horizontal,omitempty"`
}

type designProject struct {
	Version  int               `json:"version"`
	Module   string            `json:"module"`
	Forms    []*designForm     `json:"forms"`
	Handlers map[string]string `json:"handlers,omitempty"`
}

type designForm struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Title   string            `json:"title"`
	Width   int               `json:"width"`
	Height  int               `json:"height"`
	Padding int               `json:"padding"`
	Theme   string            `json:"theme"`
	Root    *designNode       `json:"root"`
	Events  map[string]string `json:"events,omitempty"`
	Menus   []*designMenu     `json:"menus,omitempty"`
}

type legacyDesignProject struct {
	Version  int               `json:"version"`
	Module   string            `json:"module"`
	Title    string            `json:"title"`
	Width    int               `json:"width"`
	Height   int               `json:"height"`
	Padding  int               `json:"padding"`
	Theme    string            `json:"theme"`
	Root     *designNode       `json:"root"`
	Handlers map[string]string `json:"handlers,omitempty"`
}

func newProject() *designProject {
	return &designProject{
		Version: designVersion,
		Module:  "example.com/myapp",
		Handlers: map[string]string{
			"ContinueClick": `rosaline.Message("Your application", "Continue clicked.")`,
		},
		Forms: []*designForm{
			{
				ID: "form-1", Name: "MainForm", Title: "My Rosaline App",
				Width: 720, Height: 520, Padding: 16, Theme: "Rosaline",
				Root: &designNode{
					ID: "root", Kind: kindColumn, Component: "MainLayout",
					Gap: 10, Padding: 12, Expand: true,
					Children: []*designNode{
						{ID: "node-1", Kind: kindLabel, Component: "WelcomeLabel", Text: "Welcome to Rosaline", Bold: true},
						{ID: "node-2", Kind: kindTextBox, Component: "NameTextBox", Text: "Your name", Name: "Name"},
						{ID: "node-3", Kind: kindButton, Component: "ContinueButton", Text: "Continue", Events: map[string]string{"OnClick": "ContinueClick"}, Primary: true},
					},
				},
			},
		},
	}
}

func (project *designProject) mainForm() *designForm {
	if project == nil || len(project.Forms) == 0 {
		return nil
	}
	return project.Forms[0]
}

func (project *designProject) form(id string) *designForm {
	if project == nil {
		return nil
	}
	for _, form := range project.Forms {
		if form != nil && form.ID == id {
			return form
		}
	}
	return nil
}

func (project *designProject) formContainingNode(id string) *designForm {
	if project == nil {
		return nil
	}
	for _, form := range project.Forms {
		if form != nil && findNode(form.Root, id) != nil {
			return form
		}
	}
	return nil
}

func (project *designProject) visitRoots(visit func(*designNode)) {
	if project == nil {
		return
	}
	for _, form := range project.Forms {
		if form != nil {
			visit(form.Root)
		}
	}
}

func defaultNode(kind widgetKind, id string) *designNode {
	node := &designNode{ID: id, Kind: kind, Gap: 8}
	switch kind {
	case kindColumn, kindRow:
		node.Padding = 8
	case kindGrid:
		node.Columns = 2
		node.Padding = 8
	case kindStack:
	case kindCard:
		node.Padding = 14
	case kindScroll:
		node.Width, node.Height = 480, 300
		node.Expand = true
	case kindTabs:
		node.Expand = true
	case kindTabPage:
		node.Text = "Page"
		node.Padding = 12
		node.Gap = 10
		node.Expand = true
	case kindLabel:
		node.Text = "Label"
	case kindImage:
		node.Text = "Choose an image"
		node.Width, node.Height = 320, 200
	case kindButton:
		node.Text = "Button"
	case kindTextBox:
		node.Text = "Enter text"
		node.Name = "Text"
	case kindTextArea:
		node.Name = "Notes"
		node.Height = 150
		node.Expand = true
	case kindCheckBox:
		node.Text = "Enabled"
		node.Name = "Enabled"
	case kindComboBox:
		node.Name = "Choice"
		node.Options = []string{"One", "Two", "Three"}
	case kindRadioGroup:
		node.Name = "DisplayMode"
		node.Data = []string{"Automatic = auto", "Light = light", "Dark = dark"}
	case kindList:
		node.Data = []string{"Rose", "Lavender", "Midnight"}
		node.Expand = true
	case kindTable:
		node.Data = []string{"Name | Type | Status", "Rosaline | Library | Ready", "Studio | Application | Editing"}
		node.Expand = true
	case kindTree:
		node.Data = []string{"Project", "Project/Forms", "Project/Forms/MainForm", "Project/Assets", "Dependencies"}
		node.Expand = true
	case kindSlider:
		node.Name = "Value"
		node.Maximum = 100
		node.Step = 1
	case kindProgressBar:
		node.Name = "Progress"
		node.Maximum = 100
	case kindSpacer:
		node.Width, node.Height = 24, 24
	}
	return node
}

func (kind widgetKind) container() bool {
	switch kind {
	case kindColumn, kindRow, kindGrid, kindStack, kindCard, kindScroll, kindTabs, kindTabPage:
		return true
	default:
		return false
	}
}

func (project *designProject) find(id string) *designNode {
	if project == nil {
		return nil
	}
	for _, form := range project.Forms {
		if form != nil {
			if found := findNode(form.Root, id); found != nil {
				return found
			}
		}
	}
	return nil
}

func findNode(node *designNode, id string) *designNode {
	if node == nil {
		return nil
	}
	if node.ID == id {
		return node
	}
	for _, child := range node.Children {
		if found := findNode(child, id); found != nil {
			return found
		}
	}
	return nil
}

func (project *designProject) parentOf(id string) *designNode {
	if project == nil {
		return nil
	}
	for _, form := range project.Forms {
		if form == nil || form.Root == nil || form.Root.ID == id {
			continue
		}
		if found := findParent(form.Root, id); found != nil {
			return found
		}
	}
	return nil
}

func findParent(node *designNode, id string) *designNode {
	if node == nil {
		return nil
	}
	for _, child := range node.Children {
		if child != nil && child.ID == id {
			return node
		}
		if found := findParent(child, id); found != nil {
			return found
		}
	}
	return nil
}

func (project *designProject) nextID() string {
	maximum := 0
	var visit func(*designNode)
	visit = func(node *designNode) {
		if node == nil {
			return
		}
		if strings.HasPrefix(node.ID, "node-") {
			if value, err := strconv.Atoi(strings.TrimPrefix(node.ID, "node-")); err == nil {
				maximum = max(maximum, value)
			}
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	project.visitRoots(visit)
	return fmt.Sprintf("node-%d", maximum+1)
}

func (project *designProject) addNear(selectedID string, kind widgetKind) (*designNode, error) {
	selected := project.find(selectedID)
	if selected == nil {
		if main := project.mainForm(); main != nil {
			selected = main.Root
		}
	}
	if selected == nil {
		return nil, errors.New("the project has no form")
	}
	parent := selected
	if !parent.Kind.container() {
		parent = project.parentOf(parent.ID)
	}
	if parent == nil {
		return nil, errors.New("select a container before adding a widget")
	}
	if parent.Kind == kindTabs {
		parent = firstTabPage(parent)
		if parent == nil {
			return nil, errors.New("add a tab page before adding a widget")
		}
	}
	if err := validateWidgetContainment(parent.Kind, kind); err != nil {
		return nil, err
	}
	if (parent.Kind == kindCard || parent.Kind == kindScroll) && len(parent.Children) != 0 {
		return nil, fmt.Errorf("%s can contain one widget; select its parent instead", parent.Kind)
	}
	node := defaultNode(kind, project.nextID())
	node.Component = project.nextComponentName(kind)
	parent.Children = append(parent.Children, node)
	if kind == kindTabs {
		project.initializeTabs(node)
	}
	return node, nil
}

func (project *designProject) isFormRoot(id string) bool {
	form := project.formContainingNode(id)
	return form != nil && form.Root != nil && form.Root.ID == id
}

func (project *designProject) nextFormID() string {
	maximum := 0
	for _, form := range project.Forms {
		if form != nil && strings.HasPrefix(form.ID, "form-") {
			if value, err := strconv.Atoi(strings.TrimPrefix(form.ID, "form-")); err == nil {
				maximum = max(maximum, value)
			}
		}
	}
	return fmt.Sprintf("form-%d", maximum+1)
}

func (project *designProject) nextFormName(base string) string {
	used := make(map[string]bool)
	for _, form := range project.Forms {
		if form != nil {
			used[form.Name] = true
		}
	}
	return uniqueGeneratedName(exportedIdentifier(base), used)
}

func (project *designProject) addForm() *designForm {
	name := project.nextFormName("Form2")
	form := &designForm{
		ID: project.nextFormID(), Name: name, Title: "New Form",
		Width: 640, Height: 420, Padding: 16, Theme: "Rosaline",
		Root: &designNode{
			ID: project.nextID(), Kind: kindColumn,
			Component: project.nextComponentNameFor(name + "Layout"),
			Gap:       10, Padding: 12, Expand: true,
		},
	}
	project.Forms = append(project.Forms, form)
	return form
}

func (project *designProject) duplicateForm(id string) (*designForm, error) {
	source := project.form(id)
	if source == nil {
		return nil, errors.New("form was not found")
	}
	clone := *source
	clone.ID = project.nextFormID()
	clone.Name = project.nextFormName(source.Name)
	clone.Title = source.Title + " Copy"
	clone.Root = cloneDesignNode(source.Root)
	clone.Events = cloneStringMap(source.Events)
	clone.Menus = cloneDesignMenus(source.Menus)
	project.prepareCopiedSubtree(clone.Root)
	project.prepareCopiedMenus(clone.Menus)
	project.Forms = append(project.Forms, &clone)
	return &clone, nil
}

func (project *designProject) removeForm(id string) error {
	if project == nil || len(project.Forms) == 0 {
		return errors.New("the project has no forms")
	}
	if project.Forms[0] != nil && project.Forms[0].ID == id {
		return errors.New("the main form cannot be deleted")
	}
	for index, form := range project.Forms {
		if form != nil && form.ID == id {
			project.Forms = append(project.Forms[:index], project.Forms[index+1:]...)
			return nil
		}
	}
	return errors.New("form was not found")
}

func cloneStringMap(source map[string]string) map[string]string {
	if source == nil {
		return nil
	}
	clone := make(map[string]string, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneDesignNode(node *designNode) *designNode {
	if node == nil {
		return nil
	}
	clone := *node
	clone.Options = append([]string(nil), node.Options...)
	clone.Data = append([]string(nil), node.Data...)
	clone.Children = make([]*designNode, 0, len(node.Children))
	clone.Events = cloneStringMap(node.Events)
	for _, child := range node.Children {
		clone.Children = append(clone.Children, cloneDesignNode(child))
	}
	return &clone
}

// insertCopy places an independent copy inside a selected container or just
// after a selected control. IDs, component names, and bound state names are
// made unique while event handler assignments remain shared intentionally.
func (project *designProject) insertCopy(selectedID string, source *designNode) (*designNode, error) {
	if project == nil || project.mainForm() == nil || source == nil {
		return nil, errors.New("there is no copied widget to paste")
	}
	selected := project.find(selectedID)
	if selected == nil {
		selected = project.mainForm().Root
	}
	parent := selected
	insertIndex := len(parent.Children)
	if !selected.Kind.container() {
		parent = project.parentOf(selected.ID)
		if parent == nil {
			return nil, errors.New("select a container or a child of one")
		}
		for index, child := range parent.Children {
			if child != nil && child.ID == selected.ID {
				insertIndex = index + 1
				break
			}
		}
	}
	if source.Kind == kindTabPage {
		selectedPage := selected
		if selected.Kind != kindTabPage {
			_, selectedPage = project.tabPageContaining(selected.ID)
		}
		if selectedPage != nil {
			parent = project.parentOf(selectedPage.ID)
			for index, child := range parent.Children {
				if child != nil && child.ID == selectedPage.ID {
					insertIndex = index + 1
					break
				}
			}
		}
	} else if parent.Kind == kindTabs {
		parent = firstTabPage(parent)
		insertIndex = len(parent.Children)
	}
	if err := validateWidgetContainment(parent.Kind, source.Kind); err != nil {
		return nil, err
	}
	if (parent.Kind == kindCard || parent.Kind == kindScroll) && len(parent.Children) != 0 {
		return nil, fmt.Errorf("%s can contain only one widget", parent.Kind)
	}

	clone := cloneDesignNode(source)
	project.prepareCopiedSubtree(clone)
	parent.Children = append(parent.Children, nil)
	copy(parent.Children[insertIndex+1:], parent.Children[insertIndex:])
	parent.Children[insertIndex] = clone
	return clone, nil
}

func (project *designProject) duplicate(id string) (*designNode, error) {
	node := project.find(id)
	if node == nil {
		return nil, errors.New("select a widget before duplicating")
	}
	if project.isFormRoot(node.ID) {
		return nil, errors.New("the root layout cannot be duplicated")
	}
	parent := project.parentOf(id)
	if parent == nil {
		return nil, errors.New("widget has no parent")
	}
	if parent.Kind == kindCard || parent.Kind == kindScroll {
		return nil, fmt.Errorf("%s can contain only one widget", parent.Kind)
	}
	insertIndex := len(parent.Children)
	for index, child := range parent.Children {
		if child != nil && child.ID == id {
			insertIndex = index + 1
			break
		}
	}
	clone := cloneDesignNode(node)
	project.prepareCopiedSubtree(clone)
	parent.Children = append(parent.Children, nil)
	copy(parent.Children[insertIndex+1:], parent.Children[insertIndex:])
	parent.Children[insertIndex] = clone
	return clone, nil
}

func (project *designProject) prepareCopiedSubtree(node *designNode) {
	usedComponents := make(map[string]bool)
	usedState := make(map[string]bool)
	nextID := 1
	var collect func(*designNode)
	collect = func(current *designNode) {
		if current == nil {
			return
		}
		usedComponents[current.Component] = true
		if stateType(current.Kind) != "" {
			uniqueGeneratedName(exportedIdentifier(current.Name), usedState)
		}
		if strings.HasPrefix(current.ID, "node-") {
			if value, err := strconv.Atoi(strings.TrimPrefix(current.ID, "node-")); err == nil {
				nextID = max(nextID, value+1)
			}
		}
		for _, child := range current.Children {
			collect(child)
		}
	}
	project.visitRoots(collect)

	var prepare func(*designNode)
	prepare = func(current *designNode) {
		if current == nil {
			return
		}
		current.ID = fmt.Sprintf("node-%d", nextID)
		nextID++
		current.Component = uniqueCopiedName(current.Component, usedComponents)
		if stateType(current.Kind) != "" {
			current.Name = uniqueGeneratedName(exportedIdentifier(current.Name), usedState)
		}
		for _, child := range current.Children {
			prepare(child)
		}
	}
	prepare(node)
}

func uniqueCopiedName(base string, used map[string]bool) string {
	base = exportedIdentifier(base)
	if base == "" || base == "Value" {
		base = "Widget"
	}
	return uniqueGeneratedName(base, used)
}

func uniqueGeneratedName(base string, used map[string]bool) string {
	if !used[base] {
		used[base] = true
		return base
	}
	prefix := strings.TrimRightFunc(base, unicode.IsDigit)
	suffix := 2
	if prefix != base {
		if number, err := strconv.Atoi(strings.TrimPrefix(base, prefix)); err == nil {
			suffix = number + 1
		}
		base = prefix
	}
	for ; ; suffix++ {
		candidate := base + strconv.Itoa(suffix)
		if !used[candidate] {
			used[candidate] = true
			return candidate
		}
	}
}

func (project *designProject) nextComponentName(kind widgetKind) string {
	return nextAvailableName(exportedIdentifier(string(kind)), project.usedComponentNames())
}

func (project *designProject) nextComponentNameFor(base string) string {
	return uniqueGeneratedName(exportedIdentifier(base), project.usedComponentNames())
}

func (project *designProject) usedComponentNames() map[string]bool {
	used := make(map[string]bool)
	var visit func(*designNode)
	visit = func(node *designNode) {
		if node == nil {
			return
		}
		if node.Component != "" {
			used[node.Component] = true
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	if project != nil {
		project.visitRoots(visit)
	}
	return used
}

func nextAvailableName(base string, used map[string]bool) string {
	base = exportedIdentifier(base)
	if base == "" || base == "Value" {
		base = "Widget"
	}
	for suffix := 1; ; suffix++ {
		candidate := base + strconv.Itoa(suffix)
		if !used[candidate] {
			used[candidate] = true
			return candidate
		}
	}
}

func validComponentName(name string) bool {
	name = strings.TrimSpace(name)
	first, _ := utf8.DecodeRuneInString(name)
	return validHandlerName(name) && unicode.IsUpper(first)
}

func (project *designProject) componentNameInUse(name, exceptID string) bool {
	var found bool
	var visit func(*designNode)
	visit = func(node *designNode) {
		if node == nil || found {
			return
		}
		if node.ID != exceptID && node.Component == name {
			found = true
			return
		}
		for _, child := range node.Children {
			visit(child)
		}
	}
	if project != nil {
		project.visitRoots(visit)
	}
	return found
}

func (project *designProject) ensureComponentNames() {
	if project == nil || project.mainForm() == nil {
		return
	}
	used := make(map[string]bool)
	var collect func(*designNode)
	collect = func(node *designNode) {
		if node == nil {
			return
		}
		if node.Component != "" {
			used[node.Component] = true
		}
		for _, child := range node.Children {
			collect(child)
		}
	}
	project.visitRoots(collect)
	var fill func(*designNode)
	fill = func(node *designNode) {
		if node == nil {
			return
		}
		if node.Component == "" {
			node.Component = nextAvailableName(exportedIdentifier(string(node.Kind)), used)
		}
		for _, child := range node.Children {
			fill(child)
		}
	}
	project.visitRoots(fill)
}

func (project *designProject) remove(id string) error {
	if project == nil || project.mainForm() == nil || project.isFormRoot(id) {
		return errors.New("the root layout cannot be deleted")
	}
	parent := project.parentOf(id)
	if parent == nil {
		return errors.New("widget was not found")
	}
	if parent.Kind == kindTabs && len(parent.Children) == 1 {
		return errors.New("Tabs must keep at least one page")
	}
	for index, child := range parent.Children {
		if child != nil && child.ID == id {
			parent.Children = append(parent.Children[:index], parent.Children[index+1:]...)
			return nil
		}
	}
	return errors.New("widget was not found")
}

// selectionAfterRemoval returns the most natural remaining selection for a
// visual editor: the next sibling, the previous sibling, or finally the
// parent when the removed node was its only child.
func (project *designProject) selectionAfterRemoval(id string) string {
	if project == nil || project.mainForm() == nil {
		return ""
	}
	parent := project.parentOf(id)
	if parent == nil {
		if form := project.formContainingNode(id); form != nil && form.Root != nil {
			return form.Root.ID
		}
		return project.mainForm().Root.ID
	}
	for index, child := range parent.Children {
		if child == nil || child.ID != id {
			continue
		}
		if index+1 < len(parent.Children) {
			return parent.Children[index+1].ID
		}
		if index > 0 {
			return parent.Children[index-1].ID
		}
		return parent.ID
	}
	return parent.ID
}

func containedWidgetCount(node *designNode) int {
	if node == nil {
		return 0
	}
	count := 0
	for _, child := range node.Children {
		if child == nil {
			continue
		}
		count += 1 + containedWidgetCount(child)
	}
	return count
}

func (project *designProject) moveBy(id string, difference int) error {
	parent := project.parentOf(id)
	if parent == nil {
		return errors.New("the selected widget cannot be moved here")
	}
	for index, child := range parent.Children {
		if child == nil || child.ID != id {
			continue
		}
		target := index + difference
		if target < 0 || target >= len(parent.Children) {
			return errors.New("the widget is already at that edge")
		}
		parent.Children[index], parent.Children[target] = parent.Children[target], parent.Children[index]
		return nil
	}
	return errors.New("widget was not found")
}

func (project *designProject) moveTo(id, targetID string) error {
	if id == "" || targetID == "" || id == targetID || project.mainForm() == nil || project.isFormRoot(id) {
		return errors.New("choose a different destination")
	}
	moving := project.find(id)
	target := project.find(targetID)
	if project.formContainingNode(id) != project.formContainingNode(targetID) {
		return errors.New("move widgets within one form; use copy and paste between forms")
	}
	oldParent := project.parentOf(id)
	if moving == nil || target == nil || oldParent == nil {
		return errors.New("widget was not found")
	}
	if findNode(moving, targetID) != nil {
		return errors.New("a widget cannot be moved inside itself")
	}

	newParent := target
	insertIndex := len(target.Children)
	if !target.Kind.container() {
		newParent = project.parentOf(targetID)
		if newParent == nil {
			return errors.New("destination has no parent")
		}
		for index, child := range newParent.Children {
			if child != nil && child.ID == targetID {
				insertIndex = index
				break
			}
		}
	}
	if newParent.Kind == kindTabs && moving.Kind != kindTabPage {
		newParent = firstTabPage(newParent)
		if newParent == nil {
			return errors.New("destination Tabs has no page")
		}
		insertIndex = len(newParent.Children)
	}
	if err := validateWidgetContainment(newParent.Kind, moving.Kind); err != nil {
		return err
	}
	if (newParent.Kind == kindCard || newParent.Kind == kindScroll) && len(newParent.Children) != 0 && newParent != oldParent {
		return fmt.Errorf("%s can contain only one widget", newParent.Kind)
	}

	oldIndex := -1
	for index, child := range oldParent.Children {
		if child != nil && child.ID == id {
			oldIndex = index
			break
		}
	}
	if oldIndex < 0 {
		return errors.New("widget was not found")
	}
	oldParent.Children = append(oldParent.Children[:oldIndex], oldParent.Children[oldIndex+1:]...)
	if newParent == oldParent && oldIndex < insertIndex {
		insertIndex--
	}
	insertIndex = min(max(0, insertIndex), len(newParent.Children))
	newParent.Children = append(newParent.Children, nil)
	copy(newParent.Children[insertIndex+1:], newParent.Children[insertIndex:])
	newParent.Children[insertIndex] = moving
	return nil
}

func (project *designProject) validate() error {
	if project == nil || len(project.Forms) == 0 {
		return errors.New("the design has no forms")
	}
	if project.Version != designVersion {
		return fmt.Errorf("unsupported design version %d", project.Version)
	}
	if strings.TrimSpace(project.Module) == "" {
		return errors.New("the Go module path cannot be empty")
	}
	seen := make(map[string]bool)
	components := make(map[string]bool)
	formIDs := make(map[string]bool)
	formNames := make(map[string]bool)
	if err := validateProjectHandlers(project); err != nil {
		return err
	}
	var visit func(*designNode, *designNode) error
	visit = func(node, parent *designNode) error {
		if node == nil {
			return errors.New("the design contains an empty widget")
		}
		if strings.TrimSpace(node.ID) == "" || seen[node.ID] {
			return fmt.Errorf("widget ID %q is empty or repeated", node.ID)
		}
		seen[node.ID] = true
		if !validComponentName(node.Component) {
			return fmt.Errorf("widget %s has invalid component name %q", node.ID, node.Component)
		}
		if components[node.Component] {
			return fmt.Errorf("component name %q is repeated", node.Component)
		}
		components[node.Component] = true
		if !knownKind(node.Kind) {
			return fmt.Errorf("widget %s has unknown kind %q", node.ID, node.Kind)
		}
		if node.Kind == kindTabPage && parent == nil {
			return fmt.Errorf("tab page %s must be a direct child of Tabs", node.ID)
		}
		if !node.Kind.container() && len(node.Children) != 0 {
			return fmt.Errorf("%s cannot contain other widgets", node.Kind)
		}
		if parent != nil {
			if err := validateWidgetContainment(parent.Kind, node.Kind); err != nil {
				return fmt.Errorf("widget %s: %w", node.ID, err)
			}
		}
		if node.Kind == kindTabs && len(node.Children) == 0 {
			return errors.New("Tabs must contain at least one TabPage")
		}
		if node.Kind == kindTabPage && strings.TrimSpace(node.Text) == "" {
			return fmt.Errorf("tab page %s has an empty title", node.ID)
		}
		if (node.Kind == kindCard || node.Kind == kindScroll) && len(node.Children) > 1 {
			return fmt.Errorf("%s can contain only one widget", node.Kind)
		}
		for event, handler := range node.Events {
			if !supportsEvent(node.Kind, event) {
				return fmt.Errorf("%s does not support event %q", node.Kind, event)
			}
			if !validHandlerName(handler) {
				return fmt.Errorf("widget %s has invalid handler name %q", node.ID, handler)
			}
		}
		if node.Asset != "" && filepath.Base(node.Asset) != node.Asset {
			return fmt.Errorf("widget %s has invalid asset name %q", node.ID, node.Asset)
		}
		for _, child := range node.Children {
			if err := visit(child, node); err != nil {
				return err
			}
		}
		return nil
	}
	for index, form := range project.Forms {
		if form == nil || form.Root == nil {
			return fmt.Errorf("form %d has no root layout", index+1)
		}
		if strings.TrimSpace(form.ID) == "" || formIDs[form.ID] {
			return fmt.Errorf("form ID %q is empty or repeated", form.ID)
		}
		formIDs[form.ID] = true
		if !validComponentName(form.Name) || formNames[form.Name] {
			return fmt.Errorf("form name %q is invalid or repeated", form.Name)
		}
		formNames[form.Name] = true
		if strings.TrimSpace(form.Title) == "" {
			return fmt.Errorf("form %s has an empty title", form.Name)
		}
		for event, handler := range form.Events {
			if !supportsFormEvent(event) {
				return fmt.Errorf("form %s does not support event %q", form.Name, event)
			}
			if !validHandlerName(handler) {
				return fmt.Errorf("form %s has invalid handler name %q", form.Name, handler)
			}
		}
		if err := validateDesignMenus(form.Menus, seen); err != nil {
			return fmt.Errorf("form %s menu: %w", form.Name, err)
		}
		if err := visit(form.Root, nil); err != nil {
			return err
		}
	}
	return nil
}

func knownKind(kind widgetKind) bool {
	if kind == kindTabPage {
		return true
	}
	for _, candidate := range paletteKinds {
		if kind == candidate {
			return true
		}
	}
	return false
}

func saveDesign(path string, project *designProject) error {
	if err := project.validate(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return fmt.Errorf("encode design: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create design folder: %w", err)
	}
	if err := writeFileAtomically(path, data, 0o644); err != nil {
		return fmt.Errorf("save design: %w", err)
	}
	return nil
}

func loadDesign(path string) (*designProject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read design: %w", err)
	}
	var header struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return nil, fmt.Errorf("decode design: %w", err)
	}
	var project designProject
	switch header.Version {
	case 2:
		var legacy legacyDesignProject
		if err := decodeStrict(data, &legacy); err != nil {
			return nil, err
		}
		project = designProject{
			Version: designVersion,
			Module:  legacy.Module,
			Forms: []*designForm{{
				ID: "form-1", Name: "MainForm", Title: legacy.Title,
				Width: legacy.Width, Height: legacy.Height, Padding: legacy.Padding,
				Theme: legacy.Theme, Root: legacy.Root,
			}},
			Handlers: legacy.Handlers,
		}
	case 3, 4, 5, designVersion:
		if err := decodeStrict(data, &project); err != nil {
			return nil, err
		}
		project.Version = designVersion
	default:
		return nil, fmt.Errorf("unsupported design version %d", header.Version)
	}
	project.normalize()
	project.ensureComponentNames()
	if err := project.validate(); err != nil {
		return nil, err
	}
	return &project, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode design: %w", err)
	}
	return nil
}

func (project *designProject) normalize() {
	if project.Handlers == nil {
		project.Handlers = make(map[string]string)
	}
	for _, form := range project.Forms {
		if form != nil && form.Theme == "" {
			form.Theme = "Rosaline"
		}
	}
}

func designSnapshot(project *designProject) []byte {
	data, _ := json.Marshal(project)
	return data
}

func restoreSnapshot(data []byte) (*designProject, error) {
	var project designProject
	if err := json.Unmarshal(data, &project); err != nil {
		return nil, err
	}
	project.normalize()
	project.ensureComponentNames()
	if err := project.validate(); err != nil {
		return nil, err
	}
	return &project, nil
}
