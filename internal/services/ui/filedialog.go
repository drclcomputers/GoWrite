// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gowrite/internal/services/uitheme"
	"gowrite/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type fileDialogMode int

const (
	fileDialogOpen fileDialogMode = iota
	fileDialogSave
	fileDialogExport
)

type fileFormatOption struct {
	extension   string
	description string
}

func (e *Editor) showFileDialog(mode fileDialogMode) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		ShowErrorModal(e.App, e.Root, fmt.Sprintf("Could not determine home directory: %v", err))
		return
	}
	docsPath := filepath.Join(homeDir, "Documents")
	if _, err := os.Stat(docsPath); os.IsNotExist(err) {
		docsPath = homeDir
	}

	currentFilename := e.Doc.GetFilename()
	currentPath := docsPath
	if currentFilename != "" && mode != fileDialogOpen {
		currentPath = filepath.Dir(currentFilename)
	}

	quickAccessList := tview.NewList()
	uitheme.StyleList(quickAccessList, "Quick access")
	fileList := tview.NewList()
	uitheme.StyleList(fileList, "Files")
	pathLabel := tview.NewTextView().
		SetDynamicColors(true).
		SetTextColor(uitheme.Text)
	uitheme.Paint(pathLabel, "")

	form := tview.NewForm()
	filenameInput := tview.NewInputField().
		SetLabel("Filename: ").
		SetFieldWidth(0)
	form.AddFormItem(filenameInput)

	var formats []fileFormatOption
	switch mode {
	case fileDialogSave:
		formats = []fileFormatOption{
			{".gdoc", "GoWrite native document"},
			{".txt", "Plain text export"},
			{".rtf", "Rich text export"},
		}
	case fileDialogExport:
		formats = []fileFormatOption{
			{".txt", "Plain text"},
			{".rtf", "Rich text"},
		}
	}

	selectedFormat := 0
	baseName := e.Doc.GetTitle()
	if currentFilename != "" {
		baseName = strings.TrimSuffix(filepath.Base(currentFilename), filepath.Ext(currentFilename))
	}
	if baseName == "" {
		baseName = "Untitled"
	}
	if mode == fileDialogSave && strings.EqualFold(filepath.Ext(currentFilename), ".gdoc") {
		selectedFormat = 0
	} else if mode == fileDialogExport && strings.EqualFold(filepath.Ext(currentFilename), ".rtf") {
		selectedFormat = 1
	}
	if mode == fileDialogOpen {
		filenameInput.SetText("")
	} else {
		filenameInput.SetText(baseName + formatsExtension(formats, selectedFormat))
	}

	var formatDropdown *tview.DropDown
	if mode != fileDialogOpen {
		options := make([]string, len(formats))
		for i, format := range formats {
			options[i] = format.extension + "  " + format.description
		}
		formatDropdown = tview.NewDropDown().
			SetLabel("Format: ").
			SetOptions(options, func(_ string, index int) {
				if index < 0 || index >= len(formats) {
					return
				}
				selectedFormat = index
				filenameInput.SetText(withFileExtension(filenameInput.GetText(), formats[index].extension, formats))
			}).
			SetCurrentOption(selectedFormat)
		form.AddFormItem(formatDropdown)
	}

	var dialog tview.Primitive
	var updateList func(string)
	updateList = func(path string) {
		fileList.Clear()
		currentPath = path
		pathLabel.SetText(fmt.Sprintf("[#f0d090]Location[-]  %s", utils.PrettyPath(path, "no")))

		if path != filepath.Dir(path) {
			parent := filepath.Dir(path)
			fileList.AddItem("..", "Parent directory", '↑', func() { updateList(parent) })
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			fileList.AddItem("Cannot read directory", err.Error(), 0, nil)
			return
		}
		for _, entry := range entries {
			if entry.IsDir() {
				name := entry.Name()
				fullPath := filepath.Join(path, name)
				fileList.AddItem("# "+name, "Directory", 0, func() { updateList(fullPath) })
			}
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			name := entry.Name()
			ext := strings.ToLower(filepath.Ext(name))
			if mode == fileDialogOpen && ext != ".gdoc" && ext != ".txt" && ext != ".md" {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			size := formatFileSize(info.Size())
			fullPath := filepath.Join(path, name)
			fileList.AddItem(name, size, 0, func() {
				filenameInput.SetText(fullPath)
				if formatDropdown != nil {
					for i, format := range formats {
						if strings.EqualFold(ext, format.extension) {
							selectedFormat = i
							formatDropdown.SetCurrentOption(i)
							break
						}
					}
				}
			})
		}
	}

	populateQuickAccess := func() {
		quickAccessList.Clear()
		quickAccessList.AddItem("Home", homeDir, 'h', func() { updateList(homeDir) })
		if _, err := os.Stat(docsPath); err == nil {
			quickAccessList.AddItem("Documents", docsPath, 'd', func() { updateList(docsPath) })
		}
		for _, item := range []struct {
			name string
			path string
		}{
			{"Desktop", filepath.Join(homeDir, "Desktop")},
			{"↓ Downloads", filepath.Join(homeDir, "Downloads")},
		} {
			if _, err := os.Stat(item.path); err == nil {
				item := item
				quickAccessList.AddItem(item.name, item.path, 0, func() { updateList(item.path) })
			}
		}
	}
	populateQuickAccess()
	updateList(currentPath)

	resolveFilename := func(extension string) (string, error) {
		filename := strings.TrimSpace(filenameInput.GetText())
		if filename == "" {
			return "", fmt.Errorf("filename cannot be empty")
		}
		if extension != "" {
			filename = withFileExtension(filename, extension, allDocumentFormats)
		}
		if strings.HasPrefix(filename, "~") {
			filename = filepath.Join(homeDir, strings.TrimLeft(strings.TrimPrefix(filename, "~"), string(os.PathSeparator)))
		} else if !filepath.IsAbs(filename) {
			filename = filepath.Join(currentPath, filename)
		}
		return filepath.Abs(filename)
	}

	showDialogError := func(message string) {
		ShowErrorModal(e.App, dialog, message)
	}
	confirmOverwrite := func(filename string, action func()) {
		info, err := os.Stat(filename)
		if err == nil {
			if info.IsDir() {
				showDialogError("The selected path is a directory")
				return
			}
			modal := tview.NewModal().
				SetText(fmt.Sprintf("'%s' already exists.\n\nOverwrite it?", filepath.Base(filename))).
				AddButtons([]string{"Overwrite", "Cancel"}).
				SetDoneFunc(func(_ int, label string) {
					if label == "Overwrite" {
						action()
					} else {
						e.App.SetRoot(dialog, true).SetFocus(form)
					}
				})
			uitheme.StyleModalWarn(modal, "Overwrite")
			e.App.SetRoot(modal, true).SetFocus(modal)
			return
		}
		if !os.IsNotExist(err) {
			showDialogError(fmt.Sprintf("Could not check the destination: %v", err))
			return
		}
		action()
	}

	if mode == fileDialogOpen {
		form.AddButton("Open", func() {
			filename, err := resolveFilename("")
			if err != nil {
				showDialogError(err.Error())
				return
			}
			info, err := os.Stat(filename)
			if err != nil {
				showDialogError(fmt.Sprintf("Could not open %s: %v", filename, err))
				return
			}
			if info.IsDir() {
				updateList(filename)
				return
			}
			if err := e.loadFromFile(filename); err != nil {
				showDialogError(fmt.Sprintf("Could not open %s: %v", filename, err))
				return
			}
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
		})
	} else {
		if mode == fileDialogSave {
			form.AddButton("Save", func() {
				if formats[selectedFormat].extension != ".gdoc" {
					showDialogError("Select .gdoc to save the native document; use Export for .txt or .rtf")
					return
				}
				filename, err := resolveFilename(".gdoc")
				if err != nil {
					showDialogError(err.Error())
					return
				}
				confirmOverwrite(filename, func() {
					if err := e.saveNativeToFile(filename); err != nil {
						showDialogError(fmt.Sprintf("Could not save %s: %v", filename, err))
						return
					}
					e.App.SetRoot(e.Root, true).SetFocus(e.View)
					e.redraw()
				})
			})
		}
		form.AddButton("Export", func() {
			extension := formats[selectedFormat].extension
			if extension == ".gdoc" {
				showDialogError("Choose .txt or .rtf to export")
				return
			}
			filename, err := resolveFilename(extension)
			if err != nil {
				showDialogError(err.Error())
				return
			}
			confirmOverwrite(filename, func() {
				if err := e.exportToFile(filename); err != nil {
					showDialogError(fmt.Sprintf("Could not export %s: %v", filename, err))
					return
				}
				e.App.SetRoot(e.Root, true).SetFocus(e.View)
				e.redraw()
				e.flashStatus(fmt.Sprintf("Exported %s", filepath.Base(filename)))
			})
		})
	}
	form.AddButton("Cancel", func() {
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
	})
	if mode == fileDialogOpen {
		uitheme.StyleForm(form, "Open file")
	} else if mode == fileDialogExport {
		uitheme.StyleForm(form, "Export document")
	} else {
		uitheme.StyleForm(form, "Save document")
	}

	leftSection := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(form, 0, 1, true).
		AddItem(quickAccessList, 0, 2, false)
	topSection := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(pathLabel, 3, 0, false).
		AddItem(leftSection, 0, 1, true)
	mainLayout := tview.NewFlex().
		AddItem(topSection, 0, 1, true).
		AddItem(fileList, 0, 1, false)
	instructions := tview.NewTextView().
		SetText(" [#f0d090]Alt+Shift+←/→[-] panels    [#f0d090]Enter[-] select    [#f0d090]Tab[-] buttons    [#f0d090]Esc[-] back").
		SetDynamicColors(true).
		SetTextAlign(tview.AlignCenter)
	container := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(mainLayout, 0, 1, true).
		AddItem(instructions, 1, 0, false)
	container.SetBorder(true).
		SetBorderColor(uitheme.Line).
		SetTitleColor(uitheme.Amber)
	switch mode {
	case fileDialogOpen:
		container.SetTitle(" Open Document ")
	case fileDialogExport:
		container.SetTitle(" Export Document ")
	default:
		container.SetTitle(" Save Document ")
	}
	focusables := []tview.Primitive{quickAccessList, fileList, form}
	currentFocus := 2
	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		if event.Modifiers()&tcell.ModAlt != 0 && event.Modifiers()&tcell.ModShift != 0 {
			switch event.Key() {
			case tcell.KeyLeft:
				currentFocus = (currentFocus - 1 + len(focusables)) % len(focusables)
				e.App.SetFocus(focusables[currentFocus])
				return nil
			case tcell.KeyRight:
				currentFocus = (currentFocus + 1) % len(focusables)
				e.App.SetFocus(focusables[currentFocus])
				return nil
			}
		}
		return event
	})
	dialog = container
	e.App.SetRoot(container, true).SetFocus(form)
}

var allDocumentFormats = []fileFormatOption{
	{".gdoc", "GoWrite native document"},
	{".txt", "Plain text"},
	{".rtf", "Rich text"},
}

func formatsExtension(formats []fileFormatOption, index int) string {
	if index < 0 || index >= len(formats) {
		return ".gdoc"
	}
	return formats[index].extension
}

func withFileExtension(filename, extension string, formats []fileFormatOption) string {
	filename = strings.TrimSpace(filename)
	ext := filepath.Ext(filename)
	if strings.EqualFold(ext, extension) {
		return filename
	}
	for _, format := range formats {
		if strings.EqualFold(ext, format.extension) {
			filename = strings.TrimSuffix(filename, ext)
			break
		}
	}
	return filename + extension
}

func formatFileSize(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}
