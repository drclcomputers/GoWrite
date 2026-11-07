// Copyright (c) 2025 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

// Package ui contains the main editor view and related dialogs.
package ui

import (
	"fmt"
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"
	"gowrite/internal/utils"
	"strconv"
	"strings"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Editor is the main writing surface.
// Uses a TextView (not TextArea) so tview style tags actually render:
// bold / italic / underline / strikethrough show up when the terminal supports them.
type Editor struct {
	App    *tview.Application
	Doc    *document.Document
	Root   *tview.Flex
	View   *tview.TextView
	Status *tview.TextView
	TitleBar *tview.TextView

	// Size of the document view the text was last laid out for. When the
	// terminal is resized we re-wrap (see NewEditor).
	lastW, lastH int
}

// NewEditor builds the full-screen editor layout.
func NewEditor(app *tview.Application, doc *document.Document) *Editor {
	e := &Editor{
		App: app,
		Doc: doc,
	}

	// Title bar
	e.TitleBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	e.TitleBar.SetBackgroundColor(uitheme.Ink)
	e.refreshTitle()

	// Document view – TextView with dynamic colours so [::b]/[::i]/[::u] work
	e.View = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetWrap(true).
		SetScrollable(true).
		SetRegions(false)
	e.View.SetBackgroundColor(uitheme.Ink)
	e.View.SetBorder(true)
	e.View.SetBorderColor(uitheme.Line)
	e.View.SetTitle(" Document ")
	e.View.SetTitleColor(uitheme.Amber)
	e.View.SetTitleAlign(tview.AlignLeft)
	e.View.SetTextColor(uitheme.Text)

	// Status bar
	e.Status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	e.Status.SetBackgroundColor(uitheme.Panel)
	e.refreshStatus()

	// Layout: title | text | status
	e.Root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(e.TitleBar, 1, 0, false).
		AddItem(e.View, 0, 1, true).
		AddItem(e.Status, 1, 0, false)

	e.Root.SetBackgroundColor(uitheme.Ink)

	// All editing goes through the document model; we redraw after every change.
	e.Root.SetInputCapture(e.handleKeys)
	e.View.SetInputCapture(e.handleKeys)

	// Ctrl+C copies while text is selected; otherwise it keeps its global
	// meaning from main.go (the exit prompt).
	prevCapture := app.GetInputCapture()
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC && app.GetFocus() == e.View && e.Doc.HasSelection() {
			e.copySelection()
			return nil
		}
		if prevCapture != nil {
			return prevCapture(event)
		}
		return event
	})

	// Text is wrapped by the document model to the view's width, so whenever
	// the view's size changes (first draw, terminal resize) lay it out again.
	app.SetAfterDrawFunc(func(_ tcell.Screen) {
		_, _, w, h := e.View.GetInnerRect()
		if w == e.lastW && h == e.lastH {
			return
		}
		e.lastW, e.lastH = w, h
		go app.QueueUpdateDraw(func() { e.redraw() })
	})

	e.redraw()
	return e
}

func (e *Editor) redraw() {
	// The document wraps and aligns text itself, so it needs the real width
	// of the view.
	_, _, w, h := e.View.GetInnerRect()
	if w > 4 {
		e.Doc.DisplayWidth = w
	}

	// Remember where the view is scrolled to (mouse wheel included), because
	// SetText resets it.
	top, _ := e.View.GetScrollOffset()

	e.View.SetText(e.Doc.StyledTextWithCursor())

	// Scroll the minimum amount needed to keep the cursor's screen row
	// (a wrapped line, not a paragraph) inside the view.
	if h > 0 {
		row := e.Doc.CursorRow()
		if row < top {
			top = row
		} else if row >= top+h {
			top = row - h + 1
		}
	}
	if top < 0 {
		top = 0
	}
	e.View.ScrollTo(top, 0)

	e.refreshStatus()
	e.refreshTitle()
}

// pageRows is how many lines Page Up / Page Down move: one screenful, minus
// one line of overlap so the reader keeps their place.
func (e *Editor) pageRows() int {
	_, _, _, h := e.View.GetInnerRect()
	if h <= 1 {
		return 1
	}
	return h - 1
}

func (e *Editor) refreshTitle() {
	dirty := ""
	if e.Doc.Dirty {
		dirty = " [yellow]•[::-]"
	}
	name := e.Doc.Title
	if e.Doc.Filename != "" {
		name = utils.PrettyPath(e.Doc.Filename, "title")
	}
	e.TitleBar.SetText(fmt.Sprintf("[::b]GoWrite[::-]  %s%s  [gray]v%s[::-]",
		name, dirty, utils.VER))
}

func (e *Editor) refreshStatus() {
	words := e.Doc.WordCount()
	chars := e.Doc.CharCount()
	paras := e.Doc.ParagraphCount()
	line := e.Doc.CursorLine()
	col := e.Doc.CursorCol()
	style := e.Doc.StyleString()
	sel := ""
	if n := e.Doc.SelectionLen(); n > 0 {
		sel = fmt.Sprintf("  [teal]Sel:%d[::-]", n)
	}
	e.Status.SetText(fmt.Sprintf(
		" [teal]Ln %d, Col %d[::-]  [teal]W:%d C:%d ¶%d[::-]%s  [yellow]%s[::-]  [gray]| Alt+C/Y colour · Alt+L/E/R align · Alt+B/I/U style · ? help[::-]",
		line, col, words, chars, paras, sel, style))
}

func (e *Editor) handleKeys(event *tcell.EventKey) *tcell.EventKey {
	// Quit
	if event.Key() == tcell.KeyEscape {
		if e.Doc.HasSelection() {
			e.Doc.ClearSelection()
			e.redraw()
			return nil
		}
		e.confirmQuit()
		return nil
	}

	// Help
	if event.Rune() == '?' || (event.Rune() == '/' && event.Modifiers()&tcell.ModAlt != 0) {
		e.showHelp()
		return nil
	}

	mod := event.Modifiers()

	// Ctrl shortcuts
	if mod&tcell.ModCtrl != 0 {
		switch event.Key() {
		case tcell.KeyCtrlB:
			e.toggleStyle(document.StyleBold)
			return nil
		case tcell.KeyCtrlI:
			e.toggleStyle(document.StyleItalic)
			return nil
		case tcell.KeyCtrlU:
			e.toggleStyle(document.StyleUnderline)
			return nil
		case tcell.KeyCtrlF:
			e.showFindDialog(false)
			return nil
		case tcell.KeyCtrlH:
			e.showFindDialog(true)
			return nil
		case tcell.KeyCtrlG:
			e.showGoToDialog()
			return nil
		case tcell.KeyCtrlS:
			e.saveDocument()
			return nil
		case tcell.KeyCtrlA:
			e.Doc.SelectAll()
			e.redraw()
			return nil
		case tcell.KeyCtrlX:
			e.cutSelection()
			return nil
		case tcell.KeyCtrlV:
			e.pasteClipboard()
			return nil
		case tcell.KeyLeft:
			e.prepareMove(mod&tcell.ModShift != 0)
			e.Doc.MoveWordLeft()
			e.redraw()
			return nil
		case tcell.KeyRight:
			e.prepareMove(mod&tcell.ModShift != 0)
			e.Doc.MoveWordRight()
			e.redraw()
			return nil
		}
	}

	// Alt shortcuts (same spirit as GoSheet)
	if mod&tcell.ModAlt != 0 {
		switch event.Rune() {
		case 's', 'S':
			e.saveDocument()
			return nil
		case 'o', 'O':
			e.openDocument()
			return nil
		case 'n', 'N':
			e.newDocument()
			return nil
		case 'f', 'F':
			e.showFindDialog(false)
			return nil
		case 'h', 'H':
			e.showFindDialog(true)
			return nil
		case 'g', 'G':
			e.showGoToDialog()
			return nil
		case 'c', 'C':
			e.showColorPicker(false) // foreground
			return nil
		case 'y', 'Y':
			e.showColorPicker(true) // background
			return nil
		case 'l', 'L':
			e.Doc.SetParagraphAlign(document.AlignLeft)
			e.redraw()
			return nil
		case 'e', 'E':
			e.Doc.SetParagraphAlign(document.AlignCenter)
			e.redraw()
			return nil
		case 'r', 'R':
			e.Doc.SetParagraphAlign(document.AlignRight)
			e.redraw()
			return nil
		case 'b', 'B':
			e.toggleStyle(document.StyleBold)
			return nil
		case 'i', 'I':
			e.toggleStyle(document.StyleItalic)
			return nil
		case 'u', 'U':
			e.toggleStyle(document.StyleUnderline)
			return nil
		case 'k', 'K':
			e.toggleStyle(document.StyleStrike)
			return nil
		case 'a', 'A':
			if e.Doc.ApplyActiveToSelection() {
				e.redraw()
				return nil
			}
			s := e.Doc.ActiveStyleFlags()
			if s != document.StyleNone {
				e.Doc.ApplyStyleToParagraph(s, true)
				e.redraw()
			}
			return nil
		}
	}

	// F3 / F4 = find next / prev
	if event.Key() == tcell.KeyF3 {
		e.findNext(false)
		return nil
	}
	if event.Key() == tcell.KeyF4 {
		e.findNext(true)
		return nil
	}

	// Navigation. Shift extends the selection; a plain move clears it
	// (Left / Right collapse it to its start / end).
	shift := mod&tcell.ModShift != 0
	switch event.Key() {
	case tcell.KeyLeft:
		if !shift && e.Doc.HasSelection() {
			e.Doc.CollapseSelection(true)
		} else {
			e.prepareMove(shift)
			e.Doc.MoveLeft()
		}
		e.redraw()
		return nil
	case tcell.KeyRight:
		if !shift && e.Doc.HasSelection() {
			e.Doc.CollapseSelection(false)
		} else {
			e.prepareMove(shift)
			e.Doc.MoveRight()
		}
		e.redraw()
		return nil
	case tcell.KeyUp:
		e.prepareMove(shift)
		e.Doc.MoveUp()
		e.redraw()
		return nil
	case tcell.KeyDown:
		e.prepareMove(shift)
		e.Doc.MoveDown()
		e.redraw()
		return nil
	case tcell.KeyHome:
		// Home = start of the wrapped line; Ctrl/Alt+Home = start of document
		e.prepareMove(shift)
		if mod&(tcell.ModCtrl|tcell.ModAlt) != 0 {
			e.Doc.MoveDocStart()
		} else {
			e.Doc.MoveLineStart()
		}
		e.redraw()
		return nil
	case tcell.KeyEnd:
		// End = end of the wrapped line; Ctrl/Alt+End = end of document
		e.prepareMove(shift)
		if mod&(tcell.ModCtrl|tcell.ModAlt) != 0 {
			e.Doc.MoveDocEnd()
		} else {
			e.Doc.MoveLineEnd()
		}
		e.redraw()
		return nil
	case tcell.KeyPgUp:
		e.prepareMove(shift)
		e.Doc.MovePage(-e.pageRows())
		e.redraw()
		return nil
	case tcell.KeyPgDn:
		e.prepareMove(shift)
		e.Doc.MovePage(e.pageRows())
		e.redraw()
		return nil
	case tcell.KeyInsert:
		// Ctrl+Insert = copy, Shift+Insert = paste
		if mod&tcell.ModCtrl != 0 {
			e.copySelection()
		} else if shift {
			e.pasteClipboard()
		}
		return nil
	case tcell.KeyEnter:
		e.Doc.InsertRune('\n')
		e.redraw()
		return nil
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		e.Doc.Backspace()
		e.redraw()
		return nil
	case tcell.KeyDelete:
		if shift {
			e.cutSelection()
			return nil
		}
		e.Doc.Delete()
		e.redraw()
		return nil
	case tcell.KeyTab:
		e.Doc.InsertRune('\t')
		e.redraw()
		return nil
	}

	// Printable runes
	r := event.Rune()
	if r != 0 && unicode.IsPrint(r) {
		// Skip if this was an Alt-combo we didn't handle (already returned above)
		if mod&tcell.ModAlt != 0 || mod&tcell.ModCtrl != 0 {
			return nil
		}
		e.Doc.InsertRune(r)
		e.redraw()
		return nil
	}

	return event
}

func (e *Editor) toggleStyle(s document.Style) {
	// With a selection the style is applied to (or removed from) the selected
	// text. Otherwise it only changes the active typing style: new characters
	// use it, existing text is untouched.
	if !e.Doc.ToggleStyleOnSelection(s) {
		e.Doc.ToggleStyle(s)
	}
	e.redraw()
}

// prepareMove starts or extends the selection (Shift held) or clears it, just
// before the cursor moves.
func (e *Editor) prepareMove(extend bool) {
	if extend {
		e.Doc.BeginSelection()
	} else {
		e.Doc.ClearSelection()
	}
}

func (e *Editor) copySelection() {
	n := e.Doc.SelectionLen()
	if !e.Doc.Copy() {
		e.flashStatus("Nothing selected")
		return
	}
	e.redraw()
	e.flashStatus(fmt.Sprintf("Copied %d characters", n))
}

func (e *Editor) cutSelection() {
	n := e.Doc.SelectionLen()
	if !e.Doc.Cut() {
		e.flashStatus("Nothing selected")
		return
	}
	e.redraw()
	e.flashStatus(fmt.Sprintf("Cut %d characters", n))
}

func (e *Editor) pasteClipboard() {
	if !e.Doc.Paste() {
		e.flashStatus("Clipboard is empty")
		return
	}
	e.redraw()
}

func (e *Editor) confirmQuit() {
	msg := "Quit GoWrite?"
	if e.Doc.Dirty {
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

func (e *Editor) showHelp() {
	helpText := `[::b]GoWrite – Keyboard Shortcuts[::-]

[yellow]Navigation[::-]
  Arrow keys            Move (↑/↓ follow wrapped lines)
  Home / End            Start / end of the line
  Ctrl+Home / Ctrl+End  Start / end of the document
  PgUp / PgDn           Move one screen up / down
  Ctrl+← / Ctrl+→       Word left / right
  Alt+G  or  Ctrl+G     Go to line (paragraph)

[yellow]Selection & clipboard[::-]
  Shift+arrows/Home/End/PgUp/PgDn   Select
  Shift+Ctrl+← / →      Select by word
  Ctrl+A                Select all
  Esc                   Clear the selection
  Ctrl+C / Ctrl+Insert  Copy (Ctrl+C with nothing selected = exit prompt)
  Ctrl+X / Shift+Delete Cut
  Ctrl+V / Shift+Insert Paste
  Typing, Backspace and Delete replace the selection.
  Styles, colours and alignment apply to the selection.

[yellow]Search[::-]
  Alt+F  or  Ctrl+F     Find
  Alt+H  or  Ctrl+H     Find & Replace
  F3 / F4               Find next / previous

[yellow]Styling[::-]
  Alt+B / Ctrl+B        Toggle Bold
  Alt+I / Ctrl+I        Toggle Italic
  Alt+U / Ctrl+U        Toggle Underline
  Alt+K                 Toggle Strikethrough
  Alt+C                 Text colour
  Alt+Y                 Background colour
  Alt+L / E / R         Align left / center / right

[yellow]File[::-]
  Alt+N                 New document
  Alt+O                 Open document
  Alt+S / Ctrl+S        Save document

[yellow]Other[::-]
  ?  or  Alt+/          This help
  Esc                   Quit (asks if dirty); clears a selection first

Word wrap is always on and follows the window size.
Alignment applies to every wrapped line of a paragraph.
Enter starts a new paragraph.
New text uses the active style + colours (status bar).
`

	tv := tview.NewTextView().
		SetDynamicColors(true).
		SetText(helpText).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)
	tv.SetBackgroundColor(uitheme.Panel)
	tv.SetTextColor(uitheme.Text)
	uitheme.Paint(tv, "Help")

	// Footer hint
	hint := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true).
		SetText("[gray]↑↓ scroll  ·  Esc or Enter to close[::-]")
	hint.SetBackgroundColor(uitheme.Panel)

	frame := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(tv, 0, 1, true).
		AddItem(hint, 1, 0, false)

	close := func() {
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
	}

	frame.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			close()
			return nil
		}
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			close()
			return nil
		}
		return event
	})
	tv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyEnter:
			close()
			return nil
		}
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			close()
			return nil
		}
		return event
	})

	// Centered panel that fits typical terminals without going off-screen
	pages := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(frame, 64, 0, true).
			AddItem(nil, 0, 1, false), 0, 3, true).
		AddItem(nil, 0, 1, false)
	pages.SetBackgroundColor(uitheme.Ink)

	e.App.SetRoot(pages, true).SetFocus(tv)
}

func (e *Editor) newDocument() {
	if e.Doc.Dirty {
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

func (e *Editor) saveDocument() {
	e.Doc.Dirty = false
	e.refreshTitle()
	modal := tview.NewModal().
		SetText("Save is not fully wired yet.\n(Document marked clean for now)").
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
		})
	uitheme.StyleModalOK(modal, "Save")
	e.App.SetRoot(modal, true).SetFocus(modal)
}

func (e *Editor) openDocument() {
	modal := tview.NewModal().
		SetText("Open dialog will appear here\nin the next iteration.").
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
		})
	uitheme.StyleModal(modal, "Open")
	e.App.SetRoot(modal, true).SetFocus(modal)
}


// ---------- Colour picker ----------

func (e *Editor) showColorPicker(background bool) {
	title := "Text colour"
	if background {
		title = "Background colour"
	}
	list := tview.NewList()
	uitheme.StyleList(list, title)
	list.SetBorder(true)

	fg, bg := e.Doc.ActiveColors()
	current := fg
	if background {
		current = bg
	}

	for _, c := range document.ColorPalette {
		name := c.Name
		val := c.Value
		label := name
		if val == current {
			label = name + "  ←"
		}
		// Preview colour in secondary text when possible
		sec := val
		if val == "" {
			sec = "(terminal default)"
		}
		v := val // capture
		list.AddItem(label, sec, 0, func() {
			// With a selection, recolour it; otherwise set the typing colour.
			if !e.Doc.SetSelectionColor(background, v) {
				if background {
					e.Doc.SetActiveBg(v)
				} else {
					e.Doc.SetActiveFg(v)
				}
			}
			e.redraw()
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
		})
	}
	list.AddItem("Close", "Esc", 'q', func() {
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		return event
	})

	pages := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(list, 40, 0, true).
			AddItem(nil, 0, 1, false), 16, 0, true).
		AddItem(nil, 0, 1, false)

	e.App.SetRoot(pages, true).SetFocus(list)
}

// ---------- Go to line ----------

func (e *Editor) showGoToDialog() {
	form := tview.NewForm()
	uitheme.StyleForm(form, "Go to line")

	lineField := tview.NewInputField().
		SetLabel("Line (paragraph) ").
		SetText(strconv.Itoa(e.Doc.CursorLine())).
		SetFieldWidth(8)
	form.AddFormItem(lineField)

	form.AddButton("Go", func() {
		n, err := strconv.Atoi(strings.TrimSpace(lineField.GetText()))
		if err != nil || n < 1 {
			return
		}
		e.Doc.GoToLine(n)
		e.redraw()
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
	})
	form.AddButton("Cancel", func() {
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
	})

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		return event
	})

	pages := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(form, 40, 0, true).
			AddItem(nil, 0, 1, false), 9, 0, true).
		AddItem(nil, 0, 1, false)

	e.App.SetRoot(pages, true).SetFocus(form)
}

// ---------- Find / Replace ----------

func (e *Editor) showFindDialog(withReplace bool) {
	form := tview.NewForm()
	title := "Find"
	if withReplace {
		title = "Find & Replace"
	}
	uitheme.StyleForm(form, title)

	findField := tview.NewInputField().
		SetLabel("Find what     ").
		SetText(e.Doc.LastSearch).
		SetFieldWidth(30)
	form.AddFormItem(findField)

	var replaceField *tview.InputField
	if withReplace {
		replaceField = tview.NewInputField().
			SetLabel("Replace with ").
			SetFieldWidth(30)
		form.AddFormItem(replaceField)
	}

	caseCheck := tview.NewCheckbox().
		SetLabel("Case sensitive").
		SetChecked(e.Doc.LastSearchCase)
	form.AddFormItem(caseCheck)

	wholeCheck := tview.NewCheckbox().
		SetLabel("Whole word    ").
		SetChecked(e.Doc.LastSearchWhole)
	form.AddFormItem(wholeCheck)

	doFind := func(backwards bool) {
		q := findField.GetText()
		if q == "" {
			return
		}
		_, _, found := e.Doc.Find(q, caseCheck.IsChecked(), wholeCheck.IsChecked(), backwards)
		e.redraw()
		if !found {
			e.flashStatus("Not found: " + q)
		} else {
			e.flashStatus(fmt.Sprintf("Found at line %d", e.Doc.CursorLine()))
		}
	}

	form.AddButton("Find Next", func() { doFind(false) })
	form.AddButton("Find Prev", func() { doFind(true) })

	if withReplace {
		form.AddButton("Replace All", func() {
			q := findField.GetText()
			if q == "" {
				return
			}
			repl := ""
			if replaceField != nil {
				repl = replaceField.GetText()
			}
			n := e.Doc.ReplaceAll(q, repl, caseCheck.IsChecked(), wholeCheck.IsChecked())
			e.redraw()
			e.flashStatus(fmt.Sprintf("Replaced %d occurrence(s)", n))
		})
	}

	form.AddButton("Close", func() {
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
	})

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			doFind(false)
			return nil
		}
		return event
	})

	height := 12
	if withReplace {
		height = 14
	}
	pages := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(form, 50, 0, true).
			AddItem(nil, 0, 1, false), height, 0, true).
		AddItem(nil, 0, 1, false)

	e.App.SetRoot(pages, true).SetFocus(form)
}

func (e *Editor) findNext(backwards bool) {
	if e.Doc.LastSearch == "" {
		e.showFindDialog(false)
		return
	}
	_, _, found := e.Doc.Find(e.Doc.LastSearch, e.Doc.LastSearchCase, e.Doc.LastSearchWhole, backwards)
	e.redraw()
	if !found {
		e.flashStatus("No more matches")
	} else {
		e.flashStatus(fmt.Sprintf("Found at line %d", e.Doc.CursorLine()))
	}
}

func (e *Editor) flashStatus(msg string) {
	e.Status.SetText(fmt.Sprintf(" [yellow]%s[::-]", msg))
}

// Mount sets the editor as the application root and focuses the view.
func (e *Editor) Mount() {
	e.App.SetRoot(e.Root, true).SetFocus(e.View)
}

// ShowErrorModal shows a styled error and returns to the given primitive.
func ShowErrorModal(app *tview.Application, returnTo tview.Primitive, message string) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Error\n\n%s", message)).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(returnTo, true)
		})
	uitheme.StyleModalWarn(modal, "Error")
	app.SetRoot(modal, true).SetFocus(modal)
}
