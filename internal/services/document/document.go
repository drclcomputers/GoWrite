// Copyright (c) 2026 @drclcomputers. All rights reserved.
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

// Align is paragraph-level horizontal alignment.
type Align int

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

// Run is a contiguous piece of text with the same style and colours.
type Run struct {
	Text  string
	Style Style
	Fg    string // "" = default; otherwise tview colour name or #rrggbb
	Bg    string
}

// Paragraph is a sequence of runs. Soft-wrapped on screen; hard break = new paragraph.
type Paragraph struct {
	Runs  []Run
	Align Align
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

	// Active style / colours for newly typed text
	ActiveStyle Style
	ActiveFg    string
	ActiveBg    string

	// DisplayWidth is the width (in cells) of the editing surface. Text is
	// wrapped and aligned to it; the UI keeps it in sync with the view.
	DisplayWidth int

	// Selection: the anchor is where it started; the other end is the cursor.
	// There is a selection only while hasAnchor is set and the anchor differs
	// from the cursor (see selection.go).
	hasAnchor  bool
	anchorPara int
	anchorOff  int

	// Goal column for Up/Down (see goalColumn in layout.go).
	goalSet  bool
	goalCol  int
	goalPara int
	goalOff  int

	// Last search (for Find Next / Prev)
	LastSearch      string
	LastSearchCase  bool
	LastSearchWhole bool

	// Undo / redo stacks (see history.go).
	history []historyEntry
	redo    []historyEntry
}

// ColorPalette is the shared swatch list used by the fg/bg colour pickers.
// Value is a tview tag value: "" means "terminal default", otherwise "#rrggbb".
var ColorPalette = []struct {
	Name  string
	Value string
}{
	{"Default", ""},
	// Neutrals
	{"Black", "#0e111a"},
	{"DkGray", "#3a4252"},
	{"Gray", "#8a94a6"},
	{"LtGray", "#b8c0cc"},
	{"White", "#e8ecf4"},
	// Reds / pinks
	{"DkRed", "#8b2f3a"},
	{"Red", "#e06c75"},
	{"Rose", "#e060a0"},
	{"Pink", "#f5a3c7"},
	// Oranges / yellows
	{"Orange", "#d19a66"},
	{"Amber", "#e8a86b"},
	{"Yellow", "#f0d090"},
	{"Cream", "#f5e6c8"},
	// Greens
	{"DkGreen", "#2f6b4f"},
	{"Green", "#5ab48c"},
	{"Mint", "#a3e0c0"},
	// Teals / cyans
	{"Teal", "#2f8b94"},
	{"Cyan", "#56b6c2"},
	{"LtCyan", "#a8d4dc"},
	// Blues
	{"Navy", "#2a3a6a"},
	{"Blue", "#61afef"},
	{"Sky", "#a3c8f0"},
	// Purples
	{"DkPurple", "#5a3a7a"},
	{"Purple", "#c678dd"},
	{"Violet", "#e0a3f0"},
	// Accents
	{"Olive", "#a3a83a"},
	{"Brown", "#8b6a4a"},
	{"Gold", "#d4b04a"},
}

// New creates an empty document with one blank paragraph.
func New() *Document {
	return &Document{
		Title:        "Untitled",
		Paragraphs:   []Paragraph{{Runs: []Run{{Text: "", Style: StyleNone}}}},
		CursorPara:   0,
		CursorOff:    0,
		DisplayWidth: 72,
	}
}

// SetDisplayWidth updates the editing surface width used for wrapping.
func (d *Document) SetDisplayWidth(w int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.DisplayWidth = w
}

// IsDirty reports whether the document has unsaved changes.
func (d *Document) IsDirty() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Dirty
}

// SetDirty marks the document as saved (false) or modified (true).
func (d *Document) SetDirty(v bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Dirty = v
}

// SetTitle changes the document title.
func (d *Document) SetTitle(s string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Title = s
}

// GetTitle returns the document title.
func (d *Document) GetTitle() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Title
}

// SetFilename changes the backing file.
func (d *Document) SetFilename(s string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Filename = s
}

// GetFilename returns the backing file, or "".
func (d *Document) GetFilename() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Filename
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

func styleTags(s Style, fg, bg string) (open, close string) {
	// Underline uses combining chars (decorateRunText) — SGR underline is sticky.
	const reset = "[-:-:-]"
	attrs := ""
	if s&StyleBold != 0 {
		attrs += "b"
	}
	if s&StyleItalic != 0 {
		attrs += "i"
	}
	if s&StyleStrike != 0 {
		attrs += "s"
	}
	// Colours
	f := fg
	b := bg
	if f == "" {
		f = "-"
	}
	if b == "" {
		b = "-"
	}
	hasColour := fg != "" || bg != ""
	hasAttr := attrs != ""
	if !hasColour && !hasAttr {
		return reset, ""
	}
	if hasAttr {
		return reset + "[" + f + ":" + b + ":" + attrs + "]", reset
	}
	return reset + "[" + f + ":" + b + "]", reset
}

// decorateRunText applies display-only underline via combining low-line.
func decorateRunText(text string, style Style) string {
	if style&StyleUnderline == 0 || text == "" {
		return text
	}
	var b strings.Builder
	b.Grow(len(text) * 2)
	for _, r := range text {
		b.WriteRune(r)
		if r != '\n' && r != '\t' {
			b.WriteRune('\u0332')
		}
	}
	return b.String()
}

// SetText replaces the entire content with plain text (no styles).
func (d *Document) SetText(s string) {
	// Normalise CRLF and stray CR so the layout only ever sees \n.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

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
	d.hasAnchor = false
	d.goalSet = false
	d.history = nil
	d.redo = nil
	d.Dirty = true
}

// LoadText replaces the content with s and marks the document clean. It is
// used when opening a file.
func (d *Document) LoadText(s string) {
	d.SetText(s)
	d.SetDirty(false)
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
	d.pushHistory()
	d.deleteSelectionLocked() // typing over a selection replaces it

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

	// Insert into runs at cursor using active style + colours
	d.insertIntoRuns(string(r), d.ActiveStyle, d.ActiveFg, d.ActiveBg)
	d.CursorOff++
	d.Dirty = true
}

// sameRunStyle reports whether a run matches style+colours (merge candidate).
func sameRunStyle(r Run, style Style, fg, bg string) bool {
	return r.Style == style && r.Fg == fg && r.Bg == bg
}

// insertIntoRuns inserts text with the given style and colours at the cursor.
// Caller holds lock and has called ensureCursor.
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
	// Past the end – append
	if len(p.Runs) > 0 && sameRunStyle(p.Runs[len(p.Runs)-1], style, fg, bg) {
		p.Runs[len(p.Runs)-1].Text += text
	} else {
		p.Runs = append(p.Runs, Run{Text: text, Style: style, Fg: fg, Bg: bg})
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

// Backspace deletes the character before the cursor (or merges paragraphs).
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
	// Merge with previous paragraph
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

// Delete deletes the character under the cursor (or merges with next paragraph).
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
	// Merge with next
	if !hasParaAfter {
		return
	}
	d.Paragraphs[d.CursorPara].Runs = append(d.Paragraphs[d.CursorPara].Runs, d.Paragraphs[d.CursorPara+1].Runs...)
	d.Paragraphs = append(d.Paragraphs[:d.CursorPara+1], d.Paragraphs[d.CursorPara+2:]...)
	d.Dirty = true
}

// deleteRange removes runes [from, to) in the current paragraph. Caller holds lock.
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

// MoveLeft / MoveRight / MoveWord* (vertical and line moves live in layout.go)
func (d *Document) MoveLeft() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.goalSet = false
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
	d.goalSet = false
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

// MoveWordLeft jumps to the start of the previous word.
func (d *Document) MoveWordLeft() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.goalSet = false
	plain := []rune(plainOf(d.Paragraphs[d.CursorPara]))
	if d.CursorOff == 0 {
		if d.CursorPara > 0 {
			d.CursorPara--
			d.CursorOff = len([]rune(plainOf(d.Paragraphs[d.CursorPara])))
		}
		return
	}
	i := d.CursorOff - 1
	for i > 0 && unicode.IsSpace(plain[i]) {
		i--
	}
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
	d.goalSet = false
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
	for i < n && !unicode.IsSpace(plain[i]) {
		i++
	}
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
	d.hasAnchor = false
	d.goalSet = false
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

// ApplyStyleToParagraph applies style to the whole current paragraph.
func (d *Document) ApplyStyleToParagraph(style Style, on bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.pushHistory()
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

// LastSearchQuery returns the last search term and options.
func (d *Document) LastSearchQuery() (query string, caseSensitive, wholeWord bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.LastSearch, d.LastSearchCase, d.LastSearchWhole
}

// ClearLastSearch forgets the last search term.
func (d *Document) ClearLastSearch() {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.LastSearch = ""
	d.LastSearchCase = false
	d.LastSearchWhole = false
}

// Find searches for the next occurrence of query starting after the cursor.
// It wraps around at the end (or beginning) of the document.
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

// ReplaceAll replaces every occurrence of old with new. Returns count.
// Surrounding styles are preserved; the replacement itself takes the style
// of the first character of each match.
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

// runStyleAt returns the style/colours of the run containing pos.
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

// SetActiveFg sets the foreground colour for newly typed text.
func (d *Document) SetActiveFg(fg string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ActiveFg = fg
}

// SetActiveBg sets the background colour for newly typed text.
func (d *Document) SetActiveBg(bg string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ActiveBg = bg
}

// ActiveColors returns the active fg/bg colour values.
func (d *Document) ActiveColors() (fg, bg string) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.ActiveFg, d.ActiveBg
}

// SetParagraphAlign sets alignment for the paragraph under the cursor, or for
// every paragraph the selection touches.
func (d *Document) SetParagraphAlign(a Align) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	d.pushHistory()
	first, last := d.cursorParagraphs()
	for i := first; i <= last; i++ {
		d.Paragraphs[i].Align = a
	}
	d.Dirty = true
}

// CurrentAlign returns the alignment of the paragraph under the cursor.
func (d *Document) CurrentAlign() Align {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.CursorPara < 0 || d.CursorPara >= len(d.Paragraphs) {
		return AlignLeft
	}
	return d.Paragraphs[d.CursorPara].Align
}

// ColorName returns a palette name for a colour value (or the value itself).
func ColorName(val string) string {
	if val == "" {
		return "Default"
	}
	for _, c := range ColorPalette {
		if c.Value == val {
			return c.Name
		}
	}
	return val
}

// AlignString returns a short label for an alignment.
func AlignString(a Align) string {
	switch a {
	case AlignCenter:
		return "Center"
	case AlignRight:
		return "Right"
	default:
		return "Left"
	}
}

// StyleString returns a short status label for the active style.
func (d *Document) StyleString() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var parts []string
	s := d.ActiveStyle
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
	if d.ActiveFg != "" {
		parts = append(parts, "fg:"+ColorName(d.ActiveFg))
	}
	if d.ActiveBg != "" {
		parts = append(parts, "bg:"+ColorName(d.ActiveBg))
	}
	parts = append(parts, AlignString(func() Align {
		if d.CursorPara >= 0 && d.CursorPara < len(d.Paragraphs) {
			return d.Paragraphs[d.CursorPara].Align
		}
		return AlignLeft
	}()))
	if len(parts) == 1 {
		return "Normal · " + parts[0]
	}
	return strings.Join(parts, "+")
}
