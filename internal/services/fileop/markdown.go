// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package fileop

import (
	"io"
	"strconv"
	"strings"
	"unicode"

	"gowrite/internal/services/document"
)

func ExportMarkdown(filename string, state document.State) error {
	return writeAtomic(filename, func(w io.Writer) error {
		return writeMarkdown(w, state)
	})
}

func writeMarkdown(w io.Writer, state document.State) error {
	var b strings.Builder
	previousWasList := false
	appendBlock := func(block string, isList bool) {
		if b.Len() > 0 {
			if previousWasList && isList {
				b.WriteByte('\n')
			} else {
				b.WriteString("\n\n")
			}
		}
		b.WriteString(block)
		previousWasList = isList
	}

	for _, paragraph := range state.Paragraphs {
		if paragraph.TablePart {
			continue
		}
		if paragraph.PageBreak {
			appendBlock(markdownPageBreak, false)
		}
		if paragraph.Table != nil {
			appendBlock(markdownTable(*paragraph.Table), false)
			continue
		}
		block, isList := markdownParagraph(paragraph)
		appendBlock(block, isList)
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func markdownParagraph(paragraph document.Paragraph) (string, bool) {
	plain := paragraphPlainText(paragraph)
	if paragraph.ParaStyle == document.ParagraphNormal {
		if indent, marker, bodyOffset, ok := markdownListPrefix(plain); ok {
			body := markdownStyledRuns(paragraph.Runs, bodyOffset)
			return indent + marker + body, true
		}
	}

	if paragraph.ParaStyle == document.ParagraphCode {
		fence := markdownCodeFence(plain)
		return fence + "\n" + plain + "\n" + fence, false
	}

	body := markdownStyledRuns(paragraph.Runs, 0)
	switch paragraph.ParaStyle {
	case document.ParagraphTitle, document.ParagraphHeading1:
		return "# " + body, false
	case document.ParagraphHeading2:
		return "## " + body, false
	case document.ParagraphHeading3:
		return "### " + body, false
	case document.ParagraphQuote:
		lines := strings.Split(body, "\n")
		for i := range lines {
			lines[i] = "> " + lines[i]
		}
		return strings.Join(lines, "\n"), false
	default:
		return body, false
	}
}

func markdownListPrefix(text string) (indent, marker string, bodyOffset int, ok bool) {
	runes := []rune(text)
	i := 0
	for i < len(runes) && (runes[i] == ' ' || runes[i] == '\t') {
		if runes[i] == '\t' {
			indent += "    "
		} else {
			indent += " "
		}
		i++
	}
	start := i
	rest := string(runes[i:])
	switch {
	case strings.HasPrefix(rest, "[ ] "), strings.HasPrefix(rest, "[x] "), strings.HasPrefix(rest, "[X] "):
		marker = "- " + rest[:3] + " "
		i += 4
	case strings.HasPrefix(rest, "• "), strings.HasPrefix(rest, "- "),
		strings.HasPrefix(rest, "* "), strings.HasPrefix(rest, "– "),
		strings.HasPrefix(rest, "— "):
		marker = "- "
		i += len([]rune(strings.SplitN(rest, " ", 2)[0])) + 1
		if strings.HasPrefix(rest, "– ") || strings.HasPrefix(rest, "— ") {
			i = start + 2
		}
	case startsWithOrderedMarker(rest):
		n := 0
		for n < len(rest) && ((rest[n] >= '0' && rest[n] <= '9') ||
			(rest[n] >= 'a' && rest[n] <= 'z') || (rest[n] >= 'A' && rest[n] <= 'Z')) {
			n++
		}
		if isDigits(rest[:n]) {
			marker = rest[:n] + ". "
		} else {
			marker = "1. "
		}
		i += n + 2
	default:
		return "", "", 0, false
	}
	return indent, marker, i, true
}

func isDigits(text string) bool {
	if text == "" {
		return false
	}
	for i := range text {
		if text[i] < '0' || text[i] > '9' {
			return false
		}
	}
	return true
}

func startsWithOrderedMarker(text string) bool {
	if strings.HasPrefix(text, "[") {
		return false
	}
	i := 0
	for i < len(text) && ((text[i] >= '0' && text[i] <= '9') ||
		(text[i] >= 'a' && text[i] <= 'z') || (text[i] >= 'A' && text[i] <= 'Z')) {
		i++
	}
	if i == 0 || i+1 >= len(text) || text[i] != '.' || text[i+1] != ' ' {
		return false
	}
	token := text[:i]
	if isDigits(token) || len(token) == 1 {
		return true
	}
	switch strings.ToLower(token) {
	case "i", "ii", "iii", "iv", "v", "vi", "vii", "viii", "ix", "x",
		"xi", "xii", "xiii", "xiv", "xv", "xvi", "xvii", "xviii", "xix", "xx":
		return true
	default:
		return false
	}
}

func paragraphPlainText(paragraph document.Paragraph) string {
	var b strings.Builder
	for _, run := range paragraph.Runs {
		b.WriteString(run.Text)
	}
	return b.String()
}

func markdownStyledRuns(runs []document.Run, start int) string {
	var b strings.Builder
	position := 0
	for _, run := range runs {
		runes := []rune(run.Text)
		end := position + len(runes)
		if end > start {
			from := max(0, start-position)
			b.WriteString(markdownInlineRun(run, string(runes[from:])))
		}
		position = end
	}
	return b.String()
}

func markdownInlineRun(run document.Run, text string) string {
	text = escapeMarkdown(text)
	switch run.Style & (document.StyleBold | document.StyleItalic) {
	case document.StyleBold | document.StyleItalic:
		text = wrapMarkdown(text, "***", "***")
	case document.StyleBold:
		text = wrapMarkdown(text, "**", "**")
	case document.StyleItalic:
		text = wrapMarkdown(text, "*", "*")
	}
	if run.Style&document.StyleStrike != 0 {
		text = wrapMarkdown(text, "~~", "~~")
	}
	if run.Style&document.StyleUnderline != 0 {
		text = wrapMarkdown(text, "<u>", "</u>")
	}
	styles := make([]string, 0, 2)
	if color := markdownColor(run.Fg); color != "" {
		styles = append(styles, "color:"+color)
	}
	if color := markdownColor(run.Bg); color != "" {
		styles = append(styles, "background-color:"+color)
	}
	if len(styles) > 0 {
		text = wrapMarkdown(text, `<span style="`+strings.Join(styles, ";")+`">`, `</span>`)
	}
	return text
}

func wrapMarkdown(text, open, close string) string {
	runes := []rune(text)
	start, end := 0, len(runes)
	for start < end && unicode.IsSpace(runes[start]) {
		start++
	}
	for end > start && unicode.IsSpace(runes[end-1]) {
		end--
	}
	if start == end {
		return text
	}
	return string(runes[:start]) + open + string(runes[start:end]) + close + string(runes[end:])
}

func escapeMarkdown(text string) string {
	var b strings.Builder
	for _, r := range text {
		if strings.ContainsRune("\\`*_{}[]<>#+-.!|", r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func markdownColor(value string) string {
	if len(value) != 7 || value[0] != '#' {
		return ""
	}
	if _, err := strconv.ParseUint(value[1:], 16, 24); err != nil {
		return ""
	}
	return strings.ToLower(value)
}

func markdownCodeFence(text string) string {
	maxTicks := 2
	for _, line := range strings.Split(text, "\n") {
		count := 0
		for _, r := range line {
			if r == '`' {
				count++
				if count > maxTicks {
					maxTicks = count
				}
			} else {
				count = 0
			}
		}
	}
	return strings.Repeat("`", maxTicks+1)
}

func markdownTable(table document.Table) string {
	if len(table.Rows) == 0 {
		return ""
	}
	columns := 0
	for _, row := range table.Rows {
		if len(row) > columns {
			columns = len(row)
		}
	}
	if columns == 0 {
		return ""
	}
	var b strings.Builder
	writeRow := func(ri int) {
		b.WriteString("| ")
		for ci := 0; ci < columns; ci++ {
			if ci > 0 {
				b.WriteString(" | ")
			}
			if ri >= len(table.Rows) || ci >= len(table.Rows[ri]) {
				continue
			}
			format := document.TableCellFormat{}
			if ri < len(table.CellFormats) && ci < len(table.CellFormats[ri]) {
				format = table.CellFormats[ri][ci]
			}
			style := format.Style
			switch format.ParagraphStyle {
			case document.ParagraphTitle, document.ParagraphHeading1,
				document.ParagraphHeading2, document.ParagraphHeading3:
				style |= document.StyleBold
			case document.ParagraphQuote:
				style |= document.StyleItalic
			}
			text := table.Rows[ri][ci]
			rendered := markdownInlineRun(document.Run{
				Text: text, Style: style, Fg: format.Foreground, Bg: format.Background,
			}, text)
			b.WriteString(strings.ReplaceAll(rendered, "\n", "<br>"))
		}
		b.WriteString(" |")
	}

	writeRow(0)
	b.WriteByte('\n')
	b.WriteString("|")
	for ci := 0; ci < columns; ci++ {
		align := document.AlignLeft
		if len(table.CellFormats) > 0 && ci < len(table.CellFormats[0]) {
			align = table.CellFormats[0][ci].Alignment
		}
		separator := " --- "
		switch align {
		case document.AlignCenter:
			separator = " :---: "
		case document.AlignRight:
			separator = " ---: "
		}
		b.WriteString(separator + "|")
	}
	for ri := 1; ri < len(table.Rows); ri++ {
		b.WriteByte('\n')
		writeRow(ri)
	}
	return b.String()
}

const markdownPageBreak = `<div style="page-break-before: always;"></div>`
