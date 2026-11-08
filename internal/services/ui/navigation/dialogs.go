// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package navigation

import (
	"fmt"
	"strconv"
	"strings"

	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type GoToHandlers struct {
	GoTo    func(line int)
	Invalid func()
	Cancel  func()
}

func NewGoToForm(currentLine int, handlers GoToHandlers) *tview.Form {
	form := tview.NewForm()
	uitheme.StyleForm(form, "Go to line")
	lineField := tview.NewInputField().
		SetLabel("Line (paragraph) ").
		SetText(strconv.Itoa(currentLine)).
		SetFieldWidth(8)
	form.AddFormItem(lineField)

	form.AddButton("Go", func() {
		line, err := strconv.Atoi(strings.TrimSpace(lineField.GetText()))
		if err != nil || line < 1 {
			handlers.Invalid()
			return
		}
		handlers.GoTo(line)
	})
	form.AddButton("Cancel", handlers.Cancel)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			handlers.Cancel()
			return nil
		}
		return event
	})
	return form
}

type SearchOptions struct {
	Query         string
	Replace       string
	CaseSensitive bool
	WholeWord     bool
}

type FindHandlers struct {
	Find       func(options SearchOptions, backwards bool)
	ReplaceAll func(options SearchOptions)
	Clear      func()
	Close      func()
}

func NewFindForm(initial SearchOptions, withReplace bool, handlers FindHandlers) *tview.Form {
	form := tview.NewForm()
	title := "Find"
	if withReplace {
		title = "Find & Replace"
	}
	uitheme.StyleForm(form, title)

	findField := tview.NewInputField().
		SetLabel("Find what     ").
		SetText(initial.Query).
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
		SetChecked(initial.CaseSensitive)
	form.AddFormItem(caseCheck)
	wholeCheck := tview.NewCheckbox().
		SetLabel("Whole word    ").
		SetChecked(initial.WholeWord)
	form.AddFormItem(wholeCheck)

	options := func() SearchOptions {
		result := SearchOptions{
			Query:         findField.GetText(),
			CaseSensitive: caseCheck.IsChecked(),
			WholeWord:     wholeCheck.IsChecked(),
		}
		if replaceField != nil {
			result.Replace = replaceField.GetText()
		}
		return result
	}
	form.AddButton("Find Next", func() { handlers.Find(options(), false) })
	form.AddButton("Find Previous", func() { handlers.Find(options(), true) })
	if withReplace {
		form.AddButton("Replace All", func() { handlers.ReplaceAll(options()) })
	}
	form.AddButton("Clear", func() {
		findField.SetText("")
		if replaceField != nil {
			replaceField.SetText("")
		}
		caseCheck.SetChecked(false)
		wholeCheck.SetChecked(false)
		handlers.Clear()
	})
	form.AddButton("Close", handlers.Close)
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			handlers.Close()
			return nil
		}
		if event.Key() == tcell.KeyEnter {
			handlers.Find(options(), false)
			return nil
		}
		return event
	})
	return form
}

func NewOutlineList(entries []document.OutlineEntry, onSelect func(document.OutlineEntry), onBack func()) *tview.List {
	list := tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true)
	if len(entries) == 0 {
		list.AddItem("No headings yet", "Apply Heading 1–3 to paragraphs", 0, nil)
	}
	for _, entry := range entries {
		entry := entry
		label := strings.Repeat("  ", entry.Level-1) + entry.Text
		secondary := fmt.Sprintf("Heading %d · line %d", entry.Level, entry.Paragraph+1)
		list.AddItem(label, secondary, 0, func() { onSelect(entry) })
	}
	list.AddItem("Back", "Return to Format Text", 'q', onBack)
	uitheme.StyleList(list, "Document Outline")
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			onBack()
			return nil
		}
		return event
	})
	return list
}
