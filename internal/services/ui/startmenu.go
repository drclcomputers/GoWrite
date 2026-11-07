// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package ui

import (
	"fmt"
	"gowrite/internal/services/uitheme"
	"gowrite/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const newDocButtonLabel = `
┌──────────────┐
│   ┌────╮     │
│   │    ╰┐    │
│   │ ──── │   │
│   │ ───  │   │
│   └─────┘    │
└──────────────┘
NEW DOCUMENT
`

const openDocButtonLabel = `
┌────────────┐
│ ▄▄▄        │
│ ┌────────┐ │
│ │ ────── │ │
│ └────────┘ │
└────────────┘
OPEN
`

const quitButtonLabel = `
┌────────────┐
│ ┌──────┐   │
│ │    ● │ → │
│ │      │   │
│ └──────┘   │
└────────────┘
QUIT
`

// StartMenuResult is what the start menu returns.
type StartMenuResult int

const (
	StartNew StartMenuResult = iota
	StartOpen
	StartQuit
)

func StartMenuUI(app *tview.Application) StartMenuResult {
	result := StartQuit

	container := tview.NewFlex()

	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)

	title := tview.NewTextView().
		SetText(fmt.Sprintf("[::b]GoWrite[::-] [yellow]v%s[::-]", utils.VER)).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	title.SetBorder(true).
		SetTitleColor(uitheme.Amber).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(uitheme.Ink)

	newBtn := tview.NewTextView().
		SetText(newDocButtonLabel).
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.ColorLightGreen)
	newBtn.SetBorder(true).
		SetBorderColor(uitheme.Ok).
		SetBackgroundColor(tcell.NewRGBColor(16, 28, 26))
	newBtn.SetScrollable(false)

	openBtn := tview.NewTextView().
		SetText(openDocButtonLabel).
		SetTextAlign(tview.AlignCenter).
		SetTextColor(tcell.NewRGBColor(168, 212, 220))
	openBtn.SetBorder(true).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(tcell.NewRGBColor(14, 22, 32))
	openBtn.SetScrollable(false)

	quitBtn := tview.NewTextView().
		SetText(quitButtonLabel).
		SetTextAlign(tview.AlignCenter).
		SetTextColor(uitheme.Warn)
	quitBtn.SetBorder(true).
		SetBorderColor(uitheme.Warn).
		SetBackgroundColor(tcell.NewRGBColor(28, 20, 18))
	quitBtn.SetScrollable(false)

	leftPanel.
		AddItem(title, 0, 1, false).
		AddItem(newBtn, 0, 3, true).
		AddItem(openBtn, 0, 3, true).
		AddItem(quitBtn, 0, 3, true)

	// Right side – placeholder for recent files later
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)

	recentTitle := tview.NewTextView().
		SetText("[::b]Recent Documents[::-]").
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)
	recentTitle.SetBorder(true).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(uitheme.Ink)

	recentList := tview.NewList()
	uitheme.StyleList(recentList, "")
	recentList.AddItem("No recent documents yet", "They will appear here after you save files", 0, nil)
	recentList.SetSelectedTextColor(tcell.ColorWhite)

	rightPanel.
		AddItem(recentTitle, 3, 0, false).
		AddItem(recentList, 0, 1, true)

	container.
		AddItem(leftPanel, 0, 1, true).
		AddItem(rightPanel, 0, 3, false)

	container.SetBorder(true).
		SetTitle(" GoWrite  ·  arrows or tab to move  ·  Enter to choose  ·  Esc quits ").
		SetTitleColor(uitheme.Amber).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(tcell.NewRGBColor(10, 12, 18))

	leftItems := []tview.Primitive{newBtn, openBtn, quitBtn}
	currentLeft := 0
	inLeft := true

	choose := func(r StartMenuResult) {
		result = r
		app.Stop()
	}

	newBtn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			choose(StartNew)
		}
		return event
	})
	openBtn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			choose(StartOpen)
		}
		return event
	})
	quitBtn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			choose(StartQuit)
		}
		return event
	})

	container.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch {
		case event.Key() == tcell.KeyEscape || event.Rune() == 'q' || event.Rune() == 'Q':
			choose(StartQuit)
			return nil
		case event.Rune() == 'n' && event.Modifiers()&tcell.ModAlt != 0:
			choose(StartNew)
			return nil
		case event.Rune() == 'o' && event.Modifiers()&tcell.ModAlt != 0:
			choose(StartOpen)
			return nil
		case event.Key() == tcell.KeyLeft || event.Key() == tcell.KeyBacktab:
			if inLeft {
				currentLeft--
				if currentLeft < 0 {
					inLeft = false
					app.SetFocus(recentList)
				} else {
					app.SetFocus(leftItems[currentLeft])
				}
			} else {
				inLeft = true
				currentLeft = len(leftItems) - 1
				app.SetFocus(leftItems[currentLeft])
			}
			return nil
		case event.Key() == tcell.KeyRight || event.Key() == tcell.KeyTab:
			if inLeft {
				currentLeft++
				if currentLeft >= len(leftItems) {
					inLeft = false
					app.SetFocus(recentList)
				} else {
					app.SetFocus(leftItems[currentLeft])
				}
			} else {
				inLeft = true
				currentLeft = 0
				app.SetFocus(leftItems[currentLeft])
			}
			return nil
		}
		return event
	})

	for i, item := range leftItems {
		idx := i
		if tv, ok := item.(*tview.TextView); ok {
			tv.SetFocusFunc(func() {
				currentLeft = idx
				inLeft = true
			})
		}
	}

	app.SetRoot(container, true).SetFocus(newBtn)
	if err := app.Run(); err != nil {
		return StartQuit
	}
	return result
}
