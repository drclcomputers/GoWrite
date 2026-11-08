// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

import (
	"fmt"
	"path/filepath"
	"strings"

	fileui "gowrite/internal/services/ui/file"

	"github.com/rivo/tview"
)

func (e *Editor) saveDocument() {
	filename := e.Doc.GetFilename()
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".gdoc" || ext == ".docx" || ext == ".txt" || ext == ".rtf" {
		if err := e.saveDocumentToFile(filename); err != nil {
			ShowErrorModal(e.App, e.Root, fmt.Sprintf("Could not save %s: %v", filename, err))
			return
		}
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
		e.redraw()
		return
	}
	e.showFileDialog(fileDialogSave)
}

func (e *Editor) saveAsDocument() {
	e.showFileDialog(fileDialogSave)
}

func (e *Editor) exportDocument() {
	e.showFileDialog(fileDialogExport)
}

func (e *Editor) saveNativeToFile(filename string) error {
	state := e.Doc.Snapshot()
	if state.Title == "" || state.Title == "Untitled" {
		state.Title = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
	}
	if err := fileui.SaveDocument(filename, state, e.showFormattingMarks); err != nil {
		return err
	}
	e.Doc.SetTitle(state.Title)
	e.Doc.SetFilename(filename)
	e.Doc.SetDirty(false)
	return nil
}

func (e *Editor) saveDocumentToFile(filename string) error {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".gdoc":
		return e.saveNativeToFile(filename)
	case ".docx":
		state := e.Doc.Snapshot()
		if state.Title == "" || state.Title == "Untitled" {
			state.Title = strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
		}
		if err := fileui.SaveDocument(filename, state, e.showFormattingMarks); err != nil {
			return err
		}
		e.Doc.SetTitle(state.Title)
		e.Doc.SetFilename(filename)
		e.Doc.SetDirty(false)
		return nil
	case ".txt", ".rtf":
		if err := fileui.SaveDocument(filename, e.Doc.Snapshot(), e.showFormattingMarks); err != nil {
			return err
		}
		e.Doc.SetFilename(filename)
		e.Doc.SetDirty(false)
		return nil
	default:
		return fmt.Errorf("unsupported save format %q", filepath.Ext(filename))
	}
}

func (e *Editor) exportToFile(filename string) error {
	return fileui.ExportDocument(filename, e.Doc.Snapshot())
}

func (e *Editor) loadFromFile(filename string) error {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".gdoc":
		file, err := fileui.LoadNative(filename)
		if err != nil {
			return err
		}
		if err := e.Doc.Restore(file.Document); err != nil {
			return fmt.Errorf("invalid GoWrite document: %w", err)
		}
		e.showFormattingMarks = file.Editor.ShowFormattingMarks
	case ".docx":
		state, err := fileui.LoadWord(filename)
		if err != nil {
			return err
		}
		if err := e.Doc.Restore(state); err != nil {
			return fmt.Errorf("invalid DOCX document: %w", err)
		}
		if state.Title == "" {
			e.Doc.SetTitle(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
		}
		e.showFormattingMarks = false
	case ".txt", ".md":
		text, err := fileui.ReadText(filename)
		if err != nil {
			return err
		}
		e.Doc.LoadText(text)
		e.Doc.SetTitle(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
		e.showFormattingMarks = false
	default:
		return fmt.Errorf("unsupported file format %q", filepath.Ext(filename))
	}
	e.Doc.SetFilename(filename)
	return nil
}

func (e *Editor) OpenDocument(returnTo, returnFocus tview.Primitive) {
	e.showFileDialogWithReturn(fileDialogOpen, nil, returnTo, returnFocus)
}

func (e *Editor) openDocument() {
	e.showFileDialog(fileDialogOpen)
}
