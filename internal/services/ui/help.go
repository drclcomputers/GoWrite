package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) showHelp() {
	helpText := `[#f0d090::b]EVERY DAY[-]
  Arrows          Move
  Shift+Arrows    Select text
  Type            Insert text
  Alt+Enter       Format text
  Alt+S           Save
  Alt+F           Find
  ?               This help
  Esc             Quit (or clear selection)

[#7dcea0::b]STYLING[-]
  Alt+B           Toggle Bold
  Alt+I           Toggle Italic
  Alt+U           Toggle Underline
  Alt+K           Toggle Strikethrough
  Alt+A           Apply active style to paragraph

[#8ecae6::b]LESS OFTEN[-]  (Alt + the letter)
  O open     N new     G go to line     H replace
  C copy     X cut     V paste          Z undo
  Shift+Z redo

[#8aa0b8]Alt+Shift+←/→ moves between panels. Plain Alt+arrow is swallowed by the terminal.[-]
Everything else is Alt, or a plain key. F3 / F4 repeat a search.[-]

[#5c6b7a]Esc closes this.[-]`

	textView := tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText).
		SetScrollable(true).
		SetWrap(true)

	textView.SetBorder(true).
		SetTitle(" GoWrite  ·  ? or Esc closes ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(tcell.NewRGBColor(240, 208, 144)).
		SetBorderColor(tcell.NewRGBColor(70, 160, 170)).
		SetBackgroundColor(tcell.NewRGBColor(14, 17, 26))

	textView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		case tcell.KeyEnter:
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		return event
	})

	e.App.SetRoot(textView, true).SetFocus(textView)
}
