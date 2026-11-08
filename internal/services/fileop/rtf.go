// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package fileop

import (
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf16"

	"gowrite/internal/services/document"
)

func ExportRTF(filename string, state document.State) error {
	rtf := encodeRTF(state)
	return writeAtomic(filename, func(w io.Writer) error {
		_, err := io.WriteString(w, rtf)
		return err
	})
}

func encodeRTF(state document.State) string {
	page := state.Page.Normalized()
	colors, indices := collectRTFColors(state)
	pageWidthMM, pageHeightMM := rtfPageDimensions(page)
	var b strings.Builder
	b.WriteString(`{\rtf1\ansi\deff0\uc1`)
	b.WriteString(`{\fonttbl{\f0\fnil Arial;}{\f1\fmodern Courier New;}}`)
	writeRTFColorTable(&b, colors)
	b.WriteString(`{\info{\title `)
	writeRTFText(&b, state.Title)
	b.WriteString(`}}`)
	fmt.Fprintf(&b, `\paperw%d\paperh%d\margl%d\margr%d\margt%d\margb%d`,
		mmToTwips(pageWidthMM),
		mmToTwips(pageHeightMM),
		mmToTwips(float64(page.MarginLeft)*210/100),
		mmToTwips(float64(page.MarginRight)*210/100),
		mmToTwips(float64(page.MarginTop)*297/59),
		mmToTwips(float64(page.MarginBottom)*297/59))

	pageWidth := mmToTwips(pageWidthMM) -
		mmToTwips(float64(page.MarginLeft)*210/100) -
		mmToTwips(float64(page.MarginRight)*210/100)
	writeRTFHeaderFooter(&b, "header", page.Header, page.PageNumbers, page.PageNumbers.Where == document.NumHeader, state.Title, pageWidth)
	writeRTFHeaderFooter(&b, "footer", page.Footer, page.PageNumbers, page.PageNumbers.Where == document.NumFooter, state.Title, pageWidth)

	for i, paragraph := range state.Paragraphs {
		if i > 0 && paragraph.PageBreak {
			b.WriteString(`\page `)
		}
		b.WriteString(`\pard`)
		switch paragraph.Align {
		case document.AlignCenter:
			b.WriteString(`\qc`)
		case document.AlignRight:
			b.WriteString(`\qr`)
		default:
			b.WriteString(`\ql`)
		}
		if paragraph.SpaceAfter > 0 {
			fmt.Fprintf(&b, `\sa%d`, rowsToTwips(paragraph.SpaceAfter))
		}
		if paragraph.ParaStyle == document.ParagraphQuote {
			b.WriteString(`\li360\ri360`)
		}
		switch paragraph.ParaStyle {
		case document.ParagraphTitle:
			b.WriteString(`\keepn`)
		case document.ParagraphHeading1:
			b.WriteString(`\outlinelevel0\keepn`)
		case document.ParagraphHeading2:
			b.WriteString(`\outlinelevel1\keepn`)
		case document.ParagraphHeading3:
			b.WriteString(`\outlinelevel2\keepn`)
		}
		b.WriteByte(' ')
		for _, run := range paragraph.Runs {
			style, fg, bg, font := effectiveRTFRun(paragraph.ParaStyle, run)
			b.WriteString(`{\f`)
			b.WriteString(strconv.Itoa(font))
			if idx := indices[fg]; idx > 0 {
				fmt.Fprintf(&b, `\cf%d`, idx)
			}
			if idx := indices[bg]; idx > 0 {
				fmt.Fprintf(&b, `\highlight%d`, idx)
			}
			if style&document.StyleBold != 0 {
				b.WriteString(`\b`)
			}
			if style&document.StyleItalic != 0 {
				b.WriteString(`\i`)
			}
			if style&document.StyleUnderline != 0 {
				b.WriteString(`\ul`)
			}
			if style&document.StyleStrike != 0 {
				b.WriteString(`\strike`)
			}
			b.WriteByte(' ')
			writeRTFText(&b, run.Text)
			b.WriteByte('}')
		}
		b.WriteString(`\par `)
	}
	b.WriteByte('}')
	return b.String()
}

func rtfPageDimensions(page document.PageSettings) (float64, float64) {
	var width, height float64
	switch page.Size {
	case document.SizeA3:
		width, height = 297, 420
	case document.SizeA4:
		width, height = 210, 297
	case document.SizeA5:
		width, height = 148, 210
	case document.SizeLetter:
		width, height = 215.9, 279.4
	case document.SizeLegal:
		width, height = 215.9, 355.6
	default:
		width, height = float64(page.Width)*210/100, float64(page.Height)*297/59
	}
	if page.Orientation == document.Landscape {
		width, height = height, width
	}
	return width, height
}

func collectRTFColors(state document.State) ([][3]int, map[string]int) {
	var colors [][3]int
	indices := make(map[string]int)
	add := func(value string) {
		if value == "" {
			return
		}
		if _, ok := indices[value]; ok {
			return
		}
		rgb, ok := parseRTFColor(value)
		if !ok {
			return
		}
		colors = append(colors, rgb)
		indices[value] = len(colors)
	}
	for _, paragraph := range state.Paragraphs {
		_, defaultFg, _ := paragraphDefaults(paragraph.ParaStyle)
		for _, run := range paragraph.Runs {
			fg := run.Fg
			if fg == "" {
				fg = defaultFg
			}
			add(fg)
			add(run.Bg)
		}
	}
	return colors, indices
}

func writeRTFColorTable(b *strings.Builder, colors [][3]int) {
	b.WriteString(`{\colortbl;`)
	for _, color := range colors {
		fmt.Fprintf(b, `\red%d\green%d\blue%d;`, color[0], color[1], color[2])
	}
	b.WriteByte('}')
}

func effectiveRTFRun(style document.ParagraphStyle, run document.Run) (document.Style, string, string, int) {
	runStyle, defaultFg, _ := paragraphDefaults(style)
	fg := run.Fg
	if fg == "" {
		fg = defaultFg
	}
	font := 0
	if style == document.ParagraphCode {
		font = 1
	}
	return run.Style | runStyle, fg, run.Bg, font
}

func paragraphDefaults(style document.ParagraphStyle) (document.Style, string, string) {
	switch style {
	case document.ParagraphTitle:
		return document.StyleBold, "#f0d090", ""
	case document.ParagraphHeading1:
		return document.StyleBold, "#5ec8c8", ""
	case document.ParagraphHeading2:
		return document.StyleBold, "#8ecae6", ""
	case document.ParagraphHeading3:
		return document.StyleBold, "", ""
	case document.ParagraphQuote:
		return document.StyleItalic, "", ""
	case document.ParagraphCode:
		return document.StyleNone, "#8ecae6", ""
	default:
		return document.StyleNone, "", ""
	}
}

func parseRTFColor(value string) ([3]int, bool) {
	var rgb [3]int
	if len(value) != 7 || value[0] != '#' {
		return rgb, false
	}
	for i := range rgb {
		component, err := strconv.ParseUint(value[1+i*2:3+i*2], 16, 8)
		if err != nil {
			return rgb, false
		}
		rgb[i] = int(component)
	}
	return rgb, true
}

func writeRTFHeaderFooter(b *strings.Builder, name string, value document.HeaderFooter, numbers document.PageNumbers, showNumber bool, title string, pageWidth int) {
	if value.Left == "" && value.Center == "" && value.Right == "" && !showNumber {
		return
	}
	slots := []string{value.Left, value.Center, value.Right}
	if showNumber {
		index := 0
		switch numbers.Align {
		case document.AlignCenter:
			index = 1
		case document.AlignRight:
			index = 2
		}
		number := rtfPageNumber(numbers.Format)
		if slots[index] != "" {
			slots[index] += " "
		}
		slots[index] += number
	}

	fmt.Fprintf(b, `{\%s\pard\plain\ql\tqc\tx%d\tqr\tx%d `, name, pageWidth/2, pageWidth)
	writeRTFHeaderText(b, slots[0], title)
	b.WriteString(`\tab `)
	writeRTFHeaderText(b, slots[1], title)
	b.WriteString(`\tab `)
	writeRTFHeaderText(b, slots[2], title)
	b.WriteString(`\par}`)
}

func rtfPageNumber(format document.NumberFormat) string {
	page := `{\field{\*\fldinst PAGE}{\fldrslt 1}}`
	pages := `{\field{\*\fldinst NUMPAGES}{\fldrslt 1}}`
	switch format {
	case document.FormatPage:
		return "Page " + page
	case document.FormatOf:
		return page + " of " + pages
	case document.FormatPageOf:
		return "Page " + page + " of " + pages
	default:
		return page
	}
}

func writeRTFHeaderText(b *strings.Builder, text, title string) {
	for len(text) > 0 {
		tokenAt, token := len(text), ""
		for _, candidate := range []string{"{page}", "{pages}", "{title}"} {
			if i := strings.Index(text, candidate); i >= 0 && i < tokenAt {
				tokenAt, token = i, candidate
			}
		}
		writeRTFText(b, text[:tokenAt])
		if token == "" {
			break
		}
		switch token {
		case "{page}":
			b.WriteString(rtfPageNumber(document.FormatPlain))
		case "{pages}":
			b.WriteString(`{\field{\*\fldinst NUMPAGES}{\fldrslt 1}}`)
		case "{title}":
			writeRTFText(b, title)
		}
		text = text[tokenAt+len(token):]
	}
}

func writeRTFText(b *strings.Builder, text string) {
	for _, r := range text {
		switch r {
		case '\\', '{', '}':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\t':
			b.WriteString(`\tab `)
		case '\n':
			b.WriteString(`\line `)
		case '\r':
		default:
			if r >= 0x20 && r < 0x7f {
				b.WriteRune(r)
				continue
			}
			for _, unit := range utf16.Encode([]rune{r}) {
				signed := int(int16(unit))
				fmt.Fprintf(b, `\u%d?`, signed)
			}
		}
	}
}

func mmToTwips(mm float64) int {
	return int(math.Round(mm * 1440 / 25.4))
}

func rowsToTwips(rows int) int {
	return mmToTwips(float64(rows) * 297 / 59)
}
