// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package ui

import (
	"fmt"
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Editor struct {
	App    *tview.Application
	Doc    *document.Document
	Root   *tview.Flex
	TopBar *tview.TextView
	View   *tview.TextView
	Status *tview.TextView

	lastW, lastH int
	flashSeq     int
}

func NewEditor(app *tview.Application, doc *document.Document) *Editor {
	e := &Editor{App: app, Doc: doc}

	// Top Bar (Stats & Style)
	e.TopBar = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	e.TopBar.SetBackgroundColor(uitheme.Panel)

	// Main Document View
	e.View = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		SetWrap(true).
		SetScrollable(true).
		SetRegions(false)
	e.View.SetBackgroundColor(uitheme.Ink)
	e.View.SetBorder(true)
	e.View.SetBorderColor(uitheme.Line)
	e.View.SetTitleColor(uitheme.Amber)
	e.View.SetTitleAlign(tview.AlignCenter)
	e.View.SetTextColor(uitheme.Text)

	// Status Bar
	e.Status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	e.Status.SetBackgroundColor(uitheme.Panel)

	// Layout: TopBar | View | Status
	e.Root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(e.TopBar, 1, 0, false).
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
	e.refreshUI()
}

func (e *Editor) pageRows() int {
	_, _, _, h := e.View.GetInnerRect()
	if h <= 1 {
		return 1
	}
	return h - 1
}

// refreshUI synchronizes the TopBar, Title, and StatusBar.
func (e *Editor) refreshUI() {
	name := e.Doc.GetTitle()
	if fn := e.Doc.GetFilename(); fn != "" {
		name = filepath.Base(fn)
	}

	dirty := ""
	if e.Doc.IsDirty() {
		dirty = "●"
	}

	// Update View Title
	e.View.SetTitle(fmt.Sprintf(" %s %s ", name, dirty))

	// Update Top Bar
	line := e.Doc.CursorLine()
	col := e.Doc.CursorCol()
	words := e.Doc.WordCount()
	chars := e.Doc.CharCount()
	style := e.Doc.StyleString()
	e.TopBar.SetText(fmt.Sprintf(
		" [#5ec8c8]Ln[::-] [#f0d090]%d[::-]  [#5ec8c8]Col[::-] [#f0d090]%d[::-]   [white]|[::-]   [#5ec8c8]Words:[::-] [#f0d090]%d[::-]  [#5ec8c8]Chars:[::-] [#f0d090]%d[::-]   [white]|[::-]   [#5ec8c8]Style[::-] [#ffffff]-[::-] [#f0d090]%s[::-]",
		line, col, words, chars, style,
	))

	e.Status.SetText(" [#7dcea0]Document[::-]    [#8aa0b8]type[-] edit    [#f0d090]Alt+Enter[::-] [#ffffff]format[::-]    [#f0d090]Alt+S[::-] [#ffffff]save[::-]    [#f0d090]Alt+F[::-] [#ffffff]find[::-]    [#f0d090]?[::-] [#ffffff]help[::-]")
}

func (e *Editor) Mount() {
	e.App.SetRoot(e.Root, true).SetFocus(e.View)
}
