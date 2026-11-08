// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) showHelp() {
	helpText := `[#f0d090::b]EVERY DAY[-]
  Arrows           Move cursor
  Shift + Arrows   Select text
  Type             Insert text
  Alt + Enter      Format (style, lists, view, insert, page setup)
  Alt + P          Page setup (size, margins)
  Alt + D          Header, footer, page numbers
  Alt + S          Save .gdoc / .docx
  Alt + Shift + S  Save as
  Alt + Shift + E  Open save dialog (text format selected)
  Alt + F          Find
  ?                This help
  Esc              Save dialog (location, format, exit)

[#7dcea0::b]STYLING[-]
  Alt + B          Toggle Bold
  Alt + I          Toggle Italic
  Alt + U          Toggle Underline
  Alt + K          Toggle Strikethrough
  Alt + A          Apply active style to selection
  Alt + C          Text color
  Alt + Y          Background color
  Alt + L / E / R  Align left / center / right
  Alt + Shift + P  Insert page break

[#8ecae6::b]FORMAT SCREEN  (Alt+Enter)[-]
  Style / Transform / Colors
  Paragraph        Title, headings, quote, code; alignment; space after (rows)
  Lists            numbered, lettered, bullets, checkboxes, roman
                   (lists auto-continue on Enter; empty item exits)
  View             Formatting marks and heading outline
  Insert           Date & Time · Symbols · Table · Page break
  Page Setup       Page size, margins, header, footer, page numbers

[#8ecae6::b]LESS OFTEN[-]  (Alt + the letter)
  O open          N new            G go to line       H replace
  Shift + C copy  Shift + X cut    Shift + V paste    Shift + A select all
  Z undo          Shift + Z redo
  F3 / F4         Find next / previous

[#8aa0b8]Alt+Shift+←/→ selects whole words. Plain Alt+arrow is swallowed by the terminal.[-]
Everything else is Alt, or a plain key.[-]

[#5c6b7a]Esc closes this.[-]`

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText).
		SetScrollable(true).
		SetWrap(true)

	textView.SetBorder(true).
		SetTitle(" GoWrite  ·  Esc to close ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(tcell.NewRGBColor(240, 208, 144)).
		SetBorderColor(tcell.NewRGBColor(70, 160, 170)).
		SetBackgroundColor(tcell.NewRGBColor(14, 17, 26))

	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		return event
	})

	e.App.SetRoot(textView, true).SetFocus(textView)
}
