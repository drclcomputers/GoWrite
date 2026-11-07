package ui

import (
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) showFormatDialog() {
	form := tview.NewForm()

	// Alignment
	alignIndex := 0
	switch e.Doc.CurrentAlign() {
	case document.AlignCenter:
		alignIndex = 1
	case document.AlignRight:
		alignIndex = 2
	}
	form.AddDropDown("Align", []string{"Left", "Center", "Right"}, alignIndex, func(option string, _ int) {
		switch option {
		case "Left":
			e.Doc.SetParagraphAlign(document.AlignLeft)
		case "Center":
			e.Doc.SetParagraphAlign(document.AlignCenter)
		case "Right":
			e.Doc.SetParagraphAlign(document.AlignRight)
		}
	})

	// Styles
	form.AddCheckbox("Bold", e.Doc.ActiveStyleFlags()&document.StyleBold != 0, func(checked bool) {
		if checked {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() | document.StyleBold)
		} else {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() &^ document.StyleBold)
		}
	})
	form.AddCheckbox("Italic", e.Doc.ActiveStyleFlags()&document.StyleItalic != 0, func(checked bool) {
		if checked {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() | document.StyleItalic)
		} else {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() &^ document.StyleItalic)
		}
	})
	form.AddCheckbox("Underline", e.Doc.ActiveStyleFlags()&document.StyleUnderline != 0, func(checked bool) {
		if checked {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() | document.StyleUnderline)
		} else {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() &^ document.StyleUnderline)
		}
	})
	form.AddCheckbox("Strikethrough", e.Doc.ActiveStyleFlags()&document.StyleStrike != 0, func(checked bool) {
		if checked {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() | document.StyleStrike)
		} else {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() &^ document.StyleStrike)
		}
	})

	form.AddButton("Text Color", func() { e.showColorPicker(false) })
	form.AddButton("Background Color", func() { e.showColorPicker(true) })

	// Actions
	form.AddButton("Page Break", func() {
		e.Doc.InsertPageBreak()
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
		e.redraw()
	})
	form.AddButton("Close", func() {
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
		e.redraw()
	})

	uitheme.StyleForm(form, "Format Text")
	form.SetBorder(true)

	modal := centredModal(form, 42, 18)

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
			return nil
		}
		return event
	})

	e.App.SetRoot(modal, true).SetFocus(form)
}
