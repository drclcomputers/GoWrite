package ui

import (
	"fmt"
	"gowrite/internal/services/uitheme"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) showColorPicker(background bool) {
	form := tview.NewForm()
	title := "Text color"
	if background {
		title = "Background color"
	}

	var currentColor string
	fg, bg := e.Doc.ActiveColors()
	if background {
		currentColor = bg
	} else {
		currentColor = fg
	}
	if currentColor == "" {
		currentColor = "#000000"
	}

	var r, g, b int64
	if len(currentColor) == 7 && currentColor[0] == '#' {
		r, _ = strconv.ParseInt(currentColor[1:3], 16, 0)
		g, _ = strconv.ParseInt(currentColor[3:5], 16, 0)
		b, _ = strconv.ParseInt(currentColor[5:7], 16, 0)
	}

	preview := tview.NewTextView()
	preview.SetBackgroundColor(tcell.NewRGBColor(int32(r), int32(g), int32(b)))
	preview.SetBorder(true).SetTitle(" Preview ")

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
		SetText(currentColor).
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

	form.AddFormItem(redInput).
		AddFormItem(greenInput).
		AddFormItem(blueInput).
		AddFormItem(hexInput)

	form.AddButton("Apply", func() {
		hex := hexInput.GetText()
		if !e.Doc.SetSelectionColor(background, hex) {
			if background { e.Doc.SetActiveBg(hex) } else { e.Doc.SetActiveFg(hex) }
		}
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
		e.redraw()
	})
	form.AddButton("Cancel", func() {
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
	})

	uitheme.StyleForm(form, title)
	preview.SetBorderColor(uitheme.Line).SetTitleColor(uitheme.Amber).SetTitleAlign(tview.AlignCenter)

	topRow := tview.NewFlex().
		AddItem(form, 0, 3, true).
		AddItem(preview, 0, 1, false)

	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(topRow, 0, 1, true)
	container.SetBackgroundColor(uitheme.Ink)

	container.SetBorder(true).
		SetBorderColor(uitheme.Line).
		SetTitle(" Color ").
		SetTitleColor(uitheme.Amber).
		SetTitleAlign(tview.AlignCenter)

	e.App.SetRoot(container, true).SetFocus(form)
}
