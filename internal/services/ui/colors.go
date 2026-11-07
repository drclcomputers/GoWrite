// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package ui

import (
	"fmt"
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func parseHexColor(s string) tcell.Color {
	if len(s) != 7 || s[0] != '#' {
		return tcell.ColorDefault
	}
	var r, g, b int
	if _, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return tcell.ColorDefault
	}
	return tcell.NewRGBColor(int32(r), int32(g), int32(b))
}

func contrastColor(c tcell.Color) tcell.Color {
	r, g, b := c.RGB()
	lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	if lum > 140 {
		return tcell.ColorBlack
	}
	return tcell.ColorWhite
}

func (e *Editor) showColorPicker(background bool) {
	title := "Text Colour"
	if background {
		title = "Background Colour"
	}

	palette := document.ColorPalette
	const cols = 5

	fg, bg := e.Doc.ActiveColors()
	current := fg
	if background {
		current = bg
	}

	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, true)

	selectedRow, selectedCol := 0, 0
	for i, c := range palette {
		row, col := i/cols, i%cols
		label := c.Name
		if c.Value == current {
			label = "* " + c.Name
			selectedRow, selectedCol = row, col
		}
		cell := tview.NewTableCell(" " + label + " ").
			SetAlign(tview.AlignCenter).
			SetExpansion(1)
		if c.Value == "" {
			cell.SetBackgroundColor(uitheme.Field)
			cell.SetTextColor(uitheme.Text)
		} else {
			rgb := parseHexColor(c.Value)
			cell.SetBackgroundColor(rgb)
			cell.SetTextColor(contrastColor(rgb))
		}
		table.SetCell(row, col, cell)
	}
	table.Select(selectedRow, selectedCol)

	preview := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	preview.SetBackgroundColor(uitheme.Panel)

	updatePreview := func() {
		row, col := table.GetSelection()
		idx := row*cols + col
		if idx < 0 || idx >= len(palette) {
			preview.SetText("")
			return
		}
		c := palette[idx]
		if c.Value == "" {
			preview.SetText(" [gray](terminal default)[-]")
			return
		}
		preview.SetText(fmt.Sprintf(" [#%s]███[::-]  %s  [gray]%s[-]",
			strings.TrimPrefix(c.Value, "#"), c.Name, c.Value))
	}
	updatePreview()
	table.SetSelectionChangedFunc(func(row, col int) { updatePreview() })

	apply := func() {
		row, col := table.GetSelection()
		idx := row*cols + col
		if idx < 0 || idx >= len(palette) {
			return
		}
		c := palette[idx]
		if !e.Doc.SetSelectionColor(background, c.Value) {
			if background {
				e.Doc.SetActiveBg(c.Value)
			} else {
				e.Doc.SetActiveFg(c.Value)
			}
		}
		e.redraw()
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
	}
	table.SetSelectedFunc(func(row, col int) { apply() })
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		return event
	})

	hint := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[gray]arrows move · Enter applies · Esc closes[::-]")
	hint.SetBackgroundColor(uitheme.Panel)

	body := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(table, 0, 1, true).
		AddItem(preview, 1, 0, false).
		AddItem(hint, 1, 0, false)
	uitheme.Paint(body, title)

	e.App.SetRoot(centredModal(body, 62, 12), true).SetFocus(table)
}
