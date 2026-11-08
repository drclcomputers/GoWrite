package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gowrite/internal/services/document"
)

func TestNativeSaveAndLoadRestoreDocumentAndViewSettings(t *testing.T) {
	doc := document.New()
	doc.LoadText("Heading\nBody")
	doc.SetTitle("My Document")
	doc.CursorPara, doc.CursorOff = 1, 2
	doc.SetParagraphStyle(document.ParagraphHeading2)
	doc.SetParagraphSpaceAfter(2)
	doc.Page.Footer.Center = "{page} of {pages}"
	doc.ActiveStyle = document.StyleBold
	doc.SetDirty(true)

	filename := filepath.Join(t.TempDir(), "sample.gdoc")
	editor := &Editor{Doc: doc, showFormattingMarks: true}
	if err := editor.saveNativeToFile(filename); err != nil {
		t.Fatal(err)
	}
	if doc.IsDirty() {
		t.Fatal("native save did not clear the dirty state")
	}

	loaded := &Editor{Doc: document.New()}
	if err := loaded.loadFromFile(filename); err != nil {
		t.Fatal(err)
	}
	if !loaded.showFormattingMarks {
		t.Fatal("native load did not restore formatting marks")
	}
	if got := loaded.Doc.Snapshot(); got.Title != "My Document" ||
		got.Paragraphs[1].ParaStyle != document.ParagraphHeading2 ||
		got.Paragraphs[1].SpaceAfter != 2 ||
		got.Page.Footer.Center != "{page} of {pages}" ||
		got.ActiveStyle != document.StyleBold {
		t.Fatalf("native load lost document state: %#v", got)
	}
}

func TestExportDoesNotChangeDocumentSaveState(t *testing.T) {
	doc := document.New()
	doc.LoadText("plain\nrich")
	doc.Paragraphs[1].Runs[0].Style = document.StyleItalic
	doc.SetFilename("original.gdoc")
	doc.SetDirty(true)
	editor := &Editor{Doc: doc}

	filename := filepath.Join(t.TempDir(), "export.rtf")
	if err := editor.exportToFile(filename); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), `{\rtf1`) {
		t.Fatal("RTF export has an invalid header")
	}
	if !doc.IsDirty() || doc.GetFilename() != "original.gdoc" {
		t.Fatal("export changed the native document's save state")
	}
}

func TestDOCXSaveAndLoadRestoreDocument(t *testing.T) {
	doc := document.New()
	doc.LoadText("Heading\nBody")
	doc.SetTitle("Word document")
	doc.Paragraphs[0].ParaStyle = document.ParagraphHeading1
	doc.Paragraphs[1].Runs[0].Style = document.StyleItalic
	doc.SetDirty(true)

	filename := filepath.Join(t.TempDir(), "sample.docx")
	editor := &Editor{Doc: doc}
	if err := editor.saveDocumentToFile(filename); err != nil {
		t.Fatal(err)
	}
	if doc.IsDirty() {
		t.Fatal("DOCX save did not clear the dirty state")
	}

	loaded := &Editor{Doc: document.New()}
	if err := loaded.loadFromFile(filename); err != nil {
		t.Fatal(err)
	}
	state := loaded.Doc.Snapshot()
	if state.Title != "Word document" ||
		state.Paragraphs[0].ParaStyle != document.ParagraphHeading1 ||
		state.Paragraphs[1].Runs[0].Style != document.StyleItalic {
		t.Fatalf("DOCX load lost document content: %#v", state)
	}
}
