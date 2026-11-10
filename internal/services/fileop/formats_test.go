package fileop

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
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

func TestTableExportsAsStructuredData(t *testing.T) {
	doc := document.New()
	if err := doc.InsertTable(2, 2); err != nil {
		t.Fatal(err)
	}
	paragraph, _ := doc.TableParagraph()
	table, _ := doc.TableAt(paragraph)
	table.Rows[0] = []string{"left", "right"}
	table.Rows[1] = []string{"second row", "last"}
	table.EnsureCellFormats()
	table.CellFormats[0][0] = document.TableCellFormat{
		Style: document.StyleBold, Foreground: "#123456", Background: "#abcdef",
		ParagraphStyle: document.ParagraphHeading1, Alignment: document.AlignCenter,
	}
	if err := doc.UpdateTableAt(paragraph, table); err != nil {
		t.Fatal(err)
	}
	state := doc.Snapshot()
	if got, want := doc.Text(), "left\tright\nsecond row\tlast"; got != want {
		t.Fatalf("plain document text = %q, want %q", got, want)
	}

	txtPath := filepath.Join(t.TempDir(), "table.txt")
	if err := ExportTXT(txtPath, state); err != nil {
		t.Fatal(err)
	}
	text, err := os.ReadFile(txtPath)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(text), "left\tright\nsecond row\tlast"; got != want {
		t.Fatalf("TXT table = %q, want %q", got, want)
	}
	rtf := encodeRTF(state)
	if !strings.Contains(rtf, `\trowd`) || !strings.Contains(rtf, `\cell`) ||
		!strings.Contains(rtf, `\outlinelevel0`) || !strings.Contains(rtf, `\cf`) {
		t.Fatalf("RTF output did not contain table rows: %s", rtf)
	}
	gdocPath := filepath.Join(t.TempDir(), "table.gdoc")
	if err := SaveGDoc(gdocPath, state, false); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadGDoc(gdocPath)
	if err != nil {
		t.Fatal(err)
	}
	restored := document.New()
	if err := restored.Restore(loaded.Document); err != nil {
		t.Fatal(err)
	}
	if got := restored.Snapshot(); !reflect.DeepEqual(got, state) {
		t.Fatalf("native table round trip differs:\ngot:  %#v\nwant: %#v", got, state)
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

func TestDOCXRoundTripPreservesSupportedDocumentContent(t *testing.T) {
	doc := document.New()
	doc.SetTitle("DOCX round trip")
	doc.LoadText("Heading\nBody")
	doc.Paragraphs[0].ParaStyle = document.ParagraphHeading1
	doc.Paragraphs[0].Runs[0].Style = document.StyleBold | document.StyleUnderline
	doc.Paragraphs[0].Runs[0].Fg = "#123456"
	doc.Paragraphs[1].Align = document.AlignRight
	doc.Paragraphs[1].SpaceAfter = 2
	doc.Paragraphs[1].PageBreak = true
	doc.Paragraphs[1].Runs[0].Text = "Body\twith\nline"
	doc.Paragraphs[1].Runs[0].Bg = "#abcdef"
	page := doc.GetPageSettings()
	page.Size, page.Orientation = document.SizeLegal, document.Landscape
	page.Background = "#4a5b6c"
	page.Width, page.Height = document.GetDimensions(page.Size, page.Orientation)
	page.Footer.Center = "{title}"
	page.PageNumbers = document.PageNumbers{
		Where: document.NumFooter, Align: document.AlignCenter, Format: document.FormatPageOf,
	}
	doc.SetPageSettings(page)

	filename := filepath.Join(t.TempDir(), "sample.docx")
	if err := SaveDOCX(filename, doc.Snapshot()); err != nil {
		t.Fatal(err)
	}
	state, err := LoadDOCX(filename)
	if err != nil {
		t.Fatal(err)
	}
	if state.Title != "DOCX round trip" ||
		state.Paragraphs[0].ParaStyle != document.ParagraphHeading1 ||
		state.Paragraphs[0].Runs[0].Style != document.StyleBold|document.StyleUnderline ||
		state.Paragraphs[0].Runs[0].Fg != "#123456" ||
		state.Paragraphs[1].Align != document.AlignRight ||
		state.Paragraphs[1].SpaceAfter != 2 ||
		!state.Paragraphs[1].PageBreak ||
		state.Paragraphs[1].Runs[0].Text != "Body\twith\nline" ||
		state.Paragraphs[1].Runs[0].Bg != "#abcdef" {
		t.Fatalf("DOCX round trip lost paragraph formatting: %#v", state)
	}
	if state.Page.Size != document.SizeLegal || state.Page.Orientation != document.Landscape ||
		state.Page.Background != "#4a5b6c" ||
		state.Page.Footer.Center != "DOCX round trip" ||
		state.Page.PageNumbers.Where != document.NumFooter ||
		state.Page.PageNumbers.Align != document.AlignCenter ||
		state.Page.PageNumbers.Format != document.FormatPageOf {
		t.Fatalf("DOCX round trip lost page settings: %#v", state.Page)
	}
}

func TestDOCXImportPreservesTableCells(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "table.docx")
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	part, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, err = io.WriteString(part, `<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body><w:tbl><w:tr><w:tc><w:p><w:r><w:t>left</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>right</w:t></w:r></w:p></w:tc></w:tr></w:tbl><w:tbl><w:tr><w:tc><w:p><w:r><w:t>another</w:t></w:r></w:p></w:tc></w:tr></w:tbl></w:body></w:document>`)
	if err != nil {
		t.Fatal(err)
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	state, err := LoadDOCX(filename)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Paragraphs) != 2 || state.Paragraphs[0].Table == nil || state.Paragraphs[1].Table == nil {
		t.Fatalf("DOCX table was not preserved as a table: %#v", state.Paragraphs)
	}
	if got := state.Paragraphs[0].Table.Rows; !reflect.DeepEqual(got, [][]string{{"left", "right"}}) {
		t.Fatalf("imported table cells = %#v, want [[left right]]", got)
	}
	if got := state.Paragraphs[1].Table.Rows; !reflect.DeepEqual(got, [][]string{{"another"}}) {
		t.Fatalf("second imported table cells = %#v, want [[another]]", got)
	}
	restored := document.New()
	if err := restored.Restore(state); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(restored.Text(), "left") || !strings.Contains(restored.Text(), "right") {
		t.Fatalf("restored table was not rendered: %q", restored.Text())
	}
}

func TestDOCXExportWritesStructuredTable(t *testing.T) {
	doc := document.New()
	if err := doc.InsertTable(1, 2); err != nil {
		t.Fatal(err)
	}
	paragraph, _ := doc.TableParagraph()
	table, _ := doc.TableAt(paragraph)
	table.Rows[0][0], table.Rows[0][1] = "left", "right"
	table.EnsureCellFormats()
	table.CellFormats[0][0] = document.TableCellFormat{
		Style: document.StyleBold, Foreground: "#123456", Background: "#abcdef",
		ParagraphStyle: document.ParagraphHeading2, Alignment: document.AlignRight,
	}
	if err := doc.UpdateTableAt(paragraph, table); err != nil {
		t.Fatal(err)
	}
	filename := filepath.Join(t.TempDir(), "table.docx")
	if err := SaveDOCX(filename, doc.Snapshot()); err != nil {
		t.Fatal(err)
	}
	archive, err := zip.OpenReader(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	for _, part := range archive.File {
		if part.Name != "word/document.xml" {
			continue
		}
		reader, err := part.Open()
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(reader)
		reader.Close()
		if err != nil {
			t.Fatal(err)
		}
		xml := string(data)
		if !strings.Contains(xml, "<w:tbl>") || !strings.Contains(xml, "<w:tc>") ||
			!strings.Contains(xml, "left") || !strings.Contains(xml, "right") ||
			!strings.Contains(xml, `<w:pStyle w:val="Heading2"/>`) ||
			!strings.Contains(xml, `<w:color w:val="123456"/>`) ||
			!strings.Contains(xml, `<w:tblLayout w:type="fixed"/>`) ||
			!strings.Contains(xml, `<w:shd w:val="clear" w:color="auto" w:fill="ABCDEF"/>`) {
			t.Fatalf("DOCX table structure missing from document.xml: %s", xml)
		}
		tcProps := strings.Index(xml, "<w:tcPr>")
		cellShading := strings.Index(xml, `<w:shd w:val="clear" w:color="auto" w:fill="ABCDEF"/>`)
		cellParagraph := strings.Index(xml, "<w:p>")
		if tcProps < 0 || cellShading < tcProps || cellShading > cellParagraph {
			t.Fatalf("table background is not encoded as cell shading: %s", xml)
		}
		loaded, err := LoadDOCX(filename)
		if err != nil {
			t.Fatal(err)
		}
		format := loaded.Paragraphs[0].Table.CellFormats[0][0]
		if format.Style&document.StyleBold == 0 || format.Foreground != "#123456" ||
			format.Background != "#abcdef" ||
			format.ParagraphStyle != document.ParagraphHeading2 || format.Alignment != document.AlignRight {
			t.Fatalf("DOCX table formatting not restored: %#v", format)
		}
		return
	}
	t.Fatal("DOCX has no word/document.xml")
}
