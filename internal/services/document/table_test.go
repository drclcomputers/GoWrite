package document

import (
	"strings"
	"testing"
)

func TestInsertTableCreatesEditableGrid(t *testing.T) {
	d := New()
	if err := d.InsertTable(2, 3); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(d.Text(), "\n")
	if len(lines) != 5 {
		t.Fatalf("table has %d lines, want 5: %q", len(lines), d.Text())
	}
	if !strings.HasPrefix(lines[0], "┌") || !strings.HasPrefix(lines[1], "│") ||
		!strings.HasPrefix(lines[2], "├") || !strings.HasPrefix(lines[3], "│") ||
		!strings.HasPrefix(lines[4], "└") {
		t.Fatalf("unexpected table structure: %q", lines)
	}
	for i, line := range lines {
		if got := widthOf([]rune(line)); got > d.wrapWidth() {
			t.Errorf("table line %d is %d cells wide, page text width is %d", i, got, d.wrapWidth())
		}
	}

	if d.CursorPara != 1 || d.CursorOff != 2 {
		t.Fatalf("cursor = (%d,%d), want first cell at (1,2)", d.CursorPara, d.CursorOff)
	}
	d.InsertRune('x')
	if !strings.Contains(d.CurrentPlain(), "x") {
		t.Fatal("inserted text was not placed in the first table cell")
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
