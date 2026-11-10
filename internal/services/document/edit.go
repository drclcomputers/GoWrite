// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

import (
	"fmt"
	"strconv"
	"strings"
)

func (d *Document) InsertRune(r rune) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if d.inTableLocked() {
		return
	}
	if r == '\n' {
		d.pushHistory()
	} else {
		d.pushHistoryCoalesced(editTyping, r)
	}
	d.deleteSelectionLocked()

	if r == '\n' {
		// List auto-continuation: if the current paragraph is a list item,
		// either continue with the next marker or exit the list when empty.
		plain := plainOf(d.Paragraphs[d.CursorPara])
		marker, rest, ok := parseListMarker(plain)
		if ok {
			// Cursor at end of a non-empty list item → continue list.
			// Empty list item (only the marker) → stop the list.
			content := strings.TrimSpace(rest)
			atEnd := d.CursorOff >= len([]rune(plain))
			if content == "" && (d.CursorOff <= len([]rune(marker)) || atEnd) {
				// Exit list: turn this paragraph into a plain empty one.
				d.Paragraphs[d.CursorPara].Runs = []Run{{Text: "", Style: StyleNone}}
				d.CursorOff = 0
				d.Dirty = true
				return
			}
			if atEnd {
				next := nextListMarker(marker)
				left, _ := d.splitParaAtCursor()
				d.Paragraphs[d.CursorPara] = left
				newPara := Paragraph{
					Runs:       []Run{{Text: next, Style: d.ActiveStyle, Fg: d.ActiveFg, Bg: d.ActiveBg}},
					Align:      left.Align,
					ParaStyle:  left.ParaStyle,
					SpaceAfter: left.SpaceAfter,
				}
				d.Paragraphs = append(d.Paragraphs[:d.CursorPara+1],
					append([]Paragraph{newPara}, d.Paragraphs[d.CursorPara+1:]...)...)
				d.CursorPara++
				d.CursorOff = len([]rune(next))
				d.Dirty = true
				return
			}
		}

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
	d.endCoalescedEdit()
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
			return Paragraph{
				Runs:       leftRuns,
				Align:      p.Align,
				ParaStyle:  p.ParaStyle,
				SpaceAfter: p.SpaceAfter,
				PageBreak:  p.PageBreak,
			}, Paragraph{
				Runs:       rightRuns,
				Align:      p.Align,
				ParaStyle:  p.ParaStyle,
				SpaceAfter: p.SpaceAfter,
			}
		}
		pos += runLen
	}
	return p, Paragraph{
		Runs:       []Run{{Text: "", Style: StyleNone}},
		Align:      p.Align,
		ParaStyle:  p.ParaStyle,
		SpaceAfter: p.SpaceAfter,
	}
}

func (d *Document) Backspace() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if d.inTableLocked() {
		return
	}
	hasSel := d.hasAnchor && (d.anchorPara != d.CursorPara || d.anchorOff != d.CursorOff)
	if !hasSel && d.CursorPara == 0 && d.CursorOff == 0 {
		return
	}
	if hasSel || d.CursorOff == 0 {
		d.pushHistory()
	} else {
		d.pushHistoryCoalesced(editBackspace, 0)
	}
	if d.deleteSelectionLocked() {
		return
	}

	// If the cursor is immediately after a list marker, remove the whole marker
	// (exits the list item cleanly instead of leaving a dangling "1.").
	if d.CursorOff > 0 {
		plain := plainOf(d.Paragraphs[d.CursorPara])
		if marker, _, ok := parseListMarker(plain); ok {
			mLen := len([]rune(marker))
			if d.CursorOff == mLen {
				d.deleteRange(0, mLen)
				d.CursorOff = 0
				d.Dirty = true
				return
			}
		}
		d.deleteRange(d.CursorOff-1, d.CursorOff)
		d.CursorOff--
		d.Dirty = true
		d.endCoalescedEdit()
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
	if d.inTableLocked() {
		return
	}
	hasSel := d.hasAnchor && (d.anchorPara != d.CursorPara || d.anchorOff != d.CursorOff)
	plain := []rune(plainOf(d.Paragraphs[d.CursorPara]))
	hasCharAfter := d.CursorOff < len(plain)
	hasParaAfter := d.CursorPara+1 < len(d.Paragraphs)
	if !hasSel && !hasCharAfter && !hasParaAfter {
		return
	}
	if hasSel || !hasCharAfter {
		d.pushHistory()
	} else {
		d.pushHistoryCoalesced(editDelete, 0)
	}
	if d.deleteSelectionLocked() {
		return
	}

	if hasCharAfter {
		d.deleteRange(d.CursorOff, d.CursorOff+1)
		d.Dirty = true
		d.endCoalescedEdit()
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

	para := d.Paragraphs[d.CursorPara]
	paraLen := len([]rune(plainOf(para)))
	if paraLen == 0 && para.Table == nil {
		if d.CursorPara+1 < len(d.Paragraphs) && d.Paragraphs[d.CursorPara+1].PageBreak {
			return
		}
		d.pushHistory()
		next := Paragraph{
			Runs:       []Run{{Text: "", Style: StyleNone}},
			Align:      para.Align,
			ParaStyle:  para.ParaStyle,
			SpaceAfter: para.SpaceAfter,
			PageBreak:  true,
		}
		insertAt := d.CursorPara + 1
		d.Paragraphs = append(d.Paragraphs, Paragraph{})
		copy(d.Paragraphs[insertAt+1:], d.Paragraphs[insertAt:len(d.Paragraphs)-1])
		d.Paragraphs[insertAt] = next
		d.CursorPara = insertAt
		d.CursorOff = 0
		d.Dirty = true
		return
	}
	if d.CursorOff == 0 && para.PageBreak {
		return
	}
	if d.CursorOff == paraLen && d.CursorPara+1 < len(d.Paragraphs) &&
		d.Paragraphs[d.CursorPara+1].PageBreak {
		return
	}

	d.pushHistory()
	switch {
	case d.CursorOff == 0:
		d.Paragraphs[d.CursorPara].PageBreak = true
		if d.CursorPara == 0 {
			d.ensureLeadingPageBreakAnchor()
		}
	case d.CursorOff == paraLen && d.CursorPara+1 < len(d.Paragraphs):
		d.CursorPara++
		d.CursorOff = 0
		d.Paragraphs[d.CursorPara].PageBreak = true
	case d.CursorOff == paraLen:
		next := Paragraph{
			Runs:       []Run{{Text: "", Style: StyleNone}},
			Align:      para.Align,
			ParaStyle:  para.ParaStyle,
			SpaceAfter: para.SpaceAfter,
			PageBreak:  true,
		}
		d.Paragraphs = append(d.Paragraphs, next)
		d.CursorPara++
		d.CursorOff = 0
	default:
		left, right := d.splitParaAtCursor()
		right.PageBreak = true
		d.Paragraphs[d.CursorPara] = left
		d.Paragraphs = append(d.Paragraphs[:d.CursorPara+1],
			append([]Paragraph{right}, d.Paragraphs[d.CursorPara+1:]...)...)
		d.CursorPara++
		d.CursorOff = 0
	}
	d.Dirty = true
}

func (d *Document) ensureLeadingPageBreakAnchor() {
	if len(d.Paragraphs) == 0 || !d.Paragraphs[0].PageBreak {
		return
	}
	d.Paragraphs = append(d.Paragraphs, Paragraph{})
	copy(d.Paragraphs[1:], d.Paragraphs[:len(d.Paragraphs)-1])
	d.Paragraphs[0] = Paragraph{Runs: []Run{{Text: "", Style: StyleNone}}}
	if d.CursorPara >= 0 {
		d.CursorPara++
	}
}

// InsertText inserts plain text at the cursor (replacing any selection).
// Newlines create new paragraphs. The active style/colours are applied.
func (d *Document) InsertText(s string) {
	if s == "" {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if d.inTableLocked() {
		return
	}
	d.pushHistory()
	d.deleteSelectionLocked()
	d.insertTextContentLocked(s)
}

func (d *Document) insertTextContentLocked(s string) {
	parts := strings.Split(strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n"), "\n")
	style, fg, bg := d.ActiveStyle, d.ActiveFg, d.ActiveBg

	if len(parts) == 1 {
		d.insertIntoRuns(parts[0], style, fg, bg)
		d.CursorOff += len([]rune(parts[0]))
	} else {
		left, right := d.splitParaAtCursor()
		// First fragment stays on the current paragraph.
		firstRuns := normalizeRuns(append(cloneRuns(left.Runs), Run{Text: parts[0], Style: style, Fg: fg, Bg: bg}))
		d.Paragraphs[d.CursorPara] = Paragraph{
			Runs:       firstRuns,
			Align:      left.Align,
			ParaStyle:  left.ParaStyle,
			SpaceAfter: left.SpaceAfter,
			PageBreak:  left.PageBreak,
		}

		// Middle full paragraphs.
		middles := make([]Paragraph, 0, len(parts)-2)
		for _, part := range parts[1 : len(parts)-1] {
			middles = append(middles, Paragraph{
				Runs:       []Run{{Text: part, Style: style, Fg: fg, Bg: bg}},
				Align:      left.Align,
				ParaStyle:  left.ParaStyle,
				SpaceAfter: left.SpaceAfter,
			})
		}

		// Last fragment merges with the right half.
		lastRuns := normalizeRuns(append([]Run{{Text: parts[len(parts)-1], Style: style, Fg: fg, Bg: bg}}, right.Runs...))
		lastP := Paragraph{Runs: lastRuns, Align: left.Align, ParaStyle: left.ParaStyle, SpaceAfter: left.SpaceAfter}

		tail := append([]Paragraph(nil), d.Paragraphs[d.CursorPara+1:]...)
		d.Paragraphs = append(append(d.Paragraphs[:d.CursorPara+1], middles...), append([]Paragraph{lastP}, tail...)...)
		d.CursorPara += len(parts) - 1
		d.CursorOff = len([]rune(parts[len(parts)-1]))
	}
	d.goalSet = false
	d.Dirty = true
}

// TransformSelection applies fn to the plain text of the selection (or current
// paragraph if nothing is selected) and replaces the content with the result.
// Styles are not preserved; the active style is used for the new text.
func (d *Document) TransformSelection(fn func(string) string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if d.inTableLocked() {
		return false
	}

	var text string
	if _, _, _, _, ok := d.selectionRange(); ok {
		ps := d.selectedParagraphs()
		parts := make([]string, len(ps))
		for i, p := range ps {
			parts[i] = plainOf(p)
		}
		text = strings.Join(parts, "\n")
	} else {
		text = plainOf(d.Paragraphs[d.CursorPara])
		// Select the whole paragraph so Replace can work uniformly.
		d.hasAnchor = true
		d.anchorPara = d.CursorPara
		d.anchorOff = 0
		d.CursorOff = len([]rune(text))
	}

	out := fn(text)
	if out == text {
		d.hasAnchor = false
		return false
	}

	d.pushHistory()
	d.deleteSelectionLocked()

	// Re-insert transformed text (same logic as InsertText, but lock already held).
	parts := strings.Split(strings.ReplaceAll(strings.ReplaceAll(out, "\r\n", "\n"), "\r", "\n"), "\n")
	style, fg, bg := d.ActiveStyle, d.ActiveFg, d.ActiveBg

	if len(parts) == 1 {
		d.insertIntoRuns(parts[0], style, fg, bg)
		d.CursorOff += len([]rune(parts[0]))
	} else {
		left, right := d.splitParaAtCursor()
		firstRuns := normalizeRuns(append(cloneRuns(left.Runs), Run{Text: parts[0], Style: style, Fg: fg, Bg: bg}))
		d.Paragraphs[d.CursorPara] = Paragraph{
			Runs:       firstRuns,
			Align:      left.Align,
			ParaStyle:  left.ParaStyle,
			SpaceAfter: left.SpaceAfter,
			PageBreak:  left.PageBreak,
		}

		middles := make([]Paragraph, 0, len(parts)-2)
		for _, part := range parts[1 : len(parts)-1] {
			middles = append(middles, Paragraph{
				Runs:       []Run{{Text: part, Style: style, Fg: fg, Bg: bg}},
				Align:      left.Align,
				ParaStyle:  left.ParaStyle,
				SpaceAfter: left.SpaceAfter,
			})
		}

		lastRuns := normalizeRuns(append([]Run{{Text: parts[len(parts)-1], Style: style, Fg: fg, Bg: bg}}, right.Runs...))
		lastP := Paragraph{Runs: lastRuns, Align: left.Align, ParaStyle: left.ParaStyle, SpaceAfter: left.SpaceAfter}

		tail := append([]Paragraph(nil), d.Paragraphs[d.CursorPara+1:]...)
		d.Paragraphs = append(append(d.Paragraphs[:d.CursorPara+1], middles...), append([]Paragraph{lastP}, tail...)...)
		d.CursorPara += len(parts) - 1
		d.CursorOff = len([]rune(parts[len(parts)-1]))
	}
	d.goalSet = false
	d.Dirty = true
	return true
}

// ApplyListPrefix turns the selected paragraphs (or the current one) into a list
// by prepending a marker produced by prefix(i) for the i-th item (0-based).
func (d *Document) ApplyListPrefix(prefix func(i int) string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.pushHistory()

	first, last := d.cursorParagraphs()
	if _, _, _, _, ok := d.selectionRange(); ok {
		sp, _, ep, _, _ := d.selectionRange()
		first, last = sp, ep
	}

	for i := first; i <= last; i++ {
		marker := prefix(i - first)
		plain := plainOf(d.Paragraphs[i])
		// Strip any existing common list marker so re-applying is clean.
		plain = stripListMarker(plain)
		newText := marker + plain
		d.Paragraphs[i].Runs = []Run{{Text: newText, Style: d.ActiveStyle, Fg: d.ActiveFg, Bg: d.ActiveBg}}
	}
	d.hasAnchor = false
	d.CursorPara = last
	d.CursorOff = len([]rune(plainOf(d.Paragraphs[last])))
	d.goalSet = false
	d.Dirty = true
}

// parseListMarker returns the marker (including trailing space), the rest of
// the line, and whether a recognised list marker was found at the start.
func parseListMarker(s string) (marker, rest string, ok bool) {
	s = strings.TrimLeft(s, " \t")
	if s == "" {
		return "", "", false
	}
	// Numbered: "1. ", "12. "
	i := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i > 0 && i+1 < len(s) && s[i] == '.' && s[i+1] == ' ' {
		return s[:i+2], s[i+2:], true
	}
	// Lettered: "a. ", "A. "
	if len(s) >= 3 {
		c := s[0]
		if ((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) && s[1] == '.' && s[2] == ' ' {
			return s[:3], s[3:], true
		}
	}
	// Simple roman (i. ii. iii. …) – only common short forms
	romans := []string{"xviii. ", "xvii. ", "xvi. ", "xv. ", "xiv. ", "xiii. ", "xii. ", "xi. ",
		"viii. ", "vii. ", "iii. ", "ii. ", "ix. ", "iv. ", "vi. ", "xx. ", "x. ", "v. ", "i. "}
	for _, r := range romans {
		if strings.HasPrefix(strings.ToLower(s), r) {
			// Preserve original case of the marker portion
			return s[:len(r)], s[len(r):], true
		}
	}
	// Bullets / checkboxes
	for _, prefix := range []string{"[ ] ", "[x] ", "[X] ", "• ", "- ", "* ", "– ", "— "} {
		if strings.HasPrefix(s, prefix) {
			return prefix, s[len(prefix):], true
		}
	}
	return "", s, false
}

func stripListMarker(s string) string {
	_, rest, ok := parseListMarker(s)
	if ok {
		return rest
	}
	return s
}

// nextListMarker returns the following marker in the same style.
func nextListMarker(marker string) string {
	m := strings.TrimSpace(marker)
	// Numbered
	if n, err := strconv.Atoi(strings.TrimSuffix(m, ".")); err == nil {
		return fmt.Sprintf("%d. ", n+1)
	}
	// Single letter
	if len(m) == 2 && m[1] == '.' {
		c := m[0]
		if c >= 'a' && c < 'z' {
			return string(c+1) + ". "
		}
		if c >= 'A' && c < 'Z' {
			return string(c+1) + ". "
		}
		// wrap
		if c == 'z' {
			return "a. "
		}
		if c == 'Z' {
			return "A. "
		}
	}
	// Roman (very small table)
	romans := []string{"i", "ii", "iii", "iv", "v", "vi", "vii", "viii", "ix", "x",
		"xi", "xii", "xiii", "xiv", "xv", "xvi", "xvii", "xviii", "xix", "xx"}
	lower := strings.ToLower(strings.TrimSuffix(m, "."))
	for i, r := range romans {
		if r == lower && i+1 < len(romans) {
			next := romans[i+1]
			if m[0] >= 'A' && m[0] <= 'Z' {
				next = strings.ToUpper(next)
			}
			return next + ". "
		}
	}
	// Bullets / checkboxes stay the same
	if strings.HasSuffix(marker, " ") {
		return marker
	}
	return marker + " "
}
