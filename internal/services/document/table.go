// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

import (
	"fmt"
	"strings"
)

const (
	maxTableRows    = 20
	maxTableColumns = 10
	minTableCellW   = 3
)

// InsertTable inserts a fixed-width text table at the cursor. Its cells remain
// ordinary editable document text.
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
	cellSpace := bodyWidth - columns - 1
	if cellSpace < columns*minTableCellW {
		return fmt.Errorf("table with %d columns does not fit the page text width", columns)
	}

	cellWidths := make([]int, columns)
	for i := range cellWidths {
		cellWidths[i] = cellSpace / columns
		if i < cellSpace%columns {
			cellWidths[i]++
		}
	}

	var table strings.Builder
	writeBorder := func(left, join, right rune) {
		table.WriteRune(left)
		for i, width := range cellWidths {
			table.WriteString(strings.Repeat("─", width))
			switch {
			case i == len(cellWidths)-1:
				table.WriteRune(right)
			default:
				table.WriteRune(join)
			}
		}
	}
	writeRow := func() {
		table.WriteRune('│')
		for _, width := range cellWidths {
			table.WriteString(strings.Repeat(" ", width))
			table.WriteRune('│')
		}
	}

	writeBorder('┌', '┬', '┐')
	for row := 0; row < rows; row++ {
		table.WriteRune('\n')
		writeRow()
		if row+1 < rows {
			table.WriteRune('\n')
			writeBorder('├', '┼', '┤')
		}
	}
	table.WriteRune('\n')
	writeBorder('└', '┴', '┘')

	d.pushHistory()
	d.deleteSelectionLocked()
	plain := []rune(plainOf(d.Paragraphs[d.CursorPara]))
	before := d.CursorOff > 0
	after := d.CursorOff < len(plain)
	startPara := d.CursorPara
	inserted := table.String()
	if before {
		inserted = "\n" + inserted
	}
	if after {
		inserted += "\n"
	}
	d.insertTextContentLocked(inserted)

	tableStart := startPara
	if before {
		tableStart++
	}
	tableEnd := tableStart + 2*rows
	for pi := tableStart; pi <= tableEnd; pi++ {
		p := &d.Paragraphs[pi]
		pageBreak := pi == tableStart && p.PageBreak
		p.Align = AlignLeft
		p.ParaStyle = ParagraphNormal
		p.SpaceAfter = 0
		p.PageBreak = pageBreak
	}

	d.CursorPara = startPara + 1
	if before {
		d.CursorPara++
	}
	d.CursorOff = 2
	d.goalSet = false
	d.Dirty = true
	return nil
}
