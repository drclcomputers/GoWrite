// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

import (
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"

	"github.com/rivo/tview"
)

func (e *Editor) showHeaderFooter() {
	e.showHeaderFooterWithClose(e.closeDialog)
}

func (e *Editor) showHeaderFooterFromFormat() {
	e.showHeaderFooterWithClose(e.showFormatDialog)
}

func (e *Editor) showHeaderFooterWithClose(close func()) {
	ps := e.Doc.GetPageSettings()
	form := tview.NewForm()
	form.SetItemPadding(0)

	form.AddTextView("Tokens", "{page}  {pages}  {title}", 36, 1, false, false)

	for _, f := range []struct {
		label string
		v     *string
	}{
		{"Header left", &ps.Header.Left},
		{"Header center", &ps.Header.Center},
		{"Header right", &ps.Header.Right},
		{"Footer left", &ps.Footer.Left},
		{"Footer center", &ps.Footer.Center},
		{"Footer right", &ps.Footer.Right},
	} {
		v := f.v
		form.AddInputField(f.label, *v, 36, nil, func(text string) { *v = text })
	}

	wheres := []string{"Off", "Header", "Footer"}
	form.AddDropDown("Page numbers", wheres, indexOf(wheres, string(ps.PageNumbers.Where)), func(option string, _ int) {
		ps.PageNumbers.Where = document.NumberWhere(option)
	})
	aligns := []string{"Left", "Center", "Right"}
	form.AddDropDown("Number align", aligns, int(ps.PageNumbers.Align), func(_ string, i int) {
		ps.PageNumbers.Align = document.Align(i)
	})
	formats := []string{"1", "Page 1", "1/N", "1 of N", "Page 1 of N"}
	form.AddDropDown("Number format", formats, indexOf(formats, string(ps.PageNumbers.Format)), func(option string, _ int) {
		ps.PageNumbers.Format = document.NumberFormat(option)
	})

	form.AddTextView("", "", 0, 1, false, false)
	form.AddButton("Apply", func() {
		e.Doc.SetPageSettings(ps)
		shrunk := e.Doc.GetPageSettings().MarginTop != ps.MarginTop ||
			e.Doc.GetPageSettings().MarginBottom != ps.MarginBottom
		close()
		if shrunk {
			e.flashStatus("Margins reduced to fit the header and footer")
		}
	})
	form.AddButton("Cancel", close)
	form.SetButtonsAlign(tview.AlignCenter)

	uitheme.StyleForm(form, "Header & Footer")
	form.SetBorder(true)
	escapeCloses(form, close)

	e.App.SetRoot(centredModal(form, 62, 19), true).SetFocus(form)
}
