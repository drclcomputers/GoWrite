// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package ui

import (
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) showPageSetup() {
	form := tview.NewForm()

	ps := e.Doc.GetPageSettings()

	// Size
	sizes := []string{"A5", "A4", "A3", "Letter", "Legal"}
	sizeIndex := 1
	for i, s := range sizes {
		if string(ps.Size) == s {
			sizeIndex = i
			break
		}
	}
	form.AddDropDown("Size", sizes, sizeIndex, func(option string, _ int) {
		ps.Size = document.PageSize(option)
		ps.Width, ps.Height = document.GetDimensions(ps.Size, ps.Orientation)
		e.Doc.SetPageSettings(ps)
	})

	// Orientation
	orients := []string{"Portrait", "Landscape"}
	orientIndex := 0
	if ps.Orientation == document.Landscape {
		orientIndex = 1
	}
	form.AddDropDown("Orientation", orients, orientIndex, func(option string, _ int) {
		ps.Orientation = document.Orientation(option)
		ps.Width, ps.Height = document.GetDimensions(ps.Size, ps.Orientation)
		e.Doc.SetPageSettings(ps)
	})

	form.AddButton("Apply", func() {
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
		e.redraw()
	})

	uitheme.StyleForm(form, "Page Setup")
	form.SetBorder(true)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
			return nil
		}
		return event
	})

	modal := centredModal(form, 40, 10)
	e.App.SetRoot(modal, true).SetFocus(form)
}
