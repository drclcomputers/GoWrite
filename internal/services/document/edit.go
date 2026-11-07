// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

func (d *Document) InsertRune(r rune) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.pushHistory()
	d.deleteSelectionLocked()

	if r == '\n' {
		left, right := d.splitParaAtCursor()
		d.Paragraphs[d.CursorPara] = left
		d.Paragraphs = append(d.Paragraphs[:d.CursorPara+1],
			append([]Paragraph{right}, d.Paragraphs[d.CursorPara+1:]...)...)
		d.CursorPara++
		d.CursorOff = 0
		d.Dirty = true
		return
	}

	d.insertIntoRuns(string(r), d.ActiveStyle, d.ActiveFg, d.ActiveBg)
	d.CursorOff++
	d.Dirty = true
}

func sameRunStyle(r Run, style Style, fg, bg string) bool {
	return r.Style == style && r.Fg == fg && r.Bg == bg
}

func (d *Document) insertIntoRuns(text string, style Style, fg, bg string) {
	p := &d.Paragraphs[d.CursorPara]
	off := d.CursorOff
	pos := 0
	for i := range p.Runs {
		runLen := len([]rune(p.Runs[i].Text))
		if pos+runLen >= off {
			local := off - pos
			runes := []rune(p.Runs[i].Text)
			if sameRunStyle(p.Runs[i], style, fg, bg) {
				p.Runs[i].Text = string(runes[:local]) + text + string(runes[local:])
				return
			}
			left := string(runes[:local])
			right := string(runes[local:])
			newRuns := make([]Run, 0, len(p.Runs)+2)
			newRuns = append(newRuns, p.Runs[:i]...)
			if left != "" {
				newRuns = append(newRuns, Run{Text: left, Style: p.Runs[i].Style, Fg: p.Runs[i].Fg, Bg: p.Runs[i].Bg})
			}
			newRuns = append(newRuns, Run{Text: text, Style: style, Fg: fg, Bg: bg})
			if right != "" {
				newRuns = append(newRuns, Run{Text: right, Style: p.Runs[i].Style, Fg: p.Runs[i].Fg, Bg: p.Runs[i].Bg})
			}
			newRuns = append(newRuns, p.Runs[i+1:]...)
			p.Runs = newRuns
			return
		}
		pos += runLen
	}
	if len(p.Runs) > 0 && sameRunStyle(p.Runs[len(p.Runs)-1], style, fg, bg) {
		p.Runs[len(p.Runs)-1].Text += text
	} else {
		p.Runs = append(p.Runs, Run{Text: text, Style: style, Fg: fg, Bg: bg})
	}
}

func (d *Document) splitParaAtCursor() (left, right Paragraph) {
	p := d.Paragraphs[d.CursorPara]
	off := d.CursorOff
	pos := 0
	for i, run := range p.Runs {
		runes := []rune(run.Text)
		runLen := len(runes)
		if pos+runLen >= off {
			local := off - pos
			leftRuns := append([]Run{}, p.Runs[:i]...)
			if local > 0 {
				leftRuns = append(leftRuns, Run{Text: string(runes[:local]), Style: run.Style, Fg: run.Fg, Bg: run.Bg})
			}
			if len(leftRuns) == 0 {
				leftRuns = []Run{{Text: "", Style: StyleNone}}
			}
			rightRuns := []Run{}
			if local < runLen {
				rightRuns = append(rightRuns, Run{Text: string(runes[local:]), Style: run.Style, Fg: run.Fg, Bg: run.Bg})
			}
			rightRuns = append(rightRuns, p.Runs[i+1:]...)
			if len(rightRuns) == 0 {
				rightRuns = []Run{{Text: "", Style: StyleNone}}
			}
			return Paragraph{Runs: leftRuns, Align: p.Align}, Paragraph{Runs: rightRuns, Align: p.Align}
		}
		pos += runLen
	}
	return p, Paragraph{Runs: []Run{{Text: "", Style: StyleNone}}, Align: p.Align}
}

func (d *Document) Backspace() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	hasSel := d.hasAnchor && (d.anchorPara != d.CursorPara || d.anchorOff != d.CursorOff)
	if !hasSel && d.CursorPara == 0 && d.CursorOff == 0 {
		return
	}
	d.pushHistory()
	if d.deleteSelectionLocked() {
		return
	}

	if d.CursorOff > 0 {
		d.deleteRange(d.CursorOff-1, d.CursorOff)
		d.CursorOff--
		d.Dirty = true
		return
	}
	if d.CursorPara == 0 {
		return
	}
	prevPlain := plainOf(d.Paragraphs[d.CursorPara-1])
	d.Paragraphs[d.CursorPara-1].Runs = append(d.Paragraphs[d.CursorPara-1].Runs, d.Paragraphs[d.CursorPara].Runs...)
	d.Paragraphs = append(d.Paragraphs[:d.CursorPara], d.Paragraphs[d.CursorPara+1:]...)
	d.CursorPara--
	d.CursorOff = len([]rune(prevPlain))
	d.Dirty = true
}

func (d *Document) Delete() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	hasSel := d.hasAnchor && (d.anchorPara != d.CursorPara || d.anchorOff != d.CursorOff)
	plain := []rune(plainOf(d.Paragraphs[d.CursorPara]))
	hasCharAfter := d.CursorOff < len(plain)
	hasParaAfter := d.CursorPara+1 < len(d.Paragraphs)
	if !hasSel && !hasCharAfter && !hasParaAfter {
		return
	}
	d.pushHistory()
	if d.deleteSelectionLocked() {
		return
	}

	if hasCharAfter {
		d.deleteRange(d.CursorOff, d.CursorOff+1)
		d.Dirty = true
		return
	}
	if !hasParaAfter {
		return
	}
	d.Paragraphs[d.CursorPara].Runs = append(d.Paragraphs[d.CursorPara].Runs, d.Paragraphs[d.CursorPara+1].Runs...)
	d.Paragraphs = append(d.Paragraphs[:d.CursorPara+1], d.Paragraphs[d.CursorPara+2:]...)
	d.Dirty = true
}

func (d *Document) deleteRange(from, to int) {
	p := &d.Paragraphs[d.CursorPara]
	var newRuns []Run
	pos := 0
	for _, run := range p.Runs {
		runes := []rune(run.Text)
		runLen := len(runes)
		runStart, runEnd := pos, pos+runLen
		pos = runEnd

		if runEnd <= from || runStart >= to {
			if run.Text != "" {
				newRuns = append(newRuns, run)
			}
			continue
		}
		keepLeft := from - runStart
		if keepLeft < 0 {
			keepLeft = 0
		}
		keepRight := to - runStart
		if keepRight > runLen {
			keepRight = runLen
		}
		if rest := string(runes[:keepLeft]) + string(runes[keepRight:]); rest != "" {
			kept := run
			kept.Text = rest
			newRuns = append(newRuns, kept)
		}
	}
	if len(newRuns) == 0 {
		newRuns = []Run{{Text: "", Style: StyleNone}}
	}
	p.Runs = newRuns
}

func (d *Document) InsertPageBreak() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.pushHistory()
	d.Paragraphs[d.CursorPara].PageBreak = true
	d.Dirty = true
}
