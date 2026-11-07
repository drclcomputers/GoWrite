// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package ui

import (
	"fmt"
	"gowrite/internal/services/uitheme"

	"github.com/rivo/tview"
)

// ShowErrorModal shows a styled error and returns to the given primitive.
func ShowErrorModal(app *tview.Application, returnTo tview.Primitive, message string) {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Error\n\n%s", message)).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			app.SetRoot(returnTo, true)
		})
	uitheme.StyleModalWarn(modal, "Error")
	app.SetRoot(modal, true).SetFocus(modal)
}
