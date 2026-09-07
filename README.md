# Rosaline Studio

Rosaline Studio is a beginner-friendly visual application designer for
[Rosaline](https://github.com/SeraphinaDX/Rosaline). Arrange widgets, edit their
properties, save the design, and generate a normal Go project that remains easy
to read and extend.

Rosaline Studio is itself a pure-Go Rosaline application. It builds with
`CGO_ENABLED=0` and treats Linux as a first-class platform.

## What v0.1 can do

- Add controls and layouts from a compact widget palette
- Select widgets from the hierarchy or the visual preview
- Drag widgets onto a container or sibling to rearrange the design
- Edit text, state names, actions, sizing, spacing, and common options
- Configure the application title, module path, window size, and theme
- Undo and redo up to 100 design changes
- Save strict, readable `.rosaline` design files
- Generate and run a complete Rosaline Go application
- Preserve developer-owned behavior across every regeneration

The v0.1 preview is a structural wireframe. It shows the layout, hierarchy,
selection, and theme without pretending to be a pixel-perfect rendering of the
generated native window. Press F5 to see the real application.

## Requirements

- Go 1.25 or newer
- Rosaline v0.15.0 or newer
- A graphical desktop supported by Rosaline

Rosaline v0.15.0 must exist as a Git tag before a fresh clone can download the
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

## The generated project

Studio draws a hard line between generated and handwritten code:

| File | Ownership | Regeneration behavior |
|---|---|---|
| `ui_generated.go` | Studio | Replaced with the current layout, theme, and window settings |
| `state_generated.go` | Studio | Replaced with the state fields required by controls |
| `handlers.go` | Developer | Created once and never overwritten |
| `main.go` | Developer | Created once and never overwritten |
| `go.mod` | Developer | Created once and never overwritten |
| `README.md` | Developer | Created once and never overwritten |

Buttons call one stable method:

```go
func (app *Application) Action(name string) {
	switch name {
	case "save":
		app.save()
	case "open":
		app.open()
	}
}
```

Adding or removing a designer state field cannot erase the rest of your
application logic. Generated state lives in the `UIState` value embedded by the
developer-owned `Application` type.

## Supported designer widgets

Layouts: `Column`, `Row`, `Grid`, `Stack`, `Card`, and `Scroll`.

Controls: `Label`, `Button`, `TextBox`, `TextArea`, `CheckBox`, `ComboBox`,
`Slider`, `ProgressBar`, and `Spacer`.

Rosaline has more features than the v0.1 palette. Menus, dialogs, images,
canvases, timers, tables, tabs, and custom widgets can already be added by hand
to the generated project and are candidates for later Studio releases.

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

Rosaline Studio v0.1.1 is an intentionally small early release. Generated code
and saved designs are designed to stay understandable while the visual tooling
grows. The design schema is versioned, but the Studio API and file format remain
experimental until v1.0.

## License

Rosaline Studio is free software licensed under the
[GNU Lesser General Public License v3.0 or later](LICENSE).

Generated application code is intended for the application developer to own
and license as they choose. Rosaline itself remains under its own
LGPL-3.0-or-later license.
Dependencies retain their own licenses; see
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

Copyright (C) 2026 Britney Lozza and Rosaline Studio contributors.
