// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

// This file is the layout layer: it turns paragraphs into wrapped "visual
// lines", renders them (with per-line alignment and the caret), and provides
// cursor movement that works on those visual lines instead of whole
// paragraphs.
//
// Wrapping is done here rather than by tview so that the document model knows
// exactly which screen row every character lands on. That is what makes
// aligned wrapped paragraphs, Up/Down, Home/End, Page Up/Down and
// scroll-to-cursor possible.

import (
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

const (
	// tabWidth is the number of cells a tab occupies on screen.
	tabWidth = 4

	// Caret is drawn as a highlighted cell (or a block after the last char).
	caretOpen  = "[-:-:-][black:#f0d090]"
	caretClose = "[-:-:-]"
	caretGlyph = "▌"

	// Selected text is drawn in these colours (attributes such as bold are kept).
	selFg = "#e8ecf4"
	selBg = "#3e5c7a"
)

// visualLine is one wrapped row of a paragraph, expressed as rune offsets
// into the paragraph's plain text.
//
//	Start  first rune on the row
//	End    one past the last rune on the row (includes trailing blanks that
//	       "hang" off the end of a wrapped line)
//	CEnd   one past the last visible (non-blank) rune; used for alignment
type visualLine struct {
	Start int
	End   int
	CEnd  int
}

func isBlank(r rune) bool { return r == ' ' || r == '\t' }

// cellWidth is the number of terminal cells a rune takes.
func cellWidth(r rune) int {
	if r == '\t' {
		return tabWidth
	}
	return runewidth.RuneWidth(r)
}

// widthOf is the number of terminal cells a run of runes takes.
func widthOf(rs []rune) int {
	w := 0
	for _, r := range rs {
		w += cellWidth(r)
	}
	return w
}

// wrapRunes splits text into visual lines no wider than width cells.
// Lines break after blanks where possible; a word longer than a whole line is
// broken mid-word. Blanks at a break hang off the end of the line instead of
// pushing the next word over. An empty paragraph is one empty line.
func wrapRunes(rs []rune, width int) []visualLine {
	if width < 1 {
		width = 1
	}
	n := len(rs)
	if n == 0 {
		return []visualLine{{}}
	}
	lines := make([]visualLine, 0, n/width+1)
	start := 0
	for start < n {
		end := n
		lineW := 0
		hasContent := false
		breakAt := -1
		for i := start; i < n; i++ {
			r := rs[i]
			w := cellWidth(r)
			if isBlank(r) {
				// Allow one hanging cell past width; beyond that, break.
				if i > start && lineW+w > width+1 {
					end = i
					break
				}
				lineW += w
				if hasContent {
					breakAt = i + 1
				}
				continue
			}
			// (A line that is only leading blanks may break before a word
			// that does not fit after them, so lines never exceed width.)
			if (hasContent || lineW > 0) && lineW+w > width {
				if breakAt > start {
					end = breakAt
				} else {
					end = i
				}
				break
			}
			lineW += w
			hasContent = true
		}
		cend := end
		for cend > start && isBlank(rs[cend-1]) {
			cend--
		}
		lines = append(lines, visualLine{Start: start, End: end, CEnd: cend})
		start = end
	}
	return lines
}

// lineIndexFor returns the index of the visual line that owns offset off.
// An offset that sits exactly on a soft-wrap boundary belongs to the line
// that starts there.
func lineIndexFor(lines []visualLine, off int) int {
	for i := len(lines) - 1; i > 0; i-- {
		if off >= lines[i].Start {
			return i
		}
	}
	return 0
}

// lineMaxOffset is the furthest offset the cursor may take on a line.
// On the last line of a paragraph that is the end of the paragraph; on a
// wrapped line it is the end of the visible text (the offset one past it
// belongs to the next line).
func lineMaxOffset(ln visualLine, last bool) int {
	if last {
		return ln.End
	}
	m := ln.End - 1
	if ln.CEnd < m {
		m = ln.CEnd
	}
	if m < ln.Start {
		m = ln.Start
	}
	return m
}

// offsetAtCol returns the cursor offset on a line for a content column
// (cells from the start of the line's text).
func offsetAtCol(rs []rune, ln visualLine, last bool, col int) int {
	limit := lineMaxOffset(ln, last)
	o := ln.Start
	cells := 0
	for o < limit {
		w := cellWidth(rs[o])
		if cells+w > col {
			break
		}
		cells += w
		o++
	}
	return o
}

// alignPad is the number of blank cells put before a line's text so it is
// left/centre/right aligned within wrapWidth cells.
func alignPad(a Align, wrapWidth, contentWidth int) int {
	if wrapWidth <= contentWidth {
		return 0
	}
	switch a {
	case AlignRight:
		return wrapWidth - contentWidth
	case AlignCenter:
		return (wrapWidth - contentWidth) / 2
	}
	return 0
}

// wrapWidth is the widest a line of text may be. Two cells are kept free on
// the right: one for the caret block and one of slack, so a full line plus
// the caret can never spill onto the next screen row.
func (d *Document) wrapWidth() int {
	w := d.DisplayWidth - 2
	if w < 1 {
		w = 1
	}
	return w
}

// displayText prepares text for tview: underline decoration, tag escaping
// (so "[red]" typed by the user is shown literally) and tab expansion.
func displayText(s string, style Style) string {
	s = decorateRunText(s, style)
	s = tview.Escape(s)
	return strings.ReplaceAll(s, "\t", strings.Repeat(" ", tabWidth))
}

// writeRange writes the runes [from, to) of paragraph p with their styles.
// The rune at offset caret is drawn as the caret cell, and runes in
// [selFrom, selTo) are drawn highlighted. Pass -1 to turn either off.
func writeRange(b *strings.Builder, p Paragraph, from, to, caret, selFrom, selTo int) {
	pos := 0
	for _, run := range p.Runs {
		runes := []rune(run.Text)
		rs, re := pos, pos+len(runes)
		pos = re
		if re <= from || rs >= to {
			continue
		}
		lo := max(from, rs) - rs
		hi := min(to, re) - rs

		// Cut the visible part of this run wherever the look changes.
		cuts := []int{lo, hi}
		for _, c := range []int{caret - rs, caret - rs + 1, selFrom - rs, selTo - rs} {
			if c > lo && c < hi {
				cuts = append(cuts, c)
			}
		}
		sort.Ints(cuts)

		for i := 0; i+1 < len(cuts); i++ {
			a, z := cuts[i], cuts[i+1]
			if a == z {
				continue
			}
			seg := runes[a:z]
			abs := rs + a

			switch {
			case caret >= 0 && abs == caret:
				if seg[0] == '\t' {
					// Highlight one cell of the tab, not all of it.
					b.WriteString(caretOpen + " " + caretClose)
					b.WriteString(strings.Repeat(" ", tabWidth-1))
				} else {
					b.WriteString(caretOpen)
					b.WriteString(displayText(string(seg), run.Style))
					b.WriteString(caretClose)
				}
			case selFrom >= 0 && abs >= selFrom && rs+z <= selTo:
				open, cl := styleTags(run.Style, selFg, selBg)
				b.WriteString(open)
				b.WriteString(displayText(string(seg), run.Style))
				b.WriteString(cl)
			default:
				open, cl := styleTags(run.Style, run.Fg, run.Bg)
				b.WriteString(open)
				b.WriteString(displayText(string(seg), run.Style))
				b.WriteString(cl)
			}
		}
	}
}

// render builds the tview-tagged text for the whole document, one screen row
// per visual line, and returns it together with the screen row (0-based) the
// cursor is on. Caller holds the lock.
func (d *Document) render(withCaret bool) (string, int) {
	var b strings.Builder
	if len(d.Paragraphs) == 0 {
		if withCaret {
			b.WriteString(caretOpen + caretGlyph + caretClose)
		}
		return b.String(), 0
	}

	ww := d.wrapWidth()
	maxCol := d.DisplayWidth - 1 // never draw further right than this column
	row, cursorRow := 0, 0
	sp, so, ep, eo, selOK := d.selectionRange()

	for pi, p := range d.Paragraphs {
		rs := []rune(plainOf(p))
		lines := wrapRunes(rs, ww)

		cur, curLine := -1, -1
		if pi == d.CursorPara {
			cur = min(max(d.CursorOff, 0), len(rs))
			curLine = lineIndexFor(lines, cur)
		}

		// Which part of this paragraph is selected, and does the selection
		// continue past its end (over the paragraph break)?
		selFrom, selTo, breakSel := -1, -1, false
		if selOK && pi >= sp && pi <= ep {
			from, to := 0, len(rs)
			if pi == sp {
				from = so
			}
			if pi == ep {
				to = eo
			}
			if from < to {
				selFrom, selTo = from, to
			}
			breakSel = pi < ep
		}

		for li, ln := range lines {
			if row > 0 {
				b.WriteByte('\n')
			}
			caret := -1
			hasCaret := false
			if li == curLine {
				cursorRow = row
				if withCaret {
					hasCaret = true
					caret = cur
				}
			}
			row++

			cw := widthOf(rs[ln.Start:ln.CEnd])
			pad := alignPad(p.Align, ww, cw)
			if pad > 0 {
				b.WriteString(strings.Repeat(" ", pad))
			}

			// Show hanging blanks only as far as they fit on screen.
			vEnd := ln.CEnd
			room := maxCol - pad - cw
			for vEnd < ln.End {
				w := cellWidth(rs[vEnd])
				if w > room {
					break
				}
				room -= w
				vEnd++
			}

			writeRange(&b, p, ln.Start, vEnd, caret, selFrom, selTo)
			glyph := hasCaret && (caret < ln.Start || caret >= vEnd)
			if glyph {
				// Caret is past the visible text (end of paragraph, or in
				// clipped blanks): draw it as a block after the text.
				b.WriteString(caretOpen + caretGlyph + caretClose)
			}

			// Show a selected paragraph break as one highlighted blank at the
			// end of the paragraph, if there is room for it.
			if breakSel && li == len(lines)-1 {
				used := pad + cw + widthOf(rs[ln.CEnd:vEnd])
				if glyph {
					used++
				}
				if used+1 <= d.DisplayWidth {
					open, cl := styleTags(StyleNone, selFg, selBg)
					b.WriteString(open + " " + cl)
				}
			}
		}
	}
	// Final reset so styles cannot bleed past the last cell.
	b.WriteString("[-:-:-]")
	return b.String(), cursorRow
}

// StyledText returns a tview-friendly string with colour/style tags for
// display, wrapped to DisplayWidth and aligned per line.
func (d *Document) StyledText() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	s, _ := d.render(false)
	return s
}

// StyledTextWithCursor is like StyledText but also draws a visible caret at
// the current position.
func (d *Document) StyledTextWithCursor() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	s, _ := d.render(true)
	return s
}

// CursorRow returns the 0-based screen row (visual line, counted from the top
// of the document) that the cursor is on.
func (d *Document) CursorRow() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ww := d.wrapWidth()
	row := 0
	for pi, p := range d.Paragraphs {
		rs := []rune(plainOf(p))
		lines := wrapRunes(rs, ww)
		if pi == d.CursorPara {
			off := min(max(d.CursorOff, 0), len(rs))
			return row + lineIndexFor(lines, off)
		}
		row += len(lines)
	}
	return row
}

// ---------------------------------------------------------------------------
// Navigation on visual lines
// ---------------------------------------------------------------------------

// curLayout returns the cursor paragraph's runes, its visual lines and the
// index of the line the cursor is on. Caller holds the lock and has called
// ensureCursor.
func (d *Document) curLayout() ([]rune, []visualLine, int) {
	rs := []rune(plainOf(d.Paragraphs[d.CursorPara]))
	lines := wrapRunes(rs, d.wrapWidth())
	return rs, lines, lineIndexFor(lines, d.CursorOff)
}

// goalColumn is the screen column Up/Down try to keep. It is remembered
// across consecutive vertical moves so a short line in between does not make
// the cursor forget where it was; any other movement or edit changes the
// cursor position and so naturally invalidates it.
func (d *Document) goalColumn(p Paragraph, rs []rune, ln visualLine) int {
	if d.goalSet && d.goalPara == d.CursorPara && d.goalOff == d.CursorOff {
		return d.goalCol
	}
	pad := alignPad(p.Align, d.wrapWidth(), widthOf(rs[ln.Start:ln.CEnd]))
	return pad + widthOf(rs[ln.Start:d.CursorOff])
}

// moveVertical moves the cursor one visual line up (dir < 0) or down
// (dir > 0), crossing paragraph boundaries. At the very top / bottom of the
// document it jumps to the start / end. Caller holds the lock and has called
// ensureCursor.
func (d *Document) moveVertical(dir int) {
	ww := d.wrapWidth()
	p := d.Paragraphs[d.CursorPara]
	rs := []rune(plainOf(p))
	lines := wrapRunes(rs, ww)
	li := lineIndexFor(lines, d.CursorOff)
	col := d.goalColumn(p, rs, lines[li])

	tpi, trs, tlines, tl := d.CursorPara, rs, lines, li+dir
	switch {
	case tl < 0:
		if tpi == 0 {
			d.CursorOff = 0
			return
		}
		tpi--
		trs = []rune(plainOf(d.Paragraphs[tpi]))
		tlines = wrapRunes(trs, ww)
		tl = len(tlines) - 1
	case tl >= len(lines):
		if tpi+1 >= len(d.Paragraphs) {
			d.CursorOff = len(rs)
			return
		}
		tpi++
		trs = []rune(plainOf(d.Paragraphs[tpi]))
		tlines = wrapRunes(trs, ww)
		tl = 0
	}

	ln := tlines[tl]
	pad := alignPad(d.Paragraphs[tpi].Align, ww, widthOf(trs[ln.Start:ln.CEnd]))
	d.CursorPara = tpi
	d.CursorOff = offsetAtCol(trs, ln, tl == len(tlines)-1, max(col-pad, 0))
	d.goalSet = true
	d.goalCol = col
	d.goalPara = d.CursorPara
	d.goalOff = d.CursorOff
}

// MoveUp moves the cursor up one visual (wrapped) line.
func (d *Document) MoveUp() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.moveVertical(-1)
}

// MoveDown moves the cursor down one visual (wrapped) line.
func (d *Document) MoveDown() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.moveVertical(1)
}

// MovePage moves the cursor by rows visual lines (negative = up), keeping the
// column. It stops at the start / end of the document.
func (d *Document) MovePage(rows int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	dir := 1
	if rows < 0 {
		dir = -1
		rows = -rows
	}
	for i := 0; i < rows; i++ {
		pp, po := d.CursorPara, d.CursorOff
		d.moveVertical(dir)
		if d.CursorPara == pp && d.CursorOff == po {
			break
		}
	}
}

// MoveLineStart moves to the start of the current visual line.
func (d *Document) MoveLineStart() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.goalSet = false
	_, lines, li := d.curLayout()
	d.CursorOff = lines[li].Start
}

// MoveLineEnd moves to the end of the current visual line.
func (d *Document) MoveLineEnd() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.goalSet = false
	_, lines, li := d.curLayout()
	d.CursorOff = lineMaxOffset(lines[li], li == len(lines)-1)
}

// MoveDocStart moves to the very start of the document.
func (d *Document) MoveDocStart() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.goalSet = false
	d.CursorPara = 0
	d.CursorOff = 0
}

// MoveDocEnd moves to the very end of the document.
func (d *Document) MoveDocEnd() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.goalSet = false
	d.CursorPara = len(d.Paragraphs) - 1
	d.CursorOff = len([]rune(plainOf(d.Paragraphs[d.CursorPara])))
}
