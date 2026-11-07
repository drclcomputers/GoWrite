// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package ui

import (
	"fmt"
	"gowrite/internal/services/uitheme"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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
