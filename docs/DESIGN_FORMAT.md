# Rosaline design format

Rosaline Studio stores visual projects as UTF-8 JSON files ending in
`.rosaline`. The current schema version is `8`.

## Project fields

| Field | Type | Meaning |
|---|---|---|
| `version` | integer | Design schema version; currently `8` |
| `module` | string | Generated Go module path |
| `actions` | action array | Reusable commands shared by menu items and toolbars |
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
| `toolbar` | toolbar item array | Ordered action buttons and separators for this form |
| `components` | component array | Form-owned timers and file dialogs |

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
| `action` | items | Optional shared project-action ID |
| `handler` | items | Named no-result click method |
| `shortcut` | items | Rosaline shortcut such as `Primary+S` or `F5` |

Separators have no other fields. Menus cannot have handlers or shortcuts, and
items cannot contain children. When an item has `action`, the shared action's
caption, handler, and shortcut take precedence over its local fallback values.

## Actions and toolbars

Every project action has a globally unique internal `id` and a unique exported
`name` such as `SaveAction`.

| Field | Meaning |
|---|---|
| `id` | Stable internal action identity |
| `name` | Exported designer name |
| `text` | Caption used by linked menu items and toolbar buttons |
| `handler` | Optional no-result execute method |
| `shortcut` | Optional menu shortcut such as `Primary+S` |

Each form's `toolbar` contains items with a unique `id` and a `kind` of
`Action` or `Separator`. Action items also have an `action` field referring to
a project-action ID. Separators have no action field.

## Nonvisual component fields

Every nonvisual component has a globally unique internal `id`, a unique
exported `name`, and a `kind` of `Timer`, `OpenFileDialog`, or
`SaveFileDialog`. Components belong to a form but are exposed together through
the generated `app.Components()` registry.

Timer fields:

| Field | Meaning |
|---|---|
| `interval` | Positive interval in milliseconds |
| `repeating` | Use a repeating timer instead of a one-shot timer |
| `enabled` | Start when the owning form opens |
| `handler` | Optional no-result tick method |

File-dialog fields:

| Field | Meaning |
|---|---|
| `title` | Native dialog title |
| `initialDirectory` | Optional starting directory |
| `initialFile` | Optional starting filename |
| `defaultExtension` | Extension supplied when saving without one |
| `filters` | Lines formatted as `Name | .ext, .ext` |

```json
"components": [
  {
    "id": "component-1",
    "kind": "Timer",
    "name": "RefreshTimer",
    "interval": 1000,
    "repeating": true,
    "enabled": true,
    "handler": "RefreshTimerTick"
  },
  {
    "id": "component-2",
    "kind": "OpenFileDialog",
    "name": "OpenDocumentDialog",
    "title": "Open a document",
    "filters": ["Text files | .txt, .md", "All files | *"]
  }
]
```

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
| `data` | lists, tables, trees, radio groups | Designer data, one source line per array item |
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
| `horizontal` | radio groups | Arrange choices from left to right |

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

## Data controls

The `data` array uses a small line-oriented format chosen by `kind`:

| Kind | Line format |
|---|---|
| `List` | One visible item per line |
| `RadioGroup` | `Label = value`, or one string used for both |
| `Table` | Cells separated by `|`; the first line contains headings |
| `Tree` | A path separated by `/`; shared path segments become shared nodes |

For example:

```json
{
  "id": "node-7",
  "kind": "Table",
  "component": "ComponentTable",
  "events": {"OnSelect": "ComponentSelected"},
  "data": [
    "Component | Kind | Status",
    "MainLayout | Column | Ready",
    "ProjectTree | Tree | Selected"
  ],
  "expand": true
}
```

Lists and tables support `OnSelect` and `OnActivate`. Trees support
`OnSelect`, `OnActivate`, and `OnExpand`. Radio groups bind a generated Go
string named by `name` and support `OnChange`.

## Example

```json
{
  "version": 8,
  "module": "example.com/greeting",
  "actions": [
    {
      "id": "action-1",
      "name": "SettingsAction",
      "text": "Settings",
      "handler": "ShowSettings",
      "shortcut": "Primary+,"
    }
  ],
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
              "action": "action-1",
              "text": "Settings"
            }
          ]
        }
      ],
      "toolbar": [
        {"id": "tool-1", "kind": "Action", "action": "action-1"}
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

Studio automatically migrates version-2 single-form and version-3 through
version-7 multi-form designs to version 8; save the file to keep the upgraded
structure. Version-1 designs used a generic string action dispatcher and
remain intentionally incompatible.
Commit important design files to Git before opening them in a newer Studio
version.
