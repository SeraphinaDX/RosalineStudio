# Designing menus

Rosaline Studio's Menu Designer builds a native menu bar without requiring you
to write layout code. Every form owns its own menu tree, so a primary window
and secondary forms can expose different commands.

## Open the designer

Select a form, then open the center **Menus** tab or choose
**Project > Menu Designer**. The tree shows the exact hierarchy that Studio
will generate.

## Add entries

- **Add Top Menu** creates a menu-bar heading such as File or Help.
- **Add Item** adds a clickable command to the selected menu.
- **Add Submenu** adds a nested menu to the selected menu.
- **Add Separator** visually groups nearby commands.
- **Up**, **Down**, and **Delete** organize the selected entry.

Select a menu before adding an entry. When an item is selected, Studio uses its
containing menu so adding another item remains quick.

## Edit an item

The right **Menu** inspector provides:

| Property | Applies to | Example |
|---|---|---|
| Shared action | Items | `SaveAction` |
| Caption | Menus and items | `Save As...` |
| Shortcut | Items | `Primary+Shift+S` |
| Click handler | Items | `SaveAsClick` |

Choose **Apply Menu Properties** after changing fields. Choose
**Assign and Edit Click**, or double-click an item, to create and open its
normal Go event method in the Code view.

When **Shared action** is selected, that action supplies the item's caption,
shortcut, and handler. Editing the action updates every linked menu item and
toolbar button. Choose **(no shared action)** to use the item's local fields
instead. See [Shared actions and visual toolbars](ACTIONS_TOOLBARS.md).

```go
app.Widgets().StatusLabel.SetText("Saved")
rosaline.Message("Save", "The menu item worked.")
```

Menu events can use `app.State`, `app.Widgets()`, and `app.Windows()` exactly
like button and form events. Studio validates the Go body before saving it.

## Shortcuts

Use Rosaline's readable shortcut notation:

- `Primary+S` means Control+S on Linux and Windows, Command+S on macOS.
- `Primary+Shift+Z` combines multiple modifiers.
- `F5` names a function key directly.

Leaving Shortcut empty is valid.

## Generated code

Studio converts the visual tree into normal Rosaline calls:

```go
rosaline.MenuBar(
	rosaline.Menu("File",
		rosaline.MenuItem("Save", func() { app.SaveClick() }).Shortcut("Primary+S"),
		rosaline.MenuSeparator(),
		rosaline.Menu("Recent",
			rosaline.MenuItem("Notes", func() { app.OpenNotes() }),
		),
	),
)
```

The structure lives in `ui_generated.go`; named click methods live in
`events_generated.go`. Continue using Studio to edit both. Developer-owned
files remain untouched.

## Complete example

Open `examples/notepad.rosaline`, press F5, and try its File, Edit, and Help
menus and toolbar. It demonstrates shared actions, nested menus, separators,
shortcuts, editor operations, and a close-request event in one small
application.
