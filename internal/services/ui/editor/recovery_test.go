// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"gowrite/internal/services/document"
	fileui "gowrite/internal/services/ui/file"
)

func TestRecoverySnapshotRoundTripAndRemovalWhenClean(t *testing.T) {
	doc := document.New()
	doc.LoadText("Unsaved work")
	doc.SetTitle("Recovery test")
	doc.SetDirty(true)
	path := filepath.Join(t.TempDir(), "~temp~.godoc")
	editor := &Editor{Doc: doc}

	if err := editor.writeRecovery(path, doc, doc.Snapshot(), true, editor.recoveryGeneration.Load()); err != nil {
		t.Fatal(err)
	}
	recovered, err := fileui.LoadNative(path)
	if err != nil {
		t.Fatal(err)
	}
	if recovered.Document.Title != "Recovery test" ||
		recovered.Document.Paragraphs[0].Runs[0].Text != "Unsaved work" ||
		!recovered.Editor.ShowFormattingMarks {
		t.Fatalf("recovery copy lost document state: %#v", recovered)
	}

	doc.SetDirty(false)
	if err := editor.writeRecovery(path, doc, doc.Snapshot(), true, editor.recoveryGeneration.Load()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("recovery file still exists after document became clean: %v", err)
	}
}

func TestRecoverySnapshotFromDiscardedDocumentIsIgnored(t *testing.T) {
	doc := document.New()
	doc.LoadText("Old work")
	doc.SetDirty(true)
	path := filepath.Join(t.TempDir(), "~temp~.godoc")
	editor := &Editor{Doc: doc, recoveryPath: path}
	state := doc.Snapshot()
	generation := editor.recoveryGeneration.Load()

	if err := editor.clearRecovery(); err != nil {
		t.Fatal(err)
	}
	doc.LoadText("New work")
	doc.SetDirty(true)
	if err := editor.writeRecovery(path, doc, state, false, generation); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stale recovery snapshot was written after discard: %v", err)
	}
}
