# Rosaline design format

Rosaline Studio stores visual projects as UTF-8 JSON files ending in
`.rosaline`. The current schema version is `5`.

## Project fields

| Field | Type | Meaning |
|---|---|---|
| `version` | integer | Design schema version; currently `5` |
| `module` | string | Generated Go module path |
| `forms` | form array | Primary form first, followed by reusable secondary forms |
| `handlers` | object | Go event bodies keyed by handler method name |

The first item in `forms` is always the primary application window. It cannot
be deleted. Other forms generate as reusable `rosaline.Window` handles owned
by the primary form.

## Form fields

| Field | Type | Meaning |
|---|---|---|
| `id` | string | Stable internal form identity |
| `name` | string | Unique exported Go name used by `app.Windows()` |
| `title` | string | Native window title |
| `width`, `height` | integer | Initial window size in pixels |
| `padding` | integer | Window content padding |
| `theme` | string | `Rosaline`, `Lavender`, or `Midnight` |
| `root` | widget | Root layout for this form |
| `events` | object | `OnOpen`, `OnCloseRequest`, or `OnClose` handler methods |
| `menus` | menu array | Top-level menus for this form's native menu bar |

`OnCloseRequest` is the one boolean event. Its handler must return `true` to
allow the close or `false` to keep the form open. Other form and widget event
handlers do not return a value.

## Menu fields

Every menu entry has a globally unique `id` and a `kind` of `Menu`, `Item`, or
`Separator`. Only `Menu` entries may appear at the top level.

| Field | Used by | Meaning |
|---|---|---|
| `text` | menus and items | Visible caption |
| `children` | menus | Nested items, submenus, and separators |
| `handler` | items | Named no-result click method |
| `shortcut` | items | Rosaline shortcut such as `Primary+S` or `F5` |

Separators have no other fields. Menus cannot have handlers or shortcuts, and
items cannot contain children.

## Widget fields

Every widget has a globally unique internal `id`, a `kind`, and an exported Go
`component` name. Fields that do not apply to a widget are omitted when empty.

| Field | Used by | Meaning |
|---|---|---|
| `component` | every widget | Unique generated component-reference name |
| `text` | labels, buttons, inputs, checks, tab pages | Visible text, placeholder, or page title |
| `name` | stateful controls | Preferred generated Go state-field name |
| `asset` | images | Safe filename in the design's `.assets` folder |
| `events` | interactive controls | Event names mapped to Go handler methods |
| `options` | combo boxes | Available choices |
| `children` | layouts, Tabs, TabPage | Nested widgets or pages in display order |
| `gap`, `padding` | layouts | Spacing in pixels |
| `columns` | grid | Number of equal columns |
| `width`, `height` | widgets | Explicit size when both are positive |
| `minimum`, `maximum`, `step` | sliders/progress | Numeric configuration |
| `expand` | layouts/selected controls | Consume available space |
| `primary` | buttons | Use primary styling |
| `bold` | labels | Use bold text |
| `password` | text boxes | Mask entered text |
| `vertical` | sliders/progress | Use vertical orientation |

Component names are exported Go identifiers such as `SaveButton`. They are
available to event code through `app.Widgets().SaveButton`. Form names work the
same way through `app.Windows().SettingsForm`.

State names are converted to exported Go identifiers. Duplicate names gain a
numeric suffix. For example, `display name` and `display-name` become
`DisplayName` and `DisplayName2`.

## Tabs and tab pages

A `Tabs` widget contains one or more direct `TabPage` children. A `TabPage`
contains ordinary controls and layouts and uses `text` for its visible header.
Tabs cannot directly contain ordinary controls, and a TabPage cannot appear
outside Tabs. Studio always preserves at least one page.

`TabPage` is structural: generated code turns it into
`rosaline.Tab(title, rosaline.Column(...))`. It therefore has no field in
`UIWidgets`. The parent Tabs does have a typed `*rosaline.TabsWidget` field and
may assign an `OnChange` handler.

```json
{
  "id": "node-3",
  "kind": "Tabs",
  "component": "PreferencesTabs",
  "events": {"OnChange": "PreferencesPageChanged"},
  "children": [
    {
      "id": "node-4",
      "kind": "TabPage",
      "component": "GeneralPage",
      "text": "General",
      "children": [
        {
          "id": "node-5",
          "kind": "CheckBox",
          "component": "StartupCheckBox",
          "text": "Open at sign in",
          "name": "OpenAtStartup"
        }
      ],
      "gap": 10,
      "padding": 12,
      "expand": true
    }
  ],
  "expand": true
}
```

## Example

```json
{
  "version": 5,
  "module": "example.com/greeting",
  "forms": [
    {
      "id": "form-1",
      "name": "MainForm",
      "title": "Greeting",
      "width": 720,
      "height": 520,
      "padding": 16,
      "theme": "Lavender",
      "root": {
        "id": "root",
        "kind": "Column",
        "component": "MainLayout",
        "children": [
          {
            "id": "node-1",
            "kind": "Button",
            "component": "SettingsButton",
            "text": "Settings",
            "events": {"OnClick": "ShowSettings"}
          }
        ],
        "gap": 10,
        "padding": 12,
        "expand": true
      },
      "menus": [
        {
          "id": "menu-1",
          "kind": "Menu",
          "text": "File",
          "children": [
            {
              "id": "menu-2",
              "kind": "Item",
              "text": "Settings",
              "handler": "ShowSettings",
              "shortcut": "Primary+,"
            }
          ]
        }
      ]
    },
    {
      "id": "form-2",
      "name": "SettingsForm",
      "title": "Settings",
      "width": 560,
      "height": 400,
      "padding": 16,
      "theme": "Rosaline",
      "events": {"OnCloseRequest": "AllowSettingsClose"},
      "root": {
        "id": "node-2",
        "kind": "Column",
        "component": "SettingsLayout",
        "gap": 10,
        "padding": 12,
        "expand": true
      }
    }
  ],
  "handlers": {
    "ShowSettings": "app.Windows().SettingsForm.Show()",
    "AllowSettingsClose": "return true"
  }
}
```

## Compatibility and validation

Studio rejects unknown JSON fields, unknown widget kinds, repeated form or
widget IDs, repeated form or component names, children inside non-container
controls, and more than one child in a `Card` or `Scroll`. It validates event
names, handler identifiers, handler return shape, and Go syntax. Widget moves
stay within a form, while copy and paste can safely cross forms.

Studio automatically migrates version-2 single-form and version-3 or version-4
multi-form designs to version 5; save the file to keep the upgraded structure.
Version-1 designs used a generic string action dispatcher and remain
intentionally incompatible.
Commit important design files to Git before opening them in a newer Studio
version.
