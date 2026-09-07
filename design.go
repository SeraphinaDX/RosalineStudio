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
)

const designVersion = 1

type widgetKind string

const (
	kindColumn      widgetKind = "Column"
	kindRow         widgetKind = "Row"
	kindGrid        widgetKind = "Grid"
	kindStack       widgetKind = "Stack"
	kindCard        widgetKind = "Card"
	kindScroll      widgetKind = "Scroll"
	kindLabel       widgetKind = "Label"
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
	ID       string        `json:"id"`
	Kind     widgetKind    `json:"kind"`
	Text     string        `json:"text,omitempty"`
	Name     string        `json:"name,omitempty"`
	Action   string        `json:"action,omitempty"`
	Options  []string      `json:"options,omitempty"`
	Children []*designNode `json:"children,omitempty"`
	Gap      int           `json:"gap,omitempty"`
	Padding  int           `json:"padding,omitempty"`
	Columns  int           `json:"columns,omitempty"`
	Width    int           `json:"width,omitempty"`
	Height   int           `json:"height,omitempty"`
	Minimum  float64       `json:"minimum,omitempty"`
	Maximum  float64       `json:"maximum,omitempty"`
	Step     float64       `json:"step,omitempty"`
	Expand   bool          `json:"expand,omitempty"`
	Primary  bool          `json:"primary,omitempty"`
	Bold     bool          `json:"bold,omitempty"`
	Password bool          `json:"password,omitempty"`
	Vertical bool          `json:"vertical,omitempty"`
}

type designProject struct {
	Version int         `json:"version"`
	Module  string      `json:"module"`
	Title   string      `json:"title"`
	Width   int         `json:"width"`
	Height  int         `json:"height"`
	Padding int         `json:"padding"`
	Theme   string      `json:"theme"`
	Root    *designNode `json:"root"`
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
		Root: &designNode{
			ID:      "root",
			Kind:    kindColumn,
			Gap:     10,
			Padding: 12,
			Expand:  true,
			Children: []*designNode{
				{ID: "node-1", Kind: kindLabel, Text: "Welcome to Rosaline", Bold: true},
				{ID: "node-2", Kind: kindTextBox, Text: "Your name", Name: "Name"},
				{ID: "node-3", Kind: kindButton, Text: "Continue", Action: "continue", Primary: true},
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
	case kindButton:
		node.Text = "Button"
		node.Action = "buttonClicked"
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
	parent.Children = append(parent.Children, node)
	return node, nil
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
	var visit func(*designNode) error
	visit = func(node *designNode) error {
		if node == nil {
			return errors.New("the design contains an empty widget")
		}
		if strings.TrimSpace(node.ID) == "" || seen[node.ID] {
			return fmt.Errorf("widget ID %q is empty or repeated", node.ID)
		}
		seen[node.ID] = true
		if !knownKind(node.Kind) {
			return fmt.Errorf("widget %s has unknown kind %q", node.ID, node.Kind)
		}
		if !node.Kind.container() && len(node.Children) != 0 {
			return fmt.Errorf("%s cannot contain other widgets", node.Kind)
		}
		if (node.Kind == kindCard || node.Kind == kindScroll) && len(node.Children) > 1 {
			return fmt.Errorf("%s can contain only one widget", node.Kind)
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
	if err := project.validate(); err != nil {
		return nil, err
	}
	return &project, nil
}
