package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gowrite/internal/services/document"
	fileui "gowrite/internal/services/ui/file"
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

func TestNativeLoadPreservesPageSizeAndMarginsOnNarrowTerminal(t *testing.T) {
	doc := document.New()
	page := doc.GetPageSettings()
	page.Size = document.SizeLegal
	page.Orientation = document.Landscape
	page.Width, page.Height = document.GetDimensions(page.Size, page.Orientation)
	page.MarginLeft, page.MarginRight = 11, 8
	page.MarginTop, page.MarginBottom = 4, 5
	doc.SetPageSettings(page)

	filename := filepath.Join(t.TempDir(), "wide.gdoc")
	if err := fileui.SaveDocument(filename, doc.Snapshot(), false); err != nil {
		t.Fatal(err)
	}

	target := document.New()
	target.SetDisplayWidth(20)
	editor := &Editor{Doc: target}
	if err := editor.loadFromFile(filename); err != nil {
		t.Fatal(err)
	}
	_, _ = target.PageInfo()
	if got := target.GetPageSettings(); got != page {
		t.Fatalf("loaded page settings = %+v, want %+v", got, page)
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

func TestRTFLoadRestoresDocumentAndFilename(t *testing.T) {
	source := document.New()
	source.LoadText("RTF text\nsecond paragraph")
	source.Paragraphs[1].Runs[0].Style = document.StyleItalic
	filename := filepath.Join(t.TempDir(), "imported.rtf")
	if err := fileui.ExportDocument(filename, source.Snapshot()); err != nil {
		t.Fatal(err)
	}

	editor := &Editor{Doc: document.New()}
	if err := editor.loadFromFile(filename); err != nil {
		t.Fatal(err)
	}
	if got, want := editor.Doc.Text(), source.Text(); got != want {
		t.Fatalf("loaded RTF text = %q, want %q", got, want)
	}
	if editor.Doc.GetTitle() != "imported" || editor.Doc.GetFilename() != filename {
		t.Fatalf("RTF metadata was not set: title=%q filename=%q",
			editor.Doc.GetTitle(), editor.Doc.GetFilename())
	}
	if editor.Doc.Paragraphs[1].Runs[0].Style&document.StyleItalic == 0 {
		t.Fatal("RTF import lost text formatting")
	}
}

func TestSaveAsTextUpdatesActiveFilenameAndCleanState(t *testing.T) {
	doc := document.New()
	doc.LoadText("plain text")
	doc.SetDirty(true)
	editor := &Editor{Doc: doc}
	filename := filepath.Join(t.TempDir(), "sample.txt")

	if err := editor.saveDocumentToFile(filename); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "plain text" || doc.GetFilename() != filename || doc.IsDirty() {
		t.Fatalf("TXT save did not update the active save state: text=%q filename=%q dirty=%v",
			string(data), doc.GetFilename(), doc.IsDirty())
	}
}
