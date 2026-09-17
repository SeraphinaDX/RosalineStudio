# Designing data controls

Rosaline Studio v0.7 adds four controls for browsing and choosing structured
information: List, Table, Tree, and RadioGroup. They use the same visual
workflow as every other component and generate ordinary Rosaline Go.

## The Data inspector

Select a data control and open **Properties > Data**. Enter one source line per
display item or row and choose **Apply Properties**. The form preview updates
immediately. Blank lines are ignored and surrounding spaces are trimmed.

The help above the editor changes for the selected control. Data entered for
one of these controls is stored as a readable `data` string array in the
`.rosaline` design.

## List

Enter one item per line:

```text
All controls
Layouts
Inputs
Data controls
```

List supports `OnSelect` and `OnActivate`. Activation means a double-click or
Enter on the selected item. Event code can read the current item:

```go
if index, value, ok := app.Widgets().KindList.Selected(); ok {
	app.Widgets().StatusLabel.SetText(value)
	_ = index
}
```

## RadioGroup

Enter one choice per line. A plain line uses the same display label and stored
value. Use `Label = value` when application state should use a shorter or more
stable value:

```text
Automatic = auto
Detailed view = details
Compact view = compact
```

Set **State field name** to the generated Go field, such as `ViewMode`. Enable
**Horizontal radio choices** under Style to arrange the choices left to right.
`OnChange` runs after the choice changes, and the new value is already in
`app.State.ViewMode`.

## Table

Use `|` between cells. The first line contains column headings and every later
line is a preview row:

```text
Component | Kind | Status
MainLayout | Column | Ready
ProjectTree | Tree | Selected
```

Short rows are padded and long rows are trimmed by Rosaline. Table supports
`OnSelect` and `OnActivate`:

```go
if row, values, ok := app.Widgets().ComponentTable.Selected(); ok {
	_ = row
	if len(values) > 0 {
		app.Widgets().StatusLabel.SetText(values[0])
	}
}
```

## Tree

Enter one path per line and use `/` between levels. Shared path segments become
one shared parent node:

```text
Project
Project/Forms
Project/Forms/MainForm
Project/Forms/SettingsForm
Project/Assets
Dependencies
```

Generated nodes use the complete path as their value. Tree supports
`OnSelect`, `OnActivate`, and `OnExpand`:

```go
if node, ok := app.Widgets().ProjectTree.Selected(); ok {
	app.Widgets().StatusLabel.SetText(node.Value())
}
```

## Generated source

Studio produces direct calls such as `rosaline.List`, `rosaline.Table`,
`rosaline.Tree`, `rosaline.Node`, `rosaline.RadioGroup`, and `rosaline.Choice`.
Every control has a typed field in `app.Widgets()`, while RadioGroup also gets a
normal string in `app.State`.

Open `examples/data_browser.rosaline` for all four controls working together
with Tabs and named event methods.
