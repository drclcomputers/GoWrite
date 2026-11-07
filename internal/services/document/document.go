// Copyright (c) 2025 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

// Package document holds the in-memory model of a word-processor document.
package document

import (
	"strings"
	"sync"
)

// Document is a simple multi-paragraph text model.
// Later we can grow this into runs, styles, pages, etc.
type Document struct {
	mu       sync.RWMutex
	Title    string
	Filename string
	Dirty    bool

	// Paragraphs are the atomic units for now.
	// Index 0 is always present (even if empty).
	Paragraphs []string

	// Cursor state (paragraph index + rune offset inside it)
	CursorPara int
	CursorOff  int
}

// New creates an empty document with one blank paragraph.
func New() *Document {
	return &Document{
		Title:      "Untitled",
		Paragraphs: []string{""},
		CursorPara: 0,
		CursorOff:  0,
	}
}

// Text returns the full document as a single string (paragraphs joined by \n).
func (d *Document) Text() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return strings.Join(d.Paragraphs, "\n")
}

// SetText replaces the entire content.
func (d *Document) SetText(s string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if s == "" {
		d.Paragraphs = []string{""}
	} else {
		d.Paragraphs = strings.Split(s, "\n")
	}
	d.CursorPara = 0
	d.CursorOff = 0
	d.Dirty = true
}

// CurrentParagraph returns the paragraph under the cursor.
func (d *Document) CurrentParagraph() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.CursorPara < 0 || d.CursorPara >= len(d.Paragraphs) {
		return ""
	}
	return d.Paragraphs[d.CursorPara]
}

// InsertRune inserts a single rune at the cursor and advances it.
func (d *Document) InsertRune(r rune) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.CursorPara < 0 || d.CursorPara >= len(d.Paragraphs) {
		return
	}
	para := d.Paragraphs[d.CursorPara]
	runes := []rune(para)

	if d.CursorOff < 0 {
		d.CursorOff = 0
	}
	if d.CursorOff > len(runes) {
		d.CursorOff = len(runes)
	}

	// Split the paragraph on newline
	if r == '\n' {
		left := string(runes[:d.CursorOff])
		right := string(runes[d.CursorOff:])
		d.Paragraphs[d.CursorPara] = left
		// insert new paragraph after current
		d.Paragraphs = append(d.Paragraphs[:d.CursorPara+1],
			append([]string{right}, d.Paragraphs[d.CursorPara+1:]...)...)
		d.CursorPara++
		d.CursorOff = 0
		d.Dirty = true
		return
	}

	// normal character
	newRunes := make([]rune, 0, len(runes)+1)
	newRunes = append(newRunes, runes[:d.CursorOff]...)
	newRunes = append(newRunes, r)
	newRunes = append(newRunes, runes[d.CursorOff:]...)
	d.Paragraphs[d.CursorPara] = string(newRunes)
	d.CursorOff++
	d.Dirty = true
}

// Backspace deletes the character before the cursor (or merges paragraphs).
func (d *Document) Backspace() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.CursorPara < 0 || d.CursorPara >= len(d.Paragraphs) {
		return
	}

	if d.CursorOff > 0 {
		runes := []rune(d.Paragraphs[d.CursorPara])
		newRunes := append(runes[:d.CursorOff-1], runes[d.CursorOff:]...)
		d.Paragraphs[d.CursorPara] = string(newRunes)
		d.CursorOff--
		d.Dirty = true
		return
	}

	// at start of paragraph → merge with previous
	if d.CursorPara == 0 {
		return
	}
	prev := d.Paragraphs[d.CursorPara-1]
	curr := d.Paragraphs[d.CursorPara]
	d.Paragraphs[d.CursorPara-1] = prev + curr
	d.Paragraphs = append(d.Paragraphs[:d.CursorPara], d.Paragraphs[d.CursorPara+1:]...)
	d.CursorPara--
	d.CursorOff = len([]rune(prev))
	d.Dirty = true
}

// Delete deletes the character under the cursor (or merges with next paragraph).
func (d *Document) Delete() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.CursorPara < 0 || d.CursorPara >= len(d.Paragraphs) {
		return
	}

	runes := []rune(d.Paragraphs[d.CursorPara])
	if d.CursorOff < len(runes) {
		newRunes := append(runes[:d.CursorOff], runes[d.CursorOff+1:]...)
		d.Paragraphs[d.CursorPara] = string(newRunes)
		d.Dirty = true
		return
	}

	// at end of paragraph → merge with next
	if d.CursorPara+1 >= len(d.Paragraphs) {
		return
	}
	curr := d.Paragraphs[d.CursorPara]
	next := d.Paragraphs[d.CursorPara+1]
	d.Paragraphs[d.CursorPara] = curr + next
	d.Paragraphs = append(d.Paragraphs[:d.CursorPara+1], d.Paragraphs[d.CursorPara+2:]...)
	d.Dirty = true
}

// MoveLeft / MoveRight / MoveUp / MoveDown keep the cursor inside valid bounds.
func (d *Document) MoveLeft() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.CursorOff > 0 {
		d.CursorOff--
		return
	}
	if d.CursorPara > 0 {
		d.CursorPara--
		d.CursorOff = len([]rune(d.Paragraphs[d.CursorPara]))
	}
}

func (d *Document) MoveRight() {
	d.mu.Lock()
	defer d.mu.Unlock()
	runes := []rune(d.Paragraphs[d.CursorPara])
	if d.CursorOff < len(runes) {
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
	if d.CursorPara > 0 {
		d.CursorPara--
		runes := []rune(d.Paragraphs[d.CursorPara])
		if d.CursorOff > len(runes) {
			d.CursorOff = len(runes)
		}
	}
}

func (d *Document) MoveDown() {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.CursorPara+1 < len(d.Paragraphs) {
		d.CursorPara++
		runes := []rune(d.Paragraphs[d.CursorPara])
		if d.CursorOff > len(runes) {
			d.CursorOff = len(runes)
		}
	}
}

// WordCount returns a rough word count.
func (d *Document) WordCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	count := 0
	for _, p := range d.Paragraphs {
		fields := strings.Fields(p)
		count += len(fields)
	}
	return count
}

// CharCount returns the total number of runes (excluding the newlines we add).
func (d *Document) CharCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	n := 0
	for _, p := range d.Paragraphs {
		n += len([]rune(p))
	}
	return n
}

// ParagraphCount returns how many paragraphs exist.
func (d *Document) ParagraphCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return len(d.Paragraphs)
}