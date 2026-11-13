package document

import "testing"

func TestListEnterContinuesAtCurrentLevel(t *testing.T) {
	d := New()
	d.LoadText("• first")
	d.CursorOff = len([]rune(plainOf(d.Paragraphs[0])))
	if !d.ChangeListLevel(1) || !d.ChangeListLevel(1) {
		t.Fatal("could not indent list item to level three")
	}

	d.InsertRune('\n')
	if got, want := d.Text(), "    • first\n    • "; got != want {
		t.Fatalf("continued list = %q, want %q", got, want)
	}
}

func TestListBackspaceOutdentsOneLevelThenExitsList(t *testing.T) {
	d := New()
	d.LoadText("    • ")
	d.CursorOff = len([]rune(plainOf(d.Paragraphs[0])))

	for _, want := range []string{"  • ", "• ", ""} {
		d.Backspace()
		if got := d.Text(); got != want {
			t.Fatalf("after Backspace, text = %q, want %q", got, want)
		}
	}
}

func TestListIndentIsLimitedToFiveLevels(t *testing.T) {
	d := New()
	d.LoadText("1. item")
	d.CursorOff = len([]rune(plainOf(d.Paragraphs[0])))

	for i := 0; i < maxListLevel+2; i++ {
		if !d.ChangeListLevel(1) {
			t.Fatal("Tab did not handle a list item")
		}
	}
	if got, want := d.Text(), "        1. item"; got != want {
		t.Fatalf("list at maximum level = %q, want %q", got, want)
	}
}

func TestShiftTabOutdentsLevelOneToNormalParagraph(t *testing.T) {
	d := New()
	d.LoadText("• item")
	d.CursorOff = len([]rune(plainOf(d.Paragraphs[0])))

	if !d.ChangeListLevel(-1) {
		t.Fatal("Shift+Tab did not handle the list item")
	}
	if got, want := d.Text(), "item"; got != want {
		t.Fatalf("outdented text = %q, want %q", got, want)
	}
	if d.CursorOff != len([]rune("item")) {
		t.Fatalf("cursor offset = %d, want %d", d.CursorOff, len([]rune("item")))
	}
}
