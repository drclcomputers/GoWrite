// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package editor

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
