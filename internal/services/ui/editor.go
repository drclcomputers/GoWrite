// Copyright (c)  @drclcomputers. All rights reserved.
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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Editor is the main writing surface.
type Editor struct {
	App      *tview.Application
	Doc      *document.Document
	Root     *tview.Flex
	View     *tview.TextView
	Status   *tview.TextView
	TitleBar *tview.TextView

	lastW, lastH int
	flashSeq     int
}

// NewEditor builds the full-screen editor layout.
func NewEditor(app *tview.Application, doc *document.Document) *Editor {
	e := &Editor{App: app, Doc: doc}

	e.TitleBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	e.TitleBar.SetBackgroundColor(uitheme.Ink)
	e.refreshTitle()

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

	e.Status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	e.Status.SetBackgroundColor(uitheme.Panel)
	e.refreshStatus()

	e.Root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(e.TitleBar, 1, 0, false).
		AddItem(e.View, 0, 1, true).
		AddItem(e.Status, 1, 0, false)
	e.Root.SetBackgroundColor(uitheme.Ink)

	e.Root.SetInputCapture(e.handleKeys)
	e.View.SetInputCapture(e.handleKeys)

	// NOTE: no Ctrl+C interceptor here. Ctrl+C falls through to main.go's
	// global handler (exit prompt) like a standard terminal interrupt.
	// Copy is now Alt+Shift+C.

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
	_, _, w, h := e.View.GetInnerRect()
	if w > 4 {
		e.Doc.SetDisplayWidth(w)
	}
	top, _ := e.View.GetScrollOffset()
	e.View.SetText(e.Doc.StyledTextWithCursor())
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

func (e *Editor) pageRows() int {
	_, _, _, h := e.View.GetInnerRect()
	if h <= 1 {
		return 1
	}
	return h - 1
}

func (e *Editor) refreshTitle() {
	dirty := ""
	if e.Doc.IsDirty() {
		dirty = " [yellow]•[::-]"
	}
	name := e.Doc.GetTitle()
	if fn := e.Doc.GetFilename(); fn != "" {
		name = utils.PrettyPath(fn, "title")
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
		" [teal]Ln %d, Col %d[::-]  [teal]W:%d C:%d ¶%d[::-]%s  [yellow]%s[::-]  [gray]| Alt+C/Y colour · Alt+L/E/R align · Alt+B/I/U style · Alt+Z undo · ? help[::-]",
		line, col, words, chars, paras, sel, style))
}

func (e *Editor) handleKeys(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyEscape {
		if e.Doc.HasSelection() {
			e.Doc.ClearSelection()
			e.redraw()
			return nil
		}
		e.confirmQuit()
		return nil
	}

	if event.Rune() == '?' || (event.Rune() == '/' && event.Modifiers()&tcell.ModAlt != 0) {
		e.showHelp()
		return nil
	}

	mod := event.Modifiers()

	// ---- Alt shortcuts (letters) -------------------------------------------
	// Every application shortcut is Alt-based. Shift is used to disambiguate
	// the "extra" bindings (copy / cut / paste / select all / redo).
	if mod&tcell.ModAlt != 0 {
		hasShift := mod&tcell.ModShift != 0
		switch event.Rune() {
		case 'a', 'A':
			if hasShift {
				e.Doc.SelectAll()
				e.redraw()
				return nil
			}
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
		case 'b', 'B':
			e.toggleStyle(document.StyleBold)
			return nil
		case 'c', 'C':
			if hasShift {
				e.copySelection()
				return nil
			}
			e.showColorPicker(false) // foreground
			return nil
		case 'e', 'E':
			e.Doc.SetParagraphAlign(document.AlignCenter)
			e.redraw()
			return nil
		case 'f', 'F':
			e.showFindDialog(false)
			return nil
		case 'g', 'G':
			e.showGoToDialog()
			return nil
		case 'h', 'H':
			e.showFindDialog(true)
			return nil
		case 'i', 'I':
			e.toggleStyle(document.StyleItalic)
			return nil
		case 'k', 'K':
			e.toggleStyle(document.StyleStrike)
			return nil
		case 'l', 'L':
			e.Doc.SetParagraphAlign(document.AlignLeft)
			e.redraw()
			return nil
		case 'n', 'N':
			e.newDocument()
			return nil
		case 'o', 'O':
			e.openDocument()
			return nil
		case 'r', 'R':
			e.Doc.SetParagraphAlign(document.AlignRight)
			e.redraw()
			return nil
		case 's', 'S':
			e.saveDocument()
			return nil
		case 'u', 'U':
			e.toggleStyle(document.StyleUnderline)
			return nil
		case 'v', 'V':
			if hasShift {
				e.pasteClipboard()
				return nil
			}
		case 'x', 'X':
			if hasShift {
				e.cutSelection()
				return nil
			}
		case 'y', 'Y':
			e.showColorPicker(true) // background
			return nil
		case 'z', 'Z':
			if hasShift {
				e.redo()
			} else {
				e.undo()
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

	// Navigation. Shift extends the selection; a plain move clears it.
	// Alt+Shift+Left/Right = word navigation (with selection extension).
	shift := mod&tcell.ModShift != 0
	altShift := mod&tcell.ModAlt != 0 && shift
	switch event.Key() {
	case tcell.KeyLeft:
		if altShift {
			e.prepareMove(true)
			e.Doc.MoveWordLeft()
			e.redraw()
			return nil
		}
		if !shift && e.Doc.HasSelection() {
			e.Doc.CollapseSelection(true)
		} else {
			e.prepareMove(shift)
			e.Doc.MoveLeft()
		}
		e.redraw()
		return nil
	case tcell.KeyRight:
		if altShift {
			e.prepareMove(true)
			e.Doc.MoveWordRight()
			e.redraw()
			return nil
		}
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
		e.prepareMove(shift)
		if mod&(tcell.ModCtrl|tcell.ModAlt) != 0 {
			e.Doc.MoveDocStart()
		} else {
			e.Doc.MoveLineStart()
		}
		e.redraw()
		return nil
	case tcell.KeyEnd:
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
		// Shift+Insert = paste (non-Ctrl, preserved).
		if shift {
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
		// Shift+Delete = cut (non-Ctrl, preserved).
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

	r := event.Rune()
	if r != 0 && unicode.IsPrint(r) {
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
	if !e.Doc.ToggleStyleOnSelection(s) {
		e.Doc.ToggleStyle(s)
	}
	e.redraw()
}

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

func (e *Editor) undo() {
	if !e.Doc.Undo() {
		e.flashStatus("Nothing to undo")
		return
	}
	e.redraw()
	e.flashStatus("Undone")
}

func (e *Editor) redo() {
	if !e.Doc.Redo() {
		e.flashStatus("Nothing to redo")
		return
	}
	e.redraw()
	e.flashStatus("Redone")
}

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

func (e *Editor) saveDocument() {
	filename := e.Doc.GetFilename()
	if filename == "" {
		e.showSaveAsDialog()
		return
	}
	e.saveToFile(filename)
}

func (e *Editor) saveToFile(filename string) {
	data := []byte(e.Doc.Text())
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		ShowErrorModal(e.App, e.Root, fmt.Sprintf("Could not save %s: %v", filename, err))
		return
	}
	e.Doc.SetFilename(filename)
	e.Doc.SetTitle(filepath.Base(filename))
	e.Doc.SetDirty(false)
	e.redraw()

	modal := tview.NewModal().
		SetText(fmt.Sprintf("Saved to %s", utils.PrettyPath(filename, "save"))).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
		})
	uitheme.StyleModalOK(modal, "Save")
	e.App.SetRoot(modal, true).SetFocus(modal)
}

func (e *Editor) showSaveAsDialog() {
	form := tview.NewForm()
	uitheme.StyleForm(form, "Save As")

	defaultName := e.Doc.GetTitle()
	if defaultName == "" || defaultName == "Untitled" {
		defaultName = "untitled.txt"
	} else if !strings.Contains(defaultName, ".") {
		defaultName += ".txt"
	}
	field := tview.NewInputField().
		SetLabel("Filename ").
		SetText(defaultName).
		SetFieldWidth(34)
	form.AddFormItem(field)

	form.AddButton("Save", func() {
		name := strings.TrimSpace(field.GetText())
		if name == "" {
			return
		}
		e.saveToFile(name)
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

	e.App.SetRoot(centredModal(form, 54, 9), true).SetFocus(form)
}

// OpenDocument is the public entry point used by main after the start menu.
func (e *Editor) OpenDocument() {
	e.openDocument()
}

func (e *Editor) openDocument() {
	form := tview.NewForm()
	uitheme.StyleForm(form, "Open")

	field := tview.NewInputField().
		SetLabel("Filename ").
		SetFieldWidth(34)
	form.AddFormItem(field)

	form.AddButton("Open", func() {
		name := strings.TrimSpace(field.GetText())
		if name == "" {
			return
		}
		data, err := os.ReadFile(name)
		if err != nil {
			ShowErrorModal(e.App, e.Root, fmt.Sprintf("Could not open %s: %v", name, err))
			return
		}
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		text = strings.ReplaceAll(text, "\r", "\n")
		e.Doc.LoadText(text)
		e.Doc.SetFilename(name)
		e.Doc.SetTitle(filepath.Base(name))
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

	e.App.SetRoot(centredModal(form, 54, 9), true).SetFocus(form)
}

// ---------- Colour picker ----------

func parseHexColor(s string) tcell.Color {
	if len(s) != 7 || s[0] != '#' {
		return tcell.ColorDefault
	}
	var r, g, b int
	if _, err := fmt.Sscanf(s, "#%02x%02x%02x", &r, &g, &b); err != nil {
		return tcell.ColorDefault
	}
	return tcell.NewRGBColor(int32(r), int32(g), int32(b))
}

func contrastColor(c tcell.Color) tcell.Color {
	r, g, b := c.RGB()
	lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	if lum > 140 {
		return tcell.ColorBlack
	}
	return tcell.ColorWhite
}

func (e *Editor) showColorPicker(background bool) {
	title := "Text Colour"
	if background {
		title = "Background Colour"
	}

	palette := document.ColorPalette
	const cols = 5

	fg, bg := e.Doc.ActiveColors()
	current := fg
	if background {
		current = bg
	}

	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, true)

	selectedRow, selectedCol := 0, 0
	for i, c := range palette {
		row, col := i/cols, i%cols
		label := c.Name
		if c.Value == current {
			label = "* " + c.Name
			selectedRow, selectedCol = row, col
		}
		cell := tview.NewTableCell(" " + label + " ").
			SetAlign(tview.AlignCenter).
			SetExpansion(1)
		if c.Value == "" {
			cell.SetBackgroundColor(uitheme.Field)
			cell.SetTextColor(uitheme.Text)
		} else {
			rgb := parseHexColor(c.Value)
			cell.SetBackgroundColor(rgb)
			cell.SetTextColor(contrastColor(rgb))
		}
		table.SetCell(row, col, cell)
	}
	table.Select(selectedRow, selectedCol)

	preview := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	preview.SetBackgroundColor(uitheme.Panel)

	updatePreview := func() {
		row, col := table.GetSelection()
		idx := row*cols + col
		if idx < 0 || idx >= len(palette) {
			preview.SetText("")
			return
		}
		c := palette[idx]
		if c.Value == "" {
			preview.SetText(" [gray](terminal default)[-]")
			return
		}
		preview.SetText(fmt.Sprintf(" [#%s]███[::-]  %s  [gray]%s[-]",
			strings.TrimPrefix(c.Value, "#"), c.Name, c.Value))
	}
	updatePreview()
	table.SetSelectionChangedFunc(func(row, col int) { updatePreview() })

	apply := func() {
		row, col := table.GetSelection()
		idx := row*cols + col
		if idx < 0 || idx >= len(palette) {
			return
		}
		c := palette[idx]
		if !e.Doc.SetSelectionColor(background, c.Value) {
			if background {
				e.Doc.SetActiveBg(c.Value)
			} else {
				e.Doc.SetActiveFg(c.Value)
			}
		}
		e.redraw()
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
	}
	table.SetSelectedFunc(func(row, col int) { apply() })
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		if event.Rune() == 'q' || event.Rune() == 'Q' {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		return event
	})

	hint := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText("[gray]arrows move · Enter applies · Esc closes[::-]")
	hint.SetBackgroundColor(uitheme.Panel)

	body := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(table, 0, 1, true).
		AddItem(preview, 1, 0, false).
		AddItem(hint, 1, 0, false)
	uitheme.Paint(body, title)

	e.App.SetRoot(centredModal(body, 62, 12), true).SetFocus(table)
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
			ShowErrorModal(e.App, e.Root, "Invalid line number")
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

	e.App.SetRoot(centredModal(form, 44, 7), true).SetFocus(form)
}

// ---------- Find / Replace ----------

func (e *Editor) showFindDialog(withReplace bool) {
	form := tview.NewForm()
	title := "Find"
	if withReplace {
		title = "Find & Replace"
	}
	uitheme.StyleForm(form, title)

	lastQuery, lastCase, lastWhole := e.Doc.LastSearchQuery()
	findField := tview.NewInputField().
		SetLabel("Find what     ").
		SetText(lastQuery).
		SetFieldWidth(28)
	form.AddFormItem(findField)

	var replaceField *tview.InputField
	if withReplace {
		replaceField = tview.NewInputField().
			SetLabel("Replace with ").
			SetFieldWidth(28)
		form.AddFormItem(replaceField)
	}

	caseCheck := tview.NewCheckbox().
		SetLabel("Case sensitive").
		SetChecked(lastCase)
	form.AddFormItem(caseCheck)

	wholeCheck := tview.NewCheckbox().
		SetLabel("Whole word    ").
		SetChecked(lastWhole)
	form.AddFormItem(wholeCheck)

	doFind := func(backwards bool) {
		q := findField.GetText()
		if q == "" {
			return
		}
		_, _, found := e.Doc.Find(q, caseCheck.IsChecked(), wholeCheck.IsChecked(), backwards)
		if !found {
			ShowErrorModal(e.App, e.Root, "No matches found")
			return
		}
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
		e.redraw()
		e.flashStatus(fmt.Sprintf("Found at line %d", e.Doc.CursorLine()))
	}

	form.AddButton("Find Next", func() { doFind(false) })
	form.AddButton("Find Previous", func() { doFind(true) })

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
			if n == 0 {
				ShowErrorModal(e.App, e.Root, "No matches found")
				return
			}
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
			e.flashStatus(fmt.Sprintf("Replaced %d occurrence(s)", n))
		})
	}

	form.AddButton("Clear", func() {
		findField.SetText("")
		if replaceField != nil {
			replaceField.SetText("")
		}
		caseCheck.SetChecked(false)
		wholeCheck.SetChecked(false)
		e.Doc.ClearLastSearch()
	})
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

	w, h := 54, 10
	if withReplace {
		h = 12
	}
	e.App.SetRoot(centredModal(form, w, h), true).SetFocus(form)
}

func (e *Editor) findNext(backwards bool) {
	q, cs, ww := e.Doc.LastSearchQuery()
	if q == "" {
		e.showFindDialog(false)
		return
	}
	_, _, found := e.Doc.Find(q, cs, ww, backwards)
	e.redraw()
	if !found {
		e.flashStatus("No more matches")
	} else {
		e.flashStatus(fmt.Sprintf("Found at line %d", e.Doc.CursorLine()))
	}
}

func (e *Editor) flashStatus(msg string) {
	e.flashSeq++
	seq := e.flashSeq
	e.Status.SetText(fmt.Sprintf(" [yellow]%s[::-]", msg))
	time.AfterFunc(3*time.Second, func() {
		e.App.QueueUpdateDraw(func() {
			if seq == e.flashSeq {
				e.refreshStatus()
			}
		})
	})
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
