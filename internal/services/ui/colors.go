package ui

import (
	"fmt"
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) showColorPicker(background bool) {
	title := "Text Colour"
	if background {
		title = "Background Colour"
	}

	pages := tview.NewPages()
	palette := document.ColorPalette
	const cols = 5

	fg, bg := e.Doc.ActiveColors()
	current := fg
	if background {
		current = bg
	}

	// --- PAGE 1: GRID PALETTE ---
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

	customRow, customCol := len(palette)/cols, len(palette)%cols
	customCell := tview.NewTableCell(" [ Custom... ] ").
		SetAlign(tview.AlignCenter).
		SetExpansion(1).
		SetBackgroundColor(uitheme.Line).
		SetTextColor(uitheme.Ink)
	table.SetCell(customRow, customCol, customCell)

	table.Select(selectedRow, selectedCol)

	gridPage := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(table, 0, 1, true)
	gridPage.SetBackgroundColor(uitheme.Ink)
	pages.AddPage("grid", gridPage, true, true)

	// --- PAGE 2: CUSTOM RGB/HEX FORM ---
	var r, g, b int64
	startHex := current
	if len(startHex) != 7 || startHex[0] != '#' {
		startHex = "#000000"
	}
	r, _ = strconv.ParseInt(startHex[1:3], 16, 0)
	g, _ = strconv.ParseInt(startHex[3:5], 16, 0)
	b, _ = strconv.ParseInt(startHex[5:7], 16, 0)

	preview := tview.NewTextView()
	preview.SetBackgroundColor(tcell.NewRGBColor(int32(r), int32(g), int32(b)))
	preview.SetBorder(true).SetTitle(" Preview ").SetTitleAlign(tview.AlignCenter)
	preview.SetBorderColor(uitheme.Line).SetTitleColor(uitheme.Amber)

	customForm := tview.NewForm()
	updating := false

	redInput := tview.NewInputField().
		SetLabel("Red (0-255): ").
		SetText(fmt.Sprintf("%d", r)).
		SetFieldWidth(5).
		SetAcceptanceFunc(func(text string, lastChar rune) bool {
			if text == "" { return true }
			if val, err := strconv.Atoi(text); err == nil { return val >= 0 && val <= 255 }
			return false
		})

	greenInput := tview.NewInputField().
		SetLabel("Green (0-255): ").
		SetText(fmt.Sprintf("%d", g)).
		SetFieldWidth(5).
		SetAcceptanceFunc(func(text string, lastChar rune) bool {
			if text == "" { return true }
			if val, err := strconv.Atoi(text); err == nil { return val >= 0 && val <= 255 }
			return false
		})

	blueInput := tview.NewInputField().
		SetLabel("Blue (0-255): ").
		SetText(fmt.Sprintf("%d", b)).
		SetFieldWidth(5).
		SetAcceptanceFunc(func(text string, lastChar rune) bool {
			if text == "" { return true }
			if val, err := strconv.Atoi(text); err == nil { return val >= 0 && val <= 255 }
			return false
		})

	hexInput := tview.NewInputField().
		SetLabel("Hex (#RRGGBB): ").
		SetText(startHex).
		SetFieldWidth(9)

	updatePreview := func() {
		if updating { return }
		updating = true

		rText := redInput.GetText()
		gText := greenInput.GetText()
		bText := blueInput.GetText()
		if rText == "" { rText = "0" }
		if gText == "" { gText = "0" }
		if bText == "" { bText = "0" }

		rV, _ := strconv.Atoi(rText)
		gV, _ := strconv.Atoi(gText)
		bV, _ := strconv.Atoi(bText)

		color := tcell.NewRGBColor(int32(rV), int32(gV), int32(bV))
		preview.SetBackgroundColor(color)
		hex := fmt.Sprintf("#%02x%02x%02x", rV, gV, bV)
		hexInput.SetText(hex)

		updating = false
	}

	updateFromHex := func() {
		if updating { return }
		updating = true
		hex := hexInput.GetText()
		if len(hex) == 7 && hex[0] == '#' {
			rV, err1 := strconv.ParseInt(hex[1:3], 16, 0)
			gV, err2 := strconv.ParseInt(hex[3:5], 16, 0)
			bV, err3 := strconv.ParseInt(hex[5:7], 16, 0)
			if err1 == nil && err2 == nil && err3 == nil {
				redInput.SetText(fmt.Sprintf("%d", rV))
				greenInput.SetText(fmt.Sprintf("%d", gV))
				blueInput.SetText(fmt.Sprintf("%d", bV))
				preview.SetBackgroundColor(tcell.NewRGBColor(int32(rV), int32(gV), int32(bV)))
			}
		}
		updating = false
	}

	redInput.SetDoneFunc(func(key tcell.Key) { if key == tcell.KeyEnter || key == tcell.KeyTab { updatePreview() } })
	greenInput.SetDoneFunc(func(key tcell.Key) { if key == tcell.KeyEnter || key == tcell.KeyTab { updatePreview() } })
	blueInput.SetDoneFunc(func(key tcell.Key) { if key == tcell.KeyEnter || key == tcell.KeyTab { updatePreview() } })
	hexInput.SetDoneFunc(func(key tcell.Key) { if key == tcell.KeyEnter || key == tcell.KeyTab { updateFromHex() } })

	customForm.AddFormItem(redInput).
		AddFormItem(greenInput).
		AddFormItem(blueInput).
		AddFormItem(hexInput)

	applyCustom := func() {
		hex := hexInput.GetText()
		if !e.Doc.SetSelectionColor(background, hex) {
			if background { e.Doc.SetActiveBg(hex) } else { e.Doc.SetActiveFg(hex) }
		}
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
		e.redraw()
	}

	customForm.AddButton("Apply", applyCustom)
	customForm.AddButton("Back", func() { pages.SwitchToPage("grid"); e.App.SetFocus(table) })

	customForm.SetBorder(false)
	customForm.SetBackgroundColor(uitheme.Panel)

	topRow := tview.NewFlex().
		AddItem(customForm, 0, 3, true).
		AddItem(preview, 0, 1, false)

	customPage := tview.NewFlex().SetDirection(tview.FlexRow).AddItem(topRow, 0, 1, true)
	customPage.SetBackgroundColor(uitheme.Ink)
	pages.AddPage("custom", customPage, true, false)

	// --- ACTIONS ---
	applyGrid := func() {
		row, col := table.GetSelection()
		if row == customRow && col == customCol {
			pages.SwitchToPage("custom")
			e.App.SetFocus(customForm)
			return
		}
		idx := row*cols + col
		if idx >= 0 && idx < len(palette) {
			c := palette[idx]
			if !e.Doc.SetSelectionColor(background, c.Value) {
				if background { e.Doc.SetActiveBg(c.Value) } else { e.Doc.SetActiveFg(c.Value) }
			}
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
		}
	}
	table.SetSelectedFunc(func(row, col int) { applyGrid() })
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		return event
	})

	customForm.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			pages.SwitchToPage("grid")
			e.App.SetFocus(table)
			return nil
		}
		return event
	})

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pages, 0, 1, true)
	container.SetBackgroundColor(uitheme.Ink)
	container.SetBorder(true).
		SetBorderColor(uitheme.Line).
		SetTitle(" " + title + " ").
		SetTitleColor(uitheme.Amber).
		SetTitleAlign(tview.AlignCenter)

	modal := centredModal(container, 62, 12)
	e.App.SetRoot(modal, true).SetFocus(table)
}

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
