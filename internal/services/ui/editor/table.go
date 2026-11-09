// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

import (
	"fmt"
	"strings"

	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) showTableEditor(paragraph int, returnToFormat bool) {
	table, ok := e.Doc.TableAt(paragraph)
	if !ok {
		ShowErrorModal(e.App, e.Root, "There is no table at the current position.")
		return
	}
	table.EnsureCellFormats()

	grid := tview.NewTable().SetBorders(true).SetSelectable(true, true)
	actions := tview.NewList().ShowSecondaryText(false)
	uitheme.StyleList(actions, "")
	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(grid, 0, 1, true).
		AddItem(actions, 8, 0, false)
	root.SetBorder(true).SetTitle(" Table  ·  Enter edits a cell  ·  Tab switches panels  ·  Esc cancels ")
	root.SetBorderColor(uitheme.Line)
	root.SetTitleColor(uitheme.Amber)
	root.SetBackgroundColor(uitheme.Ink)
	showTableError := func(message string) {
		modal := tview.NewModal().
			SetText(message).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(int, string) {
				e.App.SetRoot(root, true).SetFocus(grid)
			})
		uitheme.StyleModalWarn(modal, "Table")
		e.App.SetRoot(modal, true).SetFocus(modal)
	}

	restoreEditor := func() {
		if returnToFormat {
			e.showFormatDialog()
		} else {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
		}
	}
	refreshGrid := func(row, col int) {
		grid.Clear()
		for ri, cells := range table.Rows {
			for ci, text := range cells {
				format := table.CellFormats[ri][ci]
				fg := uitheme.Text
				if format.Foreground != "" {
					fg = parseHexColor(format.Foreground)
				}
				bg := uitheme.Panel
				if format.Background != "" {
					bg = parseHexColor(format.Background)
				}
				attributes := tcell.AttrMask(0)
				if format.Style&document.StyleBold != 0 ||
					format.ParagraphStyle == document.ParagraphTitle ||
					format.ParagraphStyle == document.ParagraphHeading1 ||
					format.ParagraphStyle == document.ParagraphHeading2 ||
					format.ParagraphStyle == document.ParagraphHeading3 {
					attributes |= tcell.AttrBold
				}
				if format.Style&document.StyleItalic != 0 || format.ParagraphStyle == document.ParagraphQuote {
					attributes |= tcell.AttrItalic
				}
				if format.Style&document.StyleUnderline != 0 {
					attributes |= tcell.AttrUnderline
				}
				if format.Style&document.StyleStrike != 0 {
					attributes |= tcell.AttrStrikeThrough
				}
				cell := tview.NewTableCell(text).
					SetTextColor(fg).
					SetBackgroundColor(bg).
					SetAttributes(attributes).
					SetAlign(int(format.Alignment)).
					SetExpansion(1)
				if ri == row && ci == col {
					cell.SetBackgroundColor(uitheme.Btn)
				}
				grid.SetCell(ri, ci, cell)
			}
		}
		if row >= len(table.Rows) {
			row = len(table.Rows) - 1
		}
		if col >= len(table.Rows[0]) {
			col = len(table.Rows[0]) - 1
		}
		grid.Select(row, col)
	}
	currentCell := func() (int, int) {
		row, col := grid.GetSelection()
		return row, col
	}
	openCellEditor := func() {
		row, col := currentCell()
		format := table.CellFormats[row][col]
		contents := tview.NewTextArea().
			SetLabel(fmt.Sprintf("Cell %d,%d: ", row+1, col+1)).
			SetText(table.Rows[row][col], false).
			SetSize(5, 68)
		contents.SetFormAttributes(12, uitheme.Label, uitheme.Field, uitheme.Text, uitheme.Field)
		bold := tview.NewCheckbox().SetLabel("Bold").SetChecked(format.Style&document.StyleBold != 0)
		italic := tview.NewCheckbox().SetLabel("Italic").SetChecked(format.Style&document.StyleItalic != 0)
		underline := tview.NewCheckbox().SetLabel("Underline").SetChecked(format.Style&document.StyleUnderline != 0)
		strike := tview.NewCheckbox().SetLabel("Strikethrough").SetChecked(format.Style&document.StyleStrike != 0)
		paragraphStyles := []string{"Normal", "Title", "Heading 1", "Heading 2", "Heading 3", "Quote", "Code"}
		paragraphStyleValues := []document.ParagraphStyle{
			document.ParagraphNormal, document.ParagraphTitle, document.ParagraphHeading1,
			document.ParagraphHeading2, document.ParagraphHeading3, document.ParagraphQuote,
			document.ParagraphCode,
		}
		paragraphStyle := tview.NewDropDown().SetLabel("Paragraph style:").SetOptions(paragraphStyles, nil)
		paragraphStyle.SetCurrentOption(indexOfParagraphStyle(paragraphStyleValues, format.ParagraphStyle))
		alignments := []string{"Left", "Center", "Right"}
		alignment := tview.NewDropDown().SetLabel("Alignment:").SetOptions(alignments, nil)
		alignment.SetCurrentOption(int(format.Alignment))
		foreground := format.Foreground
		background := format.Background

		form := tview.NewForm()
		uitheme.StyleForm(form, "Edit Table Cell")
		form.AddFormItem(contents).
			AddFormItem(bold).
			AddFormItem(italic).
			AddFormItem(underline).
			AddFormItem(strike).
			AddFormItem(paragraphStyle).
			AddFormItem(alignment)
		contents.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyTab {
				form.SetFocus(1)
				e.App.SetFocus(form)
				return nil
			}
			return event
		})
		modal := centredModal(form, 88, 24)
		insertAtCursor := func(value string) {
			_, start, end := contents.GetSelection()
			contents.Replace(start, end, value)
		}
		form.AddButton("Text color…", func() {
			e.showColorPickerWith(false, foreground, func(color string) {
				foreground = color
				e.App.SetRoot(modal, true).SetFocus(contents)
			}, func() { e.App.SetRoot(modal, true).SetFocus(contents) })
		})
		form.AddButton("Background…", func() {
			e.showColorPickerWith(true, background, func(color string) {
				background = color
				e.App.SetRoot(modal, true).SetFocus(contents)
			}, func() { e.App.SetRoot(modal, true).SetFocus(contents) })
		})
		form.AddButton("Date & time…", func() {
			e.showDateTimeDialogWith(
				func(value string) {
					insertAtCursor(value)
					e.App.SetRoot(modal, true).SetFocus(contents)
				},
				func() { e.App.SetRoot(modal, true).SetFocus(contents) },
			)
		})
		form.AddButton("Symbols…", func() {
			e.showSymbolGridWith(
				func(value string) {
					insertAtCursor(value)
					e.App.SetRoot(modal, true).SetFocus(contents)
				},
				func() { e.App.SetRoot(modal, true).SetFocus(contents) },
			)
		})
		form.AddButton("Apply", func() {
			cellFormat := document.TableCellFormat{
				Foreground: strings.ToLower(foreground),
				Background: strings.ToLower(background),
				Alignment:  document.AlignLeft,
			}
			if bold.IsChecked() {
				cellFormat.Style |= document.StyleBold
			}
			if italic.IsChecked() {
				cellFormat.Style |= document.StyleItalic
			}
			if underline.IsChecked() {
				cellFormat.Style |= document.StyleUnderline
			}
			if strike.IsChecked() {
				cellFormat.Style |= document.StyleStrike
			}
			styleIndex, _ := paragraphStyle.GetCurrentOption()
			if styleIndex >= 0 && styleIndex < len(paragraphStyleValues) {
				cellFormat.ParagraphStyle = paragraphStyleValues[styleIndex]
			}
			alignIndex, _ := alignment.GetCurrentOption()
			if alignIndex >= 0 && alignIndex < len(alignments) {
				cellFormat.Alignment = document.Align(alignIndex)
			}
			table.Rows[row][col] = contents.GetText()
			table.CellFormats[row][col] = cellFormat
			refreshGrid(row, col)
			e.App.SetRoot(root, true).SetFocus(grid)
		})
		form.AddButton("Cancel", func() { e.App.SetRoot(root, true).SetFocus(grid) })
		form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEscape {
				e.App.SetRoot(root, true).SetFocus(grid)
				return nil
			}
			return event
		})
		e.App.SetRoot(modal, true).SetFocus(contents)
	}

	grid.SetSelectedFunc(func(int, int) { openCellEditor() })
	refreshGrid(0, 0)
	addAction := func(label string, action func()) {
		actions.AddItem(label, "", 0, action)
	}
	addAction("Edit selected cell", openCellEditor)
	addAction("Insert row below", func() {
		row, col := currentCell()
		if len(table.Rows) >= e.Doc.MaxTableRows() {
			showTableError(fmt.Sprintf("A table can have at most %d rows.", e.Doc.MaxTableRows()))
			return
		}
		newRow := make([]string, len(table.Rows[0]))
		table.Rows = append(table.Rows, nil)
		copy(table.Rows[row+2:], table.Rows[row+1:])
		table.Rows[row+1] = newRow
		table.CellFormats = append(table.CellFormats, nil)
		copy(table.CellFormats[row+2:], table.CellFormats[row+1:])
		table.CellFormats[row+1] = make([]document.TableCellFormat, len(newRow))
		refreshGrid(row+1, col)
	})
	addAction("Insert column right", func() {
		row, col := currentCell()
		if len(table.Rows[0]) >= e.Doc.MaxTableColumns() {
			showTableError(fmt.Sprintf("The page can fit at most %d columns.", e.Doc.MaxTableColumns()))
			return
		}
		for ri := range table.Rows {
			table.Rows[ri] = append(table.Rows[ri], "")
			copy(table.Rows[ri][col+2:], table.Rows[ri][col+1:])
			table.Rows[ri][col+1] = ""
			table.CellFormats[ri] = append(table.CellFormats[ri], document.TableCellFormat{})
			copy(table.CellFormats[ri][col+2:], table.CellFormats[ri][col+1:])
			table.CellFormats[ri][col+1] = document.TableCellFormat{}
		}
		refreshGrid(row, col+1)
	})
	addAction("Delete selected row", func() {
		if len(table.Rows) == 1 {
			showTableError("A table must contain at least one row.")
			return
		}
		row, col := currentCell()
		table.Rows = append(table.Rows[:row], table.Rows[row+1:]...)
		table.CellFormats = append(table.CellFormats[:row], table.CellFormats[row+1:]...)
		refreshGrid(min(row, len(table.Rows)-1), col)
	})
	addAction("Delete selected column", func() {
		if len(table.Rows[0]) == 1 {
			showTableError("A table must contain at least one column.")
			return
		}
		row, col := currentCell()
		for ri := range table.Rows {
			table.Rows[ri] = append(table.Rows[ri][:col], table.Rows[ri][col+1:]...)
			table.CellFormats[ri] = append(table.CellFormats[ri][:col], table.CellFormats[ri][col+1:]...)
		}
		refreshGrid(row, min(col, len(table.Rows[0])-1))
	})
	addAction("Save table", func() {
		if err := e.Doc.UpdateTableAt(paragraph, table); err != nil {
			showTableError(err.Error())
			return
		}
		restoreEditor()
	})
	addAction("Cancel", restoreEditor)
	grid.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			restoreEditor()
			return nil
		case tcell.KeyEnter:
			openCellEditor()
			return nil
		case tcell.KeyTab:
			e.App.SetFocus(actions)
			return nil
		}
		return event
	})
	actions.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			restoreEditor()
			return nil
		case tcell.KeyTab, tcell.KeyBacktab:
			e.App.SetFocus(grid)
			return nil
		}
		return event
	})
	e.App.SetRoot(root, true).SetFocus(grid)
}

func (e *Editor) editTableAtCursor() bool {
	paragraph, ok := e.Doc.TableParagraph()
	if !ok {
		return false
	}
	e.showTableEditor(paragraph, false)
	return true
}

func indexOfParagraphStyle(styles []document.ParagraphStyle, style document.ParagraphStyle) int {
	for i, candidate := range styles {
		if candidate == style {
			return i
		}
	}
	return 0
}
