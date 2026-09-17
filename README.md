# Rosaline Studio

Rosaline Studio is a beginner-friendly visual application designer for
[Rosaline](https://github.com/SeraphinaDX/Rosaline). Arrange widgets, edit their
properties, save the design, and generate a normal Go project that remains easy
to read and extend.

Rosaline Studio is itself a pure-Go Rosaline application. It builds with
`CGO_ENABLED=0` and treats Linux as a first-class platform.

![Screenshot](rosaline-studio.avif)

## What v0.10.0 can do

- Design a primary form and any number of reusable secondary forms
- Create, duplicate, rename, select, configure, and delete forms from the
  project panel
- Add controls and layouts from a compact widget palette
- Select widgets from the hierarchy or the visual preview
- Drag widgets onto a container or sibling to rearrange the design
- Delete widgets from the form toolbar, hierarchy, Edit menu, keyboard, or a
  right-click menu, with safe confirmation and undo
- Cut, copy, paste, and duplicate complete widget subtrees
- Give every control a Lazarus-style component name
- Switch between Lazarus-style Form and Code views
- Build complete menu bars in a dedicated visual Menu Designer
- Add menu items, separators, nested submenus, shortcuts, and click handlers
- Define Lazarus-style project actions that share one caption, shortcut, and
  execute handler across menus and toolbars
- Build an ordered toolbar for each form from shared actions and separators
- Add form-owned Timer, Open File Dialog, and Save File Dialog components in a
  dedicated Lazarus-style Components designer
- Configure timer intervals, repetition, automatic startup, and tick handlers
- Configure native file-dialog titles, starting paths, filenames, extensions,
  and friendly filters
- Control nonvisual components from event code through typed
  `app.Components()` references
- Add tabbed interfaces from the palette with friendly page containers
- Add, rename, duplicate, reorder, select, and delete pages in the Pages inspector
- Switch designed pages directly in the form preview
- Design lists, tables, trees, and vertical or horizontal radio groups
- Enter realistic control data through a simple multiline Data inspector
- Edit text, state names, sizing, spacing, and common options
- Assign `OnClick`, `OnChange`, and `OnSubmit` methods in the Events inspector
- Double-click a form control to create or edit its default event
- Write event bodies in the integrated Go editor with syntax validation
- Browse the complete generated project in a Lazarus-style Project Files tree
- Open several Go files in source tabs with syntax coloring and unsaved marks
- Create, rename, delete, format, and save developer-owned Go files
- Inspect Studio-owned generated Go files safely in read-only mode
- Build without running and open compiler errors at the exact file and line
- Import PNG, JPEG, GIF, BMP, TIFF, WebP, and AVIF pictures
- Preview, size, and embed image assets in generated applications
- Configure each form's title, window size, padding, and theme
- Assign form `OnOpen`, `OnCloseRequest`, and `OnClose` methods
- Undo and redo up to 100 design changes
- Save strict, readable `.rosaline` design files
- Generate and run a complete Rosaline Go application
- Preserve developer-owned behavior across every regeneration

The form preview shows layout, hierarchy, selection, theme, and imported
pictures. Native controls remain a close structural preview rather than a
pixel-perfect copy of the generated window. Press F5 to see the real
application.

Build and Run downloads Rosaline and its Go dependencies automatically before
starting the generated application. Generated applications use
`CGO_ENABLED=0` and build independently of any surrounding Go workspace.
On the first run, Studio explains every setup action and asks for confirmation
before creating files or downloading modules.

## Requirements

- Go 1.25 or newer
- Rosaline v0.20.0 or newer
- A graphical desktop supported by Rosaline

Rosaline v0.20.0 must exist as a Git tag before a fresh clone can download the
dependency. If you keep both repositories side by side during development, a
Go workspace can use your local Rosaline checkout instead.

## Run Studio

Clone this repository, enter it, and run:

```bash
env CGO_ENABLED=0 go run .
```

The first build can take a while because Go compiles Rosaline's pure-Go window,
font, image, and AVIF dependencies. Later builds use Go's cache and are normally
much faster.

For side-by-side local development before the Rosaline tag is published:

```bash
go work init . ../Rosaline
env CGO_ENABLED=0 go run .
```

See [docs/QUICK_START.md](docs/QUICK_START.md) for the full first-project
walkthrough.

See [docs/SOURCE_EDITOR.md](docs/SOURCE_EDITOR.md) for editing handwritten Go,
viewing generated code, formatting, and following build errors.

## The generated project

Each design generates into a separate folder beside its `.rosaline` file. A
design named `greeting.rosaline` produces the Go application in `greeting/`, so
running a design saved inside the Studio repository cannot launch Studio
itself.

Studio draws a hard line between generated and handwritten code:

| File | Ownership | Regeneration behavior |
|---|---|---|
| `ui_generated.go` | Studio | Replaced with the current layout, theme, and window settings |
| `state_generated.go` | Studio | Replaced with the state fields required by controls |
| `events_generated.go` | Studio | Replaced with event methods edited in Studio's Code view |
| `handlers.go` | Developer | Created once and never overwritten |
| `main.go` | Developer | Created once and never overwritten |
| `go.mod` | Developer | Created once and never overwritten |
| `README.md` | Developer | Created once and never overwritten |

Each event calls a normal named Go method:

```go
func (app *Application) SaveClick() {
	rosaline.Message("Saved", "Your application handled OnClick.")
}
```

Every designed widget also has a typed component reference. A handler can
change another control without editing generated layout code:

```go
func (app *Application) SaveButtonClick() {
	app.Widgets().StatusLabel.SetText("Saved")
	app.Widgets().SaveButton.SetEnabled(false)
}
```

Choose memorable component names such as `StatusLabel` and `SaveButton` in the
Properties inspector. Older version-2 designs receive safe names automatically.

Every designed form has a typed reusable window reference as well. For
example, a button event on the main form can open a secondary form with:

```go
app.Windows().SettingsForm.Show()
```

The same handle supports `Close`, `Focus`, `SetTitle`, and `IsOpen`. Select a
form's root layout to edit its lifecycle events in the Events inspector.

Studio stores event bodies in the `.rosaline` design and regenerates
`events_generated.go`. Put reusable helpers, services, imports, and non-visual
logic in developer-owned `handlers.go` or another Go file. Adding or removing a
designer field cannot erase those files.

See [Designing multiple forms](docs/MULTIPLE_FORMS.md) for the complete form
workflow and lifecycle-event model.

## Visual tabs and pages

Add **Tabs** from the palette and Studio creates General and Advanced pages.
Click a page header in the form preview, or select its `TabPage` in the
hierarchy, then add controls normally. The right **Pages** inspector adds,
renames, duplicates, reorders, and deletes pages. A Tabs component also offers
`OnChange` in the normal Events inspector.

Generated code stays ordinary Rosaline Go: a `rosaline.Tabs` contains
`rosaline.Tab` pages whose content is a `rosaline.Column`. Open
[`examples/preferences.rosaline`](examples/preferences.rosaline) for a complete
three-page settings application, and see
[Designing tabs and pages](docs/TABS_DESIGNER.md) for the full workflow.

## Visual data controls

Studio can design `List`, `Table`, `Tree`, and `RadioGroup` controls. Select one
and open **Properties > Data**. The editor explains the syntax for that control:
one item per line, `Label = value` radio choices, `|`-separated table cells, or
`/`-separated tree paths.

The controls expose their native Rosaline events in the Events inspector.
Generated event methods can inspect the selected item through the typed
component in `app.Widgets()`. Open
[`examples/data_browser.rosaline`](examples/data_browser.rosaline) for a
complete project-browser interface, and see
[Designing data controls](docs/DATA_CONTROLS.md) for each data format and event.

## Visual menu designer

Open the center **Menus** tab or choose **Project > Menu Designer**. Add a
top-level menu such as File, select it, then add items, submenus, or separators.
The right **Menu** inspector edits the caption, shortcut, and click-handler
method. Double-click an item to open that method in the normal Code view.

Studio generates the same menu hierarchy for the selected form and includes
menu handlers in `events_generated.go`. See
[Designing menus](docs/MENU_DESIGNER.md) and open
[`examples/notepad.rosaline`](examples/notepad.rosaline) for a complete app.

## Shared actions and visual toolbars

Open the center **Actions** tab or choose **Project > Actions and Toolbar**.
Create an action such as `SaveAction`, give it a caption, shortcut, and execute
handler, then add it to the active form's toolbar. In the Menu inspector, link
a menu item to the same action. Editing that action updates both clients.

Toolbars are per-form, while actions belong to the whole project and can be
reused by every form. Generated source stays ordinary Rosaline Go built from a
row of buttons and separators. See
[Shared actions and visual toolbars](docs/ACTIONS_TOOLBARS.md) and open the
updated [`examples/notepad.rosaline`](examples/notepad.rosaline).

## Nonvisual components

Open the center **Components** tab or choose **Project > Nonvisual
Components**. Add a Timer, Open File Dialog, or Save File Dialog to the active
form. The compact tray below the form preview shows these behavioral components
without pretending that they are visual widgets.

Timers can repeat or fire once, start with their form or remain stopped, and
open their tick method directly in Studio's Go editor. File dialogs share one
simple call from any event:

```go
path, ok := app.Components().OpenDocumentDialog.Execute()
if ok {
	app.Widgets().StatusLabel.SetText(path)
}
```

See [Nonvisual components](docs/NONVISUAL_COMPONENTS.md) and open
[`examples/components.rosaline`](examples/components.rosaline) for a complete
timer and file-dialog application.

## Supported designer widgets

Layouts: `Column`, `Row`, `Grid`, `Stack`, `Card`, `Scroll`, and `Tabs` with
designer-managed `TabPage` containers.

Controls: `Label`, `Image`, `Button`, `TextBox`, `TextArea`, `CheckBox`,
`ComboBox`, `RadioGroup`, `List`, `Table`, `Tree`, `Slider`, `ProgressBar`, and
`Spacer`.

Rosaline has more features than the current palette. Canvases and custom
widgets can already be added by hand to the generated project and are
candidates for later Studio releases.

## Design file

The `.rosaline` file is JSON with a versioned schema. It is suitable for Git,
code review, and manual inspection. See
[docs/DESIGN_FORMAT.md](docs/DESIGN_FORMAT.md).

## Tests

Run Studio's normal test suite with:

```bash
env CGO_ENABLED=0 go test ./...
```

Maintainers with a local Rosaline source tree can also compile a freshly
generated application:

```bash
env CGO_ENABLED=0 ROSALINE_SOURCE=../Rosaline go test ./...
```

## Project status

Rosaline Studio v0.10.0 is an intentionally small early release. Generated code
and saved designs are designed to stay understandable while the visual tooling
grows. The design schema is versioned, but the Studio API and file format remain
experimental until v1.0. Studio automatically migrates version-2 through
version-7 designs to version 8; version-1 designs remain intentionally
incompatible.

## License

Rosaline Studio is free software licensed under the
[GNU Lesser General Public License v3.0 or later](LICENSE).

Generated application code is intended for the application developer to own
and license as they choose. Rosaline itself remains under its own
LGPL-3.0-or-later license.
Dependencies retain their own licenses; see
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

Copyright (C) 2026 Britney Lozza and Rosaline Studio contributors.
