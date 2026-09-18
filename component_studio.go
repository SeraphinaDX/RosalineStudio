// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"fmt"
	"strconv"
	"strings"

	rosaline "github.com/SeraphinaDX/Rosaline"
)

type componentInspectorState struct {
	Name             string
	Interval         string
	Repeating        bool
	Enabled          bool
	Handler          string
	Title            string
	InitialDirectory string
	InitialFile      string
	DefaultExtension string
	Filters          string
}

func (studio *studio) buildComponentWorkspace() rosaline.Widget {
	timerProperties := rosaline.Column(
		rosaline.Label("Timer options").Bold().Color(rosaline.Rose),
		rosaline.Label("Interval is measured in milliseconds.").Color(rosaline.DefaultTheme.Muted),
		inspectorField("Interval", rosaline.TextBox(&studio.component.Interval).Width(18)),
		rosaline.CheckBox("Repeat until stopped", &studio.component.Repeating),
		rosaline.CheckBox("Start when the form opens", &studio.component.Enabled),
		inspectorField("Tick handler", rosaline.TextBox(&studio.component.Handler).Width(26)),
		rosaline.Button("Assign and Edit Tick", studio.editTimerHandler).Primary(),
	).Gap(7)
	dialogProperties := rosaline.Column(
		rosaline.Label("File dialog options").Bold().Color(rosaline.Rose),
		inspectorField("Window title", rosaline.TextBox(&studio.component.Title).Width(27)),
		rosaline.Grid(2,
			inspectorField("Initial directory", rosaline.TextBox(&studio.component.InitialDirectory).Width(19)),
			inspectorField("Initial file", rosaline.TextBox(&studio.component.InitialFile).Width(19)),
			inspectorField("Default extension", rosaline.TextBox(&studio.component.DefaultExtension).Width(19)),
		).Gap(7),
		rosaline.Label("Filters - one per line: Name | .ext, .ext").Color(rosaline.DefaultTheme.Muted),
		rosaline.TextArea(&studio.component.Filters).Size(42, 7),
	).Gap(7)
	properties := rosaline.Column(
		rosaline.LabelFunc(studio.selectedComponentLabel).Bold(),
		inspectorField("Component name", rosaline.TextBox(&studio.component.Name).Width(28)),
		rosaline.Tabs(
			rosaline.Tab("Timer", timerProperties),
			rosaline.Tab("File Dialog", dialogProperties),
		).Expand(),
		rosaline.Button("Apply Component Properties", studio.applyComponentInspector).Primary(),
		rosaline.LabelFunc(studio.componentHelp).Color(rosaline.DefaultTheme.Muted),
	).Gap(7).Expand()
	return rosaline.Column(
		rosaline.Row(
			rosaline.Label("Nonvisual Components").Bold().Color(rosaline.Rose),
			rosaline.LabelFunc(studio.componentCountLabel).Color(rosaline.DefaultTheme.Muted),
			rosaline.Spring(),
			rosaline.Button("Add Timer", func() { studio.addNonvisualComponent(componentTimer) }).Primary(),
			rosaline.Button("Add Open Dialog", func() { studio.addNonvisualComponent(componentOpenDialog) }),
			rosaline.Button("Add Save Dialog", func() { studio.addNonvisualComponent(componentSaveDialog) }),
		).Gap(6),
		rosaline.Row(
			rosaline.Column(
				rosaline.LabelFunc(func() string {
					form := studio.activeForm()
					if form == nil {
						return "Components on this form"
					}
					return "Components on " + form.Name
				}).Bold(),
				studio.componentList,
				rosaline.Row(
					rosaline.Button("Duplicate", studio.duplicateNonvisualComponent),
					rosaline.Button("Delete", studio.deleteNonvisualComponent),
				).Gap(6),
			).Gap(7).Expand(),
			rosaline.Card(properties).Padding(8).Expand(),
		).Gap(10).Expand(),
	).Gap(8).Expand()
}

func (studio *studio) selectedComponent() *designComponent {
	component, form := studio.project.component(studio.componentID)
	if form != studio.activeForm() {
		return nil
	}
	return component
}

func (studio *studio) ensureComponentSelection() {
	if studio.selectedComponent() != nil {
		return
	}
	studio.componentID = ""
	form := studio.activeForm()
	if form != nil && len(form.Components) != 0 && form.Components[0] != nil {
		studio.componentID = form.Components[0].ID
	}
}

func (studio *studio) selectComponent(index int) {
	if studio.syncComponent || index < 0 || index >= len(studio.componentIDs) {
		return
	}
	studio.componentID = studio.componentIDs[index]
	studio.loadComponentInspector()
	studio.status = "Selected " + studio.selectedComponentLabel()
}

func (studio *studio) loadComponentInspector() {
	component := studio.selectedComponent()
	if component == nil {
		studio.component = componentInspectorState{}
		return
	}
	studio.component = componentInspectorState{
		Name: component.Name, Interval: strconv.Itoa(component.Interval),
		Repeating: component.Repeating, Enabled: component.Enabled,
		Handler: component.Handler, Title: component.Title,
		InitialDirectory: component.InitialDirectory, InitialFile: component.InitialFile,
		DefaultExtension: component.DefaultExtension,
		Filters:          strings.Join(component.Filters, "\n"),
	}
}

func (studio *studio) selectedComponentLabel() string {
	component := studio.selectedComponent()
	if component == nil {
		return "No nonvisual component selected"
	}
	return component.Name + " - " + string(component.Kind)
}

func (studio *studio) componentHelp() string {
	component := studio.selectedComponent()
	if component == nil {
		return "Add a Timer, Open File Dialog, or Save File Dialog to the active form."
	}
	if component.Kind == componentTimer {
		return "Timers call their tick handler while this form is open. Use app.Components()." + component.Name + ".Start() or .Stop() in event code."
	}
	return "Open this dialog from event code with path, ok := app.Components()." + component.Name + ".Execute()."
}

func (studio *studio) componentTrayLabel() string {
	form := studio.activeForm()
	if form == nil || len(form.Components) == 0 {
		return "Nonvisual components: none"
	}
	items := make([]string, 0, 2)
	for _, component := range form.Components {
		if component != nil {
			name := []rune(component.Name)
			if len(name) > 18 {
				name = append(name[:17], '…')
			}
			items = append(items, string(name)+" ["+string(component.Kind)+"]")
			if len(items) == 2 {
				break
			}
		}
	}
	label := "Nonvisual: " + strings.Join(items, "  •  ")
	if len(form.Components) > len(items) {
		label += fmt.Sprintf("  •  +%d more", len(form.Components)-len(items))
	}
	return label
}

func (studio *studio) rebuildComponents() {
	studio.ensureComponentSelection()
	labels := make([]string, 0)
	studio.componentIDs = studio.componentIDs[:0]
	selected := -1
	if form := studio.activeForm(); form != nil {
		for _, component := range form.Components {
			if component == nil {
				continue
			}
			labels = append(labels, component.Name+" - "+string(component.Kind))
			studio.componentIDs = append(studio.componentIDs, component.ID)
			if component.ID == studio.componentID {
				selected = len(labels) - 1
			}
		}
	}
	if studio.componentList != nil {
		studio.syncComponent = true
		studio.componentList.SetItems(labels...)
		studio.componentList.Select(selected)
		studio.syncComponent = false
	}
	studio.loadComponentInspector()
}

func (studio *studio) showComponentDesigner() {
	if !studio.confirmOpenHandler("") {
		return
	}
	if studio.workspace != nil {
		studio.workspace.Select(workspaceComponentTab)
	}
}

func (studio *studio) addNonvisualComponent(kind componentKind) {
	before := designSnapshot(studio.project)
	component, err := studio.project.addComponent(studio.activeForm(), kind)
	if err != nil {
		studio.status = "Could not add component: " + err.Error()
		return
	}
	studio.componentID = component.ID
	studio.commitChange(before, "Added "+component.Name)
	studio.showComponentDesigner()
}

func (studio *studio) duplicateNonvisualComponent() {
	if studio.selectedComponent() == nil {
		studio.status = "Select a nonvisual component first"
		return
	}
	before := designSnapshot(studio.project)
	component, err := studio.project.duplicateComponent(studio.componentID)
	if err != nil {
		studio.status = "Could not duplicate component: " + err.Error()
		return
	}
	studio.componentID = component.ID
	studio.commitChange(before, "Duplicated nonvisual component")
}

func (studio *studio) deleteNonvisualComponent() {
	component := studio.selectedComponent()
	if component == nil {
		studio.status = "Select a nonvisual component first"
		return
	}
	if !rosaline.Confirm("Delete nonvisual component?", "Delete "+component.Name+"? You can undo this change.") {
		studio.status = "Component deletion cancelled"
		return
	}
	before := designSnapshot(studio.project)
	if err := studio.project.removeComponent(component.ID); err != nil {
		studio.status = "Could not delete component: " + err.Error()
		return
	}
	studio.componentID = ""
	studio.commitChange(before, "Deleted nonvisual component - use Primary+Z to undo")
}

func (studio *studio) applyComponentInspector() {
	component := studio.selectedComponent()
	if component == nil {
		studio.status = "Select a nonvisual component first"
		return
	}
	name := strings.TrimSpace(studio.component.Name)
	if !validComponentName(name) {
		studio.status = "Component names must be exported Go identifiers such as RefreshTimer"
		return
	}
	if other := studio.project.componentNamed(name); other != nil && other.ID != component.ID {
		studio.status = "That component name is already in use"
		return
	}
	candidate := *component
	candidate.Name = name
	candidate.Filters = append([]string(nil), component.Filters...)
	newHandler := ""
	switch candidate.Kind {
	case componentTimer:
		interval, err := strconv.Atoi(strings.TrimSpace(studio.component.Interval))
		if err != nil || interval < 1 {
			studio.status = "Timer interval must be at least 1 millisecond"
			return
		}
		handler := strings.TrimSpace(studio.component.Handler)
		if handler != "" && !validHandlerName(handler) {
			studio.status = "Tick handler names must be valid Go identifiers"
			return
		}
		if handler != "" {
			if err := handlerAcceptsSignature(studio.project, handler, handlerSignature{Seen: true}); err != nil {
				studio.status = err.Error()
				return
			}
		}
		candidate.Interval, candidate.Repeating = interval, studio.component.Repeating
		candidate.Enabled, candidate.Handler = studio.component.Enabled, handler
		candidate.Title, candidate.InitialDirectory, candidate.InitialFile, candidate.DefaultExtension = "", "", "", ""
		candidate.Filters = nil
		newHandler = handler
	case componentOpenDialog, componentSaveDialog:
		filters := splitLines(studio.component.Filters)
		if _, err := parseDesignFileFilters(filters); err != nil {
			studio.status = err.Error()
			return
		}
		candidate.Title = strings.TrimSpace(studio.component.Title)
		candidate.InitialDirectory = strings.TrimSpace(studio.component.InitialDirectory)
		candidate.InitialFile = strings.TrimSpace(studio.component.InitialFile)
		candidate.DefaultExtension = strings.TrimSpace(studio.component.DefaultExtension)
		candidate.Filters = filters
		candidate.Interval, candidate.Repeating, candidate.Enabled, candidate.Handler = 0, false, false, ""
	}
	before := designSnapshot(studio.project)
	*component = candidate
	if newHandler != "" {
		if studio.project.Handlers == nil {
			studio.project.Handlers = make(map[string]string)
		}
		if _, exists := studio.project.Handlers[newHandler]; !exists {
			studio.project.Handlers[newHandler] = defaultTimerHandlerBody(component)
		}
	}
	studio.commitChange(before, "Updated nonvisual component "+name)
}

func (studio *studio) editTimerHandler() {
	component := studio.selectedComponent()
	if component == nil || component.Kind != componentTimer {
		studio.status = "Select a Timer first"
		return
	}
	handler := strings.TrimSpace(studio.component.Handler)
	if handler == "" {
		handler = strings.TrimSpace(component.Handler)
	}
	if handler == "" {
		handler = studio.project.nextHandlerName(component.Name + "Tick")
	}
	if !validHandlerName(handler) {
		studio.status = "Tick handler names must be valid Go identifiers"
		return
	}
	if !studio.confirmOpenHandler(handler) {
		return
	}
	if err := handlerAcceptsSignature(studio.project, handler, handlerSignature{Seen: true}); err != nil {
		studio.status = err.Error()
		return
	}
	before := designSnapshot(studio.project)
	changed := component.Handler != handler
	component.Handler = handler
	if studio.project.Handlers == nil {
		studio.project.Handlers = make(map[string]string)
	}
	if _, exists := studio.project.Handlers[handler]; !exists {
		studio.project.Handlers[handler] = defaultTimerHandlerBody(component)
		changed = true
	}
	if changed {
		studio.commitChange(before, "Assigned timer tick handler "+handler)
	}
	studio.component.Handler = handler
	studio.codeHandler, studio.codeReturnsBool = handler, false
	studio.codeParameters = nil
	studio.codeBody = studio.project.Handlers[handler]
	studio.codeEditor.SetText(studio.codeBody)
	studio.codeEditor.MarkSaved()
	studio.workspace.Select(workspaceCodeTab)
	studio.codeEditor.Focus()
	studio.status = "Editing " + handler
}

func splitLines(value string) []string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		if line = strings.TrimSpace(line); line != "" {
			result = append(result, line)
		}
	}
	return result
}

func (studio *studio) componentCountLabel() string {
	count := 0
	if form := studio.activeForm(); form != nil {
		count = len(form.Components)
	}
	return fmt.Sprintf("%d component(s) on this form", count)
}
