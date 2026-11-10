// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

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
	styleForm.AddDropDown("Transform", []string{
		"None", "UPPERCASE", "lowercase", "Title Case", "Sentence case",
		"iNVERT cASE", "Reverse", "Collapse spaces",
	}, 0, func(option string, _ int) {
		var transform func(string) string
		switch option {
		case "UPPERCASE":
			transform = stringsToUpper
		case "lowercase":
			transform = stringsToLower
		case "Title Case":
			transform = titleCase
		case "Sentence case":
			transform = sentenceCase
		case "iNVERT cASE":
			transform = invertCase
		case "Reverse":
			transform = reverseString
		case "Collapse spaces":
			transform = collapseSpaces
		default:
			return
		}
		e.Doc.TransformSelection(transform)
		e.closeFormatAndRedraw()
	})
	uitheme.StyleForm(styleForm, "Style")

	paragraphForm := tview.NewForm()
	paragraphStyles := []document.ParagraphStyle{
		document.ParagraphNormal,
		document.ParagraphTitle,
		document.ParagraphHeading1,
		document.ParagraphHeading2,
		document.ParagraphHeading3,
		document.ParagraphQuote,
		document.ParagraphCode,
	}
	styleNames := make([]string, len(paragraphStyles))
	styleIndex := 0
	for i, style := range paragraphStyles {
		styleNames[i] = style.String()
		if style == e.Doc.CurrentParagraphStyle() {
			styleIndex = i
		}
	}
	paragraphForm.AddDropDown("Style", styleNames, styleIndex, func(_ string, index int) {
		if index >= 0 && index < len(paragraphStyles) {
			e.Doc.SetParagraphStyle(paragraphStyles[index])
			e.redraw()
		}
	})

	alignIndex := 0
	switch e.Doc.CurrentAlign() {
	case document.AlignCenter:
		alignIndex = 1
	case document.AlignRight:
		alignIndex = 2
	}
	paragraphForm.AddDropDown("Align", []string{"Left", "Center", "Right"}, alignIndex, func(option string, _ int) {
		switch option {
		case "Left":
			e.Doc.SetParagraphAlign(document.AlignLeft)
		case "Center":
			e.Doc.SetParagraphAlign(document.AlignCenter)
		case "Right":
			e.Doc.SetParagraphAlign(document.AlignRight)
		}
	})

	spacingValues := []int{0, 1, 2, 3, 4, 5, 10}
	spacingOptions := []string{"0 rows", "1 row", "2 rows", "3 rows", "4 rows", "5 rows", "10 rows"}
	spacingIndex := 0
	for i, rows := range spacingValues {
		if rows == e.Doc.CurrentParagraphSpaceAfter() {
			spacingIndex = i
			break
		}
	}
	paragraphForm.AddDropDown("Space after", spacingOptions, spacingIndex, func(_ string, index int) {
		if index >= 0 && index < len(spacingValues) {
			e.Doc.SetParagraphSpaceAfter(spacingValues[index])
			e.redraw()
		}
	})
	uitheme.StyleForm(paragraphForm, "Paragraph")

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

	viewPanel := tview.NewForm()
	viewPanel.AddCheckbox("Formatting marks", e.showFormattingMarks, func(checked bool) {
		e.showFormattingMarks = checked
		e.redraw()
	})
	viewPanel.AddButton("Outline…", func() { e.showOutlineDialog() })
	uitheme.StyleForm(viewPanel, "View")

	insertPanel := buttonPanel(e.App, "Insert", []struct {
		label string
		fn    func()
	}{
		{"Date & Time…", func() { e.showDateTimeDialog() }},
		{"Symbols…", func() { e.showSymbolGrid() }},
		{"Table…", func() { e.showInsertTableDialog() }},
		{"Page Break", func() {
			e.Doc.InsertPageBreak()
			e.closeFormatAndRedraw()
		}},
	})
	insertBox := panelInBox(insertPanel)

	pagePanel := buttonPanel(e.App, "Page Setup", []struct {
		label string
		fn    func()
	}{
		{"Page Setup…", e.showPageSetupFromFormat},
		{"Header & Footer…", e.showHeaderFooterFromFormat},
	})
	pageBox := panelInBox(pagePanel)

	// ── Layout ──────────────────────────────────────────────────────────────
	leftCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(styleForm, 0, 2, true).
		AddItem(colorBox, 0, 1, false)

	midCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(panelInBox(listPanel), 0, 3, false).
		AddItem(panelInBox(viewPanel), 0, 1, false)

	rightCol := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(insertBox, 0, 5, false).
		AddItem(paragraphForm, 0, 4, false).
		AddItem(pageBox, 0, 3, false)

	columns := tview.NewFlex().
		AddItem(leftCol, 0, 1, false).
		AddItem(midCol, 0, 1, false).
		AddItem(rightCol, 0, 1, false)

	hint := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter).
		SetText(uitheme.Markup(" {val}Tab{clear} next   {val}Alt+Shift+←/→{clear} section   {val}Esc{clear} close"))
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

	sections := []tview.Primitive{styleForm, colorPanel, listPanel, viewPanel, insertPanel, paragraphForm, pagePanel}
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
	return uitheme.StyleButton(tview.NewButton(label).SetSelectedFunc(fn))
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
