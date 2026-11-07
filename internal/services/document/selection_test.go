package document

import (
	"reflect"
	"strings"
	"testing"
)

const (
	testRed  = "#e06c75"
	testBlue = "#61afef"
)

// selectRange selects from (sp,so) to (ep,eo), leaving the cursor at the end.
func selectRange(d *Document, sp, so, ep, eo int) {
	d.hasAnchor = true
	d.anchorPara, d.anchorOff = sp, so
	d.CursorPara, d.CursorOff = ep, eo
}

func paraTexts(d *Document) []string {
	out := make([]string, len(d.Paragraphs))
	for i, p := range d.Paragraphs {
		out[i] = plainOf(p)
	}
	return out
}

func expectTexts(t *testing.T, d *Document, want ...string) {
	t.Helper()
	if got := paraTexts(d); !reflect.DeepEqual(got, want) {
		t.Fatalf("paragraphs = %q, want %q", got, want)
	}
}

func expectCursor(t *testing.T, d *Document, para, off int) {
	t.Helper()
	if d.CursorPara != para || d.CursorOff != off {
		t.Fatalf("cursor = (%d,%d), want (%d,%d)", d.CursorPara, d.CursorOff, para, off)
	}
}

// Regression: deleting a character dropped the run's colours.
func TestDeleteKeepsColours(t *testing.T) {
	d := newTestDoc(40, "hello")
	d.Paragraphs[0].Runs = []Run{{Text: "hello", Style: StyleBold, Fg: testRed, Bg: testBlue}}

	d.CursorOff = 5
	d.Backspace() // "hell"
	d.CursorOff = 2
	d.Delete() // "hel"

	want := []Run{{Text: "hel", Style: StyleBold, Fg: testRed, Bg: testBlue}}
	if got := d.Paragraphs[0].Runs; !reflect.DeepEqual(got, want) {
		t.Fatalf("runs = %+v, want %+v", got, want)
	}
}

func TestDeleteSelectionWithinParagraph(t *testing.T) {
	d := newTestDoc(40, "hello world")
	selectRange(d, 0, 0, 0, 5)
	if !d.HasSelection() || d.SelectionLen() != 5 {
		t.Fatalf("selection = %v, len %d", d.HasSelection(), d.SelectionLen())
	}
	if !d.DeleteSelection() {
		t.Fatal("DeleteSelection reported nothing to delete")
	}
	expectTexts(t, d, " world")
	expectCursor(t, d, 0, 0)
	if d.HasSelection() {
		t.Error("selection should be gone after deleting it")
	}
}

func TestDeleteSelectionAcrossParagraphs(t *testing.T) {
	d := newTestDoc(40, "hello", "brave", "world")
	d.Paragraphs[0].Align = AlignRight
	selectRange(d, 0, 2, 2, 3)
	d.Backspace() // Backspace deletes the selection, nothing more
	expectTexts(t, d, "held")
	expectCursor(t, d, 0, 2)
	if d.Paragraphs[0].Align != AlignRight {
		t.Error("merged paragraph should keep the first paragraph's alignment")
	}
}

func TestTypingReplacesSelection(t *testing.T) {
	d := newTestDoc(40, "hello world")
	selectRange(d, 0, 0, 0, 5)
	d.InsertRune('X')
	expectTexts(t, d, "X world")
	expectCursor(t, d, 0, 1)
	if d.HasSelection() {
		t.Error("selection should be gone after typing")
	}
}

func TestEnterReplacesSelectionAndSplits(t *testing.T) {
	d := newTestDoc(40, "hello world")
	selectRange(d, 0, 3, 0, 8) // "lo wo"
	d.InsertRune('\n')
	expectTexts(t, d, "hel", "rld")
	expectCursor(t, d, 1, 0)
}

func TestSelectionBackwards(t *testing.T) {
	// Anchor after the cursor (selecting leftwards) works the same.
	d := newTestDoc(40, "hello world")
	selectRange(d, 0, 5, 0, 0)
	d.Delete()
	expectTexts(t, d, " world")
	expectCursor(t, d, 0, 0)
}

func TestSelectAllAndSelectedText(t *testing.T) {
	d := newTestDoc(40, "a", "bc")
	d.SelectAll()
	if got := d.SelectedText(); got != "a\nbc" {
		t.Errorf("SelectedText = %q", got)
	}
	if got := d.SelectionLen(); got != 4 { // 'a' + break + 'b' + 'c'
		t.Errorf("SelectionLen = %d, want 4", got)
	}
	d.DeleteSelection()
	expectTexts(t, d, "")
}

func TestCollapseSelection(t *testing.T) {
	d := newTestDoc(40, "hello world")
	selectRange(d, 0, 2, 0, 7)
	d.CollapseSelection(true)
	expectCursor(t, d, 0, 2)
	if d.HasSelection() {
		t.Error("collapse should clear the selection")
	}
	selectRange(d, 0, 2, 0, 7)
	d.CollapseSelection(false)
	expectCursor(t, d, 0, 7)
}

func TestBeginSelectionExtendsWithMoves(t *testing.T) {
	d := newTestDoc(40, "hello world")
	d.CursorOff = 2
	d.BeginSelection()
	d.MoveRight()
	d.BeginSelection() // second Shift+move must not move the anchor
	d.MoveRight()
	if got := d.SelectedText(); got != "ll" {
		t.Errorf("SelectedText = %q, want %q", got, "ll")
	}
}

func TestToggleStyleOnSelection(t *testing.T) {
	d := newTestDoc(40, "hello world")
	selectRange(d, 0, 6, 0, 11)

	if !d.ToggleStyleOnSelection(StyleBold) {
		t.Fatal("expected the style to apply to the selection")
	}
	want := []Run{{Text: "hello "}, {Text: "world", Style: StyleBold}}
	if got := d.Paragraphs[0].Runs; !reflect.DeepEqual(got, want) {
		t.Fatalf("after bold: %+v, want %+v", got, want)
	}

	d.ToggleStyleOnSelection(StyleBold) // all bold -> removes it, runs merge back
	want = []Run{{Text: "hello world"}}
	if got := d.Paragraphs[0].Runs; !reflect.DeepEqual(got, want) {
		t.Fatalf("after un-bold: %+v, want %+v", got, want)
	}

	// Partly bold: toggling bolds everything rather than un-bolding.
	d.ToggleStyleOnSelection(StyleBold)
	selectRange(d, 0, 0, 0, 11)
	d.ToggleStyleOnSelection(StyleBold)
	want = []Run{{Text: "hello world", Style: StyleBold}}
	if got := d.Paragraphs[0].Runs; !reflect.DeepEqual(got, want) {
		t.Fatalf("partly bold -> %+v, want %+v", got, want)
	}

	// No selection: nothing happens (the caller falls back to typing style).
	d.ClearSelection()
	if d.ToggleStyleOnSelection(StyleItalic) {
		t.Error("should report no selection")
	}
}

func TestSetSelectionColor(t *testing.T) {
	d := newTestDoc(40, "hello world")
	selectRange(d, 0, 0, 0, 5)
	d.SetSelectionColor(false, testRed)
	selectRange(d, 0, 3, 0, 8)
	d.SetSelectionColor(true, testBlue)

	want := []Run{
		{Text: "hel", Fg: testRed},
		{Text: "lo", Fg: testRed, Bg: testBlue},
		{Text: " wo", Bg: testBlue},
		{Text: "rld"},
	}
	if got := d.Paragraphs[0].Runs; !reflect.DeepEqual(got, want) {
		t.Fatalf("runs = %+v\nwant   %+v", got, want)
	}

	// An empty value restores the default colour and runs merge again.
	d.SetSelectionColor(true, "")
	selectRange(d, 0, 0, 0, 11)
	d.SetSelectionColor(false, "")
	want = []Run{{Text: "hello world"}}
	if got := d.Paragraphs[0].Runs; !reflect.DeepEqual(got, want) {
		t.Fatalf("after clearing: %+v, want %+v", got, want)
	}
}

func TestApplyActiveToSelection(t *testing.T) {
	d := newTestDoc(40, "hello world")
	d.ActiveStyle = StyleItalic
	d.ActiveFg = testRed
	selectRange(d, 0, 0, 0, 5)
	if !d.ApplyActiveToSelection() {
		t.Fatal("expected to apply")
	}
	want := []Run{{Text: "hello", Style: StyleItalic, Fg: testRed}, {Text: " world"}}
	if got := d.Paragraphs[0].Runs; !reflect.DeepEqual(got, want) {
		t.Fatalf("runs = %+v, want %+v", got, want)
	}
}

func TestAlignAppliesToSelectedParagraphs(t *testing.T) {
	aligns := func(d *Document) []Align {
		out := make([]Align, len(d.Paragraphs))
		for i, p := range d.Paragraphs {
			out[i] = p.Align
		}
		return out
	}

	d := newTestDoc(40, "a", "b", "c", "d")
	selectRange(d, 1, 1, 2, 1)
	d.SetParagraphAlign(AlignRight)
	if got, want := aligns(d), []Align{AlignLeft, AlignRight, AlignRight, AlignLeft}; !reflect.DeepEqual(got, want) {
		t.Errorf("selection over two paragraphs: %v, want %v", got, want)
	}

	// A selection that only touches the next paragraph's start excludes it.
	d = newTestDoc(40, "a", "b", "c")
	selectRange(d, 0, 0, 1, 0)
	d.SetParagraphAlign(AlignCenter)
	if got, want := aligns(d), []Align{AlignCenter, AlignLeft, AlignLeft}; !reflect.DeepEqual(got, want) {
		t.Errorf("selection ending at a paragraph start: %v, want %v", got, want)
	}

	// No selection: just the cursor's paragraph.
	d = newTestDoc(40, "a", "b", "c")
	d.CursorPara = 2
	d.SetParagraphAlign(AlignRight)
	if got, want := aligns(d), []Align{AlignLeft, AlignLeft, AlignRight}; !reflect.DeepEqual(got, want) {
		t.Errorf("no selection: %v, want %v", got, want)
	}
}

func TestCopyPasteKeepsStyle(t *testing.T) {
	d := newTestDoc(40, "abcd")
	d.Paragraphs[0].Runs = []Run{{Text: "ab", Style: StyleBold}, {Text: "cd"}}
	selectRange(d, 0, 1, 0, 3) // "bc"
	if !d.Copy() {
		t.Fatal("Copy found no selection")
	}
	expectTexts(t, d, "abcd") // copying changes nothing

	d.ClearSelection()
	d.CursorOff = 4
	if !d.Paste() {
		t.Fatal("Paste found an empty clipboard")
	}
	expectTexts(t, d, "abcdbc")
	expectCursor(t, d, 0, 6)
	want := []Run{{Text: "ab", Style: StyleBold}, {Text: "cd"}, {Text: "b", Style: StyleBold}, {Text: "c"}}
	if got := d.Paragraphs[0].Runs; !reflect.DeepEqual(got, want) {
		t.Fatalf("runs = %+v, want %+v", got, want)
	}
}

func TestPasteMultipleParagraphs(t *testing.T) {
	d := newTestDoc(40, "one", "two", "three")
	selectRange(d, 0, 1, 2, 2) // "ne" / "two" / "th"
	d.Copy()
	d.ClearSelection()
	d.CursorPara, d.CursorOff = 2, 5
	d.Paste()
	expectTexts(t, d, "one", "two", "threene", "two", "th")
	expectCursor(t, d, 4, 2)
}

func TestPasteReplacesSelection(t *testing.T) {
	d := newTestDoc(40, "hello world")
	selectRange(d, 0, 0, 0, 5)
	d.Copy()
	selectRange(d, 0, 6, 0, 11)
	d.Paste()
	expectTexts(t, d, "hello hello")
	expectCursor(t, d, 0, 11)
}

func TestCutThenPaste(t *testing.T) {
	d := newTestDoc(40, "hello world")
	selectRange(d, 0, 0, 0, 6)
	if !d.Cut() {
		t.Fatal("Cut found no selection")
	}
	expectTexts(t, d, "world")
	d.CursorOff = 5
	d.Paste()
	expectTexts(t, d, "worldhello ")

	// The clipboard outlives the document it came from.
	other := newTestDoc(40, "x")
	other.CursorOff = 1
	other.Paste()
	expectTexts(t, other, "xhello ")
}

func TestPasteIntoEmptyClipboardIsNoop(t *testing.T) {
	setClip(nil)
	d := newTestDoc(40, "abc")
	if d.Paste() {
		t.Error("Paste should report an empty clipboard")
	}
	expectTexts(t, d, "abc")
}

func TestSelectionIsHighlighted(t *testing.T) {
	d := newTestDoc(40, "hello world")
	if strings.Contains(d.StyledText(), selBg) {
		t.Fatal("highlight colour present with no selection")
	}
	selectRange(d, 0, 0, 0, 5)
	if !strings.Contains(d.StyledText(), selBg) {
		t.Fatal("selected text is not highlighted")
	}
}

// Highlight, caret and paragraph-break marker must not push a row past the view width.
func TestSelectionRenderNeverExceedsDisplayWidth(t *testing.T) {
	paras := []string{"the quick brown fox jumps", "", "over the lazy dog", "aaaaaaaaaaaaaaaaaaaaaaaaa"}
	for _, align := range []Align{AlignLeft, AlignCenter, AlignRight} {
		for width := 6; width <= 26; width++ {
			d := newTestDoc(width, paras...)
			for i := range d.Paragraphs {
				d.Paragraphs[i].Align = align
			}
			ranges := [][4]int{{0, 0, 3, 25}, {0, 4, 2, 3}, {1, 0, 2, 0}, {0, 25, 1, 0}, {2, 5, 2, 9}}
			for _, r := range ranges {
				selectRange(d, r[0], r[1], r[2], r[3])
				out := d.StyledTextWithCursor()
				for i, line := range strings.Split(tagRe.ReplaceAllString(out, ""), "\n") {
					if got := len([]rune(line)); got > width {
						t.Fatalf("align %d width %d sel %v: row %d is %d cells wide: %q",
							align, width, r, i, got, line)
					}
				}
			}
		}
	}
}
