# Rosaline Studio quick start

This guide creates a small greeting application without asking you to learn a
GUI framework first.

## 1. Start Studio

From the Rosaline Studio repository:

```bash
env CGO_ENABLED=0 go run .
```

Studio opens with a working starter form: a heading, a text box, and a primary
Continue button.

## 2. Understand the three panels

- The left panel contains the widget palette and application hierarchy.
- The center is a visual structural preview. Click a widget to select it.
- The right panel edits the selected widget or the whole application.

Double-clicking a palette item adds it to the selected container. If a control
is selected, Studio adds the new widget to that control's parent container.

`Card` and `Scroll` accept one child. Put a `Column`, `Row`, or `Grid` inside
when you need several controls in one of them.

## 3. Personalize the starter form

1. Select the heading in the preview or hierarchy.
2. Change **Text or placeholder** to `Welcome to my first app`.
3. Choose **Apply Widget Properties**.
4. Select the text box and keep its state field name as `Name`.
5. Select the button, change its action to `greet`, and apply.

Property edits take effect when you press their Apply button. This makes it
easy to change several related values as one undoable operation.

## 4. Configure the application

Open the right panel's **Application** tab and set:

- Window title: `My Greeting App`
- Go module path: a path you control, such as `example.com/greeting`
- Theme: `Lavender`

Choose **Apply Application Settings**.

## 5. Save the design

Choose **File > Save As** and select an empty folder. Save the file as
`greeting.rosaline`.

Studio generates application files in the same folder as the design. Using one
folder for each application keeps the design and its generated Go module
together.

## 6. Generate and run

Press F5, or choose **File > Build and Run**. Studio will:

1. Save the design.
2. Regenerate `ui_generated.go` and `state_generated.go`.
3. Create the developer files that do not exist yet.
4. Run the generated project with `CGO_ENABLED=0`.

Close the generated application window to return to editing.

## 7. Add real behavior

Open `handlers.go` in the generated application folder. Replace its action
switch with:

```go
func (app *Application) Action(name string) {
	switch name {
	case "greet":
		person := app.State.Name
		if person == "" {
			person = "friend"
		}
		rosaline.Message("Hello", "Welcome, "+person+"!")
	default:
		fmt.Println("Rosaline action:", name)
	}
}
```

`app.State.Name` is a normal Go string. The generated text box receives its
address and updates it when the user types.

Run again from Studio. Your handler stays in place because Studio never
overwrites `handlers.go`.

## 8. Keep designing safely

You can now add widgets, rename state fields, switch themes, or completely
rearrange the layout. On every generation:

- Studio-owned files are refreshed.
- Developer-owned files are preserved byte for byte.

Do not hand-edit files ending in `_generated.go`; make visual changes in Studio
instead. Put business logic, files, networking, and custom behavior in
`handlers.go` or additional `.go` files you create.

## Keyboard shortcuts

| Shortcut | Action |
|---|---|
| Primary+N | New design |
| Primary+O | Open design |
| Primary+S | Save design |
| Primary+Shift+S | Save as |
| Primary+G | Generate project |
| F5 | Generate and run |
| Primary+Z | Undo |
| Primary+Shift+Z | Redo |
| Alt+Up / Alt+Down | Reorder selected widget |
| Delete | Delete selected widget |
| F1 | Quick help |

On Linux and Windows, Primary is Control. On macOS, it is Command.

