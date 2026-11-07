// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

import (
	"strings"
	"sync"
)

// The clipboard is shared by every document in the process, so copied text
// survives File > New / Open. It keeps styles, colours and alignment.
var (
	clipMu sync.Mutex
	clip   []Paragraph
)

func setClip(ps []Paragraph) {
	clipMu.Lock()
	defer clipMu.Unlock()
	clip = ps
}

// getClip returns a deep copy of the clipboard.
func getClip() []Paragraph {
	clipMu.Lock()
	defer clipMu.Unlock()
	out := make([]Paragraph, len(clip))
	for i, p := range clip {
		out[i] = Paragraph{Runs: cloneRuns(p.Runs), Align: p.Align}
	}
	return out
}

// ---------------------------------------------------------------------------
// Run helpers
// ---------------------------------------------------------------------------

func runeLen(runs []Run) int {
	n := 0
	for _, r := range runs {
		n += len([]rune(r.Text))
	}
	return n
}

func cloneRuns(runs []Run) []Run {
	return append([]Run(nil), runs...)
}

// normalizeRuns drops empty runs and merges neighbours that look the same.
// It never returns an empty slice: an empty paragraph is one empty run.
func normalizeRuns(runs []Run) []Run {
	out := make([]Run, 0, len(runs))
	for _, r := range runs {
		if r.Text == "" {
			continue
		}
		if n := len(out); n > 0 && sameRunStyle(out[n-1], r.Style, r.Fg, r.Bg) {
			out[n-1].Text += r.Text
			continue
		}
		out = append(out, r)
	}
	if len(out) == 0 {
		return []Run{{Text: "", Style: StyleNone}}
	}
	return out
}

// sliceRuns returns copies of the parts of runs that cover runes [from, to).
func sliceRuns(runs []Run, from, to int) []Run {
	var out []Run
	pos := 0
	for _, run := range runs {
		runes := []rune(run.Text)
		rs, re := pos, pos+len(runes)
		pos = re
		if re <= from || rs >= to {
			continue
		}
		lo := max(from, rs) - rs
		hi := min(to, re) - rs
		piece := run
		piece.Text = string(runes[lo:hi])
		out = append(out, piece)
	}
	return out
}

// restyleRange applies fn to the part of paragraph pi covering [from, to),
// splitting runs at the edges and merging them back afterwards.
func (d *Document) restyleRange(pi, from, to int, fn func(*Run)) {
	p := &d.Paragraphs[pi]
	var out []Run
	pos := 0
	for _, run := range p.Runs {
		runes := []rune(run.Text)
		rs, re := pos, pos+len(runes)
		pos = re
		if re <= from || rs >= to {
			out = append(out, run)
			continue
		}
		lo := max(from, rs) - rs
		hi := min(to, re) - rs
		if lo > 0 {
			before := run
			before.Text = string(runes[:lo])
			out = append(out, before)
		}
		mid := run
		mid.Text = string(runes[lo:hi])
		fn(&mid)
		out = append(out, mid)
		if hi < len(runes) {
			after := run
			after.Text = string(runes[hi:])
			out = append(out, after)
		}
	}
	p.Runs = normalizeRuns(out)
}

// ---------------------------------------------------------------------------
// Selection range
// ---------------------------------------------------------------------------

// clampPos forces (paragraph, offset) into the document.
func (d *Document) clampPos(pi, off int) (int, int) {
	if pi < 0 {
		pi = 0
	}
	if pi >= len(d.Paragraphs) {
		pi = len(d.Paragraphs) - 1
	}
	n := len([]rune(plainOf(d.Paragraphs[pi])))
	if off < 0 {
		off = 0
	}
	if off > n {
		off = n
	}
	return pi, off
}

// selectionRange returns the selection as an ordered start and end position.
// ok is false when nothing is selected. Caller holds the lock.
func (d *Document) selectionRange() (sp, so, ep, eo int, ok bool) {
	if !d.hasAnchor || len(d.Paragraphs) == 0 {
		return
	}
	ap, ao := d.clampPos(d.anchorPara, d.anchorOff)
	cp, co := d.clampPos(d.CursorPara, d.CursorOff)
	if ap == cp && ao == co {
		return
	}
	if ap > cp || (ap == cp && ao > co) {
		ap, ao, cp, co = cp, co, ap, ao
	}
	return ap, ao, cp, co, true
}

// forEachSelected calls fn for the non-empty part of every selected
// paragraph. It reports whether there is a selection.
func (d *Document) forEachSelected(fn func(pi, from, to int)) bool {
	sp, so, ep, eo, ok := d.selectionRange()
	if !ok {
		return false
	}
	for pi := sp; pi <= ep; pi++ {
		from, to := 0, len([]rune(plainOf(d.Paragraphs[pi])))
		if pi == sp {
			from = so
		}
		if pi == ep {
			to = eo
		}
		if from < to {
			fn(pi, from, to)
		}
	}
	return true
}

// cursorParagraphs is the paragraph range that paragraph-level commands
// (alignment) apply to: the selected paragraphs, or just the cursor's. A
// selection that ends at the very start of a paragraph does not include it.
func (d *Document) cursorParagraphs() (first, last int) {
	sp, _, ep, eo, ok := d.selectionRange()
	if !ok {
		return d.CursorPara, d.CursorPara
	}
	if eo == 0 && ep > sp {
		ep--
	}
	return sp, ep
}

// selectedParagraphs returns the selected text as paragraphs (with styles).
func (d *Document) selectedParagraphs() []Paragraph {
	sp, so, ep, eo, ok := d.selectionRange()
	if !ok {
		return nil
	}
	out := make([]Paragraph, 0, ep-sp+1)
	for pi := sp; pi <= ep; pi++ {
		p := d.Paragraphs[pi]
		from, to := 0, runeLen(p.Runs)
		if pi == sp {
			from = so
		}
		if pi == ep {
			to = eo
		}
		out = append(out, Paragraph{
			Runs:  normalizeRuns(sliceRuns(p.Runs, from, to)),
			Align: p.Align,
		})
	}
	return out
}

// ---------------------------------------------------------------------------
// Public selection API
// ---------------------------------------------------------------------------

// HasSelection reports whether any text is selected.
func (d *Document) HasSelection() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	_, _, _, _, ok := d.selectionRange()
	return ok
}

// SelectionLen is the number of characters selected (paragraph breaks count
// as one each).
func (d *Document) SelectionLen() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	sp, so, ep, eo, ok := d.selectionRange()
	if !ok {
		return 0
	}
	if sp == ep {
		return eo - so
	}
	n := runeLen(d.Paragraphs[sp].Runs) - so + 1
	for pi := sp + 1; pi < ep; pi++ {
		n += runeLen(d.Paragraphs[pi].Runs) + 1
	}
	return n + eo
}

// BeginSelection starts a selection at the cursor if there is none yet. Call
// it just before moving the cursor with Shift held.
func (d *Document) BeginSelection() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if !d.hasAnchor {
		d.hasAnchor = true
		d.anchorPara = d.CursorPara
		d.anchorOff = d.CursorOff
	}
}

// ClearSelection drops the selection, leaving the cursor where it is.
func (d *Document) ClearSelection() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.hasAnchor = false
}

// CollapseSelection drops the selection and puts the cursor at its start or
// end (what Left / Right do when text is selected).
func (d *Document) CollapseSelection(toStart bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	sp, so, ep, eo, ok := d.selectionRange()
	d.hasAnchor = false
	d.goalSet = false
	if !ok {
		return
	}
	if toStart {
		d.CursorPara, d.CursorOff = sp, so
	} else {
		d.CursorPara, d.CursorOff = ep, eo
	}
}

// SelectAll selects the whole document.
func (d *Document) SelectAll() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.hasAnchor = true
	d.anchorPara, d.anchorOff = 0, 0
	d.CursorPara = len(d.Paragraphs) - 1
	d.CursorOff = runeLen(d.Paragraphs[d.CursorPara].Runs)
	d.goalSet = false
}

// SelectedText returns the selected text (paragraphs joined by \n).
func (d *Document) SelectedText() string {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	ps := d.selectedParagraphs()
	parts := make([]string, len(ps))
	for i, p := range ps {
		parts[i] = plainOf(p)
	}
	return strings.Join(parts, "\n")
}

// deleteSelectionLocked removes the selected text and puts the cursor where
// it was. It reports whether anything was deleted. Caller holds the lock and
// has called ensureCursor.
func (d *Document) deleteSelectionLocked() bool {
	sp, so, ep, eo, ok := d.selectionRange()
	d.hasAnchor = false
	if !ok {
		return false
	}
	d.goalSet = false
	if sp == ep {
		d.CursorPara = sp
		d.deleteRange(so, eo)
	} else {
		// Keep what is before the selection in the first paragraph and what
		// is after it in the last, joined into one paragraph.
		head := sliceRuns(d.Paragraphs[sp].Runs, 0, so)
		tail := sliceRuns(d.Paragraphs[ep].Runs, eo, runeLen(d.Paragraphs[ep].Runs))
		d.Paragraphs[sp] = Paragraph{
			Runs:  normalizeRuns(append(head, tail...)),
			Align: d.Paragraphs[sp].Align,
		}
		d.Paragraphs = append(d.Paragraphs[:sp+1], d.Paragraphs[ep+1:]...)
	}
	d.CursorPara, d.CursorOff = sp, so
	d.Dirty = true
	return true
}

// DeleteSelection deletes the selected text. It reports whether there was any.
func (d *Document) DeleteSelection() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if _, _, _, _, ok := d.selectionRange(); !ok {
		return false
	}
	d.pushHistory()
	return d.deleteSelectionLocked()
}

// ---------------------------------------------------------------------------
// Clipboard
// ---------------------------------------------------------------------------

// Copy copies the selection to the clipboard. It reports whether there was a
// selection.
func (d *Document) Copy() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	ps := d.selectedParagraphs()
	if len(ps) == 0 {
		return false
	}
	setClip(ps)
	return true
}

// Cut copies the selection to the clipboard and deletes it.
func (d *Document) Cut() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	ps := d.selectedParagraphs()
	if len(ps) == 0 {
		return false
	}
	d.pushHistory()
	setClip(ps)
	d.deleteSelectionLocked()
	return true
}

// Paste inserts the clipboard at the cursor, replacing the selection. It
// reports whether there was anything to paste.
func (d *Document) Paste() bool {
	items := getClip()
	if len(items) == 0 {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.pushHistory()
	d.deleteSelectionLocked()

	left, right := d.splitParaAtCursor()
	if len(items) == 1 {
		runs := append(cloneRuns(left.Runs), items[0].Runs...)
		runs = append(runs, right.Runs...)
		d.Paragraphs[d.CursorPara] = Paragraph{Runs: normalizeRuns(runs), Align: left.Align}
		d.CursorOff += runeLen(items[0].Runs)
	} else {
		last := items[len(items)-1]
		first := Paragraph{
			Runs:  normalizeRuns(append(cloneRuns(left.Runs), items[0].Runs...)),
			Align: left.Align,
		}
		lastP := Paragraph{
			Runs:  normalizeRuns(append(cloneRuns(last.Runs), right.Runs...)),
			Align: last.Align,
		}
		newParas := []Paragraph{first}
		newParas = append(newParas, items[1:len(items)-1]...)
		newParas = append(newParas, lastP)

		tail := append([]Paragraph(nil), d.Paragraphs[d.CursorPara+1:]...)
		d.Paragraphs = append(append(d.Paragraphs[:d.CursorPara], newParas...), tail...)
		d.CursorPara += len(items) - 1
		d.CursorOff = runeLen(last.Runs)
	}
	d.goalSet = false
	d.Dirty = true
	return true
}

// ---------------------------------------------------------------------------
// Formatting the selection
// ---------------------------------------------------------------------------

// ToggleStyleOnSelection toggles a style on the selected text.
func (d *Document) ToggleStyleOnSelection(s Style) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if _, _, _, _, ok := d.selectionRange(); !ok {
		return false
	}
	d.pushHistory()
	all := true
	d.forEachSelected(func(pi, from, to int) {
		for _, r := range sliceRuns(d.Paragraphs[pi].Runs, from, to) {
			if r.Text != "" && r.Style&s == 0 {
				all = false
			}
		}
	})
	d.forEachSelected(func(pi, from, to int) {
		d.restyleRange(pi, from, to, func(r *Run) {
			if all {
				r.Style &^= s
			} else {
				r.Style |= s
			}
		})
	})
	d.Dirty = true
	return true
}

// SetSelectionColor sets the text (or background) colour of the selected
// text. An empty value restores the terminal default.
func (d *Document) SetSelectionColor(background bool, value string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if _, _, _, _, ok := d.selectionRange(); !ok {
		return false
	}
	d.pushHistory()
	d.forEachSelected(func(pi, from, to int) {
		d.restyleRange(pi, from, to, func(r *Run) {
			if background {
				r.Bg = value
			} else {
				r.Fg = value
			}
		})
	})
	d.Dirty = true
	return true
}


/// ApplyActiveToSelection applies the active typing style and colours to the
// selected text.
func (d *Document) ApplyActiveToSelection() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if _, _, _, _, ok := d.selectionRange(); !ok {
		return false
	}
	d.pushHistory()
	style, fg, bg := d.ActiveStyle, d.ActiveFg, d.ActiveBg
	d.forEachSelected(func(pi, from, to int) {
		d.restyleRange(pi, from, to, func(r *Run) {
			r.Style |= style
			if fg != "" {
				r.Fg = fg
			}
			if bg != "" {
				r.Bg = bg
			}
		})
	})
	d.Dirty = true
	return true
}
