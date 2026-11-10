package document

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mattn/go-runewidth"
)

func testPage() PageSettings {
	return PageSettings{Size: SizeA4, Orientation: Portrait, Width: 30, Height: 12, MarginTop: 1, MarginBottom: 1}
}

func headerFooterPage() PageSettings {
	ps := testPage()
	ps.Header = HeaderFooter{Left: "Left", Center: "Mid", Right: "Right"}
	ps.PageNumbers = PageNumbers{Where: NumFooter, Align: AlignCenter, Format: FormatPageOf}
	return ps
}

// pagedDoc has n one-line paragraphs "p00", "p01", ... on the given page.
func pagedDoc(n int, ps PageSettings) *Document {
	d := New()
	d.DisplayWidth = 0
	d.Page = ps
	d.Paragraphs = nil
	for i := 0; i < n; i++ {
		d.Paragraphs = append(d.Paragraphs, Paragraph{Runs: []Run{{Text: fmt.Sprintf("p%02d", i)}}})
	}
	return d
}

func TestMarginsShrinkTheTextArea(t *testing.T) {
	ps := testPage()
	ps.MarginLeft, ps.MarginRight = 5, 3
	d := pagedDoc(1, ps)

	if got := d.wrapWidth(); got != 22 {
		t.Fatalf("wrapWidth = %d, want 22", got)
	}
	found := false
	for _, line := range plainLines(d) {
		if line != "" && len([]rune(line)) != ps.Width+2 {
			t.Errorf("row is %d cells wide, want %d: %q", len([]rune(line)), ps.Width+2, line)
		}
		if strings.Contains(line, "p00") {
			found = true
			if !strings.HasPrefix(line, "│     p00") {
				t.Errorf("text is not indented by the left margin: %q", line)
			}
		}
	}
	if !found {
		t.Fatal("paragraph not rendered")
	}
}

func TestHeaderFooterAndPageNumbers(t *testing.T) {
	d := pagedDoc(10, headerFooterPage())
	lines := plainLines(d)

	if _, total := d.PageInfo(); total != 2 {
		t.Fatalf("pages = %d, want 2", total)
	}
	for i, line := range lines {
		if line != "" && len([]rune(line)) != 32 {
			t.Errorf("row %d is %d cells wide, want 32: %q", i, len([]rune(line)), line)
		}
	}

	has := func(row int, want string) {
		t.Helper()
		if row >= len(lines) || !strings.Contains(lines[row], want) {
			t.Errorf("row %d should contain %q, got %q", row, want, lines[min(row, len(lines)-1)])
		}
	}
	for _, row := range []int{1, 16} {
		has(row, "Left")
		has(row, "Mid")
		has(row, "Right")
	}
	for i := 0; i < 8; i++ {
		has(3+i, fmt.Sprintf("p%02d", i))
	}
	has(12, "Page 1 of 2")
	has(18, "p08")
	has(27, "Page 2 of 2")

	for row, glyph := range map[int]string{0: "┌", 13: "└", 15: "┌", 28: "└"} {
		if row >= len(lines) || !strings.HasPrefix(lines[row], glyph) {
			t.Errorf("row %d should start with %q", row, glyph)
		}
	}
	if lines[14] != "" {
		t.Errorf("row 14 should be the gap between pages, got %q", lines[14])
	}
}

func TestCursorRowMatchesRender(t *testing.T) {
	for _, ps := range []PageSettings{testPage(), headerFooterPage()} {
		d := pagedDoc(10, ps)
		lines := plainLines(d)
		for i := range d.Paragraphs {
			d.CursorPara, d.CursorOff = i, 0
			row := d.CursorRow()
			if row >= len(lines) || !strings.Contains(lines[row], fmt.Sprintf("p%02d", i)) {
				t.Fatalf("cursor in paragraph %d: row %d is not its line", i, row)
			}
		}
	}
}

func TestPageInfo(t *testing.T) {
	d := pagedDoc(10, headerFooterPage())
	if cur, total := d.PageInfo(); cur != 1 || total != 2 {
		t.Fatalf("PageInfo at start = %d/%d, want 1/2", cur, total)
	}
	d.CursorPara = 9
	if cur, total := d.PageInfo(); cur != 2 || total != 2 {
		t.Fatalf("PageInfo at end = %d/%d, want 2/2", cur, total)
	}

	d = pagedDoc(3, testPage())
	d.Paragraphs[1].PageBreak = true
	d.Paragraphs[2].PageBreak = true
	d.CursorPara = 2
	if cur, total := d.PageInfo(); cur != 3 || total != 3 {
		t.Fatalf("PageInfo with page breaks = %d/%d, want 3/3", cur, total)
	}
}

func TestInsertPageBreakAtCursor(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		para, off  int
		want       []string
		wantCursor int
	}{
		{
			name:       "inside paragraph",
			text:       "beforeafter",
			off:        6,
			want:       []string{"before", "after"},
			wantCursor: 1,
		},
		{
			name:       "at paragraph end",
			text:       "before\nafter",
			off:        6,
			want:       []string{"before", "after"},
			wantCursor: 1,
		},
		{
			name:       "at document end",
			text:       "before",
			off:        6,
			want:       []string{"before", ""},
			wantCursor: 1,
		},
		{
			name:       "empty first paragraph",
			text:       "",
			para:       0,
			off:        0,
			want:       []string{"", ""},
			wantCursor: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := New()
			d.LoadText(tt.text)
			d.Page = testPage()
			d.CursorPara, d.CursorOff = tt.para, tt.off

			d.InsertPageBreak()

			if len(d.Paragraphs) != len(tt.want) {
				t.Fatalf("paragraph count = %d, want %d", len(d.Paragraphs), len(tt.want))
			}
			for i, want := range tt.want {
				if got := plainOf(d.Paragraphs[i]); got != want {
					t.Errorf("paragraph %d = %q, want %q", i, got, want)
				}
				if d.Paragraphs[i].PageBreak != (i == tt.wantCursor) {
					t.Errorf("paragraph %d PageBreak = %v, want %v", i,
						d.Paragraphs[i].PageBreak, i == tt.wantCursor)
				}
			}
			if d.CursorPara != tt.wantCursor || d.CursorOff != 0 {
				t.Errorf("cursor = (%d,%d), want (%d,0)", d.CursorPara, d.CursorOff, tt.wantCursor)
			}
			if cur, total := d.PageInfo(); cur != 2 || total != 2 {
				t.Errorf("PageInfo = %d/%d, want 2/2", cur, total)
			}
		})
	}
}

func TestPageBreakAtDocumentStartCanNavigateBack(t *testing.T) {
	d := New()
	d.Page = testPage()
	d.InsertPageBreak()

	if d.CursorPara != 1 {
		t.Fatalf("cursor paragraph = %d, want 1", d.CursorPara)
	}
	if current, total := d.PageInfo(); current != 2 || total != 2 {
		t.Fatalf("page info after break = %d/%d, want 2/2", current, total)
	}

	d.MoveUp()
	if d.CursorPara != 0 {
		t.Fatalf("cursor paragraph after moving up = %d, want 0", d.CursorPara)
	}
	if current, total := d.PageInfo(); current != 1 || total != 2 {
		t.Fatalf("page info after moving up = %d/%d, want 1/2", current, total)
	}
}

func TestPageBreakAdvancesFromEveryEmptyPage(t *testing.T) {
	d := New()
	d.Page = testPage()
	d.InsertPageBreak()
	d.InsertPageBreak()

	if current, total := d.PageInfo(); current != 3 || total != 3 {
		t.Fatalf("page info after two breaks = %d/%d, want 3/3", current, total)
	}
	for wantPage := 2; wantPage >= 1; wantPage-- {
		d.MoveUp()
		if current, total := d.PageInfo(); current != wantPage || total != 3 {
			t.Fatalf("page info moving back = %d/%d, want %d/3", current, total, wantPage)
		}
	}
}

func TestRestoreAddsAnchorForLegacyLeadingPageBreak(t *testing.T) {
	d := New()
	d.Page = testPage()
	d.Paragraphs[0].Runs[0].Text = "content"
	d.Paragraphs[0].PageBreak = true
	state := d.Snapshot()

	restored := New()
	if err := restored.Restore(state); err != nil {
		t.Fatal(err)
	}
	if restored.CursorPara != 1 || plainOf(restored.Paragraphs[0]) != "" {
		t.Fatalf("restored leading page break lacks a navigable anchor: cursor=%d paragraphs=%+v",
			restored.CursorPara, restored.Paragraphs)
	}
	restored.MoveUp()
	if current, total := restored.PageInfo(); current != 1 || total != 2 {
		t.Fatalf("restored page info after moving up = %d/%d, want 1/2", current, total)
	}
}

func TestComposeRow(t *testing.T) {
	cases := []struct {
		l, c, r string
		w       int
		want    string
	}{
		{"a", "b", "c", 9, "a   b   c"},
		{"", "", "x", 5, "    x"},
		{"left", "", "", 8, "left    "},
		{"", "mid", "", 9, "   mid   "},
		{"abcdefgh", "", "ij", 6, "abc ij"},
		{"abcd", "XYZ", "ef", 10, "abcd XY ef"},
	}
	for _, c := range cases {
		got := composeRow(c.l, c.c, c.r, c.w)
		if got != c.want {
			t.Errorf("composeRow(%q,%q,%q,%d) = %q, want %q", c.l, c.c, c.r, c.w, got, c.want)
		}
	}
	if got := composeRow("日本語", "", "右", 7); runewidth.StringWidth(got) != 7 {
		t.Errorf("wide runes: %q is %d cells, want 7", got, runewidth.StringWidth(got))
	}
}

func TestHeaderTokensAndEscaping(t *testing.T) {
	ps := testPage()
	ps.Header.Left = "{title}"
	d := pagedDoc(1, ps)
	d.Title = "Doc"
	if lines := plainLines(d); !strings.Contains(lines[1], "Doc") {
		t.Errorf("{title} not expanded: %q", lines[1])
	}

	ps.Header.Left = "[red]x"
	d = pagedDoc(1, ps)
	if out := d.StyledText(); strings.Contains(out, "[red]") {
		t.Errorf("header text would be parsed as a colour tag: %q", out)
	}
}

func TestNormalizedKeepsTextArea(t *testing.T) {
	for _, header := range []bool{false, true} {
		ps := PageSettings{Width: 100, Height: 59, MarginLeft: 80, MarginRight: 80, MarginTop: 70, MarginBottom: 70}
		chrome := 0
		if header {
			ps.Header.Left = "x"
			chrome = 1
		}
		n := ps.Normalized()
		if w := n.Width - n.MarginLeft - n.MarginRight; w < minBodyWidth {
			t.Errorf("header=%v: text width %d, want >= %d", header, w, minBodyWidth)
		}
		if h := n.Height - n.MarginTop - n.MarginBottom - chrome; h < minBodyRows {
			t.Errorf("header=%v: text rows %d, want >= %d", header, h, minBodyRows)
		}
		if n.Normalized() != n {
			t.Errorf("header=%v: Normalized is not idempotent", header)
		}
	}
	if n := (PageSettings{Width: 100, Height: 59, MarginLeft: -4}).Normalized(); n.MarginLeft != 0 {
		t.Errorf("negative margin = %d, want 0", n.MarginLeft)
	}
}

func TestMarginAndHeaderChangesAreUndoable(t *testing.T) {
	d := New()
	orig := d.GetPageSettings()

	ps := orig
	ps.MarginLeft = 12
	ps.Footer.Center = "{page}"
	d.SetPageSettings(ps)
	if got := d.GetPageSettings(); got.MarginLeft != 12 || got.Footer.Center != "{page}" {
		t.Fatalf("settings not stored: %+v", got)
	}

	mustUndo(t, d)
	if got := d.GetPageSettings(); got != orig {
		t.Fatalf("after undo = %+v, want %+v", got, orig)
	}
	mustRedo(t, d)
	if got := d.GetPageSettings(); got.MarginLeft != 12 {
		t.Fatalf("after redo MarginLeft = %d, want 12", got.MarginLeft)
	}

	ps.MarginLeft = 500
	d.SetPageSettings(ps)
	if got := d.GetPageSettings(); got.MarginLeft >= 500 {
		t.Fatalf("oversized margin was not clamped: %d", got.MarginLeft)
	}
}
