# Canvas Designer

Canvas is Rosaline Studio's custom 2D drawing surface. Use it for paint tools,
graphs, diagrams, image editors, board games, visualizations, and other
interfaces that do not fit ordinary controls.

The designer shows the Canvas's position, size, background, and selection. It
does not execute application Go code inside Studio. Press **F5** to see the
real drawing and interact with it.

## Add a Canvas

1. Select a layout in the form preview or Hierarchy.
2. Add **Canvas** from the Widget Palette.
3. Give it a memorable component name such as `DrawingCanvas`.
4. Under **Layout**, set Width and Height. The defaults are 480 by 300 pixels.
5. Enable **Use available space** when the drawing area should grow with its
   layout.
6. Under **Content**, enter a background such as `#fff7fb`.
7. Under **Style**, enable keyboard focus if this Canvas should initially
   receive key events.

Canvas backgrounds accept `#RGB`, `#RRGGBB`, and `#RRGGBBAA` colors. Studio
rejects an invalid value before it can reach generated code.

## Draw the scene

Select the Canvas, open **Events**, choose `OnDraw`, and choose **Assign and
Edit**. Studio creates this shape:

```go
func (app *Application) DrawingCanvasDraw(canvas *rosaline.DrawingCanvas) {
	canvas.Clear(rosaline.Hex("#fff7fb"))
}
```

Add drawing commands to the method body:

```go
canvas.Clear(rosaline.Hex("#fff7fb"))
canvas.Line(20, 20, 220, 20, 3, rosaline.Rose)
canvas.FillRect(30, 50, 100, 70, rosaline.SoftRose)
canvas.FillCircle(210, 100, 36, rosaline.Rose)
canvas.Text("Hello, Canvas!", 24, 150, rosaline.TextStyle{
	Color: rosaline.Hex("#7d3156"),
	Size:  18,
})
```

`OnDraw` must draw the complete current scene. Rosaline clears and rebuilds the
pixel image whenever the Canvas redraws.

## Redraw after state changes

The generated component reference is a `*rosaline.CanvasWidget`:

```go
app.Widgets().DrawingCanvas.Redraw()
```

Call `Redraw()` after a button, timer, slider, menu, or other non-Canvas event
changes drawing state. Rosaline automatically redraws after Canvas mouse
callbacks, so mouse handlers normally only update state.

For example, a Slider event can update a drawing immediately:

```go
func (app *Application) RadiusChanged(value float64) {
	app.Widgets().DrawingCanvas.Redraw()
}
```

The `OnDraw` method can then read `app.State.Radius`.

## Mouse input

Canvas offers `OnMouseDown`, `OnDoubleClick`, `OnMouseMove`, and `OnMouseUp`.
Each receives:

```go
event rosaline.MouseEvent
```

The event contains:

| Field | Meaning |
|---|---|
| `X`, `Y` | Position measured from the Canvas's top-left corner |
| `Button` | `MouseLeft`, `MouseMiddle`, `MouseRight`, or `MouseNone` |
| `Dragging` | A mouse button is held while the pointer moves |
| `Shift`, `Control`, `Alt` | Modifier-key state |

A draggable object can update its state directly:

```go
if event.Dragging && event.Button == rosaline.MouseLeft {
	app.State.ObjectX = event.X
	app.State.ObjectY = event.Y
}
```

## Keyboard input

Canvas offers `OnKeyDown` and `OnKeyUp`. Each receives:

```go
event rosaline.KeyEvent
```

Enable **Give canvas keyboard focus** when the drawing surface should receive
focus as soon as the window opens. A Canvas with key handlers also participates
in normal Tab focus order, and clicking it gives it focus.

```go
switch event.Key {
case rosaline.KeyLeft:
	app.State.ObjectX -= 5
case rosaline.KeyRight:
	app.State.ObjectX += 5
default:
	return
}
app.Widgets().DrawingCanvas.Redraw()
```

`event.Text` contains produced text for printable keys. The event also reports
`Shift`, `Control`, `Alt`, and cross-platform `Primary` modifiers.

## Save the picture

`Picture()` renders the Canvas off-screen using the same `OnDraw` method. It is
available even before the widget is mounted:

```go
path, ok := app.Components().SavePictureDialog.Execute()
if !ok {
	return
}
if err := app.Widgets().DrawingCanvas.Picture().SavePNG(path); err != nil {
	rosaline.Error("Could not save picture", err.Error())
}
```

Rosaline also supports `SaveAVIF`. Add a Save File Dialog under **Components**
to keep the destination and filters reusable.

## Larger drawing applications

Studio stores event bodies in the design. Put reusable types, slices of points,
file formats, algorithms, and imported packages in `handlers.go` or another
developer-owned Go file under **Source**. Generated files remain small and can
call those helpers without risking the custom code during regeneration.

Open `examples/canvas_playground.rosaline` for a complete design combining
custom drawing, mouse dragging, double-click, arrow keys, sliders, redraw, a
Save File Dialog, and PNG export.
