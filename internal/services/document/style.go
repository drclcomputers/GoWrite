// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package document

import "strings"

func styleTags(s Style, fg, bg string) (open, close string) {
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

func (d *Document) SetActiveFg(fg string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ActiveFg = fg
}

func (d *Document) SetActiveBg(bg string) {
	d.mu.Lock()
	defer d.mu.Unlock()
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
