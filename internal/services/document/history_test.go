package document

import "testing"

func typeString(d *Document, s string) {
	for _, r := range s {
		d.InsertRune(r)
	}
}

func mustUndo(t *testing.T, d *Document) {
	t.Helper()
	if !d.Undo() {
		t.Fatal("Undo() = false, want true")
	}
}

func mustRedo(t *testing.T, d *Document) {
	t.Helper()
	if !d.Redo() {
		t.Fatal("Redo() = false, want true")
	}
}

func TestTypingUndoesOneWordAtATime(t *testing.T) {
	d := New()
	typeString(d, "hello world")

	mustUndo(t, d)
	expectTexts(t, d, "hello ")
	mustUndo(t, d)
	expectTexts(t, d, "")
	if d.Undo() {
		t.Fatal("Undo() on an empty history should report false")
	}

	mustRedo(t, d)
	expectTexts(t, d, "hello ")
	mustRedo(t, d)
	expectTexts(t, d, "hello world")
}

func TestBackspaceRunIsOneUndoStep(t *testing.T) {
	d := New()
	d.LoadText("abcdef")
	d.CursorOff = 6
	d.Backspace()
	d.Backspace()
	d.Backspace()
	expectTexts(t, d, "abc")

	mustUndo(t, d)
	expectTexts(t, d, "abcdef")
	expectCursor(t, d, 0, 6)
}

func TestDeleteRunIsOneUndoStep(t *testing.T) {
	d := New()
	d.LoadText("abcdef")
	d.Delete()
	d.Delete()
	expectTexts(t, d, "cdef")

	mustUndo(t, d)
	expectTexts(t, d, "abcdef")
}

func TestMovingTheCursorStartsANewUndoStep(t *testing.T) {
	d := New()
	typeString(d, "ab")
	d.CursorOff = 1
	d.InsertRune('X')
	expectTexts(t, d, "aXb")

	mustUndo(t, d)
	expectTexts(t, d, "ab")
	mustUndo(t, d)
	expectTexts(t, d, "")
}

func TestStyleChangeStartsANewUndoStep(t *testing.T) {
	d := New()
	typeString(d, "ab")
	d.SetActiveStyle(StyleBold)
	typeString(d, "cd")

	mustUndo(t, d)
	expectTexts(t, d, "ab")
}

func TestTypingOverSelectionIsSeparateStep(t *testing.T) {
	d := New()
	typeString(d, "abc")
	selectRange(d, 0, 0, 0, 3)
	d.InsertRune('x')
	expectTexts(t, d, "x")

	mustUndo(t, d)
	expectTexts(t, d, "abc")
}

func TestEnterIsItsOwnUndoStep(t *testing.T) {
	d := New()
	typeString(d, "ab")
	d.InsertRune('\n')
	typeString(d, "cd")
	expectTexts(t, d, "ab", "cd")

	mustUndo(t, d)
	expectTexts(t, d, "ab", "")
	mustUndo(t, d)
	expectTexts(t, d, "ab")
}

func TestNewEditClearsRedo(t *testing.T) {
	d := New()
	typeString(d, "a ")
	mustUndo(t, d)
	d.InsertRune('b')
	if d.Redo() {
		t.Fatal("Redo() should be empty after a new edit")
	}
}

func TestUndoRestoresPageBreaks(t *testing.T) {
	d := New()
	d.LoadText("a\nb")
	d.CursorPara, d.CursorOff = 1, 0
	d.InsertPageBreak()
	d.CursorOff = 1
	d.InsertRune('x')

	mustUndo(t, d) // the typed letter
	if !d.Paragraphs[1].PageBreak {
		t.Fatal("undoing an edit removed the page break")
	}
	mustUndo(t, d) // the page break itself
	if d.Paragraphs[1].PageBreak {
		t.Fatal("undoing the page break left it in place")
	}
	mustRedo(t, d)
	if !d.Paragraphs[1].PageBreak {
		t.Fatal("redo did not bring the page break back")
	}
}

func TestPageBreakSurvivesEditsThatRebuildParagraphs(t *testing.T) {
	// Enter in the middle of a paragraph keeps the break on the first half.
	d := New()
	d.LoadText("hello")
	d.InsertPageBreak()
	if d.CursorPara != 1 || d.Paragraphs[0].PageBreak || !d.Paragraphs[1].PageBreak {
		t.Fatalf("leading break = cursor %d, flags %v,%v; want cursor 1 and false,true",
			d.CursorPara, d.Paragraphs[0].PageBreak, d.Paragraphs[1].PageBreak)
	}
	d.CursorOff = 2
	d.InsertRune('\n')
	if !d.Paragraphs[1].PageBreak || d.Paragraphs[2].PageBreak {
		t.Fatalf("break flags after Enter = %v,%v, want true,false",
			d.Paragraphs[1].PageBreak, d.Paragraphs[2].PageBreak)
	}

	// Deleting across paragraphs keeps the first paragraph's break.
	d = New()
	d.LoadText("one\ntwo")
	d.InsertPageBreak()
	selectRange(d, 1, 1, 2, 1)
	d.DeleteSelection()
	if !d.Paragraphs[1].PageBreak {
		t.Fatal("deleting across paragraphs dropped the page break")
	}
}

func TestUndoRestoresPageSettings(t *testing.T) {
	d := New()
	orig := d.GetPageSettings()

	ps := orig
	ps.Size, ps.Orientation = SizeLetter, Landscape
	ps.Width, ps.Height = GetDimensions(ps.Size, ps.Orientation)
	d.SetPageSettings(ps)

	mustUndo(t, d)
	if got := d.GetPageSettings(); got != orig {
		t.Fatalf("page settings after undo = %+v, want %+v", got, orig)
	}
	mustRedo(t, d)
	if got := d.GetPageSettings(); got != ps {
		t.Fatalf("page settings after redo = %+v, want %+v", got, ps)
	}
}

func TestUnchangedPageSettingsDoNothing(t *testing.T) {
	d := New()
	d.SetPageSettings(d.GetPageSettings())
	if d.IsDirty() || d.CanUndo() {
		t.Fatal("re-applying identical page settings should not dirty the document or add an undo step")
	}
}

func TestNoOpOperationsLeaveNoUndoStep(t *testing.T) {
	d := New()
	d.LoadText("a")
	d.InsertPageBreak()
	d.InsertPageBreak() // already has one: changes nothing

	mustUndo(t, d)
	if d.Paragraphs[0].PageBreak {
		t.Fatal("first Undo should have removed the page break, not a no-op step")
	}
	if d.Undo() {
		t.Fatal("nothing left to undo")
	}
}
