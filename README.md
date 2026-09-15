# Rosaline Studio

Rosaline Studio is a beginner-friendly visual application designer for
[Rosaline](https://github.com/SeraphinaDX/Rosaline). Arrange widgets, edit their
properties, save the design, and generate a normal Go project that remains easy
to read and extend.

Rosaline Studio is itself a pure-Go Rosaline application. It builds with
`CGO_ENABLED=0` and treats Linux as a first-class platform.

![Screenshot](rosaline-studio.avif)

## What v0.2.1 can do

- Add controls and layouts from a compact widget palette
- Select widgets from the hierarchy or the visual preview
- Drag widgets onto a container or sibling to rearrange the design
- Delete widgets from the form toolbar, hierarchy, Edit menu, keyboard, or a
  right-click menu, with safe confirmation and undo
- Switch between Lazarus-style Form and Code views
- Edit text, state names, sizing, spacing, and common options
- Assign `OnClick`, `OnChange`, and `OnSubmit` methods in the Events inspector
- Double-click a form control to create or edit its default event
- Write event bodies in the integrated Go editor with syntax validation
- Import PNG, JPEG, GIF, BMP, TIFF, WebP, and AVIF pictures
- Preview, size, and embed image assets in generated applications
- Configure the application title, module path, window size, and theme
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
- Rosaline v0.16.1 or newer
- A graphical desktop supported by Rosaline

Rosaline v0.16.1 must exist as a Git tag before a fresh clone can download the
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

Studio stores event bodies in the `.rosaline` design and regenerates
`events_generated.go`. Put reusable helpers, services, imports, and non-visual
logic in developer-owned `handlers.go` or another Go file. Adding or removing a
designer field cannot erase those files.

## Supported designer widgets

Layouts: `Column`, `Row`, `Grid`, `Stack`, `Card`, and `Scroll`.

Controls: `Label`, `Image`, `Button`, `TextBox`, `TextArea`, `CheckBox`,
`ComboBox`, `Slider`, `ProgressBar`, and `Spacer`.

Rosaline has more features than the v0.2 palette. Menus, dialogs, canvases,
timers, tables, tabs, and custom widgets can already be added by hand
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

Rosaline Studio v0.2.0 is an intentionally small early release. Generated code
and saved designs are designed to stay understandable while the visual tooling
grows. The design schema is versioned, but the Studio API and file format remain
experimental until v1.0. Version-1 design files are intentionally incompatible
with the cleaner event model in version 2.

## License

Rosaline Studio is free software licensed under the
[GNU Lesser General Public License v3.0 or later](LICENSE).

Generated application code is intended for the application developer to own
and license as they choose. Rosaline itself remains under its own
LGPL-3.0-or-later license.
Dependencies retain their own licenses; see
[THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).

Copyright (C) 2026 Britney Lozza and Rosaline Studio contributors.
