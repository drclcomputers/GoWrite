// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package ui

import (
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) confirmQuit() {
	msg := "Quit GoWrite?"
	if e.Doc.IsDirty() {
		msg = "You have unsaved changes.\nQuit anyway?"
	}
	modal := tview.NewModal().
		SetText(msg).
		AddButtons([]string{"Quit", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Quit" {
				e.App.Stop()
			} else {
				e.App.SetRoot(e.Root, true).SetFocus(e.View)
			}
		})
	uitheme.StyleModalWarn(modal, "Confirm")
	e.App.SetRoot(modal, true).SetFocus(modal)
}

func (e *Editor) newDocument() {
	if e.Doc.IsDirty() {
		modal := tview.NewModal().
			SetText("Current document has unsaved changes.\nDiscard and create a new one?").
			AddButtons([]string{"Discard", "Cancel"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				if buttonLabel == "Discard" {
					e.Doc = document.New()
					e.redraw()
				}
				e.App.SetRoot(e.Root, true).SetFocus(e.View)
			})
		uitheme.StyleModalWarn(modal, "New Document")
		e.App.SetRoot(modal, true).SetFocus(modal)
		return
	}
	e.Doc = document.New()
	e.redraw()
}

func (e *Editor) showHelp() {
	helpText := `[#f0d090::b]GoWrite – Keyboard Shortcuts[-]

[#7dcea0::b]NAVIGATION[-]
  Arrow keys            Move (↑/↓ follow wrapped lines)
  Home / End            Start / end of the line
  Alt+Home / Alt+End    Start / end of the document
  PgUp / PgDn           Move one screen up / down
  Alt+Shift+← / →       Word left / right
  Alt+G                 Go to line (paragraph)

[#8ecae6::b]SELECTION & CLIPBOARD[-]
  Shift+arrows/Home/End/PgUp/PgDn   Select
  Alt+Shift+← / →       Select by word
  Alt+Shift+A           Select all
  Esc                   Clear the selection
  Alt+Shift+C           Copy
  Alt+Shift+X / Shift+Delete        Cut
  Alt+Shift+V / Shift+Insert        Paste
  Typing, Backspace and Delete replace the selection.
  Styles, colours and alignment apply to the selection.

[#f0d090::b]UNDO / REDO[-]
  Alt+Z                 Undo
  Alt+Shift+Z           Redo

[#f0d090::b]SEARCH[-]
  Alt+F                 Find
  Alt+H                 Find & Replace
  F3 / F4               Find next / previous

[#8ecae6::b]STYLING[-]
  Alt+B                 Toggle Bold
  Alt+I                 Toggle Italic
  Alt+U                 Toggle Underline
  Alt+K                 Toggle Strikethrough
  Alt+C                 Text colour (grid palette)
  Alt+Y                 Background colour (grid palette)
  Alt+L / E / R         Align left / center / right
  Alt+A                 Apply active style to selection/paragraph

[#f0d090::b]FILE[-]
  Alt+N                 New document
  Alt+O                 Open document
  Alt+S                 Save document

[#8aa0b8]Word wrap is always on and follows the window size.
Alignment applies to every wrapped line of a paragraph.
Enter starts a new paragraph.
New text uses the active style + colours (status bar).
Ctrl+C is reserved for the standard terminal exit prompt.[-]

[#5c6b7a]Esc closes this.[-]`

	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)
	tv.SetBorder(true).
		SetTitle(" GoWrite · ? or Esc closes ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(tcell.NewRGBColor(240, 208, 144)).
		SetBorderColor(tcell.NewRGBColor(70, 160, 170)).
		SetBackgroundColor(tcell.NewRGBColor(14, 17, 26))
	tv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		return event
	})

	e.App.SetRoot(tv, true).SetFocus(tv)
}

// centredModal wraps a primitive in a hollow Flex so it appears as a centred
// box of the requested width and height.
func centredModal(p tview.Primitive, width, height int) tview.Primitive {
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(p, width, 0, true).
			AddItem(nil, 0, 1, false), height, 0, true).
		AddItem(nil, 0, 1, false)
}
