// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package ui

import (
	"fmt"
	"gowrite/internal/services/uitheme"
	"gowrite/internal/utils"
	"os"
	"path/filepath"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

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
