// SPDX-License-Identifier: LGPL-3.0-or-later

package main

import (
	"fmt"
	"math"
	"strings"

	rosaline "github.com/SeraphinaDX/Rosaline"
)

const (
	previewWidth  = 700
	previewHeight = 540
)

type previewRect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

func (rectangle previewRect) inset(amount float64) previewRect {
	amount = min(amount, min(rectangle.Width/2, rectangle.Height/2))
	return previewRect{
		X:      rectangle.X + amount,
		Y:      rectangle.Y + amount,
		Width:  max(1, rectangle.Width-amount*2),
		Height: max(1, rectangle.Height-amount*2),
	}
}

func (rectangle previewRect) contains(x, y float64) bool {
	return x >= rectangle.X && x <= rectangle.X+rectangle.Width && y >= rectangle.Y && y <= rectangle.Y+rectangle.Height
}

type previewBox struct {
	Node  *designNode
	Rect  previewRect
	Depth int
}

func layoutPreview(project *designProject) []previewBox {
	if project == nil || project.Root == nil {
		return nil
	}
	bounds := previewRect{X: 24, Y: 54, Width: previewWidth - 48, Height: previewHeight - 78}
	boxes := make([]previewBox, 0)
	layoutPreviewNode(project.Root, bounds, 0, &boxes)
	return boxes
}

func layoutPreviewNode(node *designNode, bounds previewRect, depth int, boxes *[]previewBox) {
	if node == nil {
		return
	}
	bounds = applyPreviewSize(bounds, node)
	*boxes = append(*boxes, previewBox{Node: node, Rect: bounds, Depth: depth})
	if len(node.Children) == 0 {
		return
	}

	padding := float64(max(0, node.Padding))
	inside := bounds.inset(padding + 5)
	gap := float64(max(0, node.Gap))
	switch node.Kind {
	case kindRow:
		layoutPreviewRow(node.Children, inside, gap, depth+1, boxes)
	case kindGrid:
		layoutPreviewGrid(node, inside, gap, depth+1, boxes)
	case kindStack:
		for _, child := range node.Children {
			layoutPreviewNode(child, inside.inset(float64(depth+1)*2), depth+1, boxes)
		}
	case kindCard, kindScroll:
		layoutPreviewNode(node.Children[0], inside.inset(3), depth+1, boxes)
	default:
		layoutPreviewColumn(node.Children, inside, gap, depth+1, boxes)
	}
}

func layoutPreviewColumn(children []*designNode, bounds previewRect, gap float64, depth int, boxes *[]previewBox) {
	if len(children) == 0 {
		return
	}
	available := max(1, bounds.Height-gap*float64(len(children)-1))
	total := 0.0
	for _, child := range children {
		total += previewWeight(child, false)
	}
	y := bounds.Y
	for _, child := range children {
		height := available * previewWeight(child, false) / total
		layoutPreviewNode(child, previewRect{X: bounds.X, Y: y, Width: bounds.Width, Height: height}, depth, boxes)
		y += height + gap
	}
}

func layoutPreviewRow(children []*designNode, bounds previewRect, gap float64, depth int, boxes *[]previewBox) {
	if len(children) == 0 {
		return
	}
	available := max(1, bounds.Width-gap*float64(len(children)-1))
	total := 0.0
	for _, child := range children {
		total += previewWeight(child, true)
	}
	x := bounds.X
	for _, child := range children {
		width := available * previewWeight(child, true) / total
		layoutPreviewNode(child, previewRect{X: x, Y: bounds.Y, Width: width, Height: bounds.Height}, depth, boxes)
		x += width + gap
	}
}

func layoutPreviewGrid(node *designNode, bounds previewRect, gap float64, depth int, boxes *[]previewBox) {
	columns := max(1, node.Columns)
	rows := (len(node.Children) + columns - 1) / columns
	cellWidth := max(1, (bounds.Width-gap*float64(columns-1))/float64(columns))
	cellHeight := max(1, (bounds.Height-gap*float64(rows-1))/float64(rows))
	for index, child := range node.Children {
		column := index % columns
		row := index / columns
		rectangle := previewRect{
			X:      bounds.X + float64(column)*(cellWidth+gap),
			Y:      bounds.Y + float64(row)*(cellHeight+gap),
			Width:  cellWidth,
			Height: cellHeight,
		}
		layoutPreviewNode(child, rectangle, depth, boxes)
	}
}

func previewWeight(node *designNode, horizontal bool) float64 {
	if node == nil {
		return 1
	}
	if horizontal && node.Width > 0 {
		return max(0.4, float64(node.Width)/120)
	}
	if !horizontal && node.Height > 0 {
		return max(0.4, float64(node.Height)/48)
	}
	switch node.Kind {
	case kindTextArea, kindScroll, kindGrid, kindStack:
		return 2.5
	case kindColumn, kindRow, kindCard:
		return 1.8
	case kindSpacer:
		return 0.55
	default:
		return 1
	}
}

func applyPreviewSize(bounds previewRect, node *designNode) previewRect {
	width, height := bounds.Width, bounds.Height
	if node.Width > 0 && node.Kind != kindScroll {
		width = min(width, float64(node.Width))
	}
	if node.Height > 0 && node.Kind != kindScroll {
		height = min(height, float64(node.Height))
	}
	return previewRect{
		X:      bounds.X + (bounds.Width-width)/2,
		Y:      bounds.Y + (bounds.Height-height)/2,
		Width:  max(1, width),
		Height: max(1, height),
	}
}

func previewBoxAt(boxes []previewBox, x, y float64) *designNode {
	for index := len(boxes) - 1; index >= 0; index-- {
		if boxes[index].Rect.contains(x, y) {
			return boxes[index].Node
		}
	}
	return nil
}

type previewPalette struct {
	background rosaline.Color
	surface    rosaline.Color
	primary    rosaline.Color
	text       rosaline.Color
	muted      rosaline.Color
	border     rosaline.Color
}

func paletteFor(theme string) previewPalette {
	switch theme {
	case "Midnight":
		return previewPalette{
			background: rosaline.Hex("#140d1d"), surface: rosaline.Hex("#24152e"),
			primary: rosaline.Hex("#ff79b7"), text: rosaline.Hex("#fff0fa"),
			muted: rosaline.Hex("#c39ab4"), border: rosaline.Hex("#73445f"),
		}
	case "Lavender":
		return previewPalette{
			background: rosaline.Hex("#f7f1ff"), surface: rosaline.White,
			primary: rosaline.Hex("#8554bd"), text: rosaline.Hex("#2d2038"),
			muted: rosaline.Hex("#74627f"), border: rosaline.Hex("#d7c3e9"),
		}
	default:
		theme := rosaline.DefaultTheme
		return previewPalette{theme.Background, theme.Surface, theme.Primary, theme.Text, theme.Muted, theme.Border}
	}
}

func drawPreview(canvas *rosaline.DrawingCanvas, project *designProject, boxes []previewBox, selectedID string) {
	outer := rosaline.Hex("#ead7e3")
	canvas.Clear(outer)
	if project == nil {
		return
	}
	colors := paletteFor(project.Theme)
	canvas.FillRect(18, 18, previewWidth-36, previewHeight-30, colors.surface)
	canvas.Rect(18, 18, previewWidth-36, previewHeight-30, 2, rosaline.Hex("#b78aa5"))
	canvas.FillRect(18, 18, previewWidth-36, 30, colors.primary)
	canvas.Text(project.Title, 30, 25, rosaline.TextStyle{Color: rosaline.White, Size: 13})
	canvas.FillRect(24, 54, previewWidth-48, previewHeight-78, colors.background)

	for _, box := range boxes {
		drawPreviewNode(canvas, box, colors, box.Node.ID == selectedID)
	}
	canvas.Text("Click to select - drag onto another widget to move", 25, previewHeight-19, rosaline.TextStyle{Color: rosaline.Hex("#74586a"), Size: 11})
}

func drawPreviewNode(canvas *rosaline.DrawingCanvas, box previewBox, colors previewPalette, selected bool) {
	node := box.Node
	rectangle := box.Rect
	if node == nil {
		return
	}
	selection := rosaline.Hex("#ff4fa3")
	outline := colors.border
	if selected {
		outline = selection
	}

	switch node.Kind {
	case kindColumn, kindRow, kindGrid, kindStack:
		canvas.Rect(rectangle.X, rectangle.Y, rectangle.Width, rectangle.Height, strokeFor(selected), outline)
		canvas.Text(string(node.Kind), rectangle.X+4, rectangle.Y+3, rosaline.TextStyle{Color: colors.muted, Size: 9})
	case kindCard:
		canvas.FillRect(rectangle.X, rectangle.Y, rectangle.Width, rectangle.Height, colors.surface)
		canvas.Rect(rectangle.X, rectangle.Y, rectangle.Width, rectangle.Height, strokeFor(selected), outline)
	case kindScroll:
		canvas.FillRect(rectangle.X, rectangle.Y, rectangle.Width, rectangle.Height, colors.surface)
		canvas.Rect(rectangle.X, rectangle.Y, rectangle.Width, rectangle.Height, strokeFor(selected), outline)
		canvas.FillRect(rectangle.X+rectangle.Width-8, rectangle.Y+4, 4, rectangle.Height-8, colors.border)
	case kindLabel:
		canvas.Text(defaultText(node.Text, "Label"), rectangle.X+5, rectangle.Y+max(4, rectangle.Height/2-7), rosaline.TextStyle{Color: colors.text, Size: fontSizeFor(rectangle)})
		if selected {
			canvas.Rect(rectangle.X, rectangle.Y, rectangle.Width, rectangle.Height, 2, selection)
		}
	case kindButton:
		fill := colors.surface
		text := colors.text
		if node.Primary {
			fill, text = colors.primary, rosaline.White
		}
		canvas.FillRect(rectangle.X+2, rectangle.Y+3, rectangle.Width-4, rectangle.Height-6, fill)
		canvas.Rect(rectangle.X+2, rectangle.Y+3, rectangle.Width-4, rectangle.Height-6, strokeFor(selected), outline)
		canvas.Text(defaultText(node.Text, "Button"), rectangle.X+10, rectangle.Y+max(7, rectangle.Height/2-7), rosaline.TextStyle{Color: text, Size: fontSizeFor(rectangle)})
	case kindTextBox:
		drawInput(canvas, rectangle, defaultText(node.Text, "Text input"), colors, outline, selected)
	case kindTextArea:
		drawInput(canvas, rectangle, "Multiline text: "+defaultText(node.Name, "Notes"), colors, outline, selected)
		for y := rectangle.Y + 31; y < rectangle.Y+rectangle.Height-6; y += 15 {
			canvas.Line(rectangle.X+8, y, rectangle.X+rectangle.Width-12, y, 1, colors.border)
		}
	case kindCheckBox:
		boxSize := min(18.0, rectangle.Height-6)
		canvas.FillRect(rectangle.X+4, rectangle.Y+(rectangle.Height-boxSize)/2, boxSize, boxSize, colors.surface)
		canvas.Rect(rectangle.X+4, rectangle.Y+(rectangle.Height-boxSize)/2, boxSize, boxSize, strokeFor(selected), outline)
		canvas.Text(defaultText(node.Text, "Check box"), rectangle.X+boxSize+10, rectangle.Y+max(4, rectangle.Height/2-7), rosaline.TextStyle{Color: colors.text, Size: fontSizeFor(rectangle)})
	case kindComboBox:
		label := "Choose an option"
		if len(node.Options) != 0 {
			label = node.Options[0]
		}
		drawInput(canvas, rectangle, label+"  v", colors, outline, selected)
	case kindSlider:
		centerY := rectangle.Y + rectangle.Height/2
		canvas.Line(rectangle.X+10, centerY, rectangle.X+rectangle.Width-10, centerY, 3, colors.border)
		canvas.FillCircle(rectangle.X+rectangle.Width*0.45, centerY, 7, colors.primary)
		if selected {
			canvas.Rect(rectangle.X, rectangle.Y, rectangle.Width, rectangle.Height, 2, selection)
		}
	case kindProgressBar:
		canvas.FillRect(rectangle.X+4, rectangle.Y+rectangle.Height/2-7, rectangle.Width-8, 14, colors.border)
		canvas.FillRect(rectangle.X+4, rectangle.Y+rectangle.Height/2-7, (rectangle.Width-8)*0.55, 14, colors.primary)
		if selected {
			canvas.Rect(rectangle.X, rectangle.Y, rectangle.Width, rectangle.Height, 2, selection)
		}
	case kindSpacer:
		canvas.Line(rectangle.X+3, rectangle.Y+rectangle.Height/2, rectangle.X+rectangle.Width-3, rectangle.Y+rectangle.Height/2, 1, colors.muted)
		canvas.Text("Spacer", rectangle.X+5, rectangle.Y+3, rosaline.TextStyle{Color: colors.muted, Size: 8})
		if selected {
			canvas.Rect(rectangle.X, rectangle.Y, rectangle.Width, rectangle.Height, 2, selection)
		}
	}
}

func drawInput(canvas *rosaline.DrawingCanvas, rectangle previewRect, label string, colors previewPalette, outline rosaline.Color, selected bool) {
	canvas.FillRect(rectangle.X+2, rectangle.Y+3, rectangle.Width-4, rectangle.Height-6, colors.surface)
	canvas.Rect(rectangle.X+2, rectangle.Y+3, rectangle.Width-4, rectangle.Height-6, strokeFor(selected), outline)
	canvas.Text(label, rectangle.X+10, rectangle.Y+max(7, rectangle.Height/2-7), rosaline.TextStyle{Color: colors.muted, Size: fontSizeFor(rectangle)})
}

func strokeFor(selected bool) float64 {
	if selected {
		return 2.5
	}
	return 1
}

func fontSizeFor(rectangle previewRect) int {
	return int(max(9, min(14, math.Min(rectangle.Height/2.7, rectangle.Width/16))))
}

func defaultText(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	if len([]rune(value)) > 54 {
		return string([]rune(value)[:51]) + "..."
	}
	return value
}

func previewDescription(node *designNode) string {
	if node == nil {
		return "No widget selected"
	}
	name := ""
	if node.Name != "" {
		name = " - " + node.Name
	}
	return fmt.Sprintf("%s%s (%s)", node.Kind, name, node.ID)
}
