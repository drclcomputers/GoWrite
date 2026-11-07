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
)

func (e *Editor) saveDocument() {
	filename := e.Doc.GetFilename()
	if filename == "" {
		e.showFileDialog(true)
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
	e.App.SetRoot(e.Root, true).SetFocus(e.View)
	e.redraw()
}

func (e *Editor) loadFromFile(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		ShowErrorModal(e.App, e.Root, fmt.Sprintf("Could not open %s: %v", filename, err))
		return
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	e.Doc.LoadText(text)
	e.Doc.SetFilename(filename)
	e.Doc.SetTitle(filepath.Base(filename))
	e.App.SetRoot(e.Root, true).SetFocus(e.View)
	e.redraw()
}

func (e *Editor) OpenDocument() {
	e.showFileDialog(false)
}

func (e *Editor) openDocument() {
	e.showFileDialog(false)
}
