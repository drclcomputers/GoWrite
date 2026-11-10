// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

import (
	"strconv"
	"strings"
	"unicode"

	"github.com/mattn/go-runewidth"
)

// pageGeo is the page layout derived from PageSettings.
type pageGeo struct {
	ps          PageSettings
	bodyW       int // text columns between the margins
	bodyH       int // text rows per page
	hdr, ftr    bool
	top, bottom int // rows above/below the text: header/footer plus margin
}

func (d *Document) geometry() pageGeo {
	ps := d.Page
	if ps.Width == 0 {
		ps = DefaultA4Portrait()
	}
	ps = ps.Normalized()
	g := pageGeo{ps: ps, hdr: ps.hasHeader(), ftr: ps.hasFooter()}
	g.top, g.bottom = ps.MarginTop, ps.MarginBottom
	if g.hdr {
		g.top++
	}
	if g.ftr {
		g.bottom++
	}
	g.bodyW = max(ps.Width-ps.MarginLeft-ps.MarginRight, 1)
	g.bodyH = max(ps.Height-g.top-g.bottom, 1)
	return g
}

// pageWalker receives the pagination events. Every callback is optional.
// Rows count screen rows from the top of the first page.
type pageWalker struct {
	para      func(pi int, rs []rune, lines []visualLine)
	startPage func(page, row int)
	line      func(li, page, row int)
	space     func(page, row int)
	endPage   func(page, used int, more bool)
}

// walk paginates the document once and reports it to w; it returns the
// page count. Rendering and cursor maths share it so they cannot disagree.
func (d *Document) walk(g pageGeo, w pageWalker) int {
	page, used, row := 0, 0, 0

	begin := func() {
		page++
		used = 0
		if w.startPage != nil {
			w.startPage(page, row)
		}
		row += 1 + g.top
	}
	end := func(more bool) {
		if w.endPage != nil {
			w.endPage(page, used, more)
		}
		row += g.bodyH - used + g.bottom + 1
		if more {
			row++
		}
	}
	addSpace := func() {
		if used >= g.bodyH {
			end(true)
			begin()
		}
		if w.space != nil {
			w.space(page, row)
		}
		row++
		used++
	}

	begin()
	for pi, p := range d.Paragraphs {
		if p.PageBreak {
			end(true)
			begin()
		}
		rs := []rune(plainOf(p))
		lines := wrapRunes(rs, g.bodyW)
		if w.para != nil {
			w.para(pi, rs, lines)
		}
		for li := range lines {
			if used >= g.bodyH {
				end(true)
				begin()
			}
			if w.line != nil {
				w.line(li, page, row)
			}
			row++
			used++
		}
		if pi+1 < len(d.Paragraphs) && !d.Paragraphs[pi+1].PageBreak {
			for range min(max(p.SpaceAfter, 0), maxParagraphSpaceAfter) {
				addSpace()
			}
		}
	}
	end(false)
	return page
}

// PageInfo returns the cursor's page and the page count (both 1-based).
func (d *Document) PageInfo() (current, total int) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	pi, cur := 0, 1
	var rs []rune
	var lines []visualLine
	total = d.walk(d.geometry(), pageWalker{
		para: func(i int, r []rune, l []visualLine) { pi, rs, lines = i, r, l },
		line: func(li, page, _ int) {
			if pi == d.CursorPara && li == lineIndexFor(lines, min(max(d.CursorOff, 0), len(rs))) {
				cur = page
			}
		},
	})
	return cur, total
}

func (pn PageNumbers) text(page, pages int) string {
	n, total := strconv.Itoa(page), strconv.Itoa(pages)
	switch pn.Format {
	case FormatPage:
		return "Page " + n
	case FormatOf:
		return n + " of " + total
	case FormatPageOf:
		return "Page " + n + " of " + total
	}
	return n
}

func cleanLine(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, s)
}

// runningLine builds the header (here) or footer text for a page: plain
// text exactly w cells wide, tokens expanded and the page number added.
func (d *Document) runningLine(h HeaderFooter, pn PageNumbers, here bool, page, pages, w int) string {
	rep := strings.NewReplacer("{page}", strconv.Itoa(page), "{pages}", strconv.Itoa(pages), "{title}", d.Title)
	slots := [3]string{rep.Replace(h.Left), rep.Replace(h.Center), rep.Replace(h.Right)}
	if here {
		i := 0
		switch pn.Align {
		case AlignCenter:
			i = 1
		case AlignRight:
			i = 2
		}
		slots[i] = strings.TrimSpace(slots[i] + " " + pn.text(page, pages))
	}
	return composeRow(cleanLine(slots[0]), cleanLine(slots[1]), cleanLine(slots[2]), w)
}

// composeRow lays out left, centre and right text on one row exactly w cells
// wide, truncating from the centre outwards when they do not all fit.
func composeRow(left, center, right string, w int) string {
	if w < 1 {
		return ""
	}
	right = runewidth.Truncate(right, w, "")
	rw := runewidth.StringWidth(right)
	gapR := 0
	if rw > 0 {
		gapR = 1
	}

	left = runewidth.Truncate(left, max(w-rw-gapR, 0), "")
	lw := runewidth.StringWidth(left)

	var b strings.Builder
	b.WriteString(left)
	pos := lw

	if center != "" {
		gapL := 0
		if lw > 0 {
			gapL = 1
		}
		cw := runewidth.StringWidth(center)
		start := max((w-cw)/2, pos+gapL)
		center = runewidth.Truncate(center, max(w-rw-gapR-start, 0), "")
		if cw = runewidth.StringWidth(center); cw > 0 {
			b.WriteString(strings.Repeat(" ", start-pos))
			b.WriteString(center)
			pos = start + cw
		}
	}

	b.WriteString(strings.Repeat(" ", max(w-rw-pos, 0)))
	b.WriteString(right)
	return b.String()
}
