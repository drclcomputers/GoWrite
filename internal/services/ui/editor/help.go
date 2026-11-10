// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

import (
	"gowrite/internal/services/uitheme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) showHelp() {
	helpText := uitheme.Markup(`{h1}EVERY DAY{clear}
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

{h2}STYLING{clear}
  Alt + B          Toggle Bold
  Alt + I          Toggle Italic
  Alt + U          Toggle Underline
  Alt + K          Toggle Strikethrough
  Alt + A          Apply active style to selection
  Alt + C          Text color
  Alt + Y          Background color
  Alt + L / E / R  Align left / center / right
  Alt + Shift + P  Insert page break

{h3}FORMAT SCREEN  (Alt+Enter){clear}
  Style / Transform / Colors
  Paragraph        Title, headings, quote, code; alignment; space after (rows)
  Lists            numbered, lettered, bullets, checkboxes, roman
                   (lists auto-continue on Enter; empty item exits)
  View             Formatting marks and heading outline
  Insert           Date & Time · Symbols · Table · Page break
                   Enter on a table edits a cell; at the table end it moves after it
  Page Setup       Page size, margins, header, footer, page numbers

{h3}LESS OFTEN{clear}  (Alt + the letter)
  O open          N new            G go to line       H replace
  Shift + C copy  Shift + X cut    Shift + V paste    Shift + A select all
  Z undo          Shift + Z redo
  F3 / F4         Find next / previous

{muted}Alt+Shift+←/→ selects whole words. Plain Alt+arrow is swallowed by the terminal.{clear}
Everything else is Alt, or a plain key.{clear}

{dim}Esc closes this.{clear}`)

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText).
		SetScrollable(true).
		SetWrap(true)

	textView.SetBorder(true).
		SetTitle(" GoWrite  ·  Esc to close ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(uitheme.Amber).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(uitheme.Ink)

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
