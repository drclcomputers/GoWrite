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

func (e *Editor) showFileDialog(saveMode bool) {
	homeDir, _ := os.UserHomeDir()
	docsPath := filepath.Join(homeDir, "Documents")
	if _, err := os.Stat(docsPath); os.IsNotExist(err) {
		docsPath = homeDir
	}

	currentPath := docsPath
	if saveMode && e.Doc.GetFilename() != "" {
		currentPath = filepath.Dir(e.Doc.GetFilename())
	}

	quickAccessList := tview.NewList()
	uitheme.StyleList(quickAccessList, "Quick access")

	fileList := tview.NewList()
	uitheme.StyleList(fileList, "Files")

	pathLabel := tview.NewTextView().
		SetDynamicColors(true).
		SetTextColor(uitheme.Text).
		SetText(fmt.Sprintf("[#f0d090]Location[-]  %s", utils.PrettyPath(filepath.Dir(currentPath), "no")))
	uitheme.Paint(pathLabel, "")

	form := tview.NewForm()

	filenameInput := tview.NewInputField().
		SetLabel("Filename: ").
		SetFieldWidth(0)

	if saveMode && e.Doc.GetFilename() != "" {
		filenameInput.SetText(strings.TrimSpace(filepath.Base(e.Doc.GetFilename())))
	}
	form.AddFormItem(filenameInput)

	var updateList func(path string)
	updateList = func(path string) {
		fileList.Clear()
		currentPath = path
		pathLabel.SetText(fmt.Sprintf("[yellow]Location:[::-] %s", utils.PrettyPath(path, "no")))

		if path != filepath.Dir(path) {
			fileList.AddItem("..", "Parent directory", '↑', func() {
				updateList(filepath.Dir(path))
			})
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}

		for _, entry := range entries {
			if entry.IsDir() {
				name := entry.Name()
				fullPath := filepath.Join(path, name)
				fileList.AddItem(fmt.Sprintf("# %s", name), "Directory", 0, func() {
					updateList(fullPath)
				})
			}
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				name := entry.Name()
				ext := strings.ToLower(filepath.Ext(name))

				if !saveMode {
					if ext != ".txt" && ext != ".md" {
						continue
					}
				}

				info, _ := entry.Info()
				var sizeStr string
				if info != nil {
					size := info.Size()
					if size < 1024 {
						sizeStr = fmt.Sprintf("%d B", size)
					} else if size < 1024*1024 {
						sizeStr = fmt.Sprintf("%.1f KB", float64(size)/1024)
					} else {
						sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
					}
				}

				fileList.AddItem(name, sizeStr, 0, func() {
					filenameInput.SetText(filepath.Join(path, name))
				})
			}
		}
	}

	populateQuickAccess := func() {
		quickAccessList.Clear()
		quickAccessList.AddItem("Home", homeDir, 'h', func() { updateList(homeDir) })
		if _, err := os.Stat(docsPath); err == nil {
			quickAccessList.AddItem("Documents", docsPath, 'd', func() { updateList(docsPath) })
		}
		desktopPath := filepath.Join(homeDir, "Desktop")
		if _, err := os.Stat(desktopPath); err == nil {
			quickAccessList.AddItem("Desktop", desktopPath, 0, func() { updateList(desktopPath) })
		}
		downloadsPath := filepath.Join(homeDir, "Downloads")
		if _, err := os.Stat(downloadsPath); err == nil {
			quickAccessList.AddItem("↓ Downloads", downloadsPath, 0, func() { updateList(downloadsPath) })
		}
	}

	populateQuickAccess()
	updateList(currentPath)

	if saveMode {
		form.AddButton("Save", func() {
			filename := strings.TrimSpace(filenameInput.GetText())
			if filename == "" {
				ShowErrorModal(e.App, e.Root, "Filename cannot be empty!")
				return
			}

			if !strings.Contains(filename, string(filepath.Separator)) {
				filename = filepath.Join(currentPath, filename)
			}

			if !strings.HasSuffix(strings.ToLower(filename), ".txt") && !strings.HasSuffix(strings.ToLower(filename), ".md") {
				filename += ".txt"
			}

			e.saveToFile(filename)
		})
	} else {
		form.AddButton("Open", func() {
			filename := strings.TrimSpace(filenameInput.GetText())
			if filename == "" {
				ShowErrorModal(e.App, e.Root, "Filename cannot be empty!")
				return
			}

			if !strings.Contains(filename, string(filepath.Separator)) {
				filename = filepath.Join(currentPath, filename)
			}

			if _, err := os.Stat(filename); os.IsNotExist(err) {
				ShowErrorModal(e.App, e.Root, fmt.Sprintf("File not found:\n%s", filename))
				return
			}

			e.loadFromFile(filename)
		})
	}

	form.AddButton("Cancel", func() {
		e.App.SetRoot(e.Root, true).SetFocus(e.View)
	})

	if saveMode {
		uitheme.StyleForm(form, "Options")
	} else {
		uitheme.StyleForm(form, "Open file")
	}

	// Layout mimicking GoSheet
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
	if saveMode {
		container.SetTitle(" Save Document ")
	} else {
		container.SetTitle(" Open Document ")
	}

	focusables := []tview.Primitive{quickAccessList, fileList, form}
	currentFocus := 2

	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			e.App.SetRoot(e.Root, true).SetFocus(e.View)
			return nil
		}
		if event.Key() == tcell.KeyLeft || (event.Key() == tcell.KeyLeft && event.Modifiers()&tcell.ModAlt != 0) {
			currentFocus = (currentFocus - 1 + len(focusables)) % len(focusables)
			e.App.SetFocus(focusables[currentFocus])
			return nil
		}
		if event.Key() == tcell.KeyRight || (event.Key() == tcell.KeyRight && event.Modifiers()&tcell.ModAlt != 0) {
			currentFocus = (currentFocus + 1) % len(focusables)
			e.App.SetFocus(focusables[currentFocus])
			return nil
		}
		return event
	})

	e.App.SetRoot(container, true).SetFocus(form)
}
