// Copyright (c) 2025 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

// A simple terminal-based word processor using tview.
// main.go is the entry point of the application.

package main

import (
	"fmt"
	"gowrite/internal/services/document"
	"gowrite/internal/services/ui"
	"log"
	"os"
	"runtime/debug"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"flag"
)

func main() {
	app := tview.NewApplication()

	defer func() {
		if r := recover(); r != nil {
			fmt.Fprintf(os.Stderr, "\nGoWrite crashed: %v\n%s\n", r, debug.Stack())
			os.Exit(1)
		}
	}()

	// Global Ctrl+C handler – same spirit as GoSheet
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

	var doc *document.Document

	if filename != "" {
		// TODO: real file loading
		doc = document.New()
		doc.Filename = filename
		doc.Title = filename
		// For now just start empty; load logic comes next
	} else {
		// Show the start menu (same pattern as GoSheet)
		choice := ui.StartMenuUI(app)
		switch choice {
		case ui.StartQuit:
			return
		case ui.StartOpen:
			// Placeholder – will open file dialog later
			doc = document.New()
		case ui.StartNew:
			doc = document.New()
		default:
			doc = document.New()
		}
	}

	if doc == nil {
		return
	}

	// Rebuild the application after the start-menu Run() finished
	app = tview.NewApplication()
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

	editor := ui.NewEditor(app, doc)
	editor.Mount()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
