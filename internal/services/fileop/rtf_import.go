// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package fileop

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf16"

	"gowrite/internal/services/document"
)

const (
	maxRTFSize      = 64 << 20
	maxRTFTableRows = 10000
	maxRTFTableCols = 256
	maxRTFDepth     = 4096
)

type rtfState struct {
	style     document.Style
	fg        string
	bg        string
	align     document.Align
	paraStyle document.ParagraphStyle
	unicode   int
}

type rtfParser struct {
	data         []byte
	pos          int
	state        rtfState
	stack        []rtfState
	colors       []string
	paragraphs   []document.Paragraph
	current      document.Paragraph
	text         strings.Builder
	pendingBreak bool
	pendingHigh  uint16
	page         document.PageSettings
	paperWidth   int
	paperHeight  int
	err          error

	tableActive   bool
	inCell        bool
	cellClosed    bool
	cellFormatSet bool
	cellText      strings.Builder
	cellFormat    document.TableCellFormat
	row           []string
	rowFormats    []document.TableCellFormat
	rows          [][]string
	formats       [][]document.TableCellFormat
}

var rtfColorPattern = regexp.MustCompile(`\\red(\d+)\\green(\d+)\\blue(\d+)`)

func LoadRTF(filename string) (document.State, error) {
	file, err := os.Open(filename)
	if err != nil {
		return document.State{}, err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return document.State{}, err
	}
	if info.Size() > maxRTFSize {
		return document.State{}, fmt.Errorf("RTF file exceeds the %d-byte limit", maxRTFSize)
	}
	data, err := io.ReadAll(io.LimitReader(file, maxRTFSize+1))
	if err != nil {
		return document.State{}, err
	}
	if len(data) > maxRTFSize {
		return document.State{}, fmt.Errorf("RTF file exceeds the %d-byte limit", maxRTFSize)
	}
	trimmed := strings.TrimSpace(string(data))
	if !strings.HasPrefix(trimmed, `{\rtf`) {
		return document.State{}, fmt.Errorf("file is not an RTF document")
	}

	parser := &rtfParser{
		data:   data,
		state:  rtfState{unicode: 1},
		colors: []string{""},
	}
	parser.resetParagraph()
	if err := parser.parse(); err != nil {
		return document.State{}, err
	}
	page := document.DefaultA4Portrait()
	if parser.page.Width > 0 && parser.page.Height > 0 {
		page = parser.page
		if parser.paperWidth > parser.paperHeight {
			page.Orientation = document.Landscape
		} else if parser.paperWidth > 0 && parser.paperHeight > 0 {
			page.Orientation = document.Portrait
		}
	}
	return document.State{
		Title:      "",
		Paragraphs: parser.paragraphs,
		Page:       page,
	}, nil
}

func (p *rtfParser) parse() error {
	for p.pos < len(p.data) {
		switch p.data[p.pos] {
		case '{':
			if end, destination := rtfDestinationEnd(p.data, p.pos); end > 0 {
				if destination == "colortbl" {
					p.parseColorTable(p.data[p.pos:end])
				}
				p.pos = end
				continue
			}
			if len(p.stack) >= maxRTFDepth {
				return fmt.Errorf("RTF nesting exceeds the %d-group limit", maxRTFDepth)
			}
			p.stack = append(p.stack, p.state)
			p.pos++
		case '}':
			p.flushText()
			p.pos++
			if len(p.stack) > 0 {
				p.state = p.stack[len(p.stack)-1]
				p.stack = p.stack[:len(p.stack)-1]
			}
		case '\\':
			if err := p.control(); err != nil {
				return err
			}
		case '\r', '\n':
			p.pos++
		default:
			if p.tableActive && !p.inCell {
				p.flushTable()
			}
			if p.data[p.pos] >= 0x80 {
				p.text.WriteRune(decodeRTFByte(p.data[p.pos]))
			} else {
				p.text.WriteByte(p.data[p.pos])
			}
			p.pos++
		}
		if p.err != nil {
			return p.err
		}
	}
	p.flushText()
	if p.tableActive {
		p.finishRow()
		p.flushTable()
	}
	if p.err != nil {
		return p.err
	}
	p.finishParagraph(false)
	if len(p.paragraphs) == 0 {
		p.resetParagraph()
		p.finishParagraph(true)
	}
	return nil
}

func (p *rtfParser) control() error {
	p.pos++
	if p.pos >= len(p.data) {
		return fmt.Errorf("incomplete RTF control")
	}
	next := p.data[p.pos]
	if next == '\'' {
		if p.pos+2 >= len(p.data) {
			return fmt.Errorf("incomplete RTF hexadecimal escape")
		}
		value, err := strconv.ParseUint(string(p.data[p.pos+1:p.pos+3]), 16, 8)
		if err != nil {
			return fmt.Errorf("invalid RTF hexadecimal escape: %w", err)
		}
		p.text.WriteRune(decodeRTFByte(byte(value)))
		p.pos += 3
		return nil
	}
	if next == '\\' || next == '{' || next == '}' {
		p.text.WriteByte(next)
		p.pos++
		return nil
	}
	if next == '~' {
		p.text.WriteRune('\u00a0')
		p.pos++
		return nil
	}
	if next == '_' {
		p.text.WriteRune('\u2011')
		p.pos++
		return nil
	}
	if next == '-' {
		p.pos++
		return nil
	}
	if next == '*' {
		p.pos++
		return nil
	}
	if !isRTFLetter(next) {
		p.pos++
		return nil
	}

	start := p.pos
	for p.pos < len(p.data) && isRTFLetter(p.data[p.pos]) {
		p.pos++
	}
	word := strings.ToLower(string(p.data[start:p.pos]))
	sign := 1
	if p.pos < len(p.data) && p.data[p.pos] == '-' {
		sign = -1
		p.pos++
	}
	numberStart := p.pos
	for p.pos < len(p.data) && p.data[p.pos] >= '0' && p.data[p.pos] <= '9' {
		p.pos++
	}
	value, hasValue := 0, numberStart != p.pos
	if hasValue {
		n, err := strconv.Atoi(string(p.data[numberStart:p.pos]))
		if err != nil {
			return fmt.Errorf("invalid RTF control value for %s: %w", word, err)
		}
		value = sign * n
	}
	if p.pos < len(p.data) && p.data[p.pos] == ' ' {
		p.pos++
	}
	if word == "bin" && hasValue {
		p.pos = min(p.pos+max(value, 0), len(p.data))
		return nil
	}
	if p.tableActive && !p.inCell && (word == "pard" || word == "par" || word == "page") {
		p.flushTable()
	}
	p.applyControl(word, value, hasValue)
	return nil
}

func (p *rtfParser) applyControl(word string, value int, hasValue bool) {
	switch word {
	case "b", "i", "ul", "strike":
		p.flushText()
		set := !hasValue || value != 0
		switch word {
		case "b":
			p.state.style = setStyle(p.state.style, document.StyleBold, set)
		case "i":
			p.state.style = setStyle(p.state.style, document.StyleItalic, set)
		case "ul":
			p.state.style = setStyle(p.state.style, document.StyleUnderline, set)
		case "strike":
			p.state.style = setStyle(p.state.style, document.StyleStrike, set)
		}
	case "ulnone":
		p.flushText()
		p.state.style &^= document.StyleUnderline
	case "plain":
		p.flushText()
		p.state.style = document.StyleNone
		p.state.fg, p.state.bg = "", ""
	case "cf":
		p.flushText()
		p.state.fg = p.color(value)
	case "highlight":
		p.flushText()
		p.state.bg = p.color(value)
	case "uc":
		p.state.unicode = max(value, 0)
	case "u":
		p.flushText()
		unit := uint16(value)
		if value < 0 {
			unit = uint16(value + 65536)
		}
		if unit >= 0xd800 && unit <= 0xdbff {
			p.pendingHigh = unit
		} else if unit >= 0xdc00 && unit <= 0xdfff && p.pendingHigh != 0 {
			p.text.WriteRune(utf16.DecodeRune(rune(p.pendingHigh), rune(unit)))
			p.pendingHigh = 0
		} else {
			if p.pendingHigh != 0 {
				p.text.WriteRune('\uFFFD')
				p.pendingHigh = 0
			}
			p.text.WriteRune(rune(unit))
		}
		p.skipFallback(p.state.unicode)
	case "par":
		if p.tableActive && p.inCell {
			p.flushText()
			p.cellText.WriteByte('\n')
		} else {
			p.finishParagraph(true)
		}
	case "line":
		p.text.WriteByte('\n')
	case "tab":
		p.text.WriteByte('\t')
	case "page":
		if p.tableActive {
			p.finishRow()
			p.flushTable()
		}
		p.finishParagraph(false)
		p.pendingBreak = true
	case "ql", "qc", "qr":
		p.flushText()
		switch word {
		case "qc":
			p.state.align = document.AlignCenter
		case "qr":
			p.state.align = document.AlignRight
		default:
			p.state.align = document.AlignLeft
		}
		if len(p.current.Runs) == 0 {
			p.current.Align = p.state.align
		}
	case "pard":
		p.flushText()
		p.state.align = document.AlignLeft
		p.state.paraStyle = document.ParagraphNormal
		p.current.Align = p.state.align
		p.current.ParaStyle = p.state.paraStyle
		p.current.SpaceAfter = 0
	case "outlinelevel":
		p.flushText()
		switch value {
		case 0:
			p.state.paraStyle = document.ParagraphHeading1
		case 1:
			p.state.paraStyle = document.ParagraphHeading2
		case 2:
			p.state.paraStyle = document.ParagraphHeading3
		default:
			p.state.paraStyle = document.ParagraphNormal
		}
		p.current.ParaStyle = p.state.paraStyle
	case "sa":
		p.current.SpaceAfter = min(max(int(float64(max(value, 0))/float64(rowsToTwips(1))+0.5), 0), 10)
	case "paperw":
		p.ensurePage()
		p.paperWidth = value
		p.page.Width = max(1, int(float64(value)*25.4/1440*100/210+0.5))
		p.page.Size = ""
	case "paperh":
		p.ensurePage()
		p.paperHeight = value
		p.page.Height = max(1, int(float64(value)*25.4/1440*59/297+0.5))
		p.page.Size = ""
	case "margl", "margr":
		p.ensurePage()
		margin := max(0, int(float64(value)*25.4/1440*100/210+0.5))
		if word == "margl" {
			p.page.MarginLeft = margin
		} else {
			p.page.MarginRight = margin
		}
	case "margt", "margb":
		p.ensurePage()
		margin := max(0, int(float64(value)*25.4/1440*59/297+0.5))
		if word == "margt" {
			p.page.MarginTop = margin
		} else {
			p.page.MarginBottom = margin
		}
	case "trowd":
		if !p.tableActive {
			p.finishParagraph(false)
			p.tableActive = true
			p.rows, p.formats = nil, nil
		}
		p.inCell = false
		p.row, p.rowFormats = nil, nil
		p.cellText.Reset()
	case "intbl":
		p.inCell = true
		p.cellClosed = false
		p.cellFormat = p.currentCellFormat()
		p.cellFormatSet = false
	case "cell", "nestcell":
		p.finishCell()
	case "row", "nestrow":
		p.finishRow()
		p.inCell = false
	}
}

func (p *rtfParser) flushText() {
	if p.text.Len() == 0 {
		return
	}
	text := p.text.String()
	p.text.Reset()
	if p.tableActive && p.inCell {
		if !p.cellFormatSet && p.cellText.Len() == 0 {
			p.cellFormat = p.currentCellFormat()
			p.cellFormatSet = true
		}
		p.cellText.WriteString(text)
		return
	}
	run := document.Run{Text: text, Style: p.state.style, Fg: p.state.fg, Bg: p.state.bg}
	n := len(p.current.Runs)
	if n > 0 && p.current.Runs[n-1].Style == run.Style &&
		p.current.Runs[n-1].Fg == run.Fg && p.current.Runs[n-1].Bg == run.Bg {
		p.current.Runs[n-1].Text += text
	} else {
		p.current.Runs = append(p.current.Runs, run)
	}
}

func (p *rtfParser) finishParagraph(force bool) {
	p.flushText()
	if !force && len(p.current.Runs) == 0 {
		return
	}
	if len(p.current.Runs) == 0 {
		p.current.Runs = []document.Run{{Text: ""}}
	}
	p.current.PageBreak = p.pendingBreak
	p.paragraphs = append(p.paragraphs, p.current)
	p.pendingBreak = false
	p.resetParagraph()
}

func (p *rtfParser) resetParagraph() {
	p.current = document.Paragraph{
		Align:     p.state.align,
		ParaStyle: p.state.paraStyle,
	}
	if p.pendingBreak {
		p.current.PageBreak = true
	}
}

func (p *rtfParser) ensurePage() {
	if p.page.Width == 0 {
		p.page = document.DefaultA4Portrait()
	}
}

func (p *rtfParser) currentCellFormat() document.TableCellFormat {
	return document.TableCellFormat{
		Style: p.state.style, Foreground: p.state.fg, Background: p.state.bg,
		ParagraphStyle: p.state.paraStyle, Alignment: p.state.align,
	}
}

func (p *rtfParser) finishCell() {
	p.flushText()
	if !p.tableActive {
		return
	}
	if !p.inCell && p.cellClosed {
		return
	}
	if !p.cellFormatSet {
		p.cellFormat = p.currentCellFormat()
	}
	p.row = append(p.row, p.cellText.String())
	p.rowFormats = append(p.rowFormats, p.cellFormat)
	p.cellText.Reset()
	p.cellClosed = true
	p.inCell = false
	p.cellFormatSet = false
}

func (p *rtfParser) finishRow() {
	if !p.tableActive {
		return
	}
	if p.inCell || len(p.row) > 0 && !p.cellClosed {
		p.finishCell()
	}
	if len(p.row) > 0 {
		p.rows = append(p.rows, p.row)
		p.formats = append(p.formats, p.rowFormats)
		if len(p.rows) > maxRTFTableRows {
			p.err = fmt.Errorf("RTF table exceeds the %d-row limit", maxRTFTableRows)
		}
	}
	p.row, p.rowFormats = nil, nil
	p.cellClosed = false
}

func (p *rtfParser) flushTable() {
	if !p.tableActive {
		return
	}
	p.finishRow()
	if len(p.rows) > 0 {
		columns := 0
		for _, row := range p.rows {
			columns = max(columns, len(row))
		}
		if columns > maxRTFTableCols {
			p.err = fmt.Errorf("RTF table exceeds the %d-column limit", maxRTFTableCols)
		} else if columns > 0 && len(p.rows) <= maxRTFTableRows {
			table := document.Table{
				Rows:        make([][]string, len(p.rows)),
				CellFormats: make([][]document.TableCellFormat, len(p.rows)),
			}
			for ri, row := range p.rows {
				table.Rows[ri] = make([]string, columns)
				table.CellFormats[ri] = make([]document.TableCellFormat, columns)
				copy(table.Rows[ri], row)
				copy(table.CellFormats[ri], p.formats[ri])
			}
			p.paragraphs = append(p.paragraphs, document.Paragraph{
				Table:      &table,
				PageBreak:  p.pendingBreak,
				TablePart:  false,
				SpaceAfter: 0,
			})
			p.pendingBreak = false
		}
	}
	p.tableActive = false
	p.inCell = false
	p.cellClosed = false
	p.cellFormatSet = false
	p.rows, p.formats = nil, nil
	p.row, p.rowFormats = nil, nil
	p.cellText.Reset()
	p.resetParagraph()
}

func (p *rtfParser) color(index int) string {
	if index >= 0 && index < len(p.colors) {
		return p.colors[index]
	}
	return ""
}

func (p *rtfParser) parseColorTable(group []byte) {
	p.colors = []string{""}
	for _, entry := range strings.Split(string(group), ";")[1:] {
		match := rtfColorPattern.FindStringSubmatch(entry)
		if len(match) != 4 {
			p.colors = append(p.colors, "")
			continue
		}
		red, _ := strconv.Atoi(match[1])
		green, _ := strconv.Atoi(match[2])
		blue, _ := strconv.Atoi(match[3])
		p.colors = append(p.colors, fmt.Sprintf("#%02x%02x%02x",
			min(max(red, 0), 255), min(max(green, 0), 255), min(max(blue, 0), 255)))
	}
}

func (p *rtfParser) skipFallback(count int) {
	for count > 0 && p.pos < len(p.data) {
		if p.data[p.pos] == '\\' {
			if p.pos+1 < len(p.data) && p.data[p.pos+1] == '\'' && p.pos+3 < len(p.data) {
				p.pos += 4
			} else if p.pos+1 < len(p.data) && strings.ContainsRune(`\{}`, rune(p.data[p.pos+1])) {
				p.pos += 2
			} else {
				break
			}
		} else if p.data[p.pos] == '{' || p.data[p.pos] == '}' {
			break
		} else {
			p.pos++
		}
		count--
	}
}

func rtfDestinationEnd(data []byte, start int) (int, string) {
	i := start + 1
	if i < len(data) && data[i] == '\\' && i+1 < len(data) && data[i+1] == '*' {
		i += 2
		for i < len(data) && data[i] == ' ' {
			i++
		}
	}
	if i >= len(data) || data[i] != '\\' {
		return 0, ""
	}
	i++
	wordStart := i
	for i < len(data) && isRTFLetter(data[i]) {
		i++
	}
	if wordStart == i {
		return 0, ""
	}
	word := strings.ToLower(string(data[wordStart:i]))
	known := map[string]bool{
		"colortbl": true, "fonttbl": true, "stylesheet": true, "info": true,
		"generator": true, "header": true, "headerl": true, "headerr": true,
		"footer": true, "footerl": true, "footerr": true, "pict": true,
		"object": true, "filetbl": true, "listtable": true, "listoverridetable": true,
		"rsidtbl": true, "xmlnstbl": true, "datastore": true,
	}
	if !known[word] {
		if start+2 < len(data) && data[start+1] == '\\' && data[start+2] == '*' {
			return findRTFGroupEnd(data, start), word
		}
		return 0, ""
	}
	return findRTFGroupEnd(data, start), word
}

func findRTFGroupEnd(data []byte, start int) int {
	depth := 0
	for i := start; i < len(data); i++ {
		switch data[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i + 1
			}
		}
	}
	return 0
}

func decodeRTFByte(value byte) rune {
	if value < 0x80 || value >= 0xa0 {
		return rune(value)
	}
	return [...]rune{
		'\u20ac', '\u0081', '\u201a', '\u0192', '\u201e', '\u2026', '\u2020', '\u2021',
		'\u02c6', '\u2030', '\u0160', '\u2039', '\u0152', '\u008d', '\u017d', '\u008f',
		'\u0090', '\u2018', '\u2019', '\u201c', '\u201d', '\u2022', '\u2013', '\u2014',
		'\u02dc', '\u2122', '\u0161', '\u203a', '\u0153', '\u009d', '\u017e', '\u0178',
	}[value-0x80]
}

func isRTFLetter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func setStyle(style, flag document.Style, enabled bool) document.Style {
	if enabled {
		return style | flag
	}
	return style &^ flag
}
