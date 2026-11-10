// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

import (
	"fmt"
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"
	"gowrite/internal/utils"
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

	lastW, lastH           int
	flashSeq               int
	showFormattingMarks    bool
	defaultFormattingMarks bool
}

func NewEditor(app *tview.Application, doc *document.Document, settings utils.Settings) *Editor {
	applyDocumentDefaults(doc, settings)
	e := &Editor{
		App:                    app,
		Doc:                    doc,
		showFormattingMarks:    settings.ShowFormattingMarks,
		defaultFormattingMarks: settings.ShowFormattingMarks,
	}

	// Top Bar
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

	// Start 1 row under the top border, stop 1 row above the bottom border
	e.View.SetBorderPadding(1, 1, 0, 0)

	// Status Bar
	e.Status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	e.Status.SetBackgroundColor(uitheme.Panel)

	// Layout
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

func applyDocumentDefaults(doc *document.Document, settings utils.Settings) {
	page := document.DefaultA4Portrait()
	page.Size = document.PageSize(settings.DefaultPageSize)
	page.Orientation = document.Orientation(settings.DefaultOrientation)
	page.Width, page.Height = document.GetDimensions(page.Size, page.Orientation)
	page.MarginLeft = settings.MarginLeft
	page.MarginRight = settings.MarginRight
	page.MarginTop = settings.MarginTop
	page.MarginBottom = settings.MarginBottom
	doc.Page = page.Normalized()
}

func (e *Editor) redraw() {
	_, _, w, h := e.View.GetInnerRect()
	if w > 4 {
		e.Doc.SetDisplayWidth(w)
	}
	top, _ := e.View.GetScrollOffset()
	e.View.SetText(e.Doc.StyledTextWithCursorAndMarks(e.showFormattingMarks))
	if h > 0 {
		row := e.Doc.CursorRow()
		if row < top {
			top = 0
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

func (e *Editor) refreshUI() {
	name := e.Doc.GetTitle()
	if fn := e.Doc.GetFilename(); fn != "" {
		name = filepath.Base(fn)
	}

	dirty := ""
	if e.Doc.IsDirty() {
		dirty = " ●"
	}

	e.View.SetTitle(fmt.Sprintf(" %s%s ", name, dirty))

	line := e.Doc.CursorLine()
	col := e.Doc.CursorCol()
	words := e.Doc.WordCount()
	chars := e.Doc.CharCount()
	style := e.Doc.StyleString()
	page, pages := e.Doc.PageInfo()

	e.TopBar.SetText(fmt.Sprintf(topBarFormat, line, col, page, pages, words, chars, style))

	e.Status.SetText(statusHint)
}

// Status-line templates, expanded once at start-up.
var (
	topBarFormat = uitheme.Markup(" {key}Ln{end} {val}%d{end}  {key}Col{end} {val}%d{end}  {key}Page{end} {val}%d/%d{end}" +
		"   {plain}|{end}   {key}Words:{end} {val}%d{end}  {key}Chars:{end} {val}%d{end}" +
		"   {plain}|{end}   {key}Style{end} {bright}-{end} {val}%s{end}")
	statusHint = uitheme.Markup(" {ok}Document{end}    {val}Alt+Enter{end} {plain}format{end}    {val}?{end} {plain}help{end}")
)

func (e *Editor) Mount() {
	e.App.SetRoot(e.Root, true).SetFocus(e.View)
}
