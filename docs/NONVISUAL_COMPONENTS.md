# Nonvisual components

Some application features have behavior but do not draw a control on the
form. Rosaline Studio keeps these in the center **Components** designer and in
a compact tray below the form preview, much like Lazarus.

Version 0.9 supports three form-owned components:

- `Timer` runs code once or repeatedly while its form is open.
- `OpenFileDialog` asks the user to choose an existing file.
- `SaveFileDialog` asks the user where a file should be saved.

## Add a timer

1. Open **Components** and choose **Add Timer**.
2. Give it an exported component name, such as `RefreshTimer`.
3. Enter the interval in milliseconds. `1000` means one second.
4. Choose whether it repeats and whether it starts with its form.
5. Choose **Assign and Edit Tick** and write the normal Go event body.

For example:

```go
app.Widgets().StatusLabel.SetText("The timer fired")
```

Studio mounts the timer on its owning form. A main-form timer starts with the
application. A secondary-form timer starts when that form opens. Event code
can control it through its typed component reference:

```go
app.Components().RefreshTimer.Stop()
app.Components().RefreshTimer.Start()

if app.Components().RefreshTimer.Running() {
	app.Widgets().StatusLabel.SetText("Timer is running")
}
```

A repeating timer uses `rosaline.Every`; a non-repeating timer uses
`rosaline.After`. The generated Go stays visible and readable.

## Add a file dialog

Choose **Add Open Dialog** or **Add Save Dialog**, then configure its title,
initial directory, initial filename, and default extension. Enter one filter
per line using:

```text
Text files | .txt, .md
Images | .png, .jpg, .webp, .avif
All files | *
```

Call the dialog from a button, action, menu, timer, or other event:

```go
path, ok := app.Components().OpenDocumentDialog.Execute()
if !ok {
	return
}
app.Widgets().StatusLabel.SetText("Opened: " + path)
```

`Execute` returns the selected path and `true`. When the user cancels, it
returns an empty path and `false`. A save dialog is called the same way:

```go
path, ok := app.Components().SaveDocumentDialog.Execute()
if ok {
	app.Widgets().StatusLabel.SetText("Save to: " + path)
}
```

The dialog chooses a path; your event code decides how to read or write the
file. That separation keeps generated UI code small and lets application code
use familiar packages such as `os` in a developer-owned Go file.

## Form ownership and generated code

Components shown in the tray belong to the currently selected form. They are
copied with that form and removed with it. Component names are unique across
the project because generated code exposes a single typed registry:

```go
app.Components().RefreshTimer
app.Components().OpenDocumentDialog
```

Studio generates the registry, timer construction, form mounting, dialog
configuration, and assigned timer event methods. It never hides those details
behind a binary project format.

Open [`../examples/components.rosaline`](../examples/components.rosaline) for
a complete application combining a repeating timer, open and save dialogs,
buttons, component references, and event code.
