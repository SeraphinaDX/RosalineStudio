# Rosaline design format

Rosaline Studio stores visual projects as UTF-8 JSON files ending in
`.rosaline`. The current schema version is `2`.

## Project fields

| Field | Type | Meaning |
|---|---|---|
| `version` | integer | Design schema version; currently `2` |
| `module` | string | Generated Go module path |
| `title` | string | Application window title |
| `width` | integer | Initial window width in pixels |
| `height` | integer | Initial window height in pixels |
| `padding` | integer | Window content padding |
| `theme` | string | `Rosaline`, `Lavender`, or `Midnight` |
| `root` | widget | Root layout widget |
| `handlers` | object | Go event bodies keyed by handler method name |

## Widget fields

Every widget has a unique `id` and a `kind`. Fields that do not apply to a
widget are omitted when empty.

| Field | Used by | Meaning |
|---|---|---|
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

State names are converted to exported Go identifiers. Duplicate names gain a
numeric suffix. For example, `display name` and `display-name` become
`DisplayName` and `DisplayName2`.

## Example

```json
{
  "version": 2,
  "module": "example.com/greeting",
  "title": "Greeting",
  "width": 720,
  "height": 520,
  "padding": 16,
  "theme": "Lavender",
  "handlers": {
    "GreetClick": "rosaline.Message(\"Hello\", \"Welcome, \"+app.State.Name+\"!\")"
  },
  "root": {
    "id": "root",
    "kind": "Column",
    "children": [
      {
        "id": "node-1",
        "kind": "Label",
        "text": "What is your name?",
        "bold": true
      },
      {
        "id": "node-2",
        "kind": "TextBox",
        "text": "Your name",
        "name": "Name"
      },
      {
        "id": "node-3",
        "kind": "Button",
        "text": "Say hello",
        "events": {
          "OnClick": "GreetClick"
        },
        "primary": true
      }
    ],
    "gap": 10,
    "padding": 12,
    "expand": true
  }
}
```

## Compatibility and validation

Studio rejects unknown JSON fields, unknown widget kinds, repeated IDs,
children inside non-container controls, and more than one child in a `Card` or
`Scroll`. It also validates event names, handler identifiers, and Go syntax.
Strict loading catches typing mistakes rather than silently losing design data.

Version-1 designs used a generic string action dispatcher and are intentionally
not compatible with version 2. Future schema changes will continue to use the
top-level version number. Commit design files to Git before opening important
work in a newer Studio version.
