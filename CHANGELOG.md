# Changelog

All notable Rosaline Studio changes are documented here.

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
