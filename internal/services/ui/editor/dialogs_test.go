package editor

import (
	"path/filepath"
	"testing"

	"gowrite/internal/services/document"
)

func TestSaveBeforeActionSavesThenContinues(t *testing.T) {
	doc := document.New()
	doc.LoadText("unsaved content")
	doc.SetDirty(true)
	filename := filepath.Join(t.TempDir(), "document.gdoc")
	doc.SetFilename(filename)
	editor := &Editor{Doc: doc}
	actionCalled := false

	editor.saveBeforeAction(func() {
		actionCalled = true
	}, nil, nil)

	if !actionCalled {
		t.Fatal("continuation did not run after saving")
	}
	if doc.IsDirty() {
		t.Fatal("document remained dirty after saving")
	}
}
