// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package ui

import (
	"fmt"
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"
	"gowrite/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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

func (e *Editor) Mount() {
	e.App.SetRoot(e.Root, true).SetFocus(e.View)
}
