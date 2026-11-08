// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package ui

import (
	"fmt"

	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// showFormatDialog opens the main Format screen (Alt+Enter)
func (e *Editor) showFormatDialog() {
	styleForm := tview.NewForm()
	styleForm.AddCheckbox("Bold", e.Doc.ActiveStyleFlags()&document.StyleBold != 0, func(checked bool) {
		if checked {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() | document.StyleBold)
		} else {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() &^ document.StyleBold)
		}
	})
	styleForm.AddCheckbox("Italic", e.Doc.ActiveStyleFlags()&document.StyleItalic != 0, func(checked bool) {
		if checked {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() | document.StyleItalic)
		} else {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() &^ document.StyleItalic)
		}
	})
	styleForm.AddCheckbox("Underline", e.Doc.ActiveStyleFlags()&document.StyleUnderline != 0, func(checked bool) {
		if checked {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() | document.StyleUnderline)
		} else {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() &^ document.StyleUnderline)
		}
	})
	styleForm.AddCheckbox("Strikethrough", e.Doc.ActiveStyleFlags()&document.StyleStrike != 0, func(checked bool) {
		if checked {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() | document.StyleStrike)
		} else {
			e.Doc.SetActiveStyle(e.Doc.ActiveStyleFlags() &^ document.StyleStrike)
		}
	})
	uitheme.StyleForm(styleForm, "Style")

	alignForm := tview.NewForm()
	alignIndex := 0
	switch e.Doc.CurrentAlign() {
	case document.AlignCenter:
		alignIndex = 1
	case document.AlignRight:
		alignIndex = 2
	}
	alignForm.AddDropDown("Align", []string{"Left", "Center", "Right"}, alignIndex, func(option string, _ int) {
		switch option {
		case "Left":
			e.Doc.SetParagraphAlign(document.AlignLeft)
		case "Center":
			e.Doc.SetParagraphAlign(document.AlignCenter)
		case "Right":
			e.Doc.SetParagraphAlign(document.AlignRight)
		}
	})
	uitheme.StyleForm(alignForm, "Alignment")

	textColorBtn := makePanelButton("Text Color", func() { e.showColorPicker(false) })
	bgColorBtn := makePanelButton("Background", func() { e.showColorPicker(true) })
	wireButtonNav(e.App, []*tview.Button{textColorBtn, bgColorBtn})

	colorPanel := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(panelSpacer(), 1, 0, false).
		AddItem(paddedButton(textColorBtn), 1, 0, true).
		AddItem(panelSpacer(), 1, 0, false).
		AddItem(paddedButton(bgColorBtn), 1, 0, false).
		AddItem(panelSpacer(), 1, 0, false)
	colorPanel.SetBorder(true).
		SetTitle(" Colors ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(uitheme.Amber).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(uitheme.Panel)
	colorBox := panelInBox(colorPanel)

	listPanel := buttonPanel(e.App, "Lists", []struct {
		label string
		fn    func()
	}{
		{"1. 2. 3. …", func() {
			e.Doc.ApplyListPrefix(func(i int) string { return fmt.Sprintf("%d. ", i+1) })
			e.closeFormatAndRedraw()
		}},
		{"a. b. c. …", func() {
			e.Doc.ApplyListPrefix(func(i int) string { return fmt.Sprintf("%c. ", 'a'+rune(i%26)) })
			e.closeFormatAndRedraw()
		}},
		{"A. B. C. …", func() {
			e.Doc.ApplyListPrefix(func(i int) string { return fmt.Sprintf("%c. ", 'A'+rune(i%26)) })
			e.closeFormatAndRedraw()
		}},
		{"• bullet", func() {
			e.Doc.ApplyListPrefix(func(int) string { return "• " })
			e.closeFormatAndRedraw()
		}},
		{"- dash", func() {
			e.Doc.ApplyListPrefix(func(int) string { return "- " })
			e.closeFormatAndRedraw()
		}},
		{"[ ] checkbox", func() {
			e.Doc.ApplyListPrefix(func(int) string { return "[ ] " })
			e.closeFormatAndRedraw()
		}},
		{"i. ii. iii.", func() {
			romans := []string{"i", "ii", "iii", "iv", "v", "vi", "vii", "viii", "ix", "x",
				"xi", "xii", "xiii", "xiv", "xv", "xvi", "xvii", "xviii", "xix", "xx"}
			e.Doc.ApplyListPrefix(func(i int) string {
				if i < len(romans) {
					return romans[i] + ". "
				}
				return fmt.Sprintf("%d. ", i+1)
			})
			e.closeFormatAndRedraw()
		}},
	})

	insertPanel := buttonPanel(e.App, "Insert", []struct {
		label string
		fn    func()
	}{
		{"Date & Time…", func() { e.showDateTimeDialog() }},
		{"Symbols…", func() { e.showSymbolGrid() }},
		{"Page Break", func() {
			e.Doc.InsertPageBreak()
			e.closeFormatAndRedraw()
		}},
	})
	insertBox := panelInBox(insertPanel)

	xformPanel := buttonPanel(e.App, "Case & Transform", []struct {
		label string
		fn    func()
	}{
		{"UPPERCASE", func() {
			e.Doc.TransformSelection(stringsToUpper)
			e.closeFormatAndRedraw()
		}},
		{"lowercase", func() {
			e.Doc.TransformSelection(stringsToLower)
			e.closeFormatAndRedraw()
		}},
		{"Title Case", func() {
			e.Doc.TransformSelection(titleCase)
			e.closeFormatAndRedraw()
		}},
		{"Sentence case", func() {
			e.Doc.TransformSelection(sentenceCase)
			e.closeFormatAndRedraw()
		}},
		{"iNVERT cASE", func() {
			e.Doc.TransformSelection(invertCase)
			e.closeFormatAndRedraw()
		}},
		{"Reverse", func() {
			e.Doc.TransformSelection(reverseString)
			e.closeFormatAndRedraw()
		}},
		{"Collapse spaces", func() {
			e.Doc.TransformSelection(collapseSpaces)
			e.closeFormatAndRedraw()
		}},
	})
	xformBox := panelInBox(xformPanel)

	// ── Layout ──────────────────────────────────────────────────────────────
	leftCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(styleForm, 0, 3, true).
		AddItem(alignForm, 0, 1, false).
		AddItem(colorBox, 7, 0, false)

	midCol := panelInBox(listPanel)

	rightCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(insertBox, 0, 1, false).
		AddItem(xformBox, 0, 2, false)

	columns := tview.NewFlex().
		AddItem(leftCol, 0, 1, true).
		AddItem(midCol, 0, 1, false).
		AddItem(rightCol, 0, 1, false)

	hint := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(" [#f0d090]Tab[-] next   [#f0d090]Alt+←/→[-] section   [#f0d090]Esc[-] close")
	hint.SetBackgroundColor(uitheme.Ink)

	root := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(columns, 0, 1, true).
		AddItem(hint, 1, 0, false)
	root.SetBackgroundColor(uitheme.Ink)
	root.SetBorder(true).
		SetTitle(" Format Text ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(uitheme.Amber).
		SetBorderColor(uitheme.Line)

	sections := []tview.Primitive{styleForm, alignForm, colorPanel, listPanel, insertPanel, xformPanel}
	secIdx := 0
	markFormatSection(sections, secIdx)

	root.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
			return nil
		}
		mod := event.Modifiers()
		if mod&tcell.ModAlt != 0 {
			switch event.Key() {
			case tcell.KeyLeft:
				secIdx = (secIdx - 1 + len(sections)) % len(sections)
				markFormatSection(sections, secIdx)
				e.App.SetFocus(sections[secIdx])
				return nil
			case tcell.KeyRight:
				secIdx = (secIdx + 1) % len(sections)
				markFormatSection(sections, secIdx)
				e.App.SetFocus(sections[secIdx])
				return nil
			}
		}
		return event
	})

	modal := centredModal(root, 90, 30)
	e.App.SetRoot(modal, true).SetFocus(styleForm)
}

func (e *Editor) closeFormatAndRedraw() {
	e.App.SetRoot(e.Root, true).SetFocus(e.View)
	e.redraw()
}

func markFormatSection(sections []tview.Primitive, active int) {
	for i, p := range sections {
		col := uitheme.Line
		if i == active {
			col = uitheme.Amber
		}
		switch v := p.(type) {
		case *tview.Form:
			v.SetBorderColor(col)
		case *tview.Flex:
			v.SetBorderColor(col)
		}
	}
}

func panelInBox(panel tview.Primitive) *tview.Pages {
	return tview.NewPages().
		AddPage("background", tview.NewBox().SetBackgroundColor(uitheme.Panel), true, true).
		AddPage("panel", panel, true, true)
}

// makePanelButton creates a themed button used inside format panels.
func makePanelButton(label string, fn func()) *tview.Button {
	b := tview.NewButton(label).SetSelectedFunc(fn)
	b.SetBackgroundColor(uitheme.Btn)
	b.SetLabelColor(tcell.ColorWhite)
	b.SetBackgroundColorActivated(uitheme.Amber)
	b.SetLabelColorActivated(uitheme.Ink)
	return b
}

// panelSpacer is a 1-cell box painted with the panel background.
func panelSpacer() tview.Primitive {
	return tview.NewBox().SetBackgroundColor(uitheme.Panel)
}

// paddedButton wraps a button with small horizontal padding (panel-colored).
func paddedButton(b *tview.Button) tview.Primitive {
	return tview.NewFlex().
		AddItem(panelSpacer(), 1, 0, false).
		AddItem(b, 0, 1, true).
		AddItem(panelSpacer(), 1, 0, false)
}

// wireButtonNav lets Tab / Shift+Tab / ↑ / ↓ move focus among a vertical button group.
func wireButtonNav(app *tview.Application, buttons []*tview.Button) {
	for i, b := range buttons {
		i := i
		b.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyTab, tcell.KeyDown:
				app.SetFocus(buttons[(i+1)%len(buttons)])
				return nil
			case tcell.KeyBacktab, tcell.KeyUp:
				app.SetFocus(buttons[(i-1+len(buttons))%len(buttons)])
				return nil
			}
			return event
		})
	}
}

// buttonPanel builds a bordered vertical stack of full-width buttons.
func buttonPanel(app *tview.Application, title string, items []struct {
	label string
	fn    func()
}) *tview.Flex {
	col := tview.NewFlex().SetDirection(tview.FlexRow)
	col.SetBackgroundColor(uitheme.Panel)
	col.AddItem(panelSpacer(), 1, 0, false)

	buttons := make([]*tview.Button, 0, len(items))
	for i, it := range items {
		btn := makePanelButton(it.label, it.fn)
		buttons = append(buttons, btn)
		col.AddItem(paddedButton(btn), 1, 0, i == 0)
		if i < len(items)-1 {
			col.AddItem(panelSpacer(), 1, 0, false)
		}
	}
	col.AddItem(panelSpacer(), 1, 0, false)
	wireButtonNav(app, buttons)

	col.SetBorder(true).
		SetTitle(" " + title + " ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(uitheme.Amber).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(uitheme.Panel)
	return col
}
