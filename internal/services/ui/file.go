// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gowrite/internal/services/fileop"
)

func (e *Editor) saveDocument() {
	filename := e.Doc.GetFilename()
	if strings.EqualFold(filepath.Ext(filename), ".gdoc") {
		if err := e.saveNativeToFile(filename); err != nil {
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
	if err := fileop.SaveGDoc(filename, state, e.showFormattingMarks); err != nil {
		return err
	}
	e.Doc.SetTitle(state.Title)
	e.Doc.SetFilename(filename)
	e.Doc.SetDirty(false)
	return nil
}

func (e *Editor) exportToFile(filename string) error {
	state := e.Doc.Snapshot()
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".txt":
		return fileop.ExportTXT(filename, state)
	case ".rtf":
		return fileop.ExportRTF(filename, state)
	default:
		return fmt.Errorf("unsupported export format %q", filepath.Ext(filename))
	}
}

func (e *Editor) loadFromFile(filename string) error {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".gdoc":
		file, err := fileop.LoadGDoc(filename)
		if err != nil {
			return err
		}
		if err := e.Doc.Restore(file.Document); err != nil {
			return fmt.Errorf("invalid GoWrite document: %w", err)
		}
		e.showFormattingMarks = file.Editor.ShowFormattingMarks
	case ".txt", ".md":
		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		text = strings.ReplaceAll(text, "\r", "\n")
		e.Doc.LoadText(text)
		e.Doc.SetTitle(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
		e.showFormattingMarks = false
	default:
		return fmt.Errorf("unsupported file format %q", filepath.Ext(filename))
	}
	e.Doc.SetFilename(filename)
	return nil
}

func (e *Editor) OpenDocument() {
	e.showFileDialog(fileDialogOpen)
}

func (e *Editor) openDocument() {
	e.showFileDialog(fileDialogOpen)
}
