# Designing tabs and pages

Rosaline Studio v0.6 adds a visual page designer for tabbed interfaces. Tabs
are useful for preferences, project properties, setup screens, and any form
that needs related groups without several windows.

## Add Tabs

Select a container in the form and add **Tabs** from the Widget Palette. Studio
creates two starter pages, General and Advanced, and selects the Tabs component.
The Tabs component may consume available space and supports an `OnChange`
event.

Each page appears as a `TabPage` beneath Tabs in the hierarchy. TabPage is a
friendly column-like container, so it accepts labels, inputs, buttons, images,
and nested layouts directly.

## Choose and fill a page

Click a page header in the form preview or select its TabPage in the hierarchy.
Only that page's controls appear in the preview. Selecting a control from
another page in the hierarchy switches the preview to its page automatically.

With a page selected, double-click a palette item or choose **Add Widget**.
When the parent Tabs is selected, Studio adds the control to the page currently
shown in the preview.

## Manage pages

Open the right **Pages** inspector after selecting Tabs, a TabPage, or any
control inside a page.

- **Apply Page Title** changes the visible header.
- **Add** creates and selects a new page.
- **Duplicate** copies the current page and every control inside it with fresh
  component and state names.
- **Move Left** and **Move Right** reorder the page headers.
- **Delete Page** removes the page after confirming when it contains controls.

Tabs must keep at least one page. Every page title must contain text.

## Handle page changes

Select the Tabs component, open **Events**, choose `OnChange`, and assign a
normal event method. For example:

```go
app.Widgets().StatusLabel.SetText("Preferences page changed")
```

The generated component reference is a `*rosaline.TabsWidget`, so advanced
event code can call its normal Rosaline methods through
`app.Widgets().PreferencesTabs`.

## Generated Go

Studio emits direct, readable Rosaline calls:

```go
rosaline.Tabs(
	rosaline.Tab("General", rosaline.Column(
		rosaline.TextBox(&app.State.DisplayName),
	).Gap(10).Padding(12).Expand()),
	rosaline.Tab("Advanced", rosaline.Column().Gap(10).Padding(12).Expand()),
).Expand().OnChange(func(int, string) { app.PreferencesPageChanged() })
```

TabPage does not become a fake widget field. It describes the title and column
that Studio passes to `rosaline.Tab`. Controls inside it remain normal named
components available through `app.Widgets()`.

Open `examples/preferences.rosaline` in Studio for a complete three-page
application with inputs, change behavior, and a save action.
