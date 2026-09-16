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

const designVersion = 2

type widgetKind string

const (
	kindColumn      widgetKind = "Column"
	kindRow         widgetKind = "Row"
	kindGrid        widgetKind = "Grid"
	kindStack       widgetKind = "Stack"
	kindCard        widgetKind = "Card"
	kindScroll      widgetKind = "Scroll"
	kindLabel       widgetKind = "Label"
	kindImage       widgetKind = "Image"
	kindButton      widgetKind = "Button"
	kindTextBox     widgetKind = "TextBox"
	kindTextArea    widgetKind = "TextArea"
	kindCheckBox    widgetKind = "CheckBox"
	kindComboBox    widgetKind = "ComboBox"
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
	kindSlider,
	kindProgressBar,
	kindSpacer,
	kindColumn,
	kindRow,
	kindGrid,
	kindStack,
	kindCard,
	kindScroll,
}

type designNode struct {
	ID        string            `json:"id"`
	Kind      widgetKind        `json:"kind"`
	Component string            `json:"component,omitempty"`
	Text      string            `json:"text,omitempty"`
	Name      string            `json:"name,omitempty"`
	Asset     string            `json:"asset,omitempty"`
	Events    map[string]string `json:"events,omitempty"`
	Options   []string          `json:"options,omitempty"`
	Children  []*designNode     `json:"children,omitempty"`
	Gap       int               `json:"gap,omitempty"`
	Padding   int               `json:"padding,omitempty"`
	Columns   int               `json:"columns,omitempty"`
	Width     int               `json:"width,omitempty"`
	Height    int               `json:"height,omitempty"`
	Minimum   float64           `json:"minimum,omitempty"`
	Maximum   float64           `json:"maximum,omitempty"`
	Step      float64           `json:"step,omitempty"`
	Expand    bool              `json:"expand,omitempty"`
	Primary   bool              `json:"primary,omitempty"`
	Bold      bool              `json:"bold,omitempty"`
	Password  bool              `json:"password,omitempty"`
	Vertical  bool              `json:"vertical,omitempty"`
}

type designProject struct {
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
		Title:   "My Rosaline App",
		Width:   720,
		Height:  520,
		Padding: 16,
		Theme:   "Rosaline",
		Handlers: map[string]string{
			"ContinueClick": `rosaline.Message("Your application", "Continue clicked.")`,
		},
		Root: &designNode{
			ID:        "root",
			Kind:      kindColumn,
			Component: "MainLayout",
			Gap:       10,
			Padding:   12,
			Expand:    true,
			Children: []*designNode{
				{ID: "node-1", Kind: kindLabel, Component: "WelcomeLabel", Text: "Welcome to Rosaline", Bold: true},
				{ID: "node-2", Kind: kindTextBox, Component: "NameTextBox", Text: "Your name", Name: "Name"},
				{ID: "node-3", Kind: kindButton, Component: "ContinueButton", Text: "Continue", Events: map[string]string{"OnClick": "ContinueClick"}, Primary: true},
			},
		},
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
	case kindColumn, kindRow, kindGrid, kindStack, kindCard, kindScroll:
		return true
	default:
		return false
	}
}

func (project *designProject) find(id string) *designNode {
	if project == nil {
		return nil
	}
	return findNode(project.Root, id)
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
	if project == nil || project.Root == nil || project.Root.ID == id {
		return nil
	}
	return findParent(project.Root, id)
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
	visit(project.Root)
	return fmt.Sprintf("node-%d", maximum+1)
}

func (project *designProject) addNear(selectedID string, kind widgetKind) (*designNode, error) {
	selected := project.find(selectedID)
	if selected == nil {
		selected = project.Root
	}
	parent := selected
	if !parent.Kind.container() {
		parent = project.parentOf(parent.ID)
	}
	if parent == nil {
		return nil, errors.New("select a container before adding a widget")
	}
	if (parent.Kind == kindCard || parent.Kind == kindScroll) && len(parent.Children) != 0 {
		return nil, fmt.Errorf("%s can contain one widget; select its parent instead", parent.Kind)
	}
	node := defaultNode(kind, project.nextID())
	node.Component = project.nextComponentName(kind)
	parent.Children = append(parent.Children, node)
	return node, nil
}

func cloneDesignNode(node *designNode) *designNode {
	if node == nil {
		return nil
	}
	clone := *node
	clone.Options = append([]string(nil), node.Options...)
	clone.Children = make([]*designNode, 0, len(node.Children))
	if node.Events != nil {
		clone.Events = make(map[string]string, len(node.Events))
		for event, handler := range node.Events {
			clone.Events[event] = handler
		}
	}
	for _, child := range node.Children {
		clone.Children = append(clone.Children, cloneDesignNode(child))
	}
	return &clone
}

// insertCopy places an independent copy inside a selected container or just
// after a selected control. IDs, component names, and bound state names are
// made unique while event handler assignments remain shared intentionally.
func (project *designProject) insertCopy(selectedID string, source *designNode) (*designNode, error) {
	if project == nil || project.Root == nil || source == nil {
		return nil, errors.New("there is no copied widget to paste")
	}
	selected := project.find(selectedID)
	if selected == nil {
		selected = project.Root
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
	if node == project.Root {
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
	collect(project.Root)

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
		visit(project.Root)
	}
	return nextAvailableName(exportedIdentifier(string(kind)), used)
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
		visit(project.Root)
	}
	return found
}

func (project *designProject) ensureComponentNames() {
	if project == nil || project.Root == nil {
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
	collect(project.Root)
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
	fill(project.Root)
}

func (project *designProject) remove(id string) error {
	if project == nil || project.Root == nil || id == project.Root.ID {
		return errors.New("the root layout cannot be deleted")
	}
	parent := project.parentOf(id)
	if parent == nil {
		return errors.New("widget was not found")
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
	if project == nil || project.Root == nil {
		return ""
	}
	parent := project.parentOf(id)
	if parent == nil {
		return project.Root.ID
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
	if id == "" || targetID == "" || id == targetID || project.Root == nil || id == project.Root.ID {
		return errors.New("choose a different destination")
	}
	moving := project.find(id)
	target := project.find(targetID)
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
	if project == nil || project.Root == nil {
		return errors.New("the design has no root layout")
	}
	if project.Version != designVersion {
		return fmt.Errorf("unsupported design version %d", project.Version)
	}
	if strings.TrimSpace(project.Module) == "" {
		return errors.New("the Go module path cannot be empty")
	}
	seen := make(map[string]bool)
	components := make(map[string]bool)
	for name, body := range project.Handlers {
		if err := validateHandler(name, body); err != nil {
			return err
		}
	}
	var visit func(*designNode) error
	visit = func(node *designNode) error {
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
		if !node.Kind.container() && len(node.Children) != 0 {
			return fmt.Errorf("%s cannot contain other widgets", node.Kind)
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
			if err := visit(child); err != nil {
				return err
			}
		}
		return nil
	}
	return visit(project.Root)
}

func knownKind(kind widgetKind) bool {
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
	var project designProject
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&project); err != nil {
		return nil, fmt.Errorf("decode design: %w", err)
	}
	if project.Handlers == nil {
		project.Handlers = make(map[string]string)
	}
	project.ensureComponentNames()
	if err := project.validate(); err != nil {
		return nil, err
	}
	return &project, nil
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
	project.ensureComponentNames()
	if err := project.validate(); err != nil {
		return nil, err
	}
	return &project, nil
}
