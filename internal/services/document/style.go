// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

import (
	"strings"

	"gowrite/internal/services/uitheme"
)

func styleTags(s Style, fg, bg, reset string) (open, close string) {
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

func (d *Document) ToggleStyle(s Style) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ActiveStyle ^= s
}

func (d *Document) SetActiveStyle(s Style) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastEdit = editOther
	d.ActiveStyle = s
}

func (d *Document) ActiveStyleFlags() Style {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.ActiveStyle
}

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

func (d *Document) CurrentAlign() Align {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.CursorPara < 0 || d.CursorPara >= len(d.Paragraphs) {
		return AlignLeft
	}
	return d.Paragraphs[d.CursorPara].Align
}

func (d *Document) SetParagraphStyle(style ParagraphStyle) {
	if style > ParagraphCode {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	first, last := d.cursorParagraphs()
	changed := false
	for i := first; i <= last; i++ {
		if d.Paragraphs[i].ParaStyle != style {
			changed = true
			break
		}
	}
	if !changed {
		return
	}
	d.pushHistory()
	for i := first; i <= last; i++ {
		d.Paragraphs[i].ParaStyle = style
	}
	d.Dirty = true
}

func (d *Document) CurrentParagraphStyle() ParagraphStyle {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.CursorPara < 0 || d.CursorPara >= len(d.Paragraphs) {
		return ParagraphNormal
	}
	return d.Paragraphs[d.CursorPara].ParaStyle
}

func (d *Document) SetParagraphSpaceAfter(rows int) {
	rows = min(max(rows, 0), maxParagraphSpaceAfter)
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	first, last := d.cursorParagraphs()
	changed := false
	for i := first; i <= last; i++ {
		if d.Paragraphs[i].SpaceAfter != rows {
			changed = true
			break
		}
	}
	if !changed {
		return
	}
	d.pushHistory()
	for i := first; i <= last; i++ {
		d.Paragraphs[i].SpaceAfter = rows
	}
	d.Dirty = true
}

func (d *Document) CurrentParagraphSpaceAfter() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	if d.CursorPara < 0 || d.CursorPara >= len(d.Paragraphs) {
		return 0
	}
	return d.Paragraphs[d.CursorPara].SpaceAfter
}

func (s ParagraphStyle) String() string {
	switch s {
	case ParagraphTitle:
		return "Title"
	case ParagraphHeading1:
		return "Heading 1"
	case ParagraphHeading2:
		return "Heading 2"
	case ParagraphHeading3:
		return "Heading 3"
	case ParagraphQuote:
		return "Quote"
	case ParagraphCode:
		return "Code"
	default:
		return "Normal"
	}
}

func (s ParagraphStyle) OutlineLevel() int {
	switch s {
	case ParagraphHeading1:
		return 1
	case ParagraphHeading2:
		return 2
	case ParagraphHeading3:
		return 3
	default:
		return 0
	}
}

func (d *Document) Outline() []OutlineEntry {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var entries []OutlineEntry
	for i, p := range d.Paragraphs {
		level := p.ParaStyle.OutlineLevel()
		text := strings.TrimSpace(plainOf(p))
		if level > 0 && text != "" {
			entries = append(entries, OutlineEntry{Paragraph: i, Level: level, Text: text})
		}
	}
	return entries
}

func (d *Document) SetActiveFg(fg string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastEdit = editOther
	d.ActiveFg = fg
}

func (d *Document) SetActiveBg(bg string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastEdit = editOther
	d.ActiveBg = bg
}

func (d *Document) ActiveColors() (fg, bg string) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.ActiveFg, d.ActiveBg
}

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
		parts = append(parts, "fg:"+uitheme.ColorName(d.ActiveFg))
	}
	if d.ActiveBg != "" {
		parts = append(parts, "bg:"+uitheme.ColorName(d.ActiveBg))
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
