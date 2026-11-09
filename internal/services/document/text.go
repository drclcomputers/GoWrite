// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

import "strings"

// plainOf returns the plain (style-stripped) text of a paragraph.
func plainOf(p Paragraph) string {
	var b strings.Builder
	for _, r := range p.Runs {
		b.WriteString(r.Text)
	}
	return b.String()
}

func tableText(table *Table) string {
	rows := make([]string, len(table.Rows))
	for i, row := range table.Rows {
		rows[i] = strings.Join(row, "\t")
	}
	return strings.Join(rows, "\n")
}

// Text returns the full document as plain text (paragraphs joined by \n).
func (d *Document) Text() string {
	d.mu.RLock()
	defer d.mu.RUnlock()
	parts := make([]string, 0, len(d.Paragraphs))
	for _, p := range d.Paragraphs {
		if p.TablePart {
			continue
		}
		if p.Table != nil {
			parts = append(parts, tableText(p.Table))
		} else {
			parts = append(parts, plainOf(p))
		}
	}
	return strings.Join(parts, "\n")
}

// AbsoluteCursorOffset returns the cursor's plain-text offset.
func (d *Document) AbsoluteCursorOffset() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	off := 0
	for i := 0; i < d.CursorPara && i < len(d.Paragraphs); i++ {
		off += len([]rune(plainOf(d.Paragraphs[i]))) + 1
	}
	off += d.CursorOff
	return off
}

// SetText replaces the content with unstyled text.
func (d *Document) SetText(s string) {
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
	d.lastEdit = editOther
	d.Dirty = true
}

// LoadText replaces the content with s and marks the document clean.
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

func (d *Document) WordCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	count := 0
	for _, p := range d.Paragraphs {
		if p.Table != nil {
			for _, row := range p.Table.Rows {
				for _, cell := range row {
					count += len(strings.Fields(cell))
				}
			}
		} else if !p.TablePart {
			count += len(strings.Fields(plainOf(p)))
		}
	}
	return count
}

func (d *Document) CharCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	n := 0
	for _, p := range d.Paragraphs {
		if p.Table != nil {
			for _, row := range p.Table.Rows {
				for _, cell := range row {
					n += len([]rune(cell))
				}
			}
		} else if !p.TablePart {
			n += len([]rune(plainOf(p)))
		}
	}
	return n
}

func (d *Document) ParagraphCount() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	count := 0
	for _, paragraph := range d.Paragraphs {
		if !paragraph.TablePart {
			count++
		}
	}
	return count
}
