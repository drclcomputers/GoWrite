// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

import (
	"slices"
	"unicode"
)

const maxHistory = 200

// historyEntry is a snapshot of everything undo can change: the paragraphs
// (text, runs, alignment, page breaks), the page setup and the cursor.
type historyEntry struct {
	Paragraphs []Paragraph
	Page       PageSettings
	CursorPara int
	CursorOff  int
}

// editKind says what sort of edit last touched the document, so consecutive
// edits of the same kind can share one undo step.
type editKind uint8

const (
	editOther     editKind = iota // anything that always gets its own undo step
	editTyping                    // inserting characters
	editBackspace                 // deleting backwards
	editDelete                    // deleting forwards
)

// takeSnapshot captures the current state. Caller holds the lock.
func (d *Document) takeSnapshot() historyEntry {
	ps := make([]Paragraph, len(d.Paragraphs))
	for i, p := range d.Paragraphs {
		ps[i] = Paragraph{Runs: cloneRuns(p.Runs), Align: p.Align, PageBreak: p.PageBreak}
	}
	return historyEntry{
		Paragraphs: ps,
		Page:       d.Page,
		CursorPara: d.CursorPara,
		CursorOff:  d.CursorOff,
	}
}

// sameContent reports whether two snapshots look identical on the page. The
// cursor is ignored: an operation that changed nothing must not leave an
// undo step behind.
func sameContent(a, b historyEntry) bool {
	if a.Page != b.Page || len(a.Paragraphs) != len(b.Paragraphs) {
		return false
	}
	for i := range a.Paragraphs {
		pa, pb := a.Paragraphs[i], b.Paragraphs[i]
		if pa.Align != pb.Align || pa.PageBreak != pb.PageBreak || !slices.Equal(pa.Runs, pb.Runs) {
			return false
		}
	}
	return true
}

// pushHistory records a snapshot before a mutation and ends any run of
// coalesced typing. Caller holds the lock.
func (d *Document) pushHistory() {
	d.history = append(d.history, d.takeSnapshot())
	if len(d.history) > maxHistory {
		d.history = d.history[1:]
	}
	d.redo = nil
	d.lastEdit = editOther
}

// pushHistoryCoalesced is pushHistory for typing and single-character
// deletes. It skips the snapshot when this edit continues the previous one:
// same kind, cursor exactly where the last edit left it, no selection being
// replaced, and (for typing) not the first letter of a new word. That way
// one undo removes a word, not a letter. Caller holds the lock, and must call
// endCoalescedEdit once the edit has moved the cursor.
func (d *Document) pushHistoryCoalesced(kind editKind, r rune) {
	hasSel := d.hasAnchor && (d.anchorPara != d.CursorPara || d.anchorOff != d.CursorOff)
	continues := !hasSel &&
		len(d.history) > 0 &&
		d.lastEdit == kind &&
		d.lastEditPara == d.CursorPara &&
		d.lastEditOff == d.CursorOff
	if continues && kind == editTyping && d.lastEditSpace && !unicode.IsSpace(r) {
		continues = false
	}
	if !continues {
		d.pushHistory()
	}
	d.lastEdit = kind
	d.lastEditSpace = unicode.IsSpace(r)
}

// endCoalescedEdit remembers where the edit left the cursor. Caller holds
// the lock.
func (d *Document) endCoalescedEdit() {
	d.lastEditPara, d.lastEditOff = d.CursorPara, d.CursorOff
}

// popHistory removes the most recent entry — used when an operation turned
// out to be a no-op. Caller holds the lock.
func (d *Document) popHistory() {
	if len(d.history) > 0 {
		d.history = d.history[:len(d.history)-1]
	}
	d.lastEdit = editOther
}

// restore applies a snapshot. Caller holds the lock.
func (d *Document) restore(h historyEntry) {
	d.Paragraphs = h.Paragraphs
	d.Page = h.Page
	d.CursorPara = h.CursorPara
	d.CursorOff = h.CursorOff
	d.hasAnchor = false
	d.goalSet = false
	d.lastEdit = editOther
	d.ensureCursor()
}

// step pops entries off from, skipping any that would change nothing, and
// restores the first real one, pushing the current state onto to. Caller
// holds the lock.
func (d *Document) step(from, to *[]historyEntry) bool {
	cur := d.takeSnapshot()
	for len(*from) > 0 {
		h := (*from)[len(*from)-1]
		*from = (*from)[:len(*from)-1]
		if sameContent(h, cur) {
			continue
		}
		*to = append(*to, cur)
		d.restore(h)
		d.Dirty = true
		return true
	}
	d.lastEdit = editOther
	return false
}

// Undo reverts the last mutating operation. Reports whether anything was
// undone.
func (d *Document) Undo() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.step(&d.history, &d.redo)
}

// Redo re-applies the last undone operation. Reports whether anything was
// redone.
func (d *Document) Redo() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.step(&d.redo, &d.history)
}

// CanUndo reports whether there is anything to undo.
func (d *Document) CanUndo() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.history) > 0
}

// CanRedo reports whether there is anything to redo.
func (d *Document) CanRedo() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.redo) > 0
}

// ClearHistory forgets the undo and redo stacks.
func (d *Document) ClearHistory() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.history = nil
	d.redo = nil
	d.lastEdit = editOther
}
