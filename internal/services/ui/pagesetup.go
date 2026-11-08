// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package ui

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

	uitheme.StyleForm(form, "Page Setup")
	form.SetBorder(true)
	escapeCloses(form, close)

	e.App.SetRoot(centredModal(form, 52, 15), true).SetFocus(form)
}
