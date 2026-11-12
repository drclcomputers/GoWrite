// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package fileop

import (
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"

	"gowrite/internal/services/document"
)

func ExportHTML(filename string, state document.State) error {
	return writeAtomic(filename, func(w io.Writer) error {
		return writeHTML(w, state)
	})
}

func writeHTML(w io.Writer, state document.State) error {
	page := state.Page.Normalized()
	width, height := rtfPageDimensions(page)
	var b strings.Builder
	b.WriteString("<!doctype html>\n<html lang=\"en\">\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	b.WriteString("<title>" + html.EscapeString(state.Title) + "</title>\n<style>\n")
	fmt.Fprintf(&b, `@page { size: %.2fmm %.2fmm; margin: %.2fmm %.2fmm %.2fmm %.2fmm; }
* { box-sizing: border-box; }
body { color: #202124; background: %s; font: 11pt/1.45 Arial, sans-serif; margin: 0 auto; max-width: %.2fmm; }
p { margin: 0 0 1em; white-space: pre-wrap; }
h1, h2, h3 { line-height: 1.2; }
h1 { font-size: 2em; } h2 { font-size: 1.6em; } h3 { font-size: 1.3em; }
blockquote { border-left: 3px solid #888; margin: 1em 0; padding-left: 1em; }
pre { white-space: pre-wrap; font-family: monospace; }
table { border-collapse: collapse; width: 100%%; margin: 1em 0; }
td { border: 1px solid #888; padding: .35em .5em; vertical-align: top; }
.page-break { break-before: page; page-break-before: always; }
.running { color: #555; font-size: 9pt; display: grid; grid-template-columns: 1fr 1fr 1fr; }
.running span:nth-child(2) { text-align: center; }
.running span:nth-child(3) { text-align: right; }
@media print {
  .page-num-plain::before { content: counter(page); }
  .page-num-page::before { content: "Page " counter(page); }
  .page-num-of::before { content: counter(page) " of " counter(pages); }
  .page-num-page-of::before { content: "Page " counter(page) " of " counter(pages); }
  .page-num-total::before { content: counter(pages); }
}
</style>\n</head>\n<body>\n`,
		width, height,
		float64(page.MarginTop)*297/59,
		float64(page.MarginRight)*210/100,
		float64(page.MarginBottom)*297/59,
		float64(page.MarginLeft)*210/100,
		cssColor(page.Background),
		width-float64(page.MarginLeft+page.MarginRight)*210/100)
	if !emptyHeaderFooter(page.Header) || page.PageNumbers.Where == document.NumHeader {
		writeHTMLRunningLine(&b, "header", page.Header, state.Title, page.PageNumbers,
			page.PageNumbers.Where == document.NumHeader)
	}
	for _, paragraph := range state.Paragraphs {
		if paragraph.TablePart {
			continue
		}
		if paragraph.PageBreak {
			b.WriteString(`<div class="page-break"></div>` + "\n")
		}
		if paragraph.Table != nil {
			writeHTMLTable(&b, *paragraph.Table)
			continue
		}
		writeHTMLParagraph(&b, paragraph)
	}
	if !emptyHeaderFooter(page.Footer) || page.PageNumbers.Where == document.NumFooter {
		writeHTMLRunningLine(&b, "footer", page.Footer, state.Title, page.PageNumbers,
			page.PageNumbers.Where == document.NumFooter)
	}
	b.WriteString("</body>\n</html>\n")
	_, err := io.WriteString(w, b.String())
	return err
}

func writeHTMLParagraph(b *strings.Builder, paragraph document.Paragraph) {
	tag := "p"
	class := ""
	switch paragraph.ParaStyle {
	case document.ParagraphTitle:
		tag, class = "h1", ` class="title"`
	case document.ParagraphHeading1:
		tag = "h1"
	case document.ParagraphHeading2:
		tag = "h2"
	case document.ParagraphHeading3:
		tag = "h3"
	case document.ParagraphQuote:
		tag = "blockquote"
	case document.ParagraphCode:
		tag = "pre"
	}
	style := make([]string, 0, 2)
	switch paragraph.Align {
	case document.AlignCenter:
		style = append(style, "text-align:center")
	case document.AlignRight:
		style = append(style, "text-align:right")
	}
	if paragraph.SpaceAfter > 0 {
		style = append(style, fmt.Sprintf("margin-bottom:%.2fem", float64(paragraph.SpaceAfter)*0.25))
	}
	b.WriteByte('<')
	b.WriteString(tag)
	b.WriteString(class)
	if len(style) > 0 {
		b.WriteString(` style="`)
		b.WriteString(strings.Join(style, ";"))
		b.WriteByte('"')
	}
	b.WriteByte('>')
	for _, run := range paragraph.Runs {
		writeHTMLRun(b, run)
	}
	b.WriteString("</" + tag + ">\n")
}

func writeHTMLRun(b *strings.Builder, run document.Run) {
	styles := make([]string, 0, 2)
	if run.Fg != "" {
		styles = append(styles, "color:"+cssColor(run.Fg))
	}
	if run.Bg != "" {
		styles = append(styles, "background-color:"+cssColor(run.Bg))
	}
	text := strings.ReplaceAll(html.EscapeString(run.Text), "\n", "<br>\n")
	text = strings.ReplaceAll(text, "\t", "&#9;")
	if len(styles) > 0 {
		b.WriteString(`<span style="` + strings.Join(styles, ";") + `">`)
	}
	if run.Style&document.StyleBold != 0 {
		b.WriteString("<strong>")
	}
	if run.Style&document.StyleItalic != 0 {
		b.WriteString("<em>")
	}
	decoration := make([]string, 0, 2)
	if run.Style&document.StyleUnderline != 0 {
		decoration = append(decoration, "underline")
	}
	if run.Style&document.StyleStrike != 0 {
		decoration = append(decoration, "line-through")
	}
	if len(decoration) > 0 {
		b.WriteString(`<span style="text-decoration:` + strings.Join(decoration, " ") + `">`)
	}
	b.WriteString(text)
	if len(decoration) > 0 {
		b.WriteString("</span>")
	}
	if run.Style&document.StyleItalic != 0 {
		b.WriteString("</em>")
	}
	if run.Style&document.StyleBold != 0 {
		b.WriteString("</strong>")
	}
	if len(styles) > 0 {
		b.WriteString("</span>")
	}
}

func writeHTMLTable(b *strings.Builder, table document.Table) {
	b.WriteString("<table>\n<tbody>\n")
	for ri, row := range table.Rows {
		b.WriteString("<tr>")
		for ci, cell := range row {
			format := document.TableCellFormat{}
			if ri < len(table.CellFormats) && ci < len(table.CellFormats[ri]) {
				format = table.CellFormats[ri][ci]
			}
			style := make([]string, 0, 3)
			if format.Foreground != "" {
				style = append(style, "color:"+cssColor(format.Foreground))
			}
			if format.Background != "" {
				style = append(style, "background-color:"+cssColor(format.Background))
			}
			switch format.Alignment {
			case document.AlignCenter:
				style = append(style, "text-align:center")
			case document.AlignRight:
				style = append(style, "text-align:right")
			}
			b.WriteString("<td")
			if len(style) > 0 {
				b.WriteString(` style="` + strings.Join(style, ";") + `"`)
			}
			b.WriteByte('>')
			cellStyle := format.Style
			switch format.ParagraphStyle {
			case document.ParagraphTitle, document.ParagraphHeading1,
				document.ParagraphHeading2, document.ParagraphHeading3:
				cellStyle |= document.StyleBold
			case document.ParagraphQuote:
				cellStyle |= document.StyleItalic
			}
			writeHTMLRun(b, document.Run{Text: cell, Style: cellStyle})
			b.WriteString("</td>")
		}
		b.WriteString("</tr>\n")
	}
	b.WriteString("</tbody>\n</table>\n")
}

func cssColor(value string) string {
	if len(value) == 7 && value[0] == '#' {
		if _, err := strconv.ParseUint(value[1:], 16, 24); err == nil {
			return strings.ToLower(value)
		}
	}
	return "#000000"
}

func emptyHeaderFooter(value document.HeaderFooter) bool {
	return value.Left == "" && value.Center == "" && value.Right == ""
}

func writeHTMLRunningLine(b *strings.Builder, tag string, value document.HeaderFooter, title string, numbers document.PageNumbers, showNumber bool) {
	slots := []string{value.Left, value.Center, value.Right}
	for i := range slots {
		slots[i] = htmlEscapeTokens(slots[i], title)
	}
	if showNumber {
		index := int(numbers.Align)
		if index < 0 || index > 2 {
			index = 0
		}
		if slots[index] != "" {
			slots[index] += " "
		}
		slots[index] += `<span class="page-num-` + htmlNumberClass(numbers.Format) + `"></span>`
	}
	b.WriteString("<" + tag + ` class="running"><span>` + slots[0] + "</span><span>" +
		slots[1] + "</span><span>" + slots[2] + "</span></" + tag + ">\n")
}

func htmlEscapeTokens(value, title string) string {
	var b strings.Builder
	for len(value) > 0 {
		index, token := len(value), ""
		for _, candidate := range []string{"{title}", "{page}", "{pages}"} {
			if i := strings.Index(value, candidate); i >= 0 && i < index {
				index, token = i, candidate
			}
		}
		b.WriteString(html.EscapeString(value[:index]))
		switch token {
		case "{title}":
			b.WriteString(html.EscapeString(title))
		case "{page}":
			b.WriteString(`<span class="page-num-plain"></span>`)
		case "{pages}":
			b.WriteString(`<span class="page-num-total"></span>`)
		}
		if token == "" {
			break
		}
		value = value[index+len(token):]
	}
	return b.String()
}

func htmlNumberClass(format document.NumberFormat) string {
	switch format {
	case document.FormatPage:
		return "page"
	case document.FormatOf:
		return "of"
	case document.FormatPageOf:
		return "page-of"
	default:
		return "plain"
	}
}
