// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

import "fmt"

func (e *Editor) copySelection() {
	n := e.Doc.SelectionLen()
	if !e.Doc.Copy() {
		e.flashStatus("Nothing selected")
		return
	}
	e.redraw()
	e.flashStatus(fmt.Sprintf("Copied %d characters", n))
}

func (e *Editor) cutSelection() {
	n := e.Doc.SelectionLen()
	if !e.Doc.Cut() {
		e.flashStatus("Nothing selected")
		return
	}
	e.redraw()
	e.flashStatus(fmt.Sprintf("Cut %d characters", n))
}

func (e *Editor) pasteClipboard() {
	if !e.Doc.Paste() {
		e.flashStatus("Clipboard is empty")
		return
	}
	e.redraw()
}

func (e *Editor) undo() {
	if !e.Doc.Undo() {
		e.flashStatus("Nothing to undo")
		return
	}
	e.redraw()
	e.flashStatus("Undone")
}

func (e *Editor) redo() {
	if !e.Doc.Redo() {
		e.flashStatus("Nothing to redo")
		return
	}
	e.redraw()
	e.flashStatus("Redone")
}
