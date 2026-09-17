// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"errors"
	"fmt"
	"strings"
)

type componentKind string

const (
	componentTimer      componentKind = "Timer"
	componentOpenDialog componentKind = "OpenFileDialog"
	componentSaveDialog componentKind = "SaveFileDialog"
)

var componentPaletteKinds = []componentKind{
	componentTimer,
	componentOpenDialog,
	componentSaveDialog,
}

// designComponent is a form-owned component that has behavior but no visual
// bounds. Studio shows these components in a Lazarus-style tray.
type designComponent struct {
	ID               string        `json:"id"`
	Kind             componentKind `json:"kind"`
	Name             string        `json:"name"`
	Interval         int           `json:"interval,omitempty"`
	Repeating        bool          `json:"repeating,omitempty"`
	Enabled          bool          `json:"enabled,omitempty"`
	Handler          string        `json:"handler,omitempty"`
	Title            string        `json:"title,omitempty"`
	InitialDirectory string        `json:"initialDirectory,omitempty"`
	InitialFile      string        `json:"initialFile,omitempty"`
	DefaultExtension string        `json:"defaultExtension,omitempty"`
	Filters          []string      `json:"filters,omitempty"`
}

type designFileFilter struct {
	Name       string
	Extensions []string
}

func (project *designProject) component(id string) (*designComponent, *designForm) {
	if project == nil {
		return nil, nil
	}
	for _, form := range project.Forms {
		if form == nil {
			continue
		}
		for _, component := range form.Components {
			if component != nil && component.ID == id {
				return component, form
			}
		}
	}
	return nil, nil
}

func (project *designProject) componentNamed(name string) *designComponent {
	if project == nil {
		return nil
	}
	for _, form := range project.Forms {
		if form == nil {
			continue
		}
		for _, component := range form.Components {
			if component != nil && component.Name == name {
				return component
			}
		}
	}
	return nil
}

func (project *designProject) nextNonvisualID() string {
	used := make(map[string]bool)
	for _, form := range project.Forms {
		if form == nil {
			continue
		}
		for _, component := range form.Components {
			if component != nil {
				used[component.ID] = true
			}
		}
	}
	for number := 1; ; number++ {
		id := fmt.Sprintf("component-%d", number)
		if !used[id] {
			return id
		}
	}
}

func (project *designProject) nextNonvisualName(base string) string {
	used := make(map[string]bool)
	for _, form := range project.Forms {
		if form == nil {
			continue
		}
		for _, component := range form.Components {
			if component != nil {
				used[component.Name] = true
			}
		}
	}
	return uniqueGeneratedName(exportedIdentifier(base), used)
}

func defaultDesignComponent(kind componentKind, id, name string) *designComponent {
	component := &designComponent{ID: id, Kind: kind, Name: name}
	switch kind {
	case componentTimer:
		component.Interval = 1000
		component.Repeating = true
		component.Enabled = true
		component.Handler = name + "Tick"
	case componentOpenDialog:
		component.Title = "Open File"
		component.Filters = []string{"All files | *"}
	case componentSaveDialog:
		component.Title = "Save File"
		component.Filters = []string{"All files | *"}
	}
	return component
}

func (project *designProject) addComponent(form *designForm, kind componentKind) (*designComponent, error) {
	if form == nil {
		return nil, errors.New("there is no active form")
	}
	if !knownComponentKind(kind) {
		return nil, fmt.Errorf("unknown component kind %q", kind)
	}
	base := string(kind)
	if kind == componentOpenDialog {
		base = "OpenDialog"
	} else if kind == componentSaveDialog {
		base = "SaveDialog"
	}
	name := project.nextNonvisualName(base)
	component := defaultDesignComponent(kind, project.nextNonvisualID(), name)
	if kind == componentTimer {
		component.Handler = project.nextHandlerName(component.Handler)
		if project.Handlers == nil {
			project.Handlers = make(map[string]string)
		}
		project.Handlers[component.Handler] = defaultTimerHandlerBody(component)
	}
	form.Components = append(form.Components, component)
	return component, nil
}

func (project *designProject) duplicateComponent(id string) (*designComponent, error) {
	source, form := project.component(id)
	if source == nil || form == nil {
		return nil, errors.New("component was not found")
	}
	clone := *source
	clone.ID = project.nextNonvisualID()
	clone.Name = project.nextNonvisualName(source.Name)
	clone.Filters = append([]string(nil), source.Filters...)
	if clone.Kind == componentTimer {
		clone.Handler = project.nextHandlerName(clone.Name + "Tick")
		if project.Handlers == nil {
			project.Handlers = make(map[string]string)
		}
		if body, ok := project.Handlers[source.Handler]; ok {
			project.Handlers[clone.Handler] = body
		} else {
			project.Handlers[clone.Handler] = defaultTimerHandlerBody(&clone)
		}
	}
	form.Components = append(form.Components, &clone)
	return &clone, nil
}

func (project *designProject) removeComponent(id string) error {
	component, form := project.component(id)
	if component == nil || form == nil {
		return errors.New("component was not found")
	}
	for index, candidate := range form.Components {
		if candidate == component {
			form.Components = append(form.Components[:index], form.Components[index+1:]...)
			return nil
		}
	}
	return errors.New("component was not found")
}

func cloneDesignComponents(source []*designComponent) []*designComponent {
	result := make([]*designComponent, 0, len(source))
	for _, component := range source {
		if component == nil {
			result = append(result, nil)
			continue
		}
		clone := *component
		clone.Filters = append([]string(nil), component.Filters...)
		result = append(result, &clone)
	}
	return result
}

func (project *designProject) prepareCopiedComponents(components []*designComponent) {
	if project.Handlers == nil {
		project.Handlers = make(map[string]string)
	}
	usedIDs := make(map[string]bool)
	usedNames := make(map[string]bool)
	for _, form := range project.Forms {
		if form == nil {
			continue
		}
		for _, component := range form.Components {
			if component != nil {
				usedIDs[component.ID] = true
				usedNames[component.Name] = true
			}
		}
	}
	nextID := 1
	for _, component := range components {
		if component == nil {
			continue
		}
		for usedIDs[fmt.Sprintf("component-%d", nextID)] {
			nextID++
		}
		component.ID = fmt.Sprintf("component-%d", nextID)
		usedIDs[component.ID] = true
		nextID++
		component.Name = uniqueGeneratedName(component.Name, usedNames)
		if component.Kind == componentTimer {
			oldHandler := component.Handler
			component.Handler = project.nextHandlerName(component.Name + "Tick")
			if body, ok := project.Handlers[oldHandler]; ok {
				project.Handlers[component.Handler] = body
			} else {
				project.Handlers[component.Handler] = defaultTimerHandlerBody(component)
			}
		}
	}
}

func knownComponentKind(kind componentKind) bool {
	for _, candidate := range componentPaletteKinds {
		if kind == candidate {
			return true
		}
	}
	return false
}

func defaultTimerHandlerBody(component *designComponent) string {
	name := "Timer"
	if component != nil && component.Name != "" {
		name = component.Name
	}
	return fmt.Sprintf("// %s fired.\n// Update application state or controls here.", name)
}

func parseDesignFileFilters(lines []string) ([]designFileFilter, error) {
	result := make([]designFileFilter, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" {
			return nil, fmt.Errorf("filter %q must use Name | .ext, .ext", line)
		}
		extensions := splitOptions(parts[1])
		if len(extensions) == 0 {
			return nil, fmt.Errorf("filter %q has no extensions", line)
		}
		result = append(result, designFileFilter{Name: strings.TrimSpace(parts[0]), Extensions: extensions})
	}
	return result, nil
}

func validateDesignComponents(form *designForm, seen, names map[string]bool) error {
	for _, component := range form.Components {
		if component == nil {
			return errors.New("contains an empty component")
		}
		if strings.TrimSpace(component.ID) == "" || seen[component.ID] {
			return fmt.Errorf("component ID %q is empty or repeated", component.ID)
		}
		seen[component.ID] = true
		if !validComponentName(component.Name) || names[component.Name] {
			return fmt.Errorf("component name %q is invalid or repeated", component.Name)
		}
		names[component.Name] = true
		if !knownComponentKind(component.Kind) {
			return fmt.Errorf("component %s has unknown kind %q", component.Name, component.Kind)
		}
		switch component.Kind {
		case componentTimer:
			if component.Interval < 1 {
				return fmt.Errorf("timer %s must have an interval of at least 1 millisecond", component.Name)
			}
			if component.Handler != "" && !validHandlerName(component.Handler) {
				return fmt.Errorf("timer %s has invalid handler name %q", component.Name, component.Handler)
			}
			if component.Title != "" || component.InitialDirectory != "" || component.InitialFile != "" || component.DefaultExtension != "" || len(component.Filters) != 0 {
				return fmt.Errorf("timer %s contains file-dialog properties", component.Name)
			}
		case componentOpenDialog, componentSaveDialog:
			if component.Interval != 0 || component.Repeating || component.Enabled || component.Handler != "" {
				return fmt.Errorf("file dialog %s contains timer properties", component.Name)
			}
			if _, err := parseDesignFileFilters(component.Filters); err != nil {
				return fmt.Errorf("file dialog %s: %w", component.Name, err)
			}
		}
	}
	return nil
}
