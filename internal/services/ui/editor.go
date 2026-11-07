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

	e.redraw()
	return e
}

func (e *Editor) redraw() {
	// Styled text with a block cursor so the user can see where they are
	e.View.SetText(e.Doc.StyledTextWithCursor())
	// Keep the cursor line roughly in view
	e.View.ScrollToBeginning()
	// Heuristic: jump toward the cursor paragraph
	line := e.Doc.CursorLine()
	if line > 3 {
		e.View.ScrollTo(line-2, 0)
	}
	e.refreshStatus()
	e.refreshTitle()
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
	e.Status.SetText(fmt.Sprintf(
		" [teal]Ln %d, Col %d[::-]  [teal]W:%d C:%d ¶%d[::-]  [yellow]%s[::-]  [gray]| Alt+C/Y colour · Alt+L/E/R align · Alt+B/I/U style · ? help[::-]",
		line, col, words, chars, paras, style))
}

func (e *Editor) handleKeys(event *tcell.EventKey) *tcell.EventKey {
	// Quit
	if event.Key() == tcell.KeyEscape {
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
		case tcell.KeyLeft:
			e.Doc.MoveWordLeft()
			e.redraw()
			return nil
		case tcell.KeyRight:
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

	// Navigation
	switch event.Key() {
	case tcell.KeyLeft:
		e.Doc.MoveLeft()
		e.redraw()
		return nil
	case tcell.KeyRight:
		e.Doc.MoveRight()
		e.redraw()
		return nil
	case tcell.KeyUp:
		e.Doc.MoveUp()
		e.redraw()
		return nil
	case tcell.KeyDown:
		e.Doc.MoveDown()
		e.redraw()
		return nil
	case tcell.KeyHome:
		// start of paragraph
		e.Doc.GoToLine(e.Doc.CursorLine())
		e.redraw()
		return nil
	case tcell.KeyEnd:
		// end of paragraph
		for {
			before := e.Doc.CursorCol()
			e.Doc.MoveRight()
			if e.Doc.CursorCol() == before {
				break
			}
			// stop if we moved to next paragraph
			if e.Doc.CursorCol() == 1 && before > 1 {
				e.Doc.MoveLeft()
				break
			}
		}
		e.redraw()
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
	// Only change the active typing style at the cursor.
	// Does NOT restyle existing text. New characters use ActiveStyle.
	// (Selection-based apply can land once we have a real selection range.)
	e.Doc.ToggleStyle(s)
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
  Arrow keys / Home / End
  Ctrl+← / Ctrl+→     Word left / right
  Alt+G  or  Ctrl+G   Go to line (paragraph)

[yellow]Search[::-]
  Alt+F  or  Ctrl+F   Find
  Alt+H  or  Ctrl+H   Find & Replace
  F3                  Find next
  F4                  Find previous

[yellow]Styling[::-]
  Alt+B / Ctrl+B      Toggle Bold
  Alt+I / Ctrl+I      Toggle Italic
  Alt+U / Ctrl+U      Toggle Underline
  Alt+K               Toggle Strikethrough
  Alt+C               Text colour
  Alt+Y               Background colour
  Alt+L / E / R       Align left / center / right (current paragraph)
  New text uses the active style + colours (see status bar)

[yellow]File[::-]
  Alt+N               New document
  Alt+O               Open document
  Alt+S  or  Ctrl+S   Save document

[yellow]Other[::-]
  ?  or  Alt+/        This help
  Esc                 Quit (asks if dirty)

Word wrap is always on. Enter starts a new paragraph.
Styles use terminal attributes – your font must support them.`

	modal := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
		})
	uitheme.StyleModal(modal, "Help")
	e.App.SetRoot(modal, true).SetFocus(modal)
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
			if background {
				e.Doc.SetActiveBg(v)
			} else {
				e.Doc.SetActiveFg(v)
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
