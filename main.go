// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

// A simple terminal-based word processor using tview.
// main.go is the entry point of the application.

package main

import (
	"flag"
	"fmt"
	"gowrite/internal/services/document"
	"gowrite/internal/services/ui"
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

	var doc *document.Document
	startChoice := ui.StartNew

	if filename != "" {
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
		startChoice = ui.StartMenuUI(app)
		switch startChoice {
		case ui.StartQuit:
			return
		case ui.StartOpen:
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

	editor := ui.NewEditor(app, doc)
	editor.Mount()

	if startChoice == ui.StartOpen {
		editor.OpenDocument()
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
