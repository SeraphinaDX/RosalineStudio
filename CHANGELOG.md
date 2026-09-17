# Changelog

All notable Rosaline Studio changes are documented here.

## v0.7.0 - 2026-09-17

- Add `List`, `Table`, `Tree`, and `RadioGroup` to the visual widget palette.
- Add a multiline Data inspector with one consistent beginner-friendly syntax:
  list items, labeled radio values, pipe-separated table cells, and slash-
  separated tree paths.
- Render representative list selection, table headings and rows, nested tree
  paths, and vertical or horizontal radio choices in the form preview.
- Generate typed Rosaline component references and normal `List`, `Table`,
  `Tree`, `Node`, `RadioGroup`, and `Choice` calls.
- Add `OnSelect` and `OnActivate` events for lists and tables, `OnSelect`,
  `OnActivate`, and `OnExpand` for trees, and `OnChange` for radio groups.
- Add a Project Browser example combining tabs, a tree, a table, a list, radio
  choices, component access, and working event methods.
- Advance the strict design schema to version 6 and automatically migrate
  version-2 through version-5 designs.

## v0.6.0 - 2026-09-17

- Add `Tabs` to the visual widget palette with General and Advanced starter
  pages so a useful tabbed form begins with one action.
- Add a dedicated Pages inspector for adding, renaming, duplicating,
  reordering, selecting, and safely deleting tab pages.
- Make tab pages friendly containers: select a page in the preview or
  hierarchy, then add ordinary controls directly to it.
- Preview tab headers and only the selected page's contents while preserving
  page selection as the designer changes controls.
- Add `OnChange` support for Tabs through the normal Events and Code workflow.
- Generate ordinary `rosaline.Tabs`, `rosaline.Tab`, and page `Column` calls,
  plus a typed `*rosaline.TabsWidget` component reference.
- Add a complete Preferences example with three designed pages and working
  change and save events.
- Advance the strict design schema to version 5 and automatically migrate
  version-2 through version-4 designs.

## v0.5.0 - 2026-09-16

- Add a Lazarus-style visual Menu Designer beside the Form and Code views.
- Create top-level menus, menu items, nested submenus, and separators through
  a hierarchical editor with reorder and delete commands.
- Edit captions, keyboard shortcuts, and named click handlers from the Menu
  inspector, then open those handlers directly in Studio's Go editor.
- Show designed menu bars in the scaled form preview and reserve their space
  when previewing widget layout.
- Generate native Rosaline menu bars for every primary or secondary form,
  including nested menus, shortcuts, separators, and application methods.
- Add a complete visual notepad example demonstrating File, Edit, Help, and
  nested menus with working text-editor commands.
- Advance the strict design schema to version 4 and automatically migrate
  version-2 and version-3 designs.
- Require Rosaline v0.19.0 for composable nested menus.

## v0.4.0 - 2026-09-16

- Add a Lazarus-style project form list with commands to create, duplicate,
  select, rename, configure, and delete secondary forms.
- Give every form its own title, resolution, padding, theme, root layout, and
  `OnOpen`, `OnCloseRequest`, and `OnClose` events.
- Generate reusable typed window references through `app.Windows()`, allowing
  event code such as `app.Windows().SettingsForm.Show()` and `.Close()`.
- Generate each secondary form as a normal `rosaline.Window` owned by the main
  form, while the first designed form remains the primary application window.
- Advance the strict design schema to version 3 and automatically migrate
  version-2 single-form designs into a `MainForm`.
- Prevent cross-form widget dragging while retaining copy and paste between
  forms with globally unique component, state, and widget identities.
- Validate boolean close-request handlers separately from ordinary event
  methods and provide a safe `return true` starter body.
- Require Rosaline v0.18.0 for primary and secondary window lifecycle events.

## v0.3.0 - 2026-09-16

- Give every visual widget a unique, editable component name and show those
  names consistently in the hierarchy, form toolbar, and inspector.
- Generate typed component references through `app.Widgets()` so event methods
  can directly change labels, buttons, inputs, checks, images, and layouts.
- Add Cut, Copy, Paste, and Duplicate to the Edit menu and right-click menus,
  plus focused designer keyboard shortcuts.
- Give pasted subtrees fresh IDs, component names, and state fields while
  intentionally retaining their assigned event handlers.
- Automatically add component names when opening older version-2 designs.
- Require Rosaline v0.17.0 for runtime text, checked-state, focus, and
  enabled-state control.

## v0.2.1 - 2026-09-15

- Put a prominent **Delete Selected** command directly above the form preview
  while retaining the hierarchy button and Edit-menu command.
- Add right-click menus to both the form and hierarchy with edit, reorder, and
  delete commands.
- Make the Delete key operate only while the form or hierarchy has focus, so it
  cannot remove a widget while editing text or Go code.
- Select the nearest remaining sibling after deletion instead of unexpectedly
  jumping back to the containing layout.
- Confirm removal of populated layouts, protect the root layout, retain event
  methods and imported assets, and make the entire subtree undoable.
- Require Rosaline v0.16.1 for canvas and tree context menus and focus-safe tree
  key events.

## v0.2.0 - 2026-09-15

- Reshape Studio around a Lazarus-style Form and Code workflow.
- Add an Events inspector with `OnClick`, `OnChange`, and `OnSubmit` events
  appropriate to each supported control.
- Open the default event by double-clicking a control in the form preview.
- Add an integrated Go event-body editor with syntax validation, saved-state
  prompts, and generated `events_generated.go` methods.
- Replace the generic string-based `Application.Action` dispatcher with named
  Go handler methods.
- Add Image to the component palette with a file chooser, real preview,
  aspect-preserving dimensions, click events, and AVIF support.
- Copy imported pictures into a design-owned asset folder and embed them in
  generated applications so deployed programs do not depend on a working
  directory.
- Copy design assets during Save As and preserve developer-owned Go files.
- Advance the strict design schema to version 2. Version-1 designs are not
  compatible with this release.
- Require Rosaline v0.16.0 for embedded images, fitted image controls, canvas
  image previews, double-click events, and mounted editor focus.

## v0.1.5 - 2026-09-07

- Added a first-run project setup prompt before Studio creates generated files
  or downloads Rosaline.
- Explain the target folder, generation, dependency download, build, and run
  steps before asking the user to continue.
- Detect incomplete setup, including a missing `go.sum`, and offer setup again.
- Added regression coverage for generated-project setup detection.

## v0.1.4 - 2026-09-07

- Made Build and Run automatically download Rosaline and all required Go
  modules before compiling the generated application.
- Build generated applications independently of an enclosing `go.work` file.
- Force generated previews to retain Rosaline's pure-Go `CGO_ENABLED=0` build.
- Include Rosaline v0.15.0 checksums in Studio's own `go.sum`.
- Added regression coverage for the generated command environment.

## v0.1.3 - 2026-09-07

- Fixed Build and Run launching Rosaline Studio when a design was saved in the
  Studio repository.
- Generate each design into a separate sibling subfolder named after the
  `.rosaline` file.
- Make Build and Run explicitly target that generated application folder.
- Report the generated folder name in Studio's status line.
- Added a regression test ensuring generation never reuses the design folder.

## v0.1.2 - 2026-09-07

- Removed the nested scrolling viewport from the widget inspector, eliminating
  its unnecessary horizontal scrollbar.
- Split widget properties into compact Content, Layout, and Style tabs.
- Kept inspector fields, selectors, and action buttons at natural widths
  instead of stretching them across the panel.
- Shortened the generated-file safety explanation so it fits naturally.
- Slightly widened the inspector panel without changing Rosaline itself.

## v0.1.1 - 2026-09-07

- Fixed labels, text boxes, buttons, and other ordinary controls stretching to
  consume the entire preview height.
- Made explicitly expandable controls absorb remaining preview space while
  ordinary controls retain natural sizes.
- Made the preview window reflect the configured application resolution and
  display that resolution in its title bar.
- Removed accidental double expansion around the fixed drawing canvas.
- Reduced Studio's initial window and side-panel sizes.
- Changed newly added nested layouts to use natural sizing by default.
- Added regression tests for resolution changes and natural sizing.

## v0.1.0 - 2026-09-07

- Added a visual widget palette, hierarchy tree, and themed wireframe preview.
- Added click selection and drag-based hierarchy rearrangement.
- Added property inspectors for widgets and application settings.
- Added versioned `.rosaline` JSON design files with strict validation.
- Added undo, redo, new, open, save, and unsaved-change handling.
- Added safe Go generation with explicit generated/developer file ownership.
- Added background build-and-run support with useful compiler output.
- Added a quick start, design-format reference, and welcome sample.
- Added Linux-friendly pure-Go builds through Rosaline v0.15.0.
