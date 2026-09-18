# Project Source Editor

Rosaline Studio includes a small, Lazarus-style source workspace for the Go
project generated beside your `.rosaline` design. It is intended for event
handlers, application logic, helpers, and other code that belongs to you.

Open it with **Project > Project Source** or select the **Source** workspace.
Studio saves and generates the project first when setup is needed.

## Project Files

The file tree separates the project into four groups:

- **Developer Source** contains handwritten `.go` files. These are editable.
- **Generated (Read Only)** contains Studio's `*_generated.go` files. You can
  inspect and copy from them, but Studio prevents accidental edits.
- **Project Files** contains files such as `go.mod`, `go.sum`, and README files.
- **Assets** contains the pictures copied into the generated application.

Select a text file to open it. Several files can stay open; use the choices
above the editor to switch between them. An asterisk marks unsaved source.

## Add application code

Enter a simple filename such as `documents.go`, then choose **New**. Studio
creates a normal Go file with `package main` and opens it immediately.

Use **Rename** or **Delete** on the active developer file. `main.go` and
`handlers.go` are protected because generated applications require them.
Names ending in `_generated.go` are reserved for Studio.

## Save and format

**Save** writes the active file. **Save All** writes every changed developer
file. Studio runs the standard Go formatter while saving valid Go source. If
the file is incomplete, its text is still preserved so the build output can
show what needs fixing.

Studio automatically saves open developer source before Generate, Build, or
Build and Run. Closing a changed file or project offers Save, Discard, and
Cancel.

## Build errors

Choose **File > Build Project** or **Build** in the Build Output page. This
downloads missing modules, builds the generated project, and leaves the result
in Studio without launching the application.

Compiler errors appear below the complete output. Double-click an error to
open its file and move the cursor to the reported line and column.

**Build and Run** performs the same build first, then launches the compiled
application only after the build succeeds.

## Event methods and generated code

The **Event Code** workspace remains the easiest place to write the body of a
designed event. Choose **View Generated Method** there to generate the project,
open `events_generated.go` read-only, and jump to the wrapper method Studio
created for that event.

Put reusable types and helper functions in your own Go files. Studio only
replaces files ending in `_generated.go`; handwritten application code remains
yours.
