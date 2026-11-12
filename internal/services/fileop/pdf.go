// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package fileop

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"

	"github.com/phpdave11/gofpdf"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/gobolditalic"
	"golang.org/x/image/font/gofont/goitalic"
	"golang.org/x/image/font/gofont/goregular"
)

type pdfChunk struct {
	text  string
	style document.Style
	fg    string
	bg    string
	width float64
}

func pdfTableStyle(format document.TableCellFormat) document.Style {
	style := format.Style
	switch format.ParagraphStyle {
	case document.ParagraphTitle, document.ParagraphHeading1,
		document.ParagraphHeading2, document.ParagraphHeading3:
		style |= document.StyleBold
	case document.ParagraphQuote:
		style |= document.StyleItalic
	}
	return style
}

type pdfWriter struct {
	pdf                    *gofpdf.Fpdf
	state                  document.State
	page                   document.PageSettings
	width, height          float64
	left, right, top, down float64
}

func ExportPDF(filename string, state document.State) error {
	return writeAtomic(filename, func(w io.Writer) error {
		return writePDF(w, state)
	})
}

func writePDF(w io.Writer, state document.State) error {
	page := state.Page.Normalized()
	width, height := rtfPageDimensions(page)
	orientation := "P"
	if page.Orientation == document.Landscape {
		orientation = "L"
		if width < height {
			width, height = height, width
		}
	} else if width > height {
		width, height = height, width
	}
	pdf := gofpdf.NewCustom(&gofpdf.InitType{
		OrientationStr: orientation,
		UnitStr:        "mm",
		Size:           gofpdf.SizeType{Wd: width, Ht: height},
	})
	pdf.AddUTF8FontFromBytes("Go", "", goregular.TTF)
	pdf.AddUTF8FontFromBytes("Go", "B", gobold.TTF)
	pdf.AddUTF8FontFromBytes("Go", "I", goitalic.TTF)
	pdf.AddUTF8FontFromBytes("Go", "BI", gobolditalic.TTF)
	if err := pdf.Error(); err != nil {
		return fmt.Errorf("load PDF fonts: %w", err)
	}
	pdf.SetTitle(state.Title, true)
	pdf.AliasNbPages("{nb}")

	topMargin := float64(page.MarginTop) * 297 / 59
	bottomMargin := float64(page.MarginBottom) * 297 / 59
	if !emptyHeaderFooter(page.Header) || page.PageNumbers.Where == document.NumHeader {
		topMargin += 5
	}
	if !emptyHeaderFooter(page.Footer) || page.PageNumbers.Where == document.NumFooter {
		bottomMargin += 5
	}
	writer := &pdfWriter{
		pdf:    pdf,
		state:  state,
		page:   page,
		width:  width,
		height: height,
		left:   float64(page.MarginLeft) * 210 / 100,
		right:  float64(page.MarginRight) * 210 / 100,
		top:    topMargin,
		down:   bottomMargin,
	}
	pdf.SetMargins(writer.left, writer.top, writer.right)
	pdf.SetAutoPageBreak(true, writer.down)
	pdf.SetHeaderFunc(writer.header)
	pdf.SetFooterFunc(writer.footer)
	pdf.AddPage()

	for _, paragraph := range state.Paragraphs {
		if paragraph.TablePart {
			continue
		}
		if paragraph.PageBreak && pdf.GetY() > writer.top+0.1 {
			pdf.AddPage()
		}
		if paragraph.Table != nil {
			writer.writeTable(*paragraph.Table)
		} else {
			writer.writeParagraph(paragraph)
		}
		if pdf.Error() != nil {
			return fmt.Errorf("render PDF: %w", pdf.Error())
		}
	}
	if err := pdf.Error(); err != nil {
		return fmt.Errorf("render PDF: %w", err)
	}
	if err := pdf.Output(w); err != nil {
		return fmt.Errorf("write PDF: %w", err)
	}
	return nil
}

func (w *pdfWriter) header() {
	w.fillPage()
	if emptyHeaderFooter(w.page.Header) && w.page.PageNumbers.Where != document.NumHeader {
		w.pdf.SetY(w.top)
		return
	}
	w.pdf.SetFont("Go", "", 9)
	text := runningPDFText(w.page.Header, w.state.Title, w.page.PageNumbers, w.page.PageNumbers.Where == document.NumHeader, w.pdf.PageNo())
	w.writeRunningLine(text, max(w.top-5, 0))
	w.pdf.SetY(w.top)
}

func (w *pdfWriter) footer() {
	if emptyHeaderFooter(w.page.Footer) && w.page.PageNumbers.Where != document.NumFooter {
		return
	}
	w.pdf.SetFont("Go", "", 9)
	text := runningPDFText(w.page.Footer, w.state.Title, w.page.PageNumbers, w.page.PageNumbers.Where == document.NumFooter, w.pdf.PageNo())
	w.writeRunningLine(text, w.height-w.down)
}

func (w *pdfWriter) fillPage() {
	r, g, b := pdfColor(w.page.Background)
	w.pdf.SetFillColor(r, g, b)
	w.pdf.Rect(0, 0, w.width, w.height, "F")
}

func (w *pdfWriter) writeRunningLine(text [3]string, y float64) {
	left, right := w.left, w.width-w.right
	w.pdf.SetY(y)
	w.pdf.SetX(left)
	w.pdf.CellFormat((right-left)/3, 5, text[0], "", 0, "L", false, 0, "")
	w.pdf.CellFormat((right-left)/3, 5, text[1], "", 0, "C", false, 0, "")
	w.pdf.CellFormat((right-left)/3, 5, text[2], "", 0, "R", false, 0, "")
}

func (w *pdfWriter) writeParagraph(paragraph document.Paragraph) {
	if len(paragraph.Runs) == 0 {
		w.advanceLine(5)
		return
	}
	size := 11.0
	switch paragraph.ParaStyle {
	case document.ParagraphTitle:
		size = 22
	case document.ParagraphHeading1:
		size = 18
	case document.ParagraphHeading2:
		size = 15
	case document.ParagraphHeading3:
		size = 13
	}
	lineHeight := size * 0.45
	available := w.width - w.left - w.right
	var line []pdfChunk
	lineWidth := 0.0
	hadContent, endedWithBreak := false, false
	flush := func() {
		w.writeLine(line, lineWidth, available, lineHeight, paragraph.Align, size)
		line, lineWidth = nil, 0
	}
	for _, run := range paragraph.Runs {
		style, fg, bg, _ := effectiveRTFRun(paragraph.ParaStyle, run)
		fontStyle := pdfFontStyle(style)
		w.pdf.SetFont("Go", fontStyle, size)
		for _, r := range run.Text {
			if r == '\n' {
				flush()
				hadContent, endedWithBreak = true, true
				continue
			}
			hadContent, endedWithBreak = true, false
			text := string(r)
			if r == '\t' {
				text = "    "
			}
			pieceWidth := w.pdf.GetStringWidth(text)
			if lineWidth+pieceWidth > available && len(line) > 0 {
				flush()
			}
			if len(line) > 0 && line[len(line)-1].style == style &&
				line[len(line)-1].fg == fg && line[len(line)-1].bg == bg {
				line[len(line)-1].text += text
				line[len(line)-1].width += pieceWidth
			} else {
				line = append(line, pdfChunk{text: text, style: style, fg: fg, bg: bg, width: pieceWidth})
			}
			lineWidth += pieceWidth
		}
	}
	if len(line) > 0 || !hadContent || endedWithBreak {
		flush()
	}
	if paragraph.SpaceAfter > 0 {
		w.advanceLine(float64(paragraph.SpaceAfter) * 2)
	}
}

func (w *pdfWriter) writeLine(chunks []pdfChunk, lineWidth, available, lineHeight float64, align document.Align, size float64) {
	if lineHeight == 0 {
		lineHeight = 5
	}
	if w.pdf.GetY()+lineHeight > w.height-w.down {
		w.pdf.AddPage()
	}
	x := w.left
	switch align {
	case document.AlignCenter:
		x += maxFloat((available-lineWidth)/2, 0)
	case document.AlignRight:
		x += maxFloat(available-lineWidth, 0)
	}
	y := w.pdf.GetY()
	for _, chunk := range chunks {
		style := pdfFontStyle(chunk.style)
		w.pdf.SetFont("Go", style, size)
		r, g, b := pdfColor(chunk.fg)
		w.pdf.SetTextColor(r, g, b)
		if chunk.bg != "" {
			r, g, b = pdfColor(chunk.bg)
			w.pdf.SetFillColor(r, g, b)
		}
		fill := chunk.bg != ""
		w.pdf.SetXY(x, y)
		w.pdf.CellFormat(chunk.width, lineHeight, chunk.text, "", 0, "L", fill, 0, "")
		if chunk.style&(document.StyleUnderline|document.StyleStrike) != 0 {
			r, g, b := pdfColor(chunk.fg)
			w.pdf.SetDrawColor(r, g, b)
			if chunk.style&document.StyleUnderline != 0 {
				w.pdf.Line(x, y+lineHeight-0.8, x+chunk.width, y+lineHeight-0.8)
			}
			if chunk.style&document.StyleStrike != 0 {
				w.pdf.Line(x, y+lineHeight/2, x+chunk.width, y+lineHeight/2)
			}
		}
		x += chunk.width
	}
	w.pdf.SetY(y + lineHeight)
	w.pdf.SetX(w.left)
	w.pdf.SetTextColor(32, 33, 36)
}

func (w *pdfWriter) writeTable(table document.Table) {
	if len(table.Rows) == 0 || len(table.Rows[0]) == 0 {
		return
	}
	columns := len(table.Rows[0])
	cellWidth := (w.width - w.left - w.right) / float64(columns)
	fontSize, lineHeight := 9.0, 4.5
	for ri, row := range table.Rows {
		linesByCell := make([][][]byte, columns)
		rowLines := 1
		for ci := 0; ci < columns; ci++ {
			format := tableFormat(table, ri, ci)
			w.pdf.SetFont("Go", pdfFontStyle(pdfTableStyle(format)), fontSize)
			linesByCell[ci] = w.pdf.SplitLines([]byte(row[ci]), cellWidth-3)
			rowLines = max(rowLines, len(linesByCell[ci]))
		}
		height := float64(rowLines)*lineHeight + 2
		if w.pdf.GetY()+height > w.height-w.down {
			w.pdf.AddPage()
		}
		y := w.pdf.GetY()
		for ci := range row {
			format := tableFormat(table, ri, ci)
			x := w.left + float64(ci)*cellWidth
			w.pdf.SetDrawColor(136, 136, 136)
			w.pdf.SetXY(x, y)
			w.pdf.Rect(x, y, cellWidth, height, "D")
			w.pdf.SetXY(x+1.5, y+1)
			w.pdf.SetFont("Go", pdfFontStyle(pdfTableStyle(format)), fontSize)
			fr, fg, fb := pdfColor(format.Foreground)
			w.pdf.SetTextColor(fr, fg, fb)
			if format.Background != "" {
				br, bg, bb := pdfColor(format.Background)
				w.pdf.SetFillColor(br, bg, bb)
			}
			align := pdfAlign(format.Alignment)
			w.pdf.MultiCell(cellWidth-3, lineHeight, row[ci], "", align, format.Background != "")
		}
		w.pdf.SetY(y + height)
		w.pdf.SetX(w.left)
	}
	w.pdf.SetTextColor(32, 33, 36)
}

func tableFormat(table document.Table, row, column int) document.TableCellFormat {
	if row >= 0 && row < len(table.CellFormats) &&
		column >= 0 && column < len(table.CellFormats[row]) {
		return table.CellFormats[row][column]
	}
	return document.TableCellFormat{}
}

func pdfFontStyle(style document.Style) string {
	bold := style&document.StyleBold != 0
	italic := style&document.StyleItalic != 0
	if bold && italic {
		return "BI"
	}
	if bold {
		return "B"
	}
	if italic {
		return "I"
	}
	return ""
}

func pdfAlign(align document.Align) string {
	switch align {
	case document.AlignCenter:
		return "C"
	case document.AlignRight:
		return "R"
	default:
		return "L"
	}
}

func (w *pdfWriter) advanceLine(height float64) {
	if w.pdf.GetY()+height > w.height-w.down {
		w.pdf.AddPage()
	}
	w.pdf.SetY(w.pdf.GetY() + height)
}

func runningPDFText(value document.HeaderFooter, title string, numbers document.PageNumbers, showNumber bool, page int) [3]string {
	replace := strings.NewReplacer(
		"{title}", title,
		"{page}", strconv.Itoa(page),
		"{pages}", "{nb}",
	)
	slots := [3]string{
		replace.Replace(value.Left),
		replace.Replace(value.Center),
		replace.Replace(value.Right),
	}
	if showNumber {
		index := int(numbers.Align)
		if index < 0 || index > 2 {
			index = 0
		}
		number := string(numbers.Format)
		number = strings.ReplaceAll(number, "1", strconv.Itoa(page))
		number = strings.ReplaceAll(number, "N", "{nb}")
		if slots[index] != "" {
			slots[index] += " "
		}
		slots[index] += number
	}
	return slots
}

func pdfColor(value string) (int, int, int) {
	if !uitheme.ValidHex(value) {
		return 32, 33, 36
	}
	r, _ := strconv.ParseUint(value[1:3], 16, 8)
	g, _ := strconv.ParseUint(value[3:5], 16, 8)
	b, _ := strconv.ParseUint(value[5:7], 16, 8)
	return int(r), int(g), int(b)
}
