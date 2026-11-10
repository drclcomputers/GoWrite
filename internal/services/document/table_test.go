package document

import (
	"strings"
	"testing"
)

func TestInsertTableCreatesStructuredGrid(t *testing.T) {
	d := New()
	if err := d.InsertTable(25, 12); err != nil {
		t.Fatal(err)
	}

	lines := make([]string, 0, len(d.Paragraphs))
	for _, paragraph := range d.Paragraphs {
		if paragraph.Table != nil || paragraph.TablePart {
			lines = append(lines, plainOf(paragraph))
		}
	}
	if len(lines) != 51 {
		t.Fatalf("table has %d rendered lines, want 51", len(lines))
	}
	paragraph, ok := d.TableParagraph()
	if !ok {
		t.Fatal("cursor is not in the inserted table")
	}
	table, ok := d.TableAt(paragraph)
	if !ok || len(table.Rows) != 25 || len(table.Rows[0]) != 12 {
		t.Fatalf("stored table dimensions = %dx%d, want 25x12", len(table.Rows), len(table.Rows[0]))
	}
	for i, line := range lines {
		if got := widthOf([]rune(line)); got > d.wrapWidth() {
			t.Errorf("table line %d is %d cells wide, page text width is %d", i, got, d.wrapWidth())
		}
	}
	if !strings.HasPrefix(lines[0], "┌") || !strings.HasPrefix(lines[1], "│") ||
		!strings.HasPrefix(lines[2], "├") || !strings.HasPrefix(lines[len(lines)-1], "└") {
		t.Fatalf("unexpected table rendering: %q", lines)
	}

	if d.CursorPara != 1 || d.CursorOff != 2 {
		t.Fatalf("cursor = (%d,%d), want first cell at (1,2)", d.CursorPara, d.CursorOff)
	}
	d.InsertRune('x')
	if got, _ := d.TableAt(paragraph); got.Rows[0][0] != "" {
		t.Fatal("ordinary text editing changed the table rendering without updating its cells")
	}
	table.Rows[0][0] = "editable cell"
	if err := d.UpdateTableAt(paragraph, table); err != nil {
		t.Fatal(err)
	}
	if got, _ := d.TableAt(paragraph); got.Rows[0][0] != "editable cell" {
		t.Fatalf("table cell = %q, want edited cell content", got.Rows[0][0])
	}
}

func TestInsertTableAtCursorAndUndo(t *testing.T) {
	d := New()
	d.LoadText("alpha omega")
	d.CursorOff = 5

	if err := d.InsertTable(1, 2); err != nil {
		t.Fatal(err)
	}
	if d.Paragraphs[0].Runs[0].Text != "alpha" || d.Paragraphs[len(d.Paragraphs)-1].Runs[0].Text != " omega" {
		t.Fatalf("table insertion did not preserve surrounding text: %q", d.Text())
	}
	if !d.Undo() || d.Text() != "alpha omega" {
		t.Fatalf("undo did not remove the table: %q", d.Text())
	}
}

func TestInsertTableRejectsInvalidDimensions(t *testing.T) {
	d := New()
	if err := d.InsertTable(0, 2); err == nil {
		t.Fatal("InsertTable accepted zero rows")
	}
	if err := d.InsertTable(2, maxTableColumns+1); err == nil {
		t.Fatal("InsertTable accepted too many columns")
	}
	if d.Text() != "" {
		t.Fatalf("invalid insertion changed document text: %q", d.Text())
	}
}

func TestInsertedTableDoesNotInheritParagraphSpacing(t *testing.T) {
	d := New()
	d.SetParagraphSpaceAfter(3)
	if err := d.InsertTable(2, 2); err != nil {
		t.Fatal(err)
	}
	for i, paragraph := range d.Paragraphs {
		if paragraph.SpaceAfter != 0 || paragraph.ParaStyle != ParagraphNormal {
			t.Errorf("table paragraph %d inherited style or spacing: %#v", i, paragraph)
		}
	}
}

func TestUpdateTableUndoAndRedo(t *testing.T) {
	d := New()
	if err := d.InsertTable(2, 2); err != nil {
		t.Fatal(err)
	}
	paragraph, _ := d.TableParagraph()
	table, _ := d.TableAt(paragraph)
	table.Rows[0][0] = "value"
	if err := d.UpdateTableAt(paragraph, table); err != nil {
		t.Fatal(err)
	}
	if !d.Undo() {
		t.Fatal("undo did not revert table edit")
	}
	paragraph, _ = d.TableParagraph()
	table, _ = d.TableAt(paragraph)
	if table.Rows[0][0] != "" {
		t.Fatalf("undo left table cell %q, want empty", table.Rows[0][0])
	}
	if !d.Redo() {
		t.Fatal("redo did not restore table edit")
	}
	paragraph, _ = d.TableParagraph()
	table, _ = d.TableAt(paragraph)
	if table.Rows[0][0] != "value" {
		t.Fatalf("redo restored table cell %q, want value", table.Rows[0][0])
	}
}

func TestDeleteTableAtRemovesBlockAndSupportsUndoRedo(t *testing.T) {
	d := New()
	d.LoadText("alpha omega")
	d.CursorOff = 5
	if err := d.InsertTable(2, 2); err != nil {
		t.Fatal(err)
	}
	paragraph, ok := d.TableParagraph()
	if !ok {
		t.Fatal("cursor is not in the inserted table")
	}

	if err := d.DeleteTableAt(paragraph); err != nil {
		t.Fatal(err)
	}
	if _, ok := d.TableParagraph(); ok {
		t.Fatal("table remained after deletion")
	}
	if len(d.Paragraphs) != 2 || plainOf(d.Paragraphs[0]) != "alpha" || plainOf(d.Paragraphs[1]) != " omega" {
		t.Fatalf("surrounding paragraphs were not preserved: %#v", d.Paragraphs)
	}
	if !d.Undo() {
		t.Fatal("undo did not restore deleted table")
	}
	if _, ok := d.TableParagraph(); !ok {
		t.Fatal("undo did not restore table")
	}
	if !d.Redo() {
		t.Fatal("redo did not delete table")
	}
	if _, ok := d.TableParagraph(); ok {
		t.Fatal("redo left table in document")
	}
}

func TestDeleteOnlyTableLeavesEmptyParagraph(t *testing.T) {
	d := New()
	if err := d.InsertTable(1, 1); err != nil {
		t.Fatal(err)
	}
	paragraph, ok := d.TableParagraph()
	if !ok {
		t.Fatal("cursor is not in the inserted table")
	}
	if err := d.DeleteTableAt(paragraph); err != nil {
		t.Fatal(err)
	}
	if len(d.Paragraphs) != 1 || plainOf(d.Paragraphs[0]) != "" {
		t.Fatalf("deleting the only table left paragraphs %#v, want one empty paragraph", d.Paragraphs)
	}
}

func TestTableWrapsCellTextAndReflowsWithPageWidth(t *testing.T) {
	d := New()
	if err := d.InsertTable(1, 2); err != nil {
		t.Fatal(err)
	}
	paragraph, _ := d.TableParagraph()
	table, _ := d.TableAt(paragraph)
	table.Rows[0][0] = strings.Repeat("long ", 20)
	if err := d.UpdateTableAt(paragraph, table); err != nil {
		t.Fatal(err)
	}
	oldLines := len(d.Paragraphs)
	page := d.GetPageSettings()
	page.MarginLeft += 30
	d.SetPageSettings(page)
	if len(d.Paragraphs) <= oldLines {
		t.Fatalf("narrower table did not wrap cell text: lines before=%d after=%d", oldLines, len(d.Paragraphs))
	}
	for i, paragraph := range d.Paragraphs {
		if got := widthOf([]rune(plainOf(paragraph))); got > d.wrapWidth() {
			t.Errorf("rendered table line %d is %d cells wide, page width is %d", i, got, d.wrapWidth())
		}
	}
}

func TestEnterAtTableEndMovesToFollowingParagraph(t *testing.T) {
	d := New()
	if err := d.InsertTable(1, 2); err != nil {
		t.Fatal(err)
	}
	d.CursorPara = len(d.Paragraphs) - 1
	d.CursorOff = len([]rune(plainOf(d.Paragraphs[d.CursorPara])))
	if !d.MoveAfterTableAtCursor() {
		t.Fatal("could not move after table at document end")
	}
	if _, ok := d.TableParagraph(); ok {
		t.Fatal("cursor remained inside the table")
	}
	d.InsertText("after")
	if got, want := d.Text(), "\t\nafter"; got != want {
		t.Fatalf("text after table = %q, want %q", got, want)
	}
}

func TestTableCellFormattingIsRendered(t *testing.T) {
	d := New()
	if err := d.InsertTable(1, 1); err != nil {
		t.Fatal(err)
	}
	paragraph, _ := d.TableParagraph()
	table, _ := d.TableAt(paragraph)
	table.Rows[0][0] = "styled"
	table.EnsureCellFormats()
	table.CellFormats[0][0] = TableCellFormat{
		Style: StyleBold, Foreground: "#123456", Background: "#abcdef",
		ParagraphStyle: ParagraphHeading1, Alignment: AlignCenter,
	}
	if err := d.UpdateTableAt(paragraph, table); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, run := range d.Paragraphs[1].Runs {
		if run.Text == "styled" {
			found = run.Style&StyleBold != 0 && run.Fg == "#123456" && run.Bg == "#abcdef"
		}
	}
	if !found {
		t.Fatalf("cell format was not reflected in rendered runs: %#v", d.Paragraphs[1].Runs)
	}
}
