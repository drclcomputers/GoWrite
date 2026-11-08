// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package ui

import (
	"fmt"
	"time"
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"gowrite/internal/services/uitheme"
)

// ── Date & Time ─────────────────────────────────────────────────────────────

func (e *Editor) showDateTimeDialog() {
	list := tview.NewList().
		ShowSecondaryText(true).
		SetHighlightFullLine(true)

	now := time.Now()
	add := func(main, secondary, text string) {
		list.AddItem(main, secondary, 0, func() {
			e.Doc.InsertText(text)
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
		})
	}

	add(now.Format("2006-01-02"), "ISO date", now.Format("2006-01-02"))
	add(now.Format("02 Jan 2006"), "Day Month Year", now.Format("02 Jan 2006"))
	add(now.Format("January 2, 2006"), "Long date", now.Format("January 2, 2006"))
	add(now.Format("Mon, 02 Jan 2006"), "RFC-style date", now.Format("Mon, 02 Jan 2006"))
	add(now.Format("15:04"), "Time 24h", now.Format("15:04"))
	add(now.Format("15:04:05"), "Time 24h + seconds", now.Format("15:04:05"))
	add(now.Format("3:04 PM"), "Time 12h", now.Format("3:04 PM"))
	add(now.Format("2006-01-02 15:04"), "Date + time", now.Format("2006-01-02 15:04"))
	add(now.Format(time.RFC3339), "RFC3339", now.Format(time.RFC3339))
	add(now.Format(time.RFC1123), "RFC1123", now.Format(time.RFC1123))

	list.AddItem("Cancel", "Back without inserting", 'q', func() {
		e.showFormatDialog()
	})

	uitheme.StyleList(list, "Insert Date & Time")
	list.SetBorder(true)

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.showFormatDialog()
			return nil
		}
		return event
	})

	modal := centredModal(list, 48, 16)
	e.App.SetRoot(modal, true).SetFocus(list)
}

// ── Symbol grid ─────────────────────────────────────────────────────────────

// Common symbols that most terminals / Unicode fonts render cleanly.
var symbolGrid = [][]string{
	{"—", "–", "…", "•", "·", "°", "′", "″", "※", "§", "¶", "†", "‡"},
	{"‘", "’", "“", "”", "‹", "›", "«", "»", "‚", "„", "″", "′", "‐"},
	{"←", "→", "↑", "↓", "↔", "↕", "⇐", "⇒", "⇑", "⇓", "⇔", "⇕", "↵"},
	{"≤", "≥", "≠", "≈", "≡", "±", "×", "÷", "∞", "√", "∑", "∏", "∂"},
	{"∈", "∉", "∋", "∩", "∪", "⊂", "⊃", "⊆", "⊇", "∧", "∨", "¬", "∀"},
	{"©", "®", "™", "€", "£", "¥", "¢", "¤", "₹", "₽", "₩", "₪", "₫"},
	{"✓", "✔", "✗", "✘", "☐", "☑", "☒", "★", "☆", "♠", "♣", "♥", "♦"},
	{"α", "β", "γ", "δ", "ε", "ζ", "η", "θ", "λ", "μ", "π", "σ", "ω"},
	{"Α", "Β", "Γ", "Δ", "Θ", "Λ", "Π", "Σ", "Φ", "Ψ", "Ω", "ℵ", "ℏ"},
	{"①", "②", "③", "④", "⑤", "⑥", "⑦", "⑧", "⑨", "⑩", "⑪", "⑫", "⑬"},
}

func (e *Editor) showSymbolGrid() {
	rows := len(symbolGrid)
	cols := len(symbolGrid[0])

	table := tview.NewTable().
		SetBorders(false).
		SetSelectable(true, true).
		SetFixed(0, 0)

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			ch := symbolGrid[r][c]
			cell := tview.NewTableCell(" " + ch + " ").
				SetAlign(tview.AlignCenter).
				SetTextColor(uitheme.Text).
				SetBackgroundColor(uitheme.Panel).
				SetSelectedStyle(tcell.StyleDefault.
					Background(uitheme.Btn).
					Foreground(tcell.ColorWhite)).
				SetExpansion(1)
			table.SetCell(r, c, cell)
		}
	}

	table.SetBorder(true).
		SetTitle(" Symbols  ·  Enter insert  ·  Esc back ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(uitheme.Amber).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(uitheme.Panel)

	status := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	status.SetBackgroundColor(uitheme.Ink)
	updateStatus := func() {
		r, c := table.GetSelection()
		if r >= 0 && r < rows && c >= 0 && c < cols {
			ch := symbolGrid[r][c]
			status.SetText(fmt.Sprintf(" [#f0d090]%s[-]   U+%04X", ch, []rune(ch)[0]))
		}
	}
	updateStatus()

	table.SetSelectionChangedFunc(func(row, col int) {
		updateStatus()
	})

	insertSelected := func() {
		r, c := table.GetSelection()
		if r >= 0 && r < rows && c >= 0 && c < cols {
			e.Doc.InsertText(symbolGrid[r][c])
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
		}
	}

	table.SetSelectedFunc(func(row, col int) {
		insertSelected()
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			e.showFormatDialog()
			return nil
		case tcell.KeyEnter:
			insertSelected()
			return nil
		}
		return event
	})

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(table, 0, 1, true).
		AddItem(status, 1, 0, false)
	root.SetBackgroundColor(uitheme.Ink)

	modal := centredModal(root, 56, rows+6)
	e.App.SetRoot(modal, true).SetFocus(table)
}

// ── Case / transform helpers (used by format dialog) ────────────────────────

func stringsToUpper(s string) string {
	return string(unicodeMap(s, unicode.ToUpper))
}

func stringsToLower(s string) string {
	return string(unicodeMap(s, unicode.ToLower))
}

func unicodeMap(s string, f func(rune) rune) []rune {
	rs := []rune(s)
	for i, r := range rs {
		rs[i] = f(r)
	}
	return rs
}

func titleCase(s string) string {
	rs := []rune(s)
	capNext := true
	for i, r := range rs {
		if unicode.IsSpace(r) || r == '-' || r == '_' {
			capNext = true
			continue
		}
		if capNext && unicode.IsLetter(r) {
			rs[i] = unicode.ToUpper(r)
			capNext = false
		} else {
			rs[i] = unicode.ToLower(r)
		}
	}
	return string(rs)
}

func invertCase(s string) string {
	rs := []rune(s)
	for i, r := range rs {
		if unicode.IsUpper(r) {
			rs[i] = unicode.ToLower(r)
		} else if unicode.IsLower(r) {
			rs[i] = unicode.ToUpper(r)
		}
	}
	return string(rs)
}

func reverseString(s string) string {
	rs := []rune(s)
	for i, j := 0, len(rs)-1; i < j; i, j = i+1, j-1 {
		rs[i], rs[j] = rs[j], rs[i]
	}
	return string(rs)
}

func collapseSpaces(s string) string {
	var b []rune
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) && r != '\n' {
			if !space {
				b = append(b, ' ')
				space = true
			}
			continue
		}
		b = append(b, r)
		space = false
	}
	return string(b)
}

func sentenceCase(s string) string {
	rs := []rune(stringsToLower(s))
	capNext := true
	for i, r := range rs {
		if r == '.' || r == '!' || r == '?' || r == '\n' {
			capNext = true
			continue
		}
		if capNext && unicode.IsLetter(r) {
			rs[i] = unicode.ToUpper(r)
			capNext = false
		}
	}
	return string(rs)
}
