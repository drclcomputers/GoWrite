package startmenu

import (
	"fmt"
	"strconv"
	"strings"

	"gowrite/internal/services/uitheme"
	"gowrite/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var pageSizes = []string{"A3", "A4", "A5", "Letter", "Legal"}
var orientations = []string{"Portrait", "Landscape"}
var saveFormats = []string{".gdoc", ".docx", ".txt", ".rtf"}

func ShowOptions(app *tview.Application, returnTo, returnFocus tview.Primitive) {
	settings, err := utils.LoadSettings()
	if err != nil {
		settings = utils.DefaultSettings()
	}
	loadError := err

	form := tview.NewForm()
	pageSize := tview.NewDropDown().SetLabel("Default paper:").SetOptions(pageSizes, nil)
	orientation := tview.NewDropDown().SetLabel("Orientation:").SetOptions(orientations, nil)
	marginLeft := newMarginField("Left margin (columns):")
	marginRight := newMarginField("Right margin (columns):")
	marginTop := newMarginField("Top margin (rows):")
	marginBottom := newMarginField("Bottom margin (rows):")
	saveFormat := tview.NewDropDown().SetLabel("Default save format:")
	formatOptions := []string{
		".gdoc - GoWrite native document",
		".docx - Microsoft Word document",
		".txt - Plain text",
		".rtf - Rich text",
	}
	saveFormat.SetOptions(formatOptions, nil)
	formattingMarks := tview.NewCheckbox().SetLabel("Show formatting marks by default")

	setValues := func(values utils.Settings) {
		pageSize.SetCurrentOption(optionIndex(pageSizes, values.DefaultPageSize))
		orientation.SetCurrentOption(optionIndex(orientations, values.DefaultOrientation))
		marginLeft.SetText(strconv.Itoa(values.MarginLeft))
		marginRight.SetText(strconv.Itoa(values.MarginRight))
		marginTop.SetText(strconv.Itoa(values.MarginTop))
		marginBottom.SetText(strconv.Itoa(values.MarginBottom))
		saveFormat.SetCurrentOption(optionIndex(saveFormats, values.DefaultSaveFormat))
		formattingMarks.SetChecked(values.ShowFormattingMarks)
	}
	setValues(settings)

	form.AddFormItem(pageSize)
	form.AddFormItem(orientation)
	form.AddFormItem(marginLeft)
	form.AddFormItem(marginRight)
	form.AddFormItem(marginTop)
	form.AddFormItem(marginBottom)
	form.AddFormItem(saveFormat)
	form.AddFormItem(formattingMarks)

	closeOptions := func() {
		app.SetRoot(returnTo, true).SetFocus(returnFocus)
	}
	showError := func(message string) {
		modal := tview.NewModal().
			SetText(message).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(int, string) {
				app.SetRoot(form, true).SetFocus(form)
			})
		uitheme.StyleModalWarn(modal, "Options")
		app.SetRoot(modal, true).SetFocus(modal)
	}

	form.AddButton("Save", func() {
		next, err := readOptions(pageSize, orientation, marginLeft, marginRight, marginTop, marginBottom, saveFormat, formattingMarks)
		if err != nil {
			showError(err.Error())
			return
		}
		if err := utils.SaveSettings(next); err != nil {
			showError(fmt.Sprintf("Could not save options: %v", err))
			return
		}
		closeOptions()
	})
	form.AddButton("Reset defaults", func() {
		setValues(utils.DefaultSettings())
	})
	form.AddButton("Cancel", closeOptions)
	uitheme.StyleForm(form, "Options  ·  defaults for new documents")
	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			closeOptions()
			return nil
		}
		return event
	})
	app.SetRoot(form, true).SetFocus(form)
	if loadError != nil {
		modal := tview.NewModal().
			SetText(fmt.Sprintf("Could not load saved options; defaults are shown.\nSave to replace the invalid settings.\n\n%v", loadError)).
			AddButtons([]string{"OK"}).
			SetDoneFunc(func(int, string) {
				app.SetRoot(form, true).SetFocus(form)
			})
		uitheme.StyleModalWarn(modal, "Options")
		app.SetRoot(modal, true).SetFocus(modal)
	}
}

func readOptions(
	pageSize, orientation *tview.DropDown,
	marginLeft, marginRight, marginTop, marginBottom *tview.InputField,
	saveFormat *tview.DropDown,
	formattingMarks *tview.Checkbox,
) (utils.Settings, error) {
	settings := utils.DefaultSettings()
	sizeIndex, _ := pageSize.GetCurrentOption()
	orientationIndex, _ := orientation.GetCurrentOption()
	formatIndex, _ := saveFormat.GetCurrentOption()
	if sizeIndex < 0 || sizeIndex >= len(pageSizes) ||
		orientationIndex < 0 || orientationIndex >= len(orientations) ||
		formatIndex < 0 || formatIndex >= len(saveFormats) {
		return settings, fmt.Errorf("select a valid default paper, orientation, and save format")
	}
	settings.DefaultPageSize = pageSizes[sizeIndex]
	settings.DefaultOrientation = orientations[orientationIndex]
	settings.DefaultSaveFormat = saveFormats[formatIndex]
	var err error
	if settings.MarginLeft, err = readMargin("Left", marginLeft); err != nil {
		return settings, err
	}
	if settings.MarginRight, err = readMargin("Right", marginRight); err != nil {
		return settings, err
	}
	if settings.MarginTop, err = readMargin("Top", marginTop); err != nil {
		return settings, err
	}
	if settings.MarginBottom, err = readMargin("Bottom", marginBottom); err != nil {
		return settings, err
	}
	settings.ShowFormattingMarks = formattingMarks.IsChecked()
	return settings, settings.Validate()
}

func readMargin(name string, field *tview.InputField) (int, error) {
	margin, err := strconv.Atoi(strings.TrimSpace(field.GetText()))
	if err != nil || margin < 0 || margin > 100 {
		return 0, fmt.Errorf("%s margin must be a number from 0 to 100", name)
	}
	return margin, nil
}

func newMarginField(label string) *tview.InputField {
	return tview.NewInputField().
		SetLabel(label).
		SetFieldWidth(4).
		SetAcceptanceFunc(func(text string, ch rune) bool {
			if ch == 0 {
				return len(text) <= 3
			}
			return ch >= '0' && ch <= '9' && len(text) <= 2
		})
}

func optionIndex(options []string, selected string) int {
	for i, option := range options {
		if strings.EqualFold(option, selected) {
			return i
		}
	}
	return 0
}
