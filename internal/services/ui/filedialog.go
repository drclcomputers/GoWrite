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
	e.showFileDialogWithReturn(mode, nil, e.Root, e.View)
}

func (e *Editor) showFileDialogWithSaveCallback(mode fileDialogMode, afterSave func()) {
	e.showFileDialogWithReturn(mode, afterSave, e.Root, e.View)
}

func (e *Editor) showFileDialogWithReturn(mode fileDialogMode, afterSave func(), returnTo, returnFocus tview.Primitive) {
	if returnTo == nil {
		returnTo = e.Root
	}
	if returnFocus == nil {
		returnFocus = e.View
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		ShowErrorModal(e.App, returnTo, fmt.Sprintf("Could not determine home directory: %v", err))
		return
	}
	docsPath := filepath.Join(homeDir, "Documents")
	if _, err := os.Stat(docsPath); os.IsNotExist(err) {
		docsPath = homeDir
	}

	currentFilename := e.Doc.GetFilename()
	currentPath, err := os.Getwd()
	if err != nil {
		ShowErrorModal(e.App, returnTo, fmt.Sprintf("Could not determine current directory: %v", err))
		return
	}
	currentPath, err = filepath.Abs(currentPath)
	if err != nil {
		ShowErrorModal(e.App, returnTo, fmt.Sprintf("Could not resolve current directory: %v", err))
		return
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
	case fileDialogSave, fileDialogExport:
		formats = allDocumentFormats
	}

	selectedFormat := 0
	baseName := e.Doc.GetTitle()
	if currentFilename != "" {
		baseName = strings.TrimSuffix(filepath.Base(currentFilename), filepath.Ext(currentFilename))
	}
	if baseName == "" {
		baseName = "Untitled"
	}
	if mode == fileDialogSave {
		switch strings.ToLower(filepath.Ext(currentFilename)) {
		case ".gdoc":
			selectedFormat = 0
		case ".docx":
			selectedFormat = 1
		case ".txt":
			selectedFormat = 2
		case ".rtf":
			selectedFormat = 3
		}
	} else if mode == fileDialogExport {
		switch strings.ToLower(filepath.Ext(currentFilename)) {
		case ".rtf":
			selectedFormat = 3
		case ".txt":
			selectedFormat = 2
		default:
			selectedFormat = 2
		}
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
			options[i] = fmt.Sprintf("%s - %s", format.extension, format.description)
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
		pathLabel.SetText(fmt.Sprintf("[#f0d090]Location:[::-] %s", utils.PrettyPath(path, "no")))

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
			if mode == fileDialogOpen && ext != ".gdoc" && ext != ".docx" && ext != ".txt" && ext != ".md" {
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
		addQuickAccess := func(name, path string, shortcut rune) {
			quickAccessList.AddItem(name, path, shortcut, func() { updateList(path) })
		}
		addQuickAccess("Home", homeDir, 'h')
		if _, err := os.Stat(docsPath); err == nil {
			addQuickAccess("Documents", docsPath, 'd')
		}
		for _, item := range []struct {
			name string
			path string
		}{
			{"Desktop", filepath.Join(homeDir, "Desktop")},
			{"↓ Downloads", filepath.Join(homeDir, "Downloads")},
		} {
			if _, err := os.Stat(item.path); err == nil {
				addQuickAccess(item.name, item.path, 0)
			}
		}
		if cwd, err := os.Getwd(); err == nil {
			addQuickAccess("Current Dir", cwd, 'c')
		}
		if filepath.Separator == '\\' {
			for drive := 'A'; drive <= 'Z'; drive++ {
				driveName := string(drive) + ":"
				drivePath := driveName + "\\"
				if _, err := os.Stat(drivePath); err == nil {
					addQuickAccess("Drive "+driveName, drivePath, 0)
				}
			}
		} else {
			addQuickAccess("Root (/)", "/", 0)
			for _, mount := range []string{"/mnt", "/media", "/Volumes"} {
				entries, err := os.ReadDir(mount)
				if err != nil || len(entries) == 0 {
					continue
				}
				mount := mount
				addQuickAccess(filepath.Base(mount), mount, 0)
				for _, entry := range entries {
					if entry.IsDir() {
						path := filepath.Join(mount, entry.Name())
						label := entry.Name()
						addQuickAccess(label, path, 0)
					}
				}
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
	writeToPath := func(filename string, checkOverwrite bool) {
		action := func() {
			extension := strings.ToLower(filepath.Ext(filename))
			var err error
			switch extension {
			case ".gdoc", ".docx", ".txt", ".rtf":
				err = e.saveDocumentToFile(filename)
			default:
				err = fmt.Errorf("unsupported file format %q", extension)
			}
			if err != nil {
				showDialogError(fmt.Sprintf("Could not write %s: %v", filename, err))
				return
			}
			if afterSave != nil {
				afterSave()
				return
			}
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			e.redraw()
			e.flashStatus(fmt.Sprintf("Saved %s", filepath.Base(filename)))
		}
		if checkOverwrite {
			confirmOverwrite(filename, action)
		} else {
			action()
		}
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
		form.AddButton("Save", func() {
			currentExtension := strings.ToLower(filepath.Ext(currentFilename))
			if formats[selectedFormat].extension == currentExtension {
				writeToPath(currentFilename, false)
				return
			}
			filename, err := resolveFilename(formats[selectedFormat].extension)
			if err != nil {
				showDialogError(err.Error())
				return
			}
			writeToPath(filename, true)
		})
		form.AddButton("Save As", func() {
			filename, err := resolveFilename(formats[selectedFormat].extension)
			if err != nil {
				showDialogError(err.Error())
				return
			}
			writeToPath(filename, true)
		})
	}
	form.AddButton("Cancel", func() { e.App.SetRoot(returnTo, true).SetFocus(returnFocus) })
	form.AddButton("Exit", func() { e.confirmQuit() })
	uitheme.StyleForm(form, "File")

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
	default:
		container.SetTitle(" Save Document ")
	}
	focusables := []tview.Primitive{quickAccessList, fileList, form}
	currentFocus := 2
	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(returnTo, true).SetFocus(returnFocus)
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
	{".docx", "Microsoft Word document"},
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
