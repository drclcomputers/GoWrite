// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package document

import "unicode"

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

func (d *Document) CursorLine() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.CursorPara + 1
}

func (d *Document) CursorCol() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.CursorOff + 1
}

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

// GoToLine moves the cursor to the start of the given 1-based line.
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
