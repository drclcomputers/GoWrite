// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

import (
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"
	"github.com/rivo/tview"
)

const (
	tabWidth  = 4
	caretOpen = "[-:-:-][black:#f0d090]"
	caretClose = "[-:-:-]"
	caretGlyph = "▌"
	selFg     = "#e8ecf4"
	selBg     = "#3e5c7a"
)

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

// lineIndexFor returns the visual line owning off; a wrap boundary belongs to the next line.
func lineIndexFor(lines []visualLine, off int) int {
	for i := len(lines) - 1; i > 0; i-- {
		if off >= lines[i].Start {
			return i
		}
	}
	return 0
}

// lineMaxOffset is the furthest cursor offset on a line.
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

// offsetAtCol returns the cursor offset for a content column on a line.
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

// alignPad is the blank cells before a line to align it within wrapWidth.
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

func (d *Document) wrapWidth() int {
	w := d.Page.Width
	if w < 1 {
		w = 1
	}
	return w
}

func displayText(s string, style Style) string {
	s = decorateRunText(s, style)
	s = tview.Escape(s)
	return strings.ReplaceAll(s, "\t", strings.Repeat(" ", tabWidth))
}

// writeRange writes the runes [from, to) of paragraph p with their styles.
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

// render builds the tview-tagged text, chunking it into pages.
func (d *Document) render(withCaret bool) (string, int) {
	if len(d.Paragraphs) == 0 {
		if withCaret {
			return caretOpen + caretGlyph + caretClose, 0
		}
		return "", 0
	}

	ps := d.Page
	if ps.Width == 0 {
		ps = DefaultA4Portrait()
	}

	var b strings.Builder
	// Calculate left margin to center the page on screen
	margin := (d.DisplayWidth - ps.Width - 2) / 2
	if margin < 0 {
		margin = 0
	}

	writeMargin := func() {
		b.WriteString(strings.Repeat(" ", margin))
	}

	drawPageTop := func() {
		writeMargin()
		b.WriteString("┌" + strings.Repeat("─", ps.Width) + "┐\n")
	}
	drawPageBottom := func() {
		writeMargin()
		b.WriteString("└" + strings.Repeat("─", ps.Width) + "┘\n")
	}

	row, cursorRow := 0, 0
	pageRows := 0

	drawPageTop()
	row++

	sp, so, ep, eo, selOK := d.selectionRange()

	for pi, p := range d.Paragraphs {
		if p.PageBreak && pageRows > 0 {
			drawPageBottom()
			b.WriteString("\n")
			drawPageTop()
			row += 2
			pageRows = 0
		}

		rs := []rune(plainOf(p))
		lines := wrapRunes(rs, ps.Width)

		cur, curLine := -1, -1
		if pi == d.CursorPara {
			cur = min(max(d.CursorOff, 0), len(rs))
			curLine = lineIndexFor(lines, cur)
		}

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
			if pageRows >= ps.Height {
				drawPageBottom()
				b.WriteString("\n")
				drawPageTop()
				row += 2
				pageRows = 0
			}

			var lineBuf strings.Builder
			caret := -1
			hasCaret := false
			if li == curLine {
				cursorRow = row
				if withCaret {
					hasCaret = true
					caret = cur
				}
			}

			cw := widthOf(rs[ln.Start:ln.CEnd])
			pad := alignPad(p.Align, ps.Width, cw)
			if pad > 0 {
				lineBuf.WriteString(strings.Repeat(" ", pad))
			}

			vEnd := ln.CEnd
			room := ps.Width - pad - cw
			for vEnd < ln.End {
				w := cellWidth(rs[vEnd])
				if w > room {
					break
				}
				room -= w
				vEnd++
			}

			writeRange(&lineBuf, p, ln.Start, vEnd, caret, selFrom, selTo)
			glyph := hasCaret && (caret < ln.Start || caret >= vEnd)
			if glyph {
				lineBuf.WriteString(caretOpen + caretGlyph + caretClose)
			}

			if breakSel && li == len(lines)-1 {
				used := pad + cw + widthOf(rs[ln.CEnd:vEnd])
				if glyph {
					used++
				}
				if used+1 <= ps.Width {
					open, cl := styleTags(StyleNone, selFg, selBg)
					lineBuf.WriteString(open + " " + cl)
				}
			}

			// Draw the actual line of text inside the page border
			writeMargin()
			b.WriteString("│")
			b.WriteString(lineBuf.String())

			// Pad the rest of the page width
			lineWidth := pad + widthOf(rs[ln.Start:vEnd])
			if glyph {
				lineWidth++
			}
			if breakSel && li == len(lines)-1 {
				lineWidth++
			}
			// Tags are zero-width, so wrapping to ps.Width cannot overflow.
			if lineWidth < ps.Width {
				b.WriteString(strings.Repeat(" ", ps.Width-lineWidth))
			}
			b.WriteString("│\n")

			row++
			pageRows++
		}
	}

	// Fill remaining page rows with blank lines
	for pageRows < ps.Height {
		writeMargin()
		b.WriteString("│" + strings.Repeat(" ", ps.Width) + "│\n")
		row++
		pageRows++
	}

	drawPageBottom()

	return b.String(), cursorRow
}

func (d *Document) StyledText() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	s, _ := d.render(false)
	return s
}

func (d *Document) StyledTextWithCursor() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	s, _ := d.render(true)
	return s
}

// CursorRow returns the visual screen row.
func (d *Document) CursorRow() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	ps := d.Page
	if ps.Width == 0 {
		ps = DefaultA4Portrait()
	}

	row := 1 // Top border
	pageRows := 0

	for pi, p := range d.Paragraphs {
		if p.PageBreak && pageRows > 0 {
			row += 2 // Bottom border + gap + top border
			pageRows = 0
		}

		rs := []rune(plainOf(p))
		lines := wrapRunes(rs, ps.Width)

		for li := range lines {
			if pageRows >= ps.Height {
				row += 2
				pageRows = 0
			}

			if pi == d.CursorPara {
				off := min(max(d.CursorOff, 0), len(rs))
				if li == lineIndexFor(lines, off) {
					return row
				}
			}
			row++
			pageRows++
		}
	}
	return row
}


// Navigation on visual lines

// curLayout returns the cursor paragraph's runes, its lines and the cursor's
// line index. Caller holds the lock.
func (d *Document) curLayout() ([]rune, []visualLine, int) {
	rs := []rune(plainOf(d.Paragraphs[d.CursorPara]))
	lines := wrapRunes(rs, d.wrapWidth())
	return rs, lines, lineIndexFor(lines, d.CursorOff)
}

// goalColumn is the column Up/Down try to keep across consecutive vertical moves.
func (d *Document) goalColumn(p Paragraph, rs []rune, ln visualLine) int {
	if d.goalSet && d.goalPara == d.CursorPara && d.goalOff == d.CursorOff {
		return d.goalCol
	}
	pad := alignPad(p.Align, d.wrapWidth(), widthOf(rs[ln.Start:ln.CEnd]))
	return pad + widthOf(rs[ln.Start:d.CursorOff])
}

// moveVertical moves one visual line up (dir < 0) or down, crossing paragraphs;
// at the document edges it jumps to the start/end.
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

// MovePage moves by rows visual lines (negative = up), keeping the column.
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
