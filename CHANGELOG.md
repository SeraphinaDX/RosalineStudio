# Changelog

All notable Rosaline Studio changes are documented here.

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
