// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package document

import "sync"

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
	Fg    string
	Bg    string
}

// Paragraph is a sequence of runs. Soft-wrapped on screen; hard break = new paragraph.
type Paragraph struct {
	Runs      []Run
	Align     Align
	PageBreak bool
}

// Document is a multi-paragraph word-processor model with basic styling.
type Document struct {
	mu       sync.RWMutex
	Title    string
	Filename string
	Dirty    bool

	Paragraphs []Paragraph
	Page       PageSettings

	CursorPara int
	CursorOff  int

	ActiveStyle Style
	ActiveFg    string
	ActiveBg    string

	DisplayWidth int

	hasAnchor  bool
	anchorPara int
	anchorOff  int

	goalSet  bool
	goalCol  int
	goalPara int
	goalOff  int

	LastSearch      string
	LastSearchCase  bool
	LastSearchWhole bool

	history []historyEntry
	redo    []historyEntry
}

// New creates an empty document with one blank paragraph.
func New() *Document {
	return &Document{
		Title:        "Untitled",
		Paragraphs:   []Paragraph{{Runs: []Run{{Text: "", Style: StyleNone}}}},
		Page:         DefaultA4Portrait(),
		CursorPara:   0,
		CursorOff:    0,
		DisplayWidth: 80,
	}
}

func (d *Document) SetPageSettings(p PageSettings) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Page = p
	d.Dirty = true
}

func (d *Document) GetPageSettings() PageSettings {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Page
}

func (d *Document) SetDisplayWidth(w int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.DisplayWidth = w
}

func (d *Document) IsDirty() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Dirty
}

func (d *Document) SetDirty(v bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Dirty = v
}

func (d *Document) SetTitle(s string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Title = s
}

func (d *Document) GetTitle() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Title
}

func (d *Document) SetFilename(s string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Filename = s
}

func (d *Document) GetFilename() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Filename
}
