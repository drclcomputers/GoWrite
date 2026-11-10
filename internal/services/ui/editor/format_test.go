package editor

import (
	"testing"

	"gowrite/internal/services/document"
	"gowrite/internal/utils"

	"github.com/rivo/tview"
)

func TestOpeningFormatMenuDoesNotDirtyDocument(t *testing.T) {
	doc := document.New()
	editor := NewEditor(tview.NewApplication(), doc, utils.DefaultSettings())

	editor.showFormatDialog()

	if doc.IsDirty() || doc.CanUndo() {
		t.Fatal("opening the format menu marked the document changed")
	}
}
