// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

import (
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) closeDialog() {
	e.App.SetRoot(e.Root, true).SetFocus(e.View)
	e.redraw()
}

func escapeCloses(form *tview.Form, close func()) {
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			close()
			return nil
		}
		return event
	})
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return 0
}

func (e *Editor) showPageSetup() {
	e.showPageSetupWithClose(e.closeDialog)
}

func (e *Editor) showPageSetupFromFormat() {
	e.showPageSetupWithClose(e.showFormatDialog)
}

type pageSetupButton struct {
	*tview.Button
}

func (b *pageSetupButton) GetLabel() string { return "" }

func (b *pageSetupButton) Draw(screen tcell.Screen) {
	x, y, width, height := b.GetRect()
	buttonWidth := tview.TaggedStringWidth(b.Button.GetLabel()) + 4
	if buttonWidth > width {
		buttonWidth = width
	}
	b.Button.SetRect(x, y, buttonWidth, height)
	b.Button.Draw(screen)
}

func (b *pageSetupButton) GetFieldWidth() int { return 0 }

func (b *pageSetupButton) GetFieldHeight() int { return 1 }

func (b *pageSetupButton) SetFormAttributes(int, tcell.Color, tcell.Color, tcell.Color, tcell.Color) tview.FormItem {
	return b
}

func (b *pageSetupButton) SetFinishedFunc(handler func(tcell.Key)) tview.FormItem {
	b.SetExitFunc(handler)
	return b
}

func (b *pageSetupButton) SetDisabled(disabled bool) tview.FormItem {
	b.Button.SetDisabled(disabled)
	return b
}

func (b *pageSetupButton) GetDisabled() bool {
	return b.Button.GetDisabled()
}

func (e *Editor) showPageSetupWithClose(close func()) {
	ps := e.Doc.GetPageSettings()
	form := tview.NewForm()
	form.SetItemPadding(0)

	sizes := []string{"A5", "A4", "A3", "Letter", "Legal"}
	form.AddDropDown("Size", sizes, indexOf(sizes, string(ps.Size)), func(option string, _ int) {
		ps.Size = document.PageSize(option)
	})
	orients := []string{"Portrait", "Landscape"}
	form.AddDropDown("Orientation", orients, indexOf(orients, string(ps.Orientation)), func(option string, _ int) {
		ps.Orientation = document.Orientation(option)
	})

	for _, m := range []struct {
		label string
		v     *int
	}{
		{"Margin left", &ps.MarginLeft},
		{"Margin right", &ps.MarginRight},
		{"Margin top", &ps.MarginTop},
		{"Margin bottom", &ps.MarginBottom},
	} {
		v := m.v
		form.AddInputField(m.label, strconv.Itoa(*v), 5, tview.InputFieldInteger, func(text string) {
			n, _ := strconv.Atoi(text)
			*v = n
		})
	}

	var pageModal tview.Primitive
	var backgroundButton *tview.Button
	form.AddTextView("", "", 0, 1, false, false)
	backgroundButton = uitheme.StyleButton(tview.NewButton("Page background: " + ps.Background).SetSelectedFunc(func() {
		e.showColorPickerWith(true, ps.Background, func(color string) {
			if color == "" {
				color = uitheme.HexPageBackground
			}
			ps.Background = color
			backgroundButton.SetLabel("Page background: " + color)
			e.App.SetRoot(pageModal, true).SetFocus(backgroundButton)
		}, func() {
			e.App.SetRoot(pageModal, true).SetFocus(backgroundButton)
		})
	}))
	form.AddFormItem(&pageSetupButton{Button: backgroundButton})

	form.AddTextView("", "", 0, 1, false, false)

	// apply stores the form's settings; it reports whether margins had to shrink.
	apply := func() bool {
		ps.Width, ps.Height = document.GetDimensions(ps.Size, ps.Orientation)
		e.Doc.SetPageSettings(ps)
		got := e.Doc.GetPageSettings()
		return got.MarginLeft != ps.MarginLeft || got.MarginRight != ps.MarginRight ||
			got.MarginTop != ps.MarginTop || got.MarginBottom != ps.MarginBottom
	}

	form.AddButton("Apply", func() {
		shrunk := apply()
		close()
		if shrunk {
			e.flashStatus("Margins reduced to fit the page")
		}
	})
	form.AddButton("Cancel", close)
	form.SetButtonsAlign(tview.AlignCenter)

	uitheme.StyleForm(form, "Page Setup")
	form.SetBorder(true)
	escapeCloses(form, close)

	pageModal = centredModal(form, 52, 15)
	e.App.SetRoot(pageModal, true).SetFocus(form)
}
