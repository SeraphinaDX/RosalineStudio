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
- The center switches between the visual Form and integrated Go Code views.
- The right inspector has Properties, Events, and Application tabs.

Double-clicking a palette item adds it to the selected container. If a control
is selected, Studio adds the new widget to that control's parent container.

Select an unwanted widget and use **Delete Selected** above the form, the
hierarchy's **Delete** button, **Edit > Delete Selected Widget**, or a
right-click menu. Delete also works while the form or hierarchy has keyboard
focus. Populated layouts ask for confirmation, and Primary+Z restores the whole
subtree. The root layout cannot be deleted.

`Card` and `Scroll` accept one child. Put a `Column`, `Row`, or `Grid` inside
when you need several controls in one of them.

## 3. Personalize the starter form

1. Select the heading in the preview or hierarchy.
2. Change **Text or placeholder** to `Welcome to my first app`.
3. Choose **Apply Widget Properties**.
4. Select the text box and keep its state field name as `Name`.
5. Select the button and open the right inspector's **Events** tab.
6. Select `OnClick`, enter `GreetClick`, and choose **Assign and Edit**.

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

Studio creates a separate application folder beside the design, named after
the design file. For example, `greeting.rosaline` generates into `greeting/`.
This keeps generated code isolated even when the design is saved inside the
Rosaline Studio repository.

## 6. Generate and run

Press F5, or choose **File > Build and Run**. Studio will:

On the first run, Studio shows this setup plan and asks you to confirm it.
Nothing is downloaded until you choose Yes.

1. Save the design.
2. Create or update the application subfolder.
3. Regenerate `ui_generated.go`, `state_generated.go`, and
   `events_generated.go`.
4. Create the developer files that do not exist yet.
5. Download Rosaline and its dependencies when needed.
6. Run that application subfolder with `CGO_ENABLED=0`.

Close the generated application window to return to editing.

## 7. Add real behavior

Studio opens the Code view for `GreetClick`. Replace the starter body with:

```go
person := app.State.Name
if person == "" {
	person = "friend"
}
rosaline.Message("Hello", "Welcome, "+person+"!")
```

Choose **Save Event Code**, then return to the Form view. Studio checks the Go
syntax before saving the handler.

`app.State.Name` is a normal Go string. The generated text box receives its
address and updates it when the user types.

Run again from Studio. The generated button calls `app.GreetClick()` directly.

## 8. Add a picture

1. Select a container and add **Image** from the palette.
2. Select the new Image control and choose **Choose Image...** in Properties.
3. Pick a PNG, JPEG, GIF, BMP, TIFF, WebP, or AVIF file.
4. Adjust Width and Height in Layout if desired.
5. To make it interactive, assign its `OnClick` event just like the button.

Studio copies the picture into `greeting.assets/`. During generation it copies
and embeds the asset in the Go application, so the compiled program can find
the picture regardless of its working directory.

## 9. Keep designing safely

You can now add widgets, rename state fields, switch themes, or completely
rearrange the layout. On every generation:

- Studio-owned files are refreshed.
- Developer-owned files are preserved byte for byte.

Do not hand-edit files ending in `_generated.go`; make visual and event changes
in Studio instead. Put reusable logic, additional imports, files, networking,
and custom behavior in `handlers.go` or additional `.go` files you create.

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
| Delete | Delete selected widget while the Form or Hierarchy has focus |
| F1 | Quick help |

On Linux and Windows, Primary is Control. On macOS, it is Command.
