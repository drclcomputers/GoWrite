// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package document

import (
	"strings"
	"unicode"
)

func (d *Document) LastSearchQuery() (query string, caseSensitive, wholeWord bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.LastSearch, d.LastSearchCase, d.LastSearchWhole
}

func (d *Document) ClearLastSearch() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.LastSearch = ""
	d.LastSearchCase = false
	d.LastSearchWhole = false
}

func (d *Document) Find(query string, caseSensitive, wholeWord, backwards bool) (int, int, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if query == "" {
		return d.CursorPara, d.CursorOff, false
	}
	d.hasAnchor = false
	d.goalSet = false
	d.LastSearch = query
	d.LastSearchCase = caseSensitive
	d.LastSearchWhole = wholeWord

	qrunes := []rune(query)
	if !caseSensitive {
		for i, r := range qrunes {
			qrunes[i] = unicode.ToLower(r)
		}
	}

	fold := func(pi int) []rune {
		rs := []rune(plainOf(d.Paragraphs[pi]))
		if caseSensitive {
			return rs
		}
		for i, r := range rs {
			rs[i] = unicode.ToLower(r)
		}
		return rs
	}

	n := len(d.Paragraphs)
	if !backwards {
		for pi := d.CursorPara; pi < n; pi++ {
			start := 0
			if pi == d.CursorPara {
				start = d.CursorOff
				if start < len([]rune(plainOf(d.Paragraphs[pi]))) {
					start++
				}
			}
			if off, ok := findForward(fold(pi), start, qrunes, wholeWord); ok {
				d.CursorPara, d.CursorOff = pi, off
				return pi, off, true
			}
		}
		for pi := 0; pi <= d.CursorPara && pi < n; pi++ {
			if off, ok := findForward(fold(pi), 0, qrunes, wholeWord); ok {
				d.CursorPara, d.CursorOff = pi, off
				return pi, off, true
			}
		}
	} else {
		for pi := d.CursorPara; pi >= 0; pi-- {
			end := len([]rune(plainOf(d.Paragraphs[pi])))
			if pi == d.CursorPara {
				end = d.CursorOff - 1
			}
			if off, ok := findBackward(fold(pi), end, qrunes, wholeWord); ok {
				d.CursorPara, d.CursorOff = pi, off
				return pi, off, true
			}
		}
		for pi := n - 1; pi >= d.CursorPara && pi >= 0; pi-- {
			end := len([]rune(plainOf(d.Paragraphs[pi])))
			if off, ok := findBackward(fold(pi), end, qrunes, wholeWord); ok {
				d.CursorPara, d.CursorOff = pi, off
				return pi, off, true
			}
		}
	}
	return d.CursorPara, d.CursorOff, false
}

func findForward(haystack []rune, start int, needle []rune, wholeWord bool) (int, bool) {
	if len(needle) == 0 {
		return 0, false
	}
	if start < 0 {
		start = 0
	}
	for i := start; i+len(needle) <= len(haystack); i++ {
		if matchAt(haystack, i, needle) && (!wholeWord || isWholeWord(haystack, i, len(needle))) {
			return i, true
		}
	}
	return 0, false
}

func findBackward(haystack []rune, end int, needle []rune, wholeWord bool) (int, bool) {
	if len(needle) == 0 {
		return 0, false
	}
	if end > len(haystack) {
		end = len(haystack)
	}
	for i := end - len(needle); i >= 0; i-- {
		if matchAt(haystack, i, needle) && (!wholeWord || isWholeWord(haystack, i, len(needle))) {
			return i, true
		}
	}
	return 0, false
}

func matchAt(haystack []rune, pos int, needle []rune) bool {
	if pos+len(needle) > len(haystack) {
		return false
	}
	for i, r := range needle {
		if haystack[pos+i] != r {
			return false
		}
	}
	return true
}

func isWholeWord(runes []rune, pos, length int) bool {
	if pos > 0 && !unicode.IsSpace(runes[pos-1]) && !isPunct(runes[pos-1]) {
		return false
	}
	end := pos + length
	if end < len(runes) && !unicode.IsSpace(runes[end]) && !isPunct(runes[end]) {
		return false
	}
	return true
}

func isPunct(r rune) bool {
	return unicode.IsPunct(r) || unicode.IsSymbol(r)
}

func (d *Document) ReplaceAll(old, new string, caseSensitive, wholeWord bool) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	if old == "" {
		return 0
	}
	new = strings.ReplaceAll(new, "\r", "")
	new = strings.ReplaceAll(new, "\n", " ")

	qrunes := []rune(old)
	if !caseSensitive {
		for i, r := range qrunes {
			qrunes[i] = unicode.ToLower(r)
		}
	}

	d.pushHistory()

	count := 0
	for pi := range d.Paragraphs {
		rs := []rune(plainOf(d.Paragraphs[pi]))
		hay := rs
		if !caseSensitive {
			hay = make([]rune, len(rs))
			for i, r := range rs {
				hay[i] = unicode.ToLower(r)
			}
		}

		type span struct{ from, to int }
		var spans []span
		for i := 0; i+len(qrunes) <= len(hay); i++ {
			if matchAt(hay, i, qrunes) && (!wholeWord || isWholeWord(hay, i, len(qrunes))) {
				spans = append(spans, span{i, i + len(qrunes)})
				i += len(qrunes) - 1
			}
		}
		if len(spans) == 0 {
			continue
		}

		var out []Run
		prev := 0
		for _, s := range spans {
			out = append(out, sliceRuns(d.Paragraphs[pi].Runs, prev, s.from)...)
			style, fg, bg := runStyleAt(d.Paragraphs[pi].Runs, s.from)
			out = append(out, Run{Text: new, Style: style, Fg: fg, Bg: bg})
			prev = s.to
			count++
		}
		out = append(out, sliceRuns(d.Paragraphs[pi].Runs, prev, len(rs))...)
		d.Paragraphs[pi].Runs = normalizeRuns(out)
	}
	if count > 0 {
		d.hasAnchor = false
		d.goalSet = false
		d.Dirty = true
		d.ensureCursor()
	} else {
		d.popHistory()
	}
	return count
}

func runStyleAt(runs []Run, pos int) (Style, string, string) {
	p := 0
	for _, r := range runs {
		n := len([]rune(r.Text))
		if pos < p+n {
			return r.Style, r.Fg, r.Bg
		}
		p += n
	}
	return StyleNone, "", ""
}
