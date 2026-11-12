// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

import (
	"gowrite/internal/services/document"
	"unicode"

	"github.com/gdamore/tcell/v2"
)

func (e *Editor) handleKeys(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyEscape {
		if e.Doc.HasSelection() {
			e.Doc.ClearSelection()
			e.redraw()
		}
		e.showFileDialog(fileDialogSave)
		return nil
	}

	if _, inTable := e.Doc.TableParagraph(); inTable {
		switch event.Key() {
		case tcell.KeyEnter:
			if e.Doc.InsertParagraphBeforeTableAtCursor() {
				e.redraw()
				return nil
			}
			if e.Doc.MoveAfterTableAtCursor() {
				e.redraw()
				return nil
			}
			e.editTableAtCursor()
			return nil
		case tcell.KeyTab, tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyDelete:
			e.editTableAtCursor()
			return nil
		default:
			if event.Rune() != 0 && unicode.IsPrint(event.Rune()) &&
				event.Modifiers()&(tcell.ModAlt|tcell.ModCtrl) == 0 {
				e.editTableAtCursor()
				return nil
			}
		}
	}

	if event.Rune() == '?' || (event.Rune() == '/' && event.Modifiers()&tcell.ModAlt != 0) {
		e.showHelp()
		return nil
	}

	mod := event.Modifiers()

	if mod&tcell.ModAlt != 0 {
		// Terminals often omit ModShift for Alt+Shift+letter and only send the
		// uppercase rune. Treat an uppercase letter under Alt as shifted.
		r := event.Rune()
		hasShift := mod&tcell.ModShift != 0 || (r >= 'A' && r <= 'Z')

		if event.Key() == tcell.KeyEnter {
			e.showFormatDialog()
			return nil
		}

		switch r {
		case 'a', 'A':
			if hasShift {
				e.Doc.SelectAll()
				e.redraw()
				return nil
			}
			if e.Doc.ApplyActiveToSelection() {
				e.redraw()
				return nil
			}
			s := e.Doc.ActiveStyleFlags()
			if s != document.StyleNone {
				e.Doc.ApplyStyleToParagraph(s, true)
				e.redraw()
			}
			return nil
		case 'b', 'B':
			e.toggleStyle(document.StyleBold)
			return nil
		case 'c', 'C':
			if hasShift {
				e.copySelection()
				return nil
			}
			e.showColorPicker(false)
			return nil
		case 'd', 'D':
			e.showHeaderFooter()
			return nil
		case 'e', 'E':
			if hasShift {
				e.exportDocument()
				return nil
			}
			e.Doc.SetParagraphAlign(document.AlignCenter)
			e.redraw()
			return nil
		case 'f', 'F':
			e.showFindDialog(false)
			return nil
		case 'g', 'G':
			e.showGoToDialog()
			return nil
		case 'h', 'H':
			e.showFindDialog(true)
			return nil
		case 'i', 'I':
			e.toggleStyle(document.StyleItalic)
			return nil
		case 'k', 'K':
			e.toggleStyle(document.StyleStrike)
			return nil
		case 'l', 'L':
			e.Doc.SetParagraphAlign(document.AlignLeft)
			e.redraw()
			return nil
		case 'n', 'N':
			e.newDocument()
			return nil
		case 'o', 'O':
			e.openDocument()
			return nil
		case 'p', 'P':
			if hasShift {
				e.Doc.InsertPageBreak()
				e.redraw()
				return nil
			}
			e.showPageSetup()
			return nil
		case 'r', 'R':
			e.Doc.SetParagraphAlign(document.AlignRight)
			e.redraw()
			return nil
		case 's', 'S':
			if hasShift {
				e.saveAsDocument()
			} else {
				e.saveDocument()
			}
			return nil
		case 'u', 'U':
			e.toggleStyle(document.StyleUnderline)
			return nil
		case 'v', 'V':
			if hasShift {
				e.pasteClipboard()
				return nil
			}
		case 'x', 'X':
			if hasShift {
				e.cutSelection()
				return nil
			}
		case 'y', 'Y':
			e.showColorPicker(true)
			return nil
		case 'z', 'Z':
			if hasShift {
				e.redo()
			} else {
				e.undo()
			}
			return nil
		}
	}

	shift := mod&tcell.ModShift != 0
	altShift := mod&tcell.ModAlt != 0 && shift
	switch event.Key() {
	case tcell.KeyLeft:
		if altShift {
			e.prepareMove(true)
			e.Doc.MoveWordLeft()
			e.redraw()
			return nil
		}
		if !shift && e.Doc.HasSelection() {
			e.Doc.CollapseSelection(true)
		} else {
			e.prepareMove(shift)
			e.Doc.MoveLeft()
		}
		e.redraw()
		return nil
	case tcell.KeyRight:
		if altShift {
			e.prepareMove(true)
			e.Doc.MoveWordRight()
			e.redraw()
			return nil
		}
		if !shift && e.Doc.HasSelection() {
			e.Doc.CollapseSelection(false)
		} else {
			e.prepareMove(shift)
			e.Doc.MoveRight()
		}
		e.redraw()
		return nil
	case tcell.KeyUp:
		e.prepareMove(shift)
		e.Doc.MoveUp()
		e.redraw()
		return nil
	case tcell.KeyDown:
		e.prepareMove(shift)
		e.Doc.MoveDown()
		e.redraw()
		return nil
	case tcell.KeyHome:
		e.prepareMove(shift)
		if mod&(tcell.ModCtrl|tcell.ModAlt) != 0 {
			e.Doc.MoveDocStart()
		} else {
			e.Doc.MoveLineStart()
		}
		e.redraw()
		return nil
	case tcell.KeyEnd:
		e.prepareMove(shift)
		if mod&(tcell.ModCtrl|tcell.ModAlt) != 0 {
			e.Doc.MoveDocEnd()
		} else {
			e.Doc.MoveLineEnd()
		}
		e.redraw()
		return nil
	case tcell.KeyPgUp:
		e.prepareMove(shift)
		e.Doc.MovePage(-e.pageRows())
		e.redraw()
		return nil
	case tcell.KeyPgDn:
		e.prepareMove(shift)
		e.Doc.MovePage(e.pageRows())
		e.redraw()
		return nil
	case tcell.KeyInsert:
		if shift {
			e.pasteClipboard()
		}
		return nil
	case tcell.KeyEnter:
		e.Doc.InsertRune('\n')
		e.redraw()
		return nil
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		e.Doc.Backspace()
		e.redraw()
		return nil
	case tcell.KeyDelete:
		if shift {
			e.cutSelection()
			return nil
		}
		e.Doc.Delete()
		e.redraw()
		return nil
	case tcell.KeyTab:
		e.Doc.InsertRune('\t')
		e.redraw()
		return nil
	}

	r := event.Rune()
	if r != 0 && unicode.IsPrint(r) {
		if mod&tcell.ModAlt != 0 || mod&tcell.ModCtrl != 0 {
			return nil
		}
		e.Doc.InsertRune(r)
		e.redraw()
		return nil
	}

	return event
}

func (e *Editor) toggleStyle(s document.Style) {
	if !e.Doc.ToggleStyleOnSelection(s) {
		e.Doc.ToggleStyle(s)
	}
	e.redraw()
}

func (e *Editor) prepareMove(extend bool) {
	if extend {
		e.Doc.BeginSelection()
	} else {
		e.Doc.ClearSelection()
	}
}
