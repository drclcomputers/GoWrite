// Copyright (c) 2025 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

// Package document holds the in-memory model of a word-processor document.
package document

import (
	"strings"
	"sync"
	"unicode"
)

// Style flags (bitmask). Can be combined.
type Style uint8

const (
	StyleNone      Style = 0
	StyleBold      Style = 1 << 0
	StyleItalic    Style = 1 << 1
	StyleUnderline Style = 1 << 2
	StyleStrike    Style = 1 << 3
)

// Run is a contiguous piece of text with the same style.
type Run struct {
	Text  string
	Style Style
}

// Paragraph is a sequence of runs. Soft-wrapped on screen; hard break = new paragraph.
type Paragraph struct {
	Runs []Run
}

// Document is a multi-paragraph word-processor model with basic styling.
type Document struct {
	mu       sync.RWMutex
	Title    string
	Filename string
	Dirty    bool

	Paragraphs []Paragraph

	// Cursor: paragraph index + rune offset into the plain text of that paragraph
	CursorPara int
	CursorOff  int

	// Active style for newly typed text
	ActiveStyle Style

	// Last search (for Find Next / Prev)
	LastSearch     string
	LastSearchCase bool
	LastSearchWhole bool
}

// New creates an empty document with one blank paragraph.
func New() *Document {
	return &Document{
		Title:      "Untitled",
		Paragraphs: []Paragraph{{Runs: []Run{{Text: "", Style: StyleNone}}}},
		CursorPara: 0,
		CursorOff:  0,
	}
}

// plainOf returns the plain (style-stripped) text of a paragraph.
func plainOf(p Paragraph) string {
	var b strings.Builder
	for _, r := range p.Runs {
		b.WriteString(r.Text)
	}
	return b.String()
}

// Text returns the full document as plain text (paragraphs joined by \n).
func (d *Document) Text() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	parts := make([]string, len(d.Paragraphs))
	for i, p := range d.Paragraphs {
		parts[i] = plainOf(p)
	}
	return strings.Join(parts, "\n")
}

// StyledText returns a tview-friendly string with colour/style tags for display.
// Bold → [::b], Italic → [::i], Underline → [::u], Strike → [::s]
func (d *Document) StyledText() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var b strings.Builder
	for i, p := range d.Paragraphs {
		if i > 0 {
			b.WriteByte('\n')
		}
		for _, r := range p.Runs {
			if r.Text == "" {
				continue
			}
			open, close := styleTags(r.Style)
			b.WriteString(open)
			b.WriteString(r.Text)
			b.WriteString(close)
		}
	}
	return b.String()
}

// AbsoluteCursorOffset returns the plain-text offset of the cursor from the
// start of the document (counting newlines between paragraphs).
func (d *Document) AbsoluteCursorOffset() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	off := 0
	for i := 0; i < d.CursorPara && i < len(d.Paragraphs); i++ {
		off += len([]rune(plainOf(d.Paragraphs[i]))) + 1 // +1 for newline
	}
	off += d.CursorOff
	return off
}

// StyledTextWithCursor is like StyledText but inserts a visible caret
// at the current position so a TextView can show where typing will go.
func (d *Document) StyledTextWithCursor() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var b strings.Builder
	// Use a simple reverse-style block that tview always understands.
	// Avoid raw hex tags that can leak as literal text on some terminals.
	const caretOpen = "[-:-:-][black:#f0d090]"
	const caretClose = "[-:-:-]"

	for pi, p := range d.Paragraphs {
		if pi > 0 {
			b.WriteByte('\n')
		}
		cursorHere := pi == d.CursorPara
		pos := 0
		cursorDrawn := false
		for _, r := range p.Runs {
			runes := []rune(r.Text)
			open, close := styleTags(r.Style)
			if !cursorHere {
				if r.Text != "" {
					b.WriteString(open)
					b.WriteString(r.Text)
					b.WriteString(close)
				}
				pos += len(runes)
				continue
			}
			// May need to split this run around the cursor
			drawnInThisRun := false
			for i, ch := range runes {
				if pos+i == d.CursorOff {
					b.WriteString(open)
					b.WriteString(string(runes[:i]))
					b.WriteString(close)
					b.WriteString(caretOpen)
					b.WriteRune(ch)
					b.WriteString(caretClose)
					if i+1 < len(runes) {
						b.WriteString(open)
						b.WriteString(string(runes[i+1:]))
						b.WriteString(close)
					}
					cursorDrawn = true
					drawnInThisRun = true
					break
				}
			}
			if !drawnInThisRun {
				if r.Text != "" {
					b.WriteString(open)
					b.WriteString(r.Text)
					b.WriteString(close)
				}
			}
			pos += len(runes)
		}
		// Cursor at end of paragraph (or empty paragraph)
		if cursorHere && !cursorDrawn {
			b.WriteString(caretOpen)
			b.WriteRune('▌')
			b.WriteString(caretClose)
		}
	}
	if len(d.Paragraphs) == 0 {
		b.WriteString(caretOpen)
		b.WriteRune('▌')
		b.WriteString(caretClose)
	}
	// Final reset so underline cannot bleed past the last cell
	b.WriteString("[-:-:-]")
	return b.String()
}

func styleTags(s Style) (open, close string) {
	// Always start with a full reset. Underline (SGR 4) is sticky on many
	// terminals if the previous run only used [::-] — the attribute can
	// bleed into following text and even to the end of the line.
	const reset = "[-:-:-]"
	if s == StyleNone {
		return reset, ""
	}
	// tview attribute flags live in the third field of a colour tag:
	//   [foreground:background:attributes]
	//   attributes: b=bold, i=italic, u=underline, s=strikethrough
	attrs := ""
	if s&StyleBold != 0 {
		attrs += "b"
	}
	if s&StyleItalic != 0 {
		attrs += "i"
	}
	if s&StyleUnderline != 0 {
		attrs += "u"
	}
	if s&StyleStrike != 0 {
		attrs += "s"
	}
	return reset + "[::" + attrs + "]", reset
}

// SetText replaces the entire content with plain text (no styles).
func (d *Document) SetText(s string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if s == "" {
		d.Paragraphs = []Paragraph{{Runs: []Run{{Text: "", Style: StyleNone}}}}
	} else {
		lines := strings.Split(s, "\n")
		d.Paragraphs = make([]Paragraph, len(lines))
		for i, line := range lines {
			d.Paragraphs[i] = Paragraph{Runs: []Run{{Text: line, Style: StyleNone}}}
		}
	}
	d.CursorPara = 0
	d.CursorOff = 0
	d.Dirty = true
}

// CurrentPlain returns the plain text of the paragraph under the cursor.
func (d *Document) CurrentPlain() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.CursorPara < 0 || d.CursorPara >= len(d.Paragraphs) {
		return ""
	}
	return plainOf(d.Paragraphs[d.CursorPara])
}

// ensureCursor clamps cursor into valid range (caller must hold lock).
func (d *Document) ensureCursor() {
	if len(d.Paragraphs) == 0 {
		d.Paragraphs = []Paragraph{{Runs: []Run{{Text: "", Style: StyleNone}}}}
	}
	if d.CursorPara < 0 {
		d.CursorPara = 0
	}
	if d.CursorPara >= len(d.Paragraphs) {
		d.CursorPara = len(d.Paragraphs) - 1
	}
	plain := []rune(plainOf(d.Paragraphs[d.CursorPara]))
	if d.CursorOff < 0 {
		d.CursorOff = 0
	}
	if d.CursorOff > len(plain) {
		d.CursorOff = len(plain)
	}
}

// InsertRune inserts a rune at the cursor using the active style.
func (d *Document) InsertRune(r rune) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()

	if r == '\n' {
		// Split paragraph
		left, right := d.splitParaAtCursor()
		d.Paragraphs[d.CursorPara] = left
		d.Paragraphs = append(d.Paragraphs[:d.CursorPara+1],
			append([]Paragraph{right}, d.Paragraphs[d.CursorPara+1:]...)...)
		d.CursorPara++
		d.CursorOff = 0
		d.Dirty = true
		return
	}

	// Insert into runs at cursor
	d.insertIntoRuns(string(r), d.ActiveStyle)
	d.CursorOff++
	d.Dirty = true
}

// insertIntoRuns inserts text with the given style at the current cursor offset.
// Caller holds lock and has called ensureCursor.
func (d *Document) insertIntoRuns(text string, style Style) {
	p := &d.Paragraphs[d.CursorPara]
	off := d.CursorOff
	// Walk runs to find insertion point
	pos := 0
	for i := range p.Runs {
		runLen := len([]rune(p.Runs[i].Text))
		if pos+runLen >= off {
			// Insert inside this run
			local := off - pos
			runes := []rune(p.Runs[i].Text)
			if p.Runs[i].Style == style {
				// Same style – just splice text
				newText := string(runes[:local]) + text + string(runes[local:])
				p.Runs[i].Text = newText
				return
			}
			// Different style – split the run
			left := string(runes[:local])
			right := string(runes[local:])
			newRuns := make([]Run, 0, len(p.Runs)+2)
			newRuns = append(newRuns, p.Runs[:i]...)
			if left != "" {
				newRuns = append(newRuns, Run{Text: left, Style: p.Runs[i].Style})
			}
			newRuns = append(newRuns, Run{Text: text, Style: style})
			if right != "" {
				newRuns = append(newRuns, Run{Text: right, Style: p.Runs[i].Style})
			}
			newRuns = append(newRuns, p.Runs[i+1:]...)
			p.Runs = newRuns
			return
		}
		pos += runLen
	}
	// Past the end – append
	if len(p.Runs) > 0 && p.Runs[len(p.Runs)-1].Style == style {
		p.Runs[len(p.Runs)-1].Text += text
	} else {
		p.Runs = append(p.Runs, Run{Text: text, Style: style})
	}
}

// splitParaAtCursor splits the current paragraph into left (up to cursor) and right (from cursor).
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
				leftRuns = append(leftRuns, Run{Text: string(runes[:local]), Style: run.Style})
			}
			if len(leftRuns) == 0 {
				leftRuns = []Run{{Text: "", Style: StyleNone}}
			}
			rightRuns := []Run{}
			if local < runLen {
				rightRuns = append(rightRuns, Run{Text: string(runes[local:]), Style: run.Style})
			}
			rightRuns = append(rightRuns, p.Runs[i+1:]...)
			if len(rightRuns) == 0 {
				rightRuns = []Run{{Text: "", Style: StyleNone}}
			}
			return Paragraph{Runs: leftRuns}, Paragraph{Runs: rightRuns}
		}
		pos += runLen
	}
	// Cursor at end
	return p, Paragraph{Runs: []Run{{Text: "", Style: StyleNone}}}
}

// Backspace deletes the character before the cursor (or merges paragraphs).
func (d *Document) Backspace() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()

	if d.CursorOff > 0 {
		d.deleteRange(d.CursorOff-1, d.CursorOff)
		d.CursorOff--
		d.Dirty = true
		return
	}
	// Merge with previous paragraph
	if d.CursorPara == 0 {
		return
	}
	prevPlain := plainOf(d.Paragraphs[d.CursorPara-1])
	// Append current runs to previous
	d.Paragraphs[d.CursorPara-1].Runs = append(d.Paragraphs[d.CursorPara-1].Runs, d.Paragraphs[d.CursorPara].Runs...)
	d.Paragraphs = append(d.Paragraphs[:d.CursorPara], d.Paragraphs[d.CursorPara+1:]...)
	d.CursorPara--
	d.CursorOff = len([]rune(prevPlain))
	d.Dirty = true
}

// Delete deletes the character under the cursor (or merges with next paragraph).
func (d *Document) Delete() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()

	plain := []rune(plainOf(d.Paragraphs[d.CursorPara]))
	if d.CursorOff < len(plain) {
		d.deleteRange(d.CursorOff, d.CursorOff+1)
		d.Dirty = true
		return
	}
	// Merge with next
	if d.CursorPara+1 >= len(d.Paragraphs) {
		return
	}
	d.Paragraphs[d.CursorPara].Runs = append(d.Paragraphs[d.CursorPara].Runs, d.Paragraphs[d.CursorPara+1].Runs...)
	d.Paragraphs = append(d.Paragraphs[:d.CursorPara+1], d.Paragraphs[d.CursorPara+2:]...)
	d.Dirty = true
}

// deleteRange removes runes [from, to) in the current paragraph. Caller holds lock.
func (d *Document) deleteRange(from, to int) {
	p := &d.Paragraphs[d.CursorPara]
	// Rebuild runs without the deleted range
	var newRuns []Run
	pos := 0
	for _, run := range p.Runs {
		runes := []rune(run.Text)
		runLen := len(runes)
		runStart, runEnd := pos, pos+runLen
		pos = runEnd

		if runEnd <= from || runStart >= to {
			// Fully outside
			if run.Text != "" {
				newRuns = append(newRuns, run)
			}
			continue
		}
		// Partial overlap
		keepLeft := from - runStart
		if keepLeft < 0 {
			keepLeft = 0
		}
		keepRight := to - runStart
		if keepRight > runLen {
			keepRight = runLen
		}
		left := string(runes[:keepLeft])
		right := string(runes[keepRight:])
		if left != "" {
			newRuns = append(newRuns, Run{Text: left, Style: run.Style})
		}
		if right != "" {
			newRuns = append(newRuns, Run{Text: right, Style: run.Style})
		}
	}
	if len(newRuns) == 0 {
		newRuns = []Run{{Text: "", Style: StyleNone}}
	}
	p.Runs = newRuns
}

// MoveLeft / MoveRight / MoveUp / MoveDown / MoveWord*
func (d *Document) MoveLeft() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if d.CursorOff > 0 {
		d.CursorOff--
		return
	}
	if d.CursorPara > 0 {
		d.CursorPara--
		d.CursorOff = len([]rune(plainOf(d.Paragraphs[d.CursorPara])))
	}
}

func (d *Document) MoveRight() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	plain := []rune(plainOf(d.Paragraphs[d.CursorPara]))
	if d.CursorOff < len(plain) {
		d.CursorOff++
		return
	}
	if d.CursorPara+1 < len(d.Paragraphs) {
		d.CursorPara++
		d.CursorOff = 0
	}
}

func (d *Document) MoveUp() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if d.CursorPara > 0 {
		d.CursorPara--
		plain := []rune(plainOf(d.Paragraphs[d.CursorPara]))
		if d.CursorOff > len(plain) {
			d.CursorOff = len(plain)
		}
	}
}

func (d *Document) MoveDown() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	if d.CursorPara+1 < len(d.Paragraphs) {
		d.CursorPara++
		plain := []rune(plainOf(d.Paragraphs[d.CursorPara]))
		if d.CursorOff > len(plain) {
			d.CursorOff = len(plain)
		}
	}
}

// MoveWordLeft jumps to the start of the previous word.
func (d *Document) MoveWordLeft() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	plain := []rune(plainOf(d.Paragraphs[d.CursorPara]))
	if d.CursorOff == 0 {
		if d.CursorPara > 0 {
			d.CursorPara--
			d.CursorOff = len([]rune(plainOf(d.Paragraphs[d.CursorPara])))
		}
		return
	}
	i := d.CursorOff - 1
	// Skip whitespace
	for i > 0 && unicode.IsSpace(plain[i]) {
		i--
	}
	// Skip word chars
	for i > 0 && !unicode.IsSpace(plain[i-1]) {
		i--
	}
	d.CursorOff = i
}

// MoveWordRight jumps to the start of the next word.
func (d *Document) MoveWordRight() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	plain := []rune(plainOf(d.Paragraphs[d.CursorPara]))
	n := len(plain)
	if d.CursorOff >= n {
		if d.CursorPara+1 < len(d.Paragraphs) {
			d.CursorPara++
			d.CursorOff = 0
		}
		return
	}
	i := d.CursorOff
	// Skip current word
	for i < n && !unicode.IsSpace(plain[i]) {
		i++
	}
	// Skip whitespace
	for i < n && unicode.IsSpace(plain[i]) {
		i++
	}
	d.CursorOff = i
}

// GoToLine moves the cursor to the start of the given 1-based line (paragraph).
func (d *Document) GoToLine(line int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if line < 1 {
		line = 1
	}
	if line > len(d.Paragraphs) {
		line = len(d.Paragraphs)
	}
	d.CursorPara = line - 1
	d.CursorOff = 0
}

// ToggleStyle toggles a style flag on the active typing style.
func (d *Document) ToggleStyle(s Style) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ActiveStyle ^= s
}

// SetActiveStyle replaces the active style.
func (d *Document) SetActiveStyle(s Style) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ActiveStyle = s
}

// ActiveStyleFlags returns the current typing style.
func (d *Document) ActiveStyleFlags() Style {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.ActiveStyle
}

// ApplyStyleToSelection applies style to a plain-text range within the current paragraph.
// For simplicity we apply to the whole current paragraph when no finer selection exists.
func (d *Document) ApplyStyleToParagraph(style Style, on bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	p := &d.Paragraphs[d.CursorPara]
	for i := range p.Runs {
		if on {
			p.Runs[i].Style |= style
		} else {
			p.Runs[i].Style &^= style
		}
	}
	d.Dirty = true
}

// Find searches for the next occurrence of query starting after the cursor.
// Returns (paraIndex, runeOffset, found).
func (d *Document) Find(query string, caseSensitive, wholeWord, backwards bool) (int, int, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if query == "" {
		return d.CursorPara, d.CursorOff, false
	}
	d.LastSearch = query
	d.LastSearchCase = caseSensitive
	d.LastSearchWhole = wholeWord

	cmp := query
	if !caseSensitive {
		cmp = strings.ToLower(query)
	}

	if backwards {
		// Search from cursor backwards
		for pi := d.CursorPara; pi >= 0; pi-- {
			plain := plainOf(d.Paragraphs[pi])
			searchIn := plain
			if !caseSensitive {
				searchIn = strings.ToLower(plain)
			}
			end := len([]rune(plain))
			if pi == d.CursorPara {
				end = d.CursorOff
			}
			// Search in runes space for correctness with multi-byte
			runes := []rune(searchIn)
			qrunes := []rune(cmp)
			for i := end - len(qrunes); i >= 0; i-- {
				if matchAt(runes, i, qrunes) && (!wholeWord || isWholeWord(runes, i, len(qrunes))) {
					d.CursorPara = pi
					d.CursorOff = i
					return pi, i, true
				}
			}
		}
		return d.CursorPara, d.CursorOff, false
	}

	// Forward
	for pi := d.CursorPara; pi < len(d.Paragraphs); pi++ {
		plain := plainOf(d.Paragraphs[pi])
		searchIn := plain
		if !caseSensitive {
			searchIn = strings.ToLower(plain)
		}
		start := 0
		if pi == d.CursorPara {
			start = d.CursorOff
			// Advance one so we don't re-find the same spot
			if start < len([]rune(plain)) {
				start++
			}
		}
		runes := []rune(searchIn)
		qrunes := []rune(cmp)
		for i := start; i+len(qrunes) <= len(runes); i++ {
			if matchAt(runes, i, qrunes) && (!wholeWord || isWholeWord(runes, i, len(qrunes))) {
				d.CursorPara = pi
				d.CursorOff = i
				return pi, i, true
			}
		}
	}
	return d.CursorPara, d.CursorOff, false
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

// ReplaceAll replaces every occurrence of old with new. Returns count.
func (d *Document) ReplaceAll(old, new string, caseSensitive, wholeWord bool) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	if old == "" {
		return 0
	}
	count := 0
	cmp := old
	if !caseSensitive {
		cmp = strings.ToLower(old)
	}
	qrunes := []rune(cmp)
	newText := new

	for pi := range d.Paragraphs {
		plain := plainOf(d.Paragraphs[pi])
		searchIn := plain
		if !caseSensitive {
			searchIn = strings.ToLower(plain)
		}
		runes := []rune(searchIn)
		// Collect replacement ranges (from the end so offsets stay valid)
		type span struct{ from, to int }
		var spans []span
		for i := 0; i+len(qrunes) <= len(runes); i++ {
			if matchAt(runes, i, qrunes) && (!wholeWord || isWholeWord(runes, i, len(qrunes))) {
				spans = append(spans, span{i, i + len(qrunes)})
				i += len(qrunes) - 1
			}
		}
		if len(spans) == 0 {
			continue
		}
		// Rebuild plain text with replacements
		origRunes := []rune(plain)
		var b strings.Builder
		prev := 0
		for _, s := range spans {
			b.WriteString(string(origRunes[prev:s.from]))
			b.WriteString(newText)
			prev = s.to
			count++
		}
		b.WriteString(string(origRunes[prev:]))
		// For simplicity after replace we collapse to a single unstyled run
		d.Paragraphs[pi] = Paragraph{Runs: []Run{{Text: b.String(), Style: StyleNone}}}
	}
	if count > 0 {
		d.Dirty = true
		d.ensureCursor()
	}
	return count
}

// WordCount / CharCount / ParagraphCount
func (d *Document) WordCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	count := 0
	for _, p := range d.Paragraphs {
		count += len(strings.Fields(plainOf(p)))
	}
	return count
}

func (d *Document) CharCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	n := 0
	for _, p := range d.Paragraphs {
		n += len([]rune(plainOf(p)))
	}
	return n
}

func (d *Document) ParagraphCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.Paragraphs)
}

// CursorLine returns 1-based paragraph number.
func (d *Document) CursorLine() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.CursorPara + 1
}

// CursorCol returns 1-based column within the paragraph.
func (d *Document) CursorCol() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.CursorOff + 1
}

// StyleString returns a short status label for the active style.
func (d *Document) StyleString() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	s := d.ActiveStyle
	if s == StyleNone {
		return "Normal"
	}
	var parts []string
	if s&StyleBold != 0 {
		parts = append(parts, "Bold")
	}
	if s&StyleItalic != 0 {
		parts = append(parts, "Italic")
	}
	if s&StyleUnderline != 0 {
		parts = append(parts, "Underline")
	}
	if s&StyleStrike != 0 {
		parts = append(parts, "Strike")
	}
	return strings.Join(parts, "+")
}
