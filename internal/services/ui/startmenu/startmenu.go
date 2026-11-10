// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package startmenu

import (
	"fmt"
	"gowrite/internal/services/uitheme"
	"gowrite/internal/utils"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const newDocButtonLabel = `┌────────────┐
│   ┌───╮    │
│   │   ╰┐   │
│   │ ── │   │
│   └────┘   │
└────────────┘
NEW DOCUMENT`

const openDocButtonLabel = `┌────────────┐
│ ▄▄▄        │
│ ┌────────┐ │
│ │ ────── │ │
│ └────────┘ │
└────────────┘
OPEN`

const optionsButtonLabel = `┌─────────────┐
│ ──●──────── │
│ ────────●── │
│ ───●─────── │
│ ────────●── │
└─────────────┘
OPTIONS`

// StartMenuResult is what the start menu returns.
type StartMenuResult int

const (
	StartNew StartMenuResult = iota
	StartOpen
	StartQuit
)

type StartMenu struct {
	View     tview.Primitive
	Focus    tview.Primitive
	onChoose func(StartMenuResult)
}

func (m *StartMenu) SetChooseHandler(handler func(StartMenuResult)) {
	m.onChoose = handler
}

func StartMenuUI(app *tview.Application) (StartMenuResult, *StartMenu) {
	result := StartQuit

	container := tview.NewFlex()

	leftPanel := tview.NewFlex().SetDirection(tview.FlexRow)

	title := tview.NewTextView().
		SetText(fmt.Sprintf("[::b]GoWrite[::-] [yellow]v%s[::-]", utils.VER)).
		SetTextAlign(tview.AlignCenter).
		SetTextStyle(tcell.StyleDefault.Background(uitheme.Panel)).
		SetDynamicColors(true)
	title.SetBorder(true).
		SetTitleColor(uitheme.Amber).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(uitheme.Panel)

	newBtn := tview.NewTextView().
		SetText(newDocButtonLabel).
		SetTextAlign(tview.AlignCenter).
		SetTextStyle(tcell.StyleDefault.Foreground(tcell.ColorLightGreen).Background(uitheme.Panel))
	newBtn.SetBorder(true).
		SetBorderColor(uitheme.Ok).
		SetBackgroundColor(uitheme.Panel)
	newBtn.SetScrollable(false)

	openBtn := tview.NewTextView().
		SetText(openDocButtonLabel).
		SetTextAlign(tview.AlignCenter).
		SetTextStyle(tcell.StyleDefault.Foreground(tcell.ColorBlue).Background(uitheme.Panel))
	openBtn.SetBorder(true).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(uitheme.Panel)
	openBtn.SetScrollable(false)

	optionsBtn := tview.NewTextView().
		SetText(optionsButtonLabel).
		SetTextAlign(tview.AlignCenter).
		SetTextStyle(tcell.StyleDefault.Foreground(uitheme.Amber).Background(uitheme.Panel))
	optionsBtn.SetBorder(true).
		SetBorderColor(tcell.NewRGBColor(160, 130, 70)).
		SetBackgroundColor(uitheme.Panel)
	optionsBtn.SetScrollable(false)
	optionsBtn.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			ShowOptions(app, container, optionsBtn)
			return nil
		}
		return event
	})

	leftPanel.
		AddItem(title, 0, 1, false).
		AddItem(newBtn, 0, 3, true).
		AddItem(openBtn, 0, 3, true).
		AddItem(optionsBtn, 0, 3, true)

	// Right side – placeholder for recent files later
	rightPanel := tview.NewFlex().SetDirection(tview.FlexRow)

	recentTitle := tview.NewTextView().
		SetText("[::b]Recent Files[::-]").
		SetTextAlign(tview.AlignCenter).
		SetTextStyle(tcell.StyleDefault.Background(uitheme.Panel)).
		SetDynamicColors(true)
	recentTitle.SetBorder(true).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(uitheme.Panel)

	recentList := tview.NewList()
	uitheme.StyleList(recentList, "")
	recentList.AddItem("No recent files", "", 0, nil)
	recentList.SetSelectedTextColor(tcell.ColorWhite)

	rightPanel.
		AddItem(recentTitle, 3, 0, false).
		AddItem(recentList, 0, 1, true)

	container.
		AddItem(leftPanel, 0, 1, true).
		AddItem(rightPanel, 0, 3, false)

	container.SetBorder(true).
		SetTitle(" GoWrite  ·  arrows or tab to move  ·  Enter to choose  ·  Esc/Q quits ").
		SetTitleColor(uitheme.Amber).
		SetBorderColor(uitheme.Line).
		SetBackgroundColor(uitheme.Panel)

	leftItems := []tview.Primitive{newBtn, openBtn, optionsBtn}
	currentLeft := 0
	inLeft := true

	var startMenu *StartMenu
	choose := func(r StartMenuResult) {
		result = r
		if startMenu != nil && startMenu.onChoose != nil {
			startMenu.onChoose(r)
		} else {
			app.Stop()
		}
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

	startMenu = &StartMenu{View: container, Focus: newBtn}
	app.SetRoot(container, true).SetFocus(newBtn)
	if err := app.Run(); err != nil {
		return StartQuit, startMenu
	}
	return result, startMenu
}
