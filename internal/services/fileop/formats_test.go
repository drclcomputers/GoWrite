package fileop

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gowrite/internal/services/document"
)

func TestGDocRoundTripPreservesDocumentAndEditorState(t *testing.T) {
	doc := document.New()
	doc.LoadText("Heading\nBody")
	doc.CursorPara, doc.CursorOff = 1, 2
	doc.SetParagraphStyle(document.ParagraphHeading1)
	doc.SetParagraphSpaceAfter(3)
	doc.Paragraphs[1].PageBreak = true
	doc.Paragraphs[0].Runs[0].Style = document.StyleUnderline
	doc.Paragraphs[0].Runs[0].Fg = "#123456"
	doc.Paragraphs[0].Runs[0].Bg = "#abcdef"
	page := doc.GetPageSettings()
	page.MarginLeft = 9
	page.Footer.Center = "{page} of {pages}"
	page.PageNumbers = document.PageNumbers{
		Where: document.NumFooter, Align: document.AlignCenter, Format: document.FormatPageOf,
	}
	doc.SetPageSettings(page)
	doc.ActiveStyle = document.StyleBold | document.StyleItalic
	doc.ActiveFg = "#fedcba"
	doc.LastSearch, doc.LastSearchCase, doc.LastSearchWhole = "heading", true, true

	filename := filepath.Join(t.TempDir(), "sample.gdoc")
	if err := SaveGDoc(filename, doc.Snapshot(), true); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 2 || data[0] != 0x1f || data[1] != 0x8b {
		t.Fatal("native document is not gzip-compressed")
	}
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	_ = gz.Close()

	loaded, err := LoadGDoc(filename)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Editor.ShowFormattingMarks {
		t.Fatal("formatting-marks setting was not restored")
	}
	restored := document.New()
	if err := restored.Restore(loaded.Document); err != nil {
		t.Fatal(err)
	}
	if got := restored.Snapshot(); !reflect.DeepEqual(got, doc.Snapshot()) {
		t.Fatalf("restored state differs:\ngot:  %#v\nwant: %#v", got, doc.Snapshot())
	}
}

func TestExportTXTAndRTF(t *testing.T) {
	doc := document.New()
	doc.LoadText("plain\nrich {text}")
	doc.Paragraphs[1].Runs[0].Text = "rich {text} 日本"
	doc.Paragraphs[1].Runs[0].Style = document.StyleBold | document.StyleItalic
	doc.Paragraphs[1].Runs[0].Fg = "#ff0000"
	doc.Paragraphs[1].ParaStyle = document.ParagraphHeading1
	page := doc.GetPageSettings()
	page.Size, page.Orientation = document.SizeLegal, document.Landscape
	page.Width, page.Height = document.GetDimensions(page.Size, page.Orientation)
	doc.SetPageSettings(page)
	state := doc.Snapshot()

	txtPath := filepath.Join(t.TempDir(), "document.txt")
	if err := ExportTXT(txtPath, state); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile(txtPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(text), "plain\nrich {text} 日本"; got != want {
		t.Fatalf("TXT = %q, want %q", got, want)
	}

	rtf := encodeRTF(state)
	for _, want := range []string{
		`{\rtf1`, `\paperw20160\paperh12240`, `\b`, `\i`, `\cf1`,
		`\outlinelevel0`, `\{text\}`, `\u26085?`,
	} {
		if !strings.Contains(rtf, want) {
			t.Errorf("RTF output does not contain %q: %s", want, rtf)
		}
	}
}

func TestTXTExportPreservesParagraphSpacingAndPageBreaks(t *testing.T) {
	doc := document.New()
	doc.LoadText("first\nsecond\nthird")
	doc.Paragraphs[0].SpaceAfter = 2
	doc.Paragraphs[2].PageBreak = true

	path := filepath.Join(t.TempDir(), "document.txt")
	if err := ExportTXT(path, doc.Snapshot()); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), "first\n\n\nsecond\fthird"; got != want {
		t.Fatalf("TXT = %q, want %q", got, want)
	}
}
