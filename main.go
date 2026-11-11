// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

// GoWrite: a terminal word processor built on tview. Entry point.

package main

import (
	"errors"
	"flag"
	"fmt"
	"gowrite/internal/services/document"
	"gowrite/internal/services/ui/editor"
	"gowrite/internal/services/ui/startmenu"
	"gowrite/internal/services/uitheme"
	"gowrite/internal/utils"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func main() {
	app := tview.NewApplication()

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\nGoWrite crashed: %v\n%s\n", r, debug.Stack())
			os.Exit(1)
		}
	}()

	// Global Ctrl+C handler
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlC {
			modal := tview.NewModal().
				SetText("Ctrl+C detected. Exiting...\nUnsaved edits will be lost.").
				AddButtons([]string{"OK"}).
				SetDoneFunc(func(buttonIndex int, buttonLabel string) {
					app.Stop()
				})
			app.SetRoot(modal, true).SetFocus(modal)
			return nil
		}
		return event
	})

	var filename string
	flag.StringVar(&filename, "file", "", "Path to a text file to open")
	flag.Parse()

	if filename == "" && len(flag.Args()) > 0 {
		filename = flag.Args()[0]
	}

	recoveryPath, recoveryPathErr := utils.RecoveryPath()
	reopenRecovery := false
	if recoveryPathErr != nil {
		fmt.Fprintf(os.Stderr, "Could not locate recovery file: %v\n", recoveryPathErr)
		recoveryPath = ""
	} else if _, err := os.Stat(recoveryPath); err == nil {
		reopenRecovery, err = promptForRecovery(app)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not resolve recovery prompt: %v\n", err)
			os.Exit(1)
		}
		if !reopenRecovery {
			if err := os.Remove(recoveryPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				fmt.Fprintf(os.Stderr, "Could not discard recovery file: %v\n", err)
				os.Exit(1)
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "Could not check recovery file: %v\n", err)
	}

	var doc *document.Document
	startChoice := startmenu.StartNew
	var startMenu *startmenu.StartMenu

	if reopenRecovery {
		doc = document.New()
	} else if filename != "" {
		data, err := os.ReadFile(filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not open %s: %v\n", filename, err)
			os.Exit(1)
		}
		text := strings.ReplaceAll(string(data), "\r\n", "\n")
		text = strings.ReplaceAll(text, "\r", "\n")
		doc = document.New()
		doc.LoadText(text)
		doc.SetFilename(filename)
		doc.SetTitle(filepath.Base(filename))
	} else {
		startChoice, startMenu = startmenu.StartMenuUI(app)
		switch startChoice {
		case startmenu.StartQuit:
			return
		case startmenu.StartOpen:
			doc = document.New()
		case startmenu.StartNew:
			doc = document.New()
		default:
			doc = document.New()
		}
	}

	if doc == nil {
		return
	}

	settings, settingsErr := utils.LoadSettings()
	if settingsErr != nil {
		fmt.Fprintf(os.Stderr, "Could not load saved options; using defaults: %v\n", settingsErr)
	}
	editorUI := editor.NewEditor(app, doc, settings)
	editorUI.ConfigureRecovery(recoveryPath)
	editorUI.Mount()

	if startMenu != nil {
		startMenu.SetChooseHandler(func(choice startmenu.StartMenuResult) {
			switch choice {
			case startmenu.StartNew:
				app.SetRoot(editorUI.Root, true).SetFocus(editorUI.View)
				editorUI.NewDocument()
			case startmenu.StartOpen:
				editorUI.OpenDocument(startMenu.View, startMenu.Focus)
			case startmenu.StartQuit:
				app.Stop()
			}
		})
	}

	if reopenRecovery {
		if err := editorUI.ReopenRecovery(); err != nil {
			editor.ShowErrorModal(app, editorUI.Root, fmt.Sprintf("Could not reopen recovery file: %v", err))
		}
	} else if startChoice == startmenu.StartOpen && startMenu != nil {
		editorUI.OpenDocument(startMenu.View, startMenu.Focus)
	}

	if err := app.Run(); err != nil {
		editorUI.StopAutosave()
		log.Fatal(err)
	}
	editorUI.StopAutosave()
	if err := editorUI.DiscardRecovery(); err != nil {
		log.Printf("Could not clear recovery file on exit: %v", err)
	}
}

func promptForRecovery(app *tview.Application) (bool, error) {
	reopen, chosen := false, false
	modal := tview.NewModal().
		SetText("An autosaved recovery copy was found.\nWould you like to reopen it or discard it?").
		AddButtons([]string{"Reopen", "Discard"}).
		SetDoneFunc(func(_ int, label string) {
			reopen = label == "Reopen"
			chosen = true
			app.Stop()
		})
	uitheme.StyleModalWarn(modal, "Document Recovery")
	app.SetRoot(modal, true).SetFocus(modal)
	if err := app.Run(); err != nil {
		return false, err
	}
	if !chosen {
		return false, fmt.Errorf("recovery choice was not completed")
	}
	return reopen, nil
}
