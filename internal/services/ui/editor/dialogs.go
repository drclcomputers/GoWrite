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
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) confirmQuit() {
	if !e.Doc.IsDirty() {
		e.App.Stop()
		return
	}
	modal := tview.NewModal().
		SetText("You have unsaved changes.\nWould you like to save before exiting?").
		AddButtons([]string{"Save", "Don't Save", "Cancel"}).
		SetDoneFunc(func(_ int, buttonLabel string) {
			switch buttonLabel {
			case "Save":
				e.saveForExit()
			case "Don't Save":
				e.App.Stop()
			default:
				e.App.SetRoot(e.Root, true).SetFocus(e.View)
			}
		})
	uitheme.StyleModalWarn(modal, "Unsaved Changes")
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		return event
	})
	e.App.SetRoot(modal, true).SetFocus(modal)
}

func (e *Editor) saveForExit() {
	filename := e.Doc.GetFilename()
	ext := filepath.Ext(filename)
	if strings.EqualFold(ext, ".gdoc") || strings.EqualFold(ext, ".docx") ||
		strings.EqualFold(ext, ".txt") || strings.EqualFold(ext, ".rtf") {
		if err := e.saveDocumentToFile(filename); err != nil {
			ShowErrorModal(e.App, e.Root, "Could not save before exiting: "+err.Error())
			return
		}
		e.App.Stop()
		return
	}
	e.showFileDialogWithSaveCallback(fileDialogSave, func() {
		e.App.Stop()
	})
}

func (e *Editor) newDocument() {
	settings, settingsErr := utils.LoadSettings()
	if settingsErr != nil {
		settings = utils.DefaultSettings()
	}
	if e.Doc.IsDirty() {
		modal := tview.NewModal().
			SetText("Current document has unsaved changes.\nDiscard and create a new one?").
			AddButtons([]string{"Discard", "Cancel"}).
			SetDoneFunc(func(buttonIndex int, buttonLabel string) {
				if buttonLabel == "Discard" {
					e.Doc = document.New()
					applyDocumentDefaults(e.Doc, settings)
					e.showFormattingMarks = settings.ShowFormattingMarks
					e.defaultFormattingMarks = settings.ShowFormattingMarks
					e.redraw()
					if settingsErr != nil {
						ShowErrorModal(e.App, e.Root, fmt.Sprintf("Could not load saved options; defaults are being used: %v", settingsErr))
						return
					}
				}
				e.App.SetRoot(e.Root, true).SetFocus(e.View)
			})
		uitheme.StyleModalWarn(modal, "New Document")
		e.App.SetRoot(modal, true).SetFocus(modal)
		return
	}
	e.Doc = document.New()
	applyDocumentDefaults(e.Doc, settings)
	e.showFormattingMarks = settings.ShowFormattingMarks
	e.defaultFormattingMarks = settings.ShowFormattingMarks
	e.redraw()
	if settingsErr != nil {
		ShowErrorModal(e.App, e.Root, fmt.Sprintf("Could not load saved options; defaults are being used: %v", settingsErr))
	}
}

func (e *Editor) NewDocument() {
	e.newDocument()
}

func centredModal(p tview.Primitive, width, height int) tview.Primitive {
	center := tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(p, width, 0, true).
		AddItem(nil, 0, 1, false)
	center.SetBackgroundColor(uitheme.Panel)
	modal := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(center, height, 0, true).
		AddItem(nil, 0, 1, false)
	modal.SetBackgroundColor(uitheme.Panel)
	return modal
}
