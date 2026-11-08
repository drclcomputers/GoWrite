// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

import (
	"fmt"
	"gowrite/internal/services/document"
	"gowrite/internal/services/ui/navigation"
	"time"
)

func (e *Editor) showGoToDialog() {
	form := navigation.NewGoToForm(e.Doc.CursorLine(), navigation.GoToHandlers{
		GoTo: func(line int) {
			e.Doc.GoToLine(line)
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
		},
		Invalid: func() {
			ShowErrorModal(e.App, e.Root, "Invalid line number")
		},
		Cancel: func() {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
		},
	})
	e.App.SetRoot(centredModal(form, 44, 7), true).SetFocus(form)
}

func (e *Editor) showOutlineDialog() {
	list := navigation.NewOutlineList(e.Doc.Outline(), func(entry document.OutlineEntry) {
		e.Doc.GoToLine(entry.Paragraph + 1)
		e.closeFormatAndRedraw()
	}, func() { e.showFormatDialog() })

	e.App.SetRoot(centredModal(list, 60, 18), true).SetFocus(list)
}

func (e *Editor) showFindDialog(withReplace bool) {
	lastQuery, lastCase, lastWhole := e.Doc.LastSearchQuery()
	form := navigation.NewFindForm(navigation.SearchOptions{
		Query:         lastQuery,
		CaseSensitive: lastCase,
		WholeWord:     lastWhole,
	}, withReplace, navigation.FindHandlers{
		Find: func(options navigation.SearchOptions, backwards bool) {
			if options.Query == "" {
				return
			}
			_, _, found := e.Doc.Find(options.Query, options.CaseSensitive, options.WholeWord, backwards)
			if !found {
				ShowErrorModal(e.App, e.Root, "No matches found")
				return
			}
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
			e.flashStatus(fmt.Sprintf("Found at line %d", e.Doc.CursorLine()))
		},
		ReplaceAll: func(options navigation.SearchOptions) {
			if options.Query == "" {
				return
			}
			n := e.Doc.ReplaceAll(options.Query, options.Replace, options.CaseSensitive, options.WholeWord)
			if n == 0 {
				ShowErrorModal(e.App, e.Root, "No matches found")
				return
			}
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
			e.flashStatus(fmt.Sprintf("Replaced %d occurrence(s)", n))
		},
		Clear: func() {
			e.Doc.ClearLastSearch()
		},
		Close: func() {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
		},
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
	e.Status.SetText(fmt.Sprintf(" [#f0d090]%s[::-]", msg))
	time.AfterFunc(3*time.Second, func() {
		e.App.QueueUpdateDraw(func() {
			if seq == e.flashSeq {
				e.refreshUI()
			}
		})
	})
}
