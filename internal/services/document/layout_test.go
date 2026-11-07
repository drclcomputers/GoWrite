package document

import (
	"reflect"
	"regexp"
	"strings"
	"testing"
)

// tagRe matches the tview colour/style tags the renderer emits.
var tagRe = regexp.MustCompile(`\[[a-zA-Z0-9#:\-]*\]`)

func newTestDoc(width int, paras ...string) *Document {
	d := New()
	d.DisplayWidth = width
	d.Paragraphs = nil
	for _, s := range paras {
		d.Paragraphs = append(d.Paragraphs, Paragraph{Runs: []Run{{Text: s}}})
	}
	return d
}

// plainLines renders without the caret and strips tags, one string per row.
func plainLines(d *Document) []string {
	return strings.Split(tagRe.ReplaceAllString(d.StyledText(), ""), "\n")
}

func TestWrapRunes(t *testing.T) {
	cases := []struct {
		text  string
		width int
		want  []visualLine
	}{
		{"hello world foo bar", 10, []visualLine{{0, 6, 5}, {6, 16, 15}, {16, 19, 19}}},
		{"abcdefghijkl mn", 5, []visualLine{{0, 5, 5}, {5, 10, 10}, {10, 15, 15}}},
		{"hello world", 5, []visualLine{{0, 6, 5}, {6, 11, 11}}},
		{"hi   ", 10, []visualLine{{0, 5, 2}}},
		{"", 5, []visualLine{{0, 0, 0}}},
	}
	for _, c := range cases {
		got := wrapRunes([]rune(c.text), c.width)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("wrapRunes(%q, %d) = %v, want %v", c.text, c.width, got, c.want)
		}
	}
}

func TestWrapRunesInvariants(t *testing.T) {
	text := []rune("the quick  brown fox jumps over the lazy dog   and   a_verylongwordthatcannotfitanywhere end ")
	for width := 1; width <= 30; width++ {
		lines := wrapRunes(text, width)
		pos := 0
		for _, ln := range lines {
			if ln.Start != pos {
				t.Fatalf("width %d: line starts at %d, expected %d", width, ln.Start, pos)
			}
			if ln.End <= ln.Start {
				t.Fatalf("width %d: empty line %v makes no progress", width, ln)
			}
			if ln.CEnd < ln.Start || ln.CEnd > ln.End {
				t.Fatalf("width %d: bad CEnd in %v", width, ln)
			}
			if w := widthOf(text[ln.Start:ln.CEnd]); w > width {
				t.Fatalf("width %d: line %v has content width %d", width, ln, w)
			}
			pos = ln.End
		}
		if pos != len(text) {
			t.Fatalf("width %d: lines cover %d of %d runes", width, pos, len(text))
		}
	}
}

func TestMoveDownUpKeepsColumn(t *testing.T) {
	// width 12 -> wrap width 10: "hello " | "world foo " | "bar"
	d := newTestDoc(12, "hello world foo bar")
	d.CursorOff = 3

	check := func(step string, wantPara, wantOff int) {
		t.Helper()
		if d.CursorPara != wantPara || d.CursorOff != wantOff {
			t.Fatalf("%s: cursor at (%d,%d), want (%d,%d)", step, d.CursorPara, d.CursorOff, wantPara, wantOff)
		}
	}
	d.MoveDown()
	check("down 1", 0, 9)
	d.MoveDown()
	check("down 2", 0, 19)
	d.MoveDown() // already on the last line: goes to the end, stays put
	check("down 3", 0, 19)
	d.MoveUp() // column 3 is remembered
	check("up", 0, 9)
}

func TestGoalColumnSurvivesShortParagraph(t *testing.T) {
	d := newTestDoc(20, "abcdefgh", "ab", "abcdefgh")
	d.CursorOff = 6

	d.MoveDown()
	if d.CursorPara != 1 || d.CursorOff != 2 {
		t.Fatalf("into short paragraph: (%d,%d)", d.CursorPara, d.CursorOff)
	}
	d.MoveDown()
	if d.CursorPara != 2 || d.CursorOff != 6 {
		t.Fatalf("out of short paragraph: (%d,%d), want (2,6)", d.CursorPara, d.CursorOff)
	}
	d.MoveUp()
	d.MoveUp()
	if d.CursorPara != 0 || d.CursorOff != 6 {
		t.Fatalf("back to start: (%d,%d), want (0,6)", d.CursorPara, d.CursorOff)
	}

	// A horizontal move forgets the remembered column.
	d.MoveDown() // (1,2)
	d.MoveLeft() // (1,1)
	d.MoveDown()
	if d.CursorPara != 2 || d.CursorOff != 1 {
		t.Fatalf("after horizontal move: (%d,%d), want (2,1)", d.CursorPara, d.CursorOff)
	}
}

func TestCursorRow(t *testing.T) {
	d := newTestDoc(12, "hello world foo bar", "x")
	d.CursorPara, d.CursorOff = 1, 0
	if got := d.CursorRow(); got != 3 {
		t.Errorf("second paragraph row = %d, want 3", got)
	}
	d.CursorPara, d.CursorOff = 0, 10
	if got := d.CursorRow(); got != 1 {
		t.Errorf("row of offset 10 = %d, want 1", got)
	}
}

func TestLineStartEnd(t *testing.T) {
	d := newTestDoc(12, "hello world foo bar")
	d.CursorOff = 8
	d.MoveLineStart()
	if d.CursorOff != 6 {
		t.Errorf("Home from 8 = %d, want 6", d.CursorOff)
	}
	d.MoveLineEnd()
	if d.CursorOff != 15 {
		t.Errorf("End on a wrapped line = %d, want 15 (after the last word)", d.CursorOff)
	}
	d.CursorOff = 17
	d.MoveLineEnd()
	if d.CursorOff != 19 {
		t.Errorf("End on the last line = %d, want 19", d.CursorOff)
	}
	d.MoveDocStart()
	if d.CursorPara != 0 || d.CursorOff != 0 {
		t.Errorf("MoveDocStart = (%d,%d)", d.CursorPara, d.CursorOff)
	}
	d.MoveDocEnd()
	if d.CursorPara != 0 || d.CursorOff != 19 {
		t.Errorf("MoveDocEnd = (%d,%d)", d.CursorPara, d.CursorOff)
	}
}

func TestMovePage(t *testing.T) {
	d := newTestDoc(20, "x", "x", "x", "x", "x")
	d.MovePage(3)
	if d.CursorPara != 3 {
		t.Errorf("MovePage(3): paragraph %d, want 3", d.CursorPara)
	}
	d.MovePage(100)
	if d.CursorPara != 4 || d.CursorOff != 1 {
		t.Errorf("MovePage(100): (%d,%d), want (4,1)", d.CursorPara, d.CursorOff)
	}
	d.MovePage(-100)
	if d.CursorPara != 0 || d.CursorOff != 0 {
		t.Errorf("MovePage(-100): (%d,%d), want (0,0)", d.CursorPara, d.CursorOff)
	}
}

// Regression: the caret pushed a full-width right-aligned line onto the next row;
// no row may exceed the view width.
func TestRenderNeverExceedsDisplayWidth(t *testing.T) {
	texts := []string{
		"",
		"x",
		"the quick brown fox jumps over the lazy dog",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"word    ",
		"a b c d e f g h i j k l m n o p q r s t u v w x y z",
		"tab\there\tand there",
	}
	for _, align := range []Align{AlignLeft, AlignCenter, AlignRight} {
		for width := 6; width <= 24; width++ {
			for _, text := range texts {
				d := newTestDoc(width, text)
				d.Paragraphs[0].Align = align
				n := len([]rune(text))
				for off := 0; off <= n; off++ {
					d.CursorOff = off
					out := d.StyledTextWithCursor()
					if c := strings.Count(out, caretOpen); c != 1 {
						t.Fatalf("align %d width %d %q off %d: %d carets", align, width, text, off, c)
					}
					for i, line := range strings.Split(tagRe.ReplaceAllString(out, ""), "\n") {
						if got := len([]rune(line)); got > width {
							t.Fatalf("align %d width %d %q off %d: row %d is %d cells wide: %q",
								align, width, text, off, i, got, line)
						}
					}
				}
			}
		}
	}
}

func TestAlignmentAppliesToEveryWrappedLine(t *testing.T) {
	// wrap width 10: rows "hello", "world foo", "bar"
	d := newTestDoc(12, "hello world foo bar")

	d.Paragraphs[0].Align = AlignRight
	for i, line := range plainLines(d) {
		if got := len([]rune(strings.TrimRight(line, " "))); got != 10 {
			t.Errorf("right: row %d ends at column %d, want 10: %q", i, got, line)
		}
	}

	d.Paragraphs[0].Align = AlignCenter
	want := []int{7, 9, 6} // pad (10-5)/2, (10-9)/2, (10-3)/2 plus the text
	for i, line := range plainLines(d) {
		if got := len([]rune(strings.TrimRight(line, " "))); got != want[i] {
			t.Errorf("center: row %d ends at column %d, want %d: %q", i, got, want[i], line)
		}
	}
}

func TestRenderEscapesTags(t *testing.T) {
	d := newTestDoc(30, "[red]x")
	if out := d.StyledText(); strings.Contains(out, "[red]") {
		t.Errorf("literal [red] would be parsed as a colour tag: %q", out)
	}
}
