// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mattn/go-runewidth"
)

const (
	maxTableRows    = 10000
	maxTableColumns = 256
	minTableCellW   = 3
)

func cloneTable(table *Table) *Table {
	if table == nil {
		return nil
	}
	copy := &Table{Rows: make([][]string, len(table.Rows))}
	for i := range table.Rows {
		copy.Rows[i] = append([]string(nil), table.Rows[i]...)
	}
	if table.CellFormats != nil {
		copy.CellFormats = make([][]TableCellFormat, len(table.CellFormats))
		for i := range table.CellFormats {
			copy.CellFormats[i] = append([]TableCellFormat(nil), table.CellFormats[i]...)
		}
	}
	return copy
}

func sameTable(a, b *Table) bool {
	if a == nil || b == nil {
		return a == b
	}
	if len(a.Rows) != len(b.Rows) {
		return false
	}
	for i := range a.Rows {
		if len(a.Rows[i]) != len(b.Rows[i]) {
			return false
		}
		for j := range a.Rows[i] {
			if a.Rows[i][j] != b.Rows[i][j] {
				return false
			}
		}
	}
	if len(a.CellFormats) != len(b.CellFormats) {
		return false
	}
	for i := range a.CellFormats {
		if len(a.CellFormats[i]) != len(b.CellFormats[i]) {
			return false
		}
		for j := range a.CellFormats[i] {
			if a.CellFormats[i][j] != b.CellFormats[i][j] {
				return false
			}
		}
	}
	return true
}

func validateTable(table *Table) error {
	if table == nil || len(table.Rows) == 0 || len(table.Rows) > maxTableRows {
		return fmt.Errorf("row count must be between 1 and %d", maxTableRows)
	}
	columns := len(table.Rows[0])
	if columns == 0 || columns > maxTableColumns {
		return fmt.Errorf("column count must be between 1 and %d", maxTableColumns)
	}
	for i, row := range table.Rows {
		if len(row) != columns {
			return fmt.Errorf("row %d has %d cells; expected %d", i+1, len(row), columns)
		}
	}
	if len(table.CellFormats) != 0 && len(table.CellFormats) != len(table.Rows) {
		return fmt.Errorf("cell-format row count does not match table rows")
	}
	for ri, row := range table.CellFormats {
		if len(row) != 0 && len(row) != columns {
			return fmt.Errorf("cell-format row %d has %d cells; expected %d", ri+1, len(row), columns)
		}
		for ci, format := range row {
			if format.Style & ^(StyleBold|StyleItalic|StyleUnderline|StyleStrike) != 0 {
				return fmt.Errorf("cell %d,%d has invalid text style", ri+1, ci+1)
			}
			if format.ParagraphStyle > ParagraphCode {
				return fmt.Errorf("cell %d,%d has invalid paragraph style", ri+1, ci+1)
			}
			if format.Alignment < AlignLeft || format.Alignment > AlignRight {
				return fmt.Errorf("cell %d,%d has invalid alignment", ri+1, ci+1)
			}
			if !validTableColor(format.Foreground) || !validTableColor(format.Background) {
				return fmt.Errorf("cell %d,%d has invalid color", ri+1, ci+1)
			}
		}
	}
	return nil
}

func validTableColor(value string) bool {
	if value == "" {
		return true
	}
	if len(value) != 7 || value[0] != '#' {
		return false
	}
	_, err := strconv.ParseUint(value[1:], 16, 24)
	return err == nil
}

// InsertTable inserts a structured table at the cursor. Cell contents are
// stored separately from the text used to draw the table in the editor.
func (d *Document) InsertTable(rows, columns int) error {
	if rows < 1 || rows > maxTableRows {
		return fmt.Errorf("rows must be between 1 and %d", maxTableRows)
	}
	if columns < 1 || columns > maxTableColumns {
		return fmt.Errorf("columns must be between 1 and %d", maxTableColumns)
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	bodyWidth := d.geometry().bodyW
	if bodyWidth < columns*minTableCellW+columns+1 {
		return fmt.Errorf("table with %d columns does not fit the page text width", columns)
	}

	d.pushHistory()
	d.deleteSelectionLocked()
	left, right := d.splitParaAtCursor()
	before, after := plainOf(left) != "", plainOf(right) != ""
	original := d.Paragraphs[d.CursorPara]
	table := &Table{Rows: make([][]string, rows)}
	for i := range table.Rows {
		table.Rows[i] = make([]string, columns)
	}
	block, err := tableParagraphs(table, bodyWidth, !before && original.PageBreak)
	if err != nil {
		d.popHistory()
		return err
	}

	replacement := make([]Paragraph, 0, len(block)+2)
	if before {
		replacement = append(replacement, left)
	}
	tableStart := d.CursorPara + len(replacement)
	replacement = append(replacement, block...)
	if after {
		replacement = append(replacement, right)
	}
	if !before && !after {
		replacement = block
		tableStart = d.CursorPara
	}
	d.Paragraphs = append(d.Paragraphs[:d.CursorPara],
		append(replacement, d.Paragraphs[d.CursorPara+1:]...)...)
	d.CursorPara = min(tableStart+1, len(d.Paragraphs)-1)
	d.CursorOff = 2
	d.goalSet = false
	d.Dirty = true
	return nil
}

// MaxTableColumns returns the number of columns that fit the current text area.
func (d *Document) MaxTableColumns() int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return max(0, (d.geometry().bodyW-1)/(minTableCellW+1))
}

// MaxTableRows returns the document model's row limit for a table.
func (d *Document) MaxTableRows() int {
	return maxTableRows
}

// TableParagraph returns the paragraph index of the table under the cursor.
func (d *Document) TableParagraph() (int, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return tableParagraphIndex(d.Paragraphs, d.CursorPara)
}

// TableAt returns an independent copy of the table at paragraph.
func (d *Document) TableAt(paragraph int) (Table, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	start, ok := tableParagraphIndex(d.Paragraphs, paragraph)
	if !ok {
		return Table{}, false
	}
	return *cloneTable(d.Paragraphs[start].Table), true
}

func (t *Table) EnsureCellFormats() {
	if len(t.CellFormats) == len(t.Rows) {
		for i := range t.Rows {
			if len(t.CellFormats[i]) == len(t.Rows[i]) {
				continue
			}
			formats := make([]TableCellFormat, len(t.Rows[i]))
			copy(formats, t.CellFormats[i])
			t.CellFormats[i] = formats
		}
		return
	}
	formats := make([][]TableCellFormat, len(t.Rows))
	for i := range t.Rows {
		formats[i] = make([]TableCellFormat, len(t.Rows[i]))
		if i < len(t.CellFormats) {
			copy(formats[i], t.CellFormats[i])
		}
	}
	t.CellFormats = formats
}

func tableCellFormat(table Table, row, column int) TableCellFormat {
	if row < 0 || row >= len(table.CellFormats) ||
		column < 0 || column >= len(table.CellFormats[row]) {
		return TableCellFormat{}
	}
	return table.CellFormats[row][column]
}

// MoveAfterTableAtCursor moves to the paragraph after a table, creating one at
// the end of the document when needed. It only acts at the table's final edge.
func (d *Document) MoveAfterTableAtCursor() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.ensureCursor()
	start, ok := tableParagraphIndex(d.Paragraphs, d.CursorPara)
	if !ok {
		return false
	}
	end := start + 1
	for end < len(d.Paragraphs) && d.Paragraphs[end].TablePart {
		end++
	}
	last := end - 1
	if d.CursorPara != last || d.CursorOff < len([]rune(plainOf(d.Paragraphs[last]))) {
		return false
	}
	if end < len(d.Paragraphs) &&
		d.Paragraphs[end].Table == nil && !d.Paragraphs[end].TablePart {
		d.CursorPara = end
		d.CursorOff = 0
		d.goalSet = false
		return true
	}
	d.pushHistory()
	paragraph := Paragraph{Runs: []Run{{Text: ""}}}
	d.Paragraphs = append(d.Paragraphs, Paragraph{})
	copy(d.Paragraphs[end+1:], d.Paragraphs[end:])
	d.Paragraphs[end] = paragraph
	d.CursorPara = end
	d.CursorOff = 0
	d.goalSet = false
	d.Dirty = true
	return true
}

// UpdateTableAt replaces a table's cell data and refreshes its editor rendering.
func (d *Document) UpdateTableAt(paragraph int, table Table) error {
	if err := validateTable(&table); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	start, ok := tableParagraphIndex(d.Paragraphs, paragraph)
	if !ok {
		return fmt.Errorf("no table at paragraph %d", paragraph+1)
	}
	block, err := tableParagraphs(&table, d.geometry().bodyW, d.Paragraphs[start].PageBreak)
	if err != nil {
		return err
	}

	end := start + 1
	for end < len(d.Paragraphs) && d.Paragraphs[end].TablePart {
		end++
	}
	d.pushHistory()
	d.Paragraphs = append(d.Paragraphs[:start],
		append(block, d.Paragraphs[end:]...)...)
	d.CursorPara = min(start+1, len(d.Paragraphs)-1)
	d.CursorOff = 2
	d.goalSet = false
	d.Dirty = true
	return nil
}

func (d *Document) reflowTablesLocked() {
	paragraphs := make([]Paragraph, 0, len(d.Paragraphs))
	cursorParagraph := 0
	for i := 0; i < len(d.Paragraphs); {
		p := d.Paragraphs[i]
		if p.Table == nil {
			if i == d.CursorPara {
				cursorParagraph = len(paragraphs)
			}
			paragraphs = append(paragraphs, p)
			i++
			continue
		}

		end := i + 1
		for end < len(d.Paragraphs) && d.Paragraphs[end].TablePart {
			end++
		}
		block, err := tableParagraphs(p.Table, d.geometry().bodyW, p.PageBreak)
		if err != nil {
			if d.CursorPara >= i && d.CursorPara < end {
				cursorParagraph = len(paragraphs) + 1
			}
			paragraphs = append(paragraphs, d.Paragraphs[i:end]...)
			i = end
			continue
		}
		if d.CursorPara >= i && d.CursorPara < end {
			cursorParagraph = len(paragraphs) + min(1, len(block)-1)
		}
		paragraphs = append(paragraphs, block...)
		i = end
	}
	d.Paragraphs = paragraphs
	d.CursorPara = min(cursorParagraph, len(paragraphs)-1)
	d.ensureCursor()
}

func tableParagraphIndex(paragraphs []Paragraph, index int) (int, bool) {
	if index < 0 || index >= len(paragraphs) {
		return 0, false
	}
	for index >= 0 && paragraphs[index].TablePart {
		index--
	}
	if index >= 0 && paragraphs[index].Table != nil {
		return index, true
	}
	return 0, false
}

func (d *Document) inTableLocked() bool {
	_, ok := tableParagraphIndex(d.Paragraphs, d.CursorPara)
	return ok
}

func tableParagraphs(table *Table, width int, pageBreak bool) ([]Paragraph, error) {
	if err := validateTable(table); err != nil {
		return nil, err
	}
	lines, err := renderTable(*table, width)
	if err != nil {
		return nil, err
	}
	paragraphs := make([]Paragraph, len(lines))
	for i, line := range lines {
		paragraphs[i] = Paragraph{
			Runs:      cloneRuns(line),
			PageBreak: i == 0 && pageBreak,
			TablePart: i != 0,
		}
		if i == 0 {
			paragraphs[i].Table = cloneTable(table)
		}
	}
	return paragraphs, nil
}

func renderTable(table Table, width int) ([][]Run, error) {
	columns := len(table.Rows[0])
	space := width - columns - 1
	if space < columns*minTableCellW {
		return nil, fmt.Errorf("table with %d columns does not fit the page text width", columns)
	}

	widths := make([]int, columns)
	for i := range widths {
		widths[i] = space / columns
		if i < space%columns {
			widths[i]++
		}
	}
	contentWidths := make([]int, columns)
	for i, w := range widths {
		contentWidths[i] = max(w-2, 1)
	}

	var lines [][]Run
	writeBorder := func(left, join, right rune) {
		var b strings.Builder
		b.WriteRune(left)
		for i, w := range widths {
			b.WriteString(strings.Repeat("─", w))
			if i == len(widths)-1 {
				b.WriteRune(right)
			} else {
				b.WriteRune(join)
			}
		}
		lines = append(lines, []Run{{Text: b.String()}})
	}
	writeBorder('┌', '┬', '┐')
	for ri, row := range table.Rows {
		wrapped := make([][]string, columns)
		height := 1
		for ci, cell := range row {
			wrapped[ci] = wrapTableCell(cell, contentWidths[ci])
			height = max(height, len(wrapped[ci]))
		}
		for line := 0; line < height; line++ {
			runs := []Run{{Text: "│"}}
			for ci := range widths {
				text := ""
				if line < len(wrapped[ci]) {
					text = wrapped[ci][line]
				}
				format := tableCellFormat(table, ri, ci)
				pad := max(contentWidths[ci]-runewidth.StringWidth(text), 0)
				leftPad, rightPad := 0, pad
				switch format.Alignment {
				case AlignCenter:
					leftPad, rightPad = pad/2, pad-pad/2
				case AlignRight:
					leftPad, rightPad = pad, 0
				}
				cellStyle, fg, bg := paragraphRunStyle(
					Paragraph{ParaStyle: format.ParagraphStyle},
					Run{Style: format.Style, Fg: format.Foreground, Bg: format.Background},
				)
				appendTableRun(&runs, Run{Text: " ", Bg: format.Background})
				appendTableRun(&runs, Run{Text: strings.Repeat(" ", leftPad), Bg: format.Background})
				appendTableRun(&runs, Run{Text: text, Style: cellStyle, Fg: fg, Bg: bg})
				appendTableRun(&runs, Run{Text: strings.Repeat(" ", rightPad+1), Bg: format.Background})
				appendTableRun(&runs, Run{Text: "│"})
			}
			lines = append(lines, runs)
		}
		if ri+1 < len(table.Rows) {
			writeBorder('├', '┼', '┤')
		}
	}
	writeBorder('└', '┴', '┘')
	return lines, nil
}

func appendTableRun(runs *[]Run, run Run) {
	if run.Text == "" {
		return
	}
	n := len(*runs)
	if n > 0 {
		last := &(*runs)[n-1]
		if last.Style == run.Style && last.Fg == run.Fg && last.Bg == run.Bg {
			last.Text += run.Text
			return
		}
	}
	*runs = append(*runs, run)
}

func wrapTableCell(text string, width int) []string {
	if text == "" {
		return []string{""}
	}
	var result []string
	for _, source := range strings.Split(text, "\n") {
		line := strings.Builder{}
		lineWidth := 0
		for _, r := range source {
			rw := runewidth.RuneWidth(r)
			if lineWidth+rw > width && lineWidth > 0 {
				result = append(result, line.String())
				line.Reset()
				lineWidth = 0
			}
			if rw > width {
				continue
			}
			line.WriteRune(r)
			lineWidth += rw
		}
		result = append(result, line.String())
	}
	return result
}
