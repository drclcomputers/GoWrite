// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

import (
	"errors"
	"fmt"
	"os"

	"gowrite/internal/services/document"
	fileui "gowrite/internal/services/ui/file"
)

func (e *Editor) ConfigureRecovery(path string) {
	e.recoveryPath = path
}

func (e *Editor) ReopenRecovery() error {
	if e.recoveryPath == "" {
		return fmt.Errorf("recovery path is not configured")
	}
	recovered, err := fileui.LoadNative(e.recoveryPath)
	if err != nil {
		return fmt.Errorf("load recovery copy: %w", err)
	}
	if err := e.restoreRecovery(recovered.Document, recovered.Editor.ShowFormattingMarks); err != nil {
		return fmt.Errorf("restore recovery copy: %w", err)
	}
	return nil
}

func (e *Editor) DiscardRecovery() error {
	return e.clearRecovery()
}

func (e *Editor) clearRecoveryAfterSave() {
	if err := e.clearRecovery(); err != nil {
		message := fmt.Sprintf("Saved, but could not clear recovery copy: %v", err)
		if e.App != nil {
			e.flashStatus(message)
		}
		fmt.Fprintln(os.Stderr, "GoWrite:", message)
	}
}

func removeRecoveryFile(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove recovery file: %w", err)
	}
	return nil
}

func (e *Editor) restoreRecovery(state document.State, showFormattingMarks bool) error {
	if err := e.Doc.Restore(state); err != nil {
		return err
	}
	e.Doc.SetFilename("")
	e.Doc.SetDirty(true)
	e.showFormattingMarks = showFormattingMarks
	e.App.SetRoot(e.Root, true).SetFocus(e.View)
	e.redraw()
	return nil
}
