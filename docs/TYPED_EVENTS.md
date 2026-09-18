# Typed event methods

Rosaline Studio generates ordinary Go methods for visual events. Events that
carry information now pass it directly into the application method instead of
making the method query the control again.

Select a component, open **Events**, and choose an event. The inspector shows
the required handler shape, and the complete method signature appears above
the Event Code editor.

## Event parameters

| Component | Event | Generated parameters |
|---|---|---|
| Button, Image | `OnClick` | none |
| TextBox | `OnChange`, `OnSubmit` | `value string` |
| TextArea, ComboBox | `OnChange` | `value string` |
| CheckBox | `OnChange` | `checked bool` |
| RadioGroup | `OnChange` | `value string` |
| Slider | `OnChange` | `value float64` |
| List | `OnSelect`, `OnActivate` | `index int, value string` |
| Table | `OnSelect`, `OnActivate` | `index int, row []string` |
| Tree | `OnSelect`, `OnActivate` | `node *rosaline.TreeNode` |
| Tree | `OnExpand` | `node *rosaline.TreeNode, expanded bool` |
| Tabs | `OnChange` | `index int, title string` |
| Menus, actions, timers, form lifecycle | their normal event | none |

The names are part of the generated method, so event code can use them like
ordinary Go variables.

## Examples

A TextBox change method can use the new text immediately:

```go
func (app *Application) NameChanged(value string) {
	app.Widgets().GreetingLabel.SetText("Hello, " + value)
}
```

A table method receives a copy of the selected cells:

```go
func (app *Application) ComponentSelected(index int, row []string) {
	if len(row) > 0 {
		app.Widgets().StatusLabel.SetText(row[0])
	}
}
```

A tree expansion method knows both which node changed and its new state:

```go
func (app *Application) ProjectExpanded(node *rosaline.TreeNode, expanded bool) {
	if expanded {
		app.Widgets().StatusLabel.SetText("Opened " + node.Label())
	}
}
```

## Shared handlers

Several events may share one method when their parameter names, parameter
types, and return value are identical. For example, TextBox `OnChange` and
`OnSubmit` can both use a method accepting `value string`.

Studio rejects incompatible sharing immediately and explains both signatures.
This prevents a List event from silently sharing a no-argument menu handler or
a string change handler from being reused for a Boolean CheckBox change.

## Existing designs

The `.rosaline` design format still stores event bodies rather than generated
method declarations, so no schema migration is necessary. Existing bodies that
ignore the new parameters continue to work because Go permits unused function
parameters. If an old body declares a top-level variable with the same name as
a new parameter, rename that local variable; Studio's syntax validation points
to the conflict before saving.
