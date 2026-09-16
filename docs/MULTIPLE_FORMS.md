# Designing multiple forms

Rosaline Studio projects can contain one primary form and any number of
reusable secondary forms. Each form has its own layout, title, size, padding,
theme, and lifecycle events.

## Create and select forms

Use **Project Forms** in the left panel:

- **New** creates an empty secondary form.
- **Duplicate** copies the selected form and all its widgets.
- **Delete** removes a secondary form after confirmation. The primary form is
  protected.
- Selecting a form switches the hierarchy, preview, properties, and events to
  that form.

Use the **Form** inspector to give every form a unique exported Go name such as
`SettingsForm`. Form names become typed fields returned by `app.Windows()`.

## Open and close a secondary form

Assign a button's `OnClick` event and write:

```go
app.Windows().SettingsForm.Show()
```

`Show` opens a closed form or focuses it when it is already open. To close it
from one of its own buttons:

```go
app.Windows().SettingsForm.Close()
```

The handle also supports `Focus`, `SetTitle`, and `IsOpen`.

## Form events

Select a form's root layout, then open the **Events** inspector:

- `OnOpen` runs after the form and its controls are mounted. It runs again
  whenever a secondary form is reopened.
- `OnCloseRequest` runs before a direct close request. Return `false` to cancel
  closing, such as when the form contains unsaved work.
- `OnClose` runs after the form has closed.

A close-request handler can be as simple as:

```go
return rosaline.Confirm("Close settings?", "Discard the current changes?")
```

## Generated structure

The first designed form becomes `rosaline.RunApp`. Every secondary form becomes
a reusable `rosaline.NewWindow` whose parent is the main window. Studio exposes
all of them through a generated structure:

```go
type UIWindows struct {
	MainForm     *rosaline.Window
	SettingsForm *rosaline.Window
}
```

Studio owns this generated wiring. Event code stays ordinary Go and can move
between forms through `app.Windows()` and update any designed component through
`app.Widgets()`.
