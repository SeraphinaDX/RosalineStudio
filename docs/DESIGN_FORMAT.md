# Rosaline design format

Rosaline Studio stores visual projects as UTF-8 JSON files ending in
`.rosaline`. The current schema version is `3`.

## Project fields

| Field | Type | Meaning |
|---|---|---|
| `version` | integer | Design schema version; currently `3` |
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

`OnCloseRequest` is the one boolean event. Its handler must return `true` to
allow the close or `false` to keep the form open. Other form and widget event
handlers do not return a value.

## Widget fields

Every widget has a globally unique internal `id`, a `kind`, and an exported Go
`component` name. Fields that do not apply to a widget are omitted when empty.

| Field | Used by | Meaning |
|---|---|---|
| `component` | every widget | Unique generated component-reference name |
| `text` | labels, buttons, inputs, checks | Visible text or placeholder |
| `name` | stateful controls | Preferred generated Go state-field name |
| `asset` | images | Safe filename in the design's `.assets` folder |
| `events` | interactive controls | Event names mapped to Go handler methods |
| `options` | combo boxes | Available choices |
| `children` | layouts | Nested widgets in display order |
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

## Example

```json
{
  "version": 3,
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
      }
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

Studio automatically migrates a version-2 single-form design to version 3 as
a `MainForm`; save the file to keep the upgraded structure. Version-1 designs
used a generic string action dispatcher and remain intentionally incompatible.
Commit important design files to Git before opening them in a newer Studio
version.
