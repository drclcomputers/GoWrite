// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

// Undo / redo support. Every mutating operation pushes a snapshot of the
// paragraphs and cursor onto the history stack. Undo pops the most recent
// snapshot and pushes the current state onto the redo stack; Redo does the
// reverse. Cursor movements and selection changes do not touch the history.

const maxHistory = 200

// historyEntry is a snapshot of the paragraph model plus cursor.
type historyEntry struct {
	Paragraphs []Paragraph
	CursorPara int
	CursorOff  int
}

// takeSnapshot captures the current state. Caller holds the lock.
func (d *Document) takeSnapshot() historyEntry {
	ps := make([]Paragraph, len(d.Paragraphs))
	for i, p := range d.Paragraphs {
		ps[i] = Paragraph{Runs: cloneRuns(p.Runs), Align: p.Align}
	}
	return historyEntry{
		Paragraphs: ps,
		CursorPara: d.CursorPara,
		CursorOff:  d.CursorOff,
	}
}

// pushHistory records a snapshot before a mutation. Caller holds the lock.
func (d *Document) pushHistory() {
	d.history = append(d.history, d.takeSnapshot())
	if len(d.history) > maxHistory {
		d.history = d.history[1:]
	}
	d.redo = nil
}

// popHistory removes the most recent entry — used when an operation turned
// out to be a no-op. Caller holds the lock.
func (d *Document) popHistory() {
	if len(d.history) > 0 {
		d.history = d.history[:len(d.history)-1]
	}
}

// restore applies a snapshot. Caller holds the lock.
func (d *Document) restore(h historyEntry) {
	d.Paragraphs = h.Paragraphs
	d.CursorPara = h.CursorPara
	d.CursorOff = h.CursorOff
	d.hasAnchor = false
	d.goalSet = false
	d.ensureCursor()
}

// Undo reverts the last mutating operation. Reports whether anything was
// undone.
func (d *Document) Undo() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.history) == 0 {
		return false
	}
	d.redo = append(d.redo, d.takeSnapshot())
	h := d.history[len(d.history)-1]
	d.history = d.history[:len(d.history)-1]
	d.restore(h)
	d.Dirty = true
	return true
}

// Redo re-applies the last undone operation. Reports whether anything was
// redone.
func (d *Document) Redo() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.redo) == 0 {
		return false
	}
	d.history = append(d.history, d.takeSnapshot())
	h := d.redo[len(d.redo)-1]
	d.redo = d.redo[:len(d.redo)-1]
	d.restore(h)
	d.Dirty = true
	return true
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
}
