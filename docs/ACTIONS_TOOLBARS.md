# Shared actions and visual toolbars

Rosaline Studio v0.8 adds Lazarus-style project actions. An action is one
reusable application command with a name, caption, keyboard shortcut, and Go
execute handler. A menu item and any number of toolbar buttons can all refer to
the same action.

Change the action once and every linked control changes with it. There is no
need to keep separate Save menu and Save toolbar handlers synchronized.

## Create an action

1. Open the center **Actions** tab or choose **Project > Actions and Toolbar**.
2. Choose **New Action**.
3. Use the action properties below the two lists.
4. Give it an exported name such as `SaveAction`.
5. Set its caption, optional shortcut, and execute-handler name.
6. Choose **Apply Action Properties**.
7. Choose **Assign and Edit Execute** or double-click the action to edit its Go
   method.

Action names identify the design entry. Execute handlers are normal methods on
the generated `Application` type. A useful action might look like this:

```text
Action name: SaveAction
Caption: Save
Shortcut: Primary+S
Execute handler: SaveDocument
```

The Code view edits `SaveDocument` exactly like a widget event:

```go
app.Widgets().StatusLabel.SetText("Saved")
```

## Build a toolbar

The right half of the Actions designer shows the active form's toolbar.

1. Select a project action on the left.
2. Choose **Add Action** under the toolbar.
3. Add separators where commands should be grouped.
4. Use **Left** and **Right** to reorder the selected toolbar item.
5. Use **Remove** to take an item off this form without deleting its action.

Each form has its own toolbar, while project actions are shared by every form.
Duplicating a form duplicates the toolbar arrangement but keeps the same
shared action links.

The form preview reserves space for the toolbar and shows its button captions.
Generated applications use ordinary `rosaline.Row`, `rosaline.Button`, and
vertical `rosaline.Separator` widgets, so the output remains readable Go.

## Reuse an action in a menu

1. Create or select a menu item in the **Menus** designer.
2. Open the right **Menu** inspector.
3. Choose the action under **Shared action**.
4. Choose **Apply Menu Properties**.

The menu item now gets its caption, shortcut, and handler from the action. Its
local values remain as a safe fallback if the action is later deleted. To make
the menu item independent again, choose **(no shared action)** and apply its
local properties.

Double-clicking a linked menu item opens the shared action's execute handler.

## Deleting safely

Deleting an action asks for confirmation and is one undoable change. Studio
removes toolbar buttons using that action and turns linked menu entries back
into ordinary menu items with their stored fallback properties. The Go handler
body remains in the design so behavior is not silently destroyed.

Open [`../examples/notepad.rosaline`](../examples/notepad.rosaline) to see New,
Undo, Redo, Cut, Copy, and Paste shared between native menus and a generated
toolbar.
