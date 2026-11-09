// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package fileop

import (
	"io"
	"strings"

	"gowrite/internal/services/document"
)

func ExportTXT(filename string, state document.State) error {
	var text strings.Builder
	for i, paragraph := range state.Paragraphs {
		if paragraph.TablePart {
			continue
		}
		if i > 0 {
			if paragraph.PageBreak {
				text.WriteByte('\f')
			} else {
				text.WriteByte('\n')
				for range min(max(state.Paragraphs[i-1].SpaceAfter, 0), 10) {
					text.WriteByte('\n')
				}
			}
		}
		if paragraph.Table != nil {
			for ri, row := range paragraph.Table.Rows {
				if ri > 0 {
					text.WriteByte('\n')
				}
				for ci, cell := range row {
					if ci > 0 {
						text.WriteByte('\t')
					}
					text.WriteString(cell)
				}
			}
			continue
		}
		for _, run := range paragraph.Runs {
			text.WriteString(run.Text)
		}
	}
	return writeAtomic(filename, func(w io.Writer) error {
		_, err := io.WriteString(w, text.String())
		return err
	})
}
