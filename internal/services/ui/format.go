package ui

import (
	"gowrite/internal/services/document"
	"gowrite/internal/services/uitheme"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (e *Editor) showFormatDialog() {
	container := tview.NewFlex()

	// Left Column - Content & Alignment
	leftForm := tview.NewForm()

	// Just showing read-only text preview here
	leftForm.AddInputField("Text", e.Doc.CurrentPlain(), 0, nil, nil)

	// Align
	alignIndex := 0
	switch e.Doc.CurrentAlign() {
	case document.AlignCenter: alignIndex = 1
	case document.AlignRight: alignIndex = 2
	}
	leftForm.AddDropDown("Align", []string{"Left", "Center", "Right"}, alignIndex, func(option string, _ int) {
		switch option {
		case "Left": e.Doc.SetParagraphAlign(document.AlignLeft)
		case "Center": e.Doc.SetParagraphAlign(document.AlignCenter)
		case "Right": e.Doc.SetParagraphAlign(document.AlignRight)
		}
	})
	uitheme.StyleForm(leftForm, "Content")

	// Right Column - Styling
	rightForm := tview.NewForm()
	activeStyle := e.Doc.ActiveStyleFlags()

	rightForm.AddCheckbox("Bold", activeStyle&document.StyleBold != 0, func(checked bool) {
		if checked { e.Doc.SetActiveStyle(activeStyle | document.StyleBold) } else { e.Doc.SetActiveStyle(activeStyle &^ document.StyleBold) }
	})
	rightForm.AddCheckbox("Italic", activeStyle&document.StyleItalic != 0, func(checked bool) {
		if checked { e.Doc.SetActiveStyle(activeStyle | document.StyleItalic) } else { e.Doc.SetActiveStyle(activeStyle &^ document.StyleItalic) }
	})
	rightForm.AddCheckbox("Underline", activeStyle&document.StyleUnderline != 0, func(checked bool) {
		if checked { e.Doc.SetActiveStyle(activeStyle | document.StyleUnderline) } else { e.Doc.SetActiveStyle(activeStyle &^ document.StyleUnderline) }
	})
	rightForm.AddCheckbox("Strikethrough", activeStyle&document.StyleStrike != 0, func(checked bool) {
		if checked { e.Doc.SetActiveStyle(activeStyle | document.StyleStrike) } else { e.Doc.SetActiveStyle(activeStyle &^ document.StyleStrike) }
	})
	uitheme.StyleForm(rightForm, "Style")

	// Color Form
	colorForm := tview.NewForm()
	colorForm.AddButton("Text Color", func() { e.showColorPicker(false) })
	colorForm.AddButton("Background Color", func() { e.showColorPicker(true) })
	uitheme.StyleForm(colorForm, "Colors")

	leftColumn := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(leftForm, 0, 1, false)

	rightColumn := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(rightForm, 0, 3, false).
		AddItem(colorForm, 0, 2, false)

	columns := tview.NewFlex().
		AddItem(leftColumn, 0, 1, false).
		AddItem(rightColumn, 0, 1, false)

	// Button bar mimicking GoSheet
	buttonBar := tview.NewFlex().SetDirection(tview.FlexRow)
	buttonText := tview.NewTextView().
		SetText(" [#f0d090]Alt+Shift+←/→[-] section    [#f0d090]Tab[-] next field    [#f0d090]Alt+S[-] apply & close    [#f0d090]Esc[-] cancel").
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	buttonText.SetBackgroundColor(tcell.NewRGBColor(14, 17, 26))
	buttonBar.AddItem(buttonText, 1, 0, false)

	container = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(columns, 0, 1, true).
		AddItem(buttonBar, 1, 0, false)
	container.SetBackgroundColor(tcell.NewRGBColor(10, 12, 18))

	container.SetBorder(true).
		SetTitle(" Format Text ").
		SetTitleAlign(tview.AlignCenter).
		SetTitleColor(tcell.NewRGBColor(240, 208, 144)).
		SetBorderColor(tcell.NewRGBColor(70, 160, 170))

	allForms := []*tview.Form{leftForm, rightForm, colorForm}
	currentFormIndex := 0

	markSection := func(forms []*tview.Form, active int) {
		for i, f := range forms {
			if i == active {
				f.SetBorderColor(tcell.NewRGBColor(240, 208, 144))
				continue
			}
			f.SetBorderColor(tcell.NewRGBColor(70, 160, 170))
		}
	}
	markSection(allForms, 0)

	moveSection := func(dir int) {
		n := len(allForms)
		currentFormIndex = (currentFormIndex + dir + n) % n
		markSection(allForms, currentFormIndex)
		e.App.SetFocus(allForms[currentFormIndex])
	}

	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
			return nil
		}
		if event.Modifiers()&tcell.ModAlt != 0 && (event.Rune() == 's' || event.Rune() == 'S') {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
			return nil
		}
		if event.Modifiers()&tcell.ModAlt != 0 && event.Modifiers()&tcell.ModShift != 0 {
			if event.Key() == tcell.KeyLeft {
				moveSection(-1)
				return nil
			}
			if event.Key() == tcell.KeyRight {
				moveSection(1)
				return nil
			}
		}
		return event
	})

	e.App.SetRoot(container, true).SetFocus(leftForm)
}
