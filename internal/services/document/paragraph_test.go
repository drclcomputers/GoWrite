package document

import (
	"strings"
	"testing"
)

func TestParagraphStylesBuildOutline(t *testing.T) {
	d := newTestDoc(30, "Document", "Introduction", "Details", "Body", "   ")
	d.CursorPara = 1
	d.SetParagraphStyle(ParagraphHeading1)
	d.CursorPara = 2
	d.SetParagraphStyle(ParagraphHeading2)
	d.CursorPara = 4
	d.SetParagraphStyle(ParagraphHeading3)

	got := d.Outline()
	want := []OutlineEntry{
		{Paragraph: 1, Level: 1, Text: "Introduction"},
		{Paragraph: 2, Level: 2, Text: "Details"},
	}
	if len(got) != len(want) {
		t.Fatalf("outline has %d entries, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("outline entry %d = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestParagraphStyleRenderingAndHistory(t *testing.T) {
	d := newTestDoc(30, "Heading")
	d.SetParagraphStyle(ParagraphHeading1)
	if !strings.Contains(d.StyledText(), "[#5ec8c8:-:b]") {
		t.Fatal("heading style is not reflected in rendered text")
	}
	d.SetParagraphSpaceAfter(3)

	if !d.Undo() {
		t.Fatal("undo spacing failed")
	}
	if d.CurrentParagraphSpaceAfter() != 0 || d.CurrentParagraphStyle() != ParagraphHeading1 {
		t.Fatal("undo spacing did not preserve the paragraph style")
	}
	if !d.Undo() {
		t.Fatal("undo style failed")
	}
	if d.CurrentParagraphStyle() != ParagraphNormal {
		t.Fatal("undo style did not restore Normal")
	}
}

func TestParagraphSpaceAfterAddsRenderedAndCursorRows(t *testing.T) {
	d := newTestDoc(30, "first", "second")
	d.CursorPara = 1
	before := d.CursorRow()

	d.CursorPara = 0
	d.SetParagraphSpaceAfter(2)
	d.CursorPara = 1
	after := d.CursorRow()
	if after-before != 2 {
		t.Fatalf("second paragraph moved by %d rows, want 2", after-before)
	}

	lines := plainLines(d)
	first, second := -1, -1
	for i, line := range lines {
		switch {
		case strings.Contains(line, "first"):
			first = i
		case strings.Contains(line, "second"):
			second = i
		}
	}
	if first < 0 || second-first != 3 {
		t.Fatalf("rendered paragraph rows are first=%d, second=%d; want three rows apart", first, second)
	}
}

func TestParagraphSpaceAfterAffectsPagination(t *testing.T) {
	d := pagedDoc(10, testPage())
	d.CursorPara = 9
	if _, total := d.PageInfo(); total != 1 {
		t.Fatalf("pages before spacing = %d, want 1", total)
	}

	d.CursorPara = 8
	d.SetParagraphSpaceAfter(1)
	d.CursorPara = 9
	if current, total := d.PageInfo(); current != 2 || total != 2 {
		t.Fatalf("PageInfo after spacing = %d/%d, want 2/2", current, total)
	}
	lines := plainLines(d)
	row := d.CursorRow()
	if row >= len(lines) || !strings.Contains(lines[row], "p09") {
		t.Fatalf("cursor row %d does not render paragraph p09", row)
	}
}
