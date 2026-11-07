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
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// Editor is the main writing surface.
type Editor struct {
	App      *tview.Application
	Doc      *document.Document
	Root     *tview.Flex
	TextArea *tview.TextArea
	Status   *tview.TextView
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

	// Main text area
	e.TextArea = tview.NewTextArea().
		SetText(doc.Text(), false).
		SetWrap(true).
		SetWordWrap(true)
	e.TextArea.SetBackgroundColor(uitheme.Ink)
	e.TextArea.SetBorder(true)
	e.TextArea.SetBorderColor(uitheme.Line)
	e.TextArea.SetTitle(" Document ")
	e.TextArea.SetTitleColor(uitheme.Amber)
	e.TextArea.SetTitleAlign(tview.AlignLeft)

	// Status bar
	e.Status = tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft)
	e.Status.SetBackgroundColor(uitheme.Panel)
	e.refreshStatus()

	// Layout: title | text | status
	e.Root = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(e.TitleBar, 1, 0, false).
		AddItem(e.TextArea, 0, 1, true).
		AddItem(e.Status, 1, 0, false)

	e.Root.SetBackgroundColor(uitheme.Ink)

	// Keep document model in sync when the user types
	e.TextArea.SetChangedFunc(func() {
		// Pull the current text back into the model
		text := e.TextArea.GetText()
		e.Doc.SetText(text)
		e.refreshStatus()
		e.refreshTitle()
	})

	// Global key bindings for the editor
	e.Root.SetInputCapture(e.handleKeys)

	return e
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
	e.Status.SetText(fmt.Sprintf(
		" [teal]Words:[::-] %d   [teal]Chars:[::-] %d   [teal]Paragraphs:[::-] %d   [gray]|  Alt+S save  ·  Alt+O open  ·  Alt+N new  ·  ? help  ·  Esc quit[::-]",
		words, chars, paras))
}

func (e *Editor) handleKeys(event *tcell.EventKey) *tcell.EventKey {
	// Quit
	if event.Key() == tcell.KeyEscape ||
		(event.Key() == tcell.KeyCtrlC) {
		e.confirmQuit()
		return nil
	}

	// Help
	if event.Rune() == '?' || (event.Rune() == '/' && event.Modifiers()&tcell.ModAlt != 0) {
		e.showHelp()
		return nil
	}

	// Alt shortcuts
	if event.Modifiers()&tcell.ModAlt != 0 {
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
		}
	}

	return event
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
				e.App.SetRoot(e.Root, true).SetFocus(e.TextArea)
			}
		})
	uitheme.StyleModalWarn(modal, "Confirm")
	e.App.SetRoot(modal, true).SetFocus(modal)
}

func (e *Editor) showHelp() {
	helpText := `[::b]GoWrite – Keyboard Shortcuts[::-]

[yellow]Navigation[::-]
  Arrow keys / Home / End / PageUp / PageDown

[yellow]Editing[::-]
  Type normally. Enter creates a new paragraph.
  Backspace / Delete work as expected.

[yellow]File[::-]
  Alt+N          New document
  Alt+O          Open document
  Alt+S          Save document

[yellow]Other[::-]
  ?  or  Alt+/   This help
  Esc            Quit (asks if dirty)

More features (styles, find/replace, markdown export, etc.)
will land in later versions.`

	modal := tview.NewModal().
		SetText(helpText).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			e.App.SetRoot(e.Root, true).SetFocus(e.TextArea)
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
					e.TextArea.SetText("", false)
					e.refreshTitle()
					e.refreshStatus()
				}
				e.App.SetRoot(e.Root, true).SetFocus(e.TextArea)
			})
		uitheme.StyleModalWarn(modal, "New Document")
		e.App.SetRoot(modal, true).SetFocus(modal)
		return
	}
	e.Doc = document.New()
	e.TextArea.SetText("", false)
	e.refreshTitle()
	e.refreshStatus()
}

// Placeholder save / open – real file dialogs come next.
func (e *Editor) saveDocument() {
	// For the base we just mark clean and show a toast-like modal.
	e.Doc.Dirty = false
	e.refreshTitle()

	modal := tview.NewModal().
		SetText("Save is not fully wired yet.\n(Document marked clean for now)").
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			e.App.SetRoot(e.Root, true).SetFocus(e.TextArea)
		})
	uitheme.StyleModalOK(modal, "Save")
	e.App.SetRoot(modal, true).SetFocus(modal)
}

func (e *Editor) openDocument() {
	modal := tview.NewModal().
		SetText("Open dialog will appear here\nin the next iteration.").
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			e.App.SetRoot(e.Root, true).SetFocus(e.TextArea)
		})
	uitheme.StyleModal(modal, "Open")
	e.App.SetRoot(modal, true).SetFocus(modal)
}

// Mount sets the editor as the application root and focuses the text area.
func (e *Editor) Mount() {
	e.App.SetRoot(e.Root, true).SetFocus(e.TextArea)
}

// Simple helper used by start menu etc.
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

// truncate for status / title if needed later
func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

// Ensure the package compiles even if some helpers are temporarily unused.
var _ = strings.TrimSpace
var _ = truncate
