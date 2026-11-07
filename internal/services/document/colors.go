// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package document

// ColorPalette is the shared swatch list used by the fg/bg colour pickers.
var ColorPalette = []struct {
	Name  string
	Value string
}{
	{"Default", ""},
	{"Black", "#0e111a"},
	{"DkGray", "#3a4252"},
	{"Gray", "#8a94a6"},
	{"LtGray", "#b8c0cc"},
	{"White", "#e8ecf4"},
	{"DkRed", "#8b2f3a"},
	{"Red", "#e06c75"},
	{"Rose", "#e060a0"},
	{"Pink", "#f5a3c7"},
	{"Orange", "#d19a66"},
	{"Amber", "#e8a86b"},
	{"Yellow", "#f0d090"},
	{"Cream", "#f5e6c8"},
	{"DkGreen", "#2f6b4f"},
	{"Green", "#5ab48c"},
	{"Mint", "#a3e0c0"},
	{"Teal", "#2f8b94"},
	{"Cyan", "#56b6c2"},
	{"LtCyan", "#a8d4dc"},
	{"Navy", "#2a3a6a"},
	{"Blue", "#61afef"},
	{"Sky", "#a3c8f0"},
	{"DkPurple", "#5a3a7a"},
	{"Purple", "#c678dd"},
	{"Violet", "#e0a3f0"},
	{"Olive", "#a3a83a"},
	{"Brown", "#8b6a4a"},
	{"Gold", "#d4b04a"},
}

// ColorName returns a palette name for a colour value (or the value itself).
func ColorName(val string) string {
	if val == "" {
		return "Default"
	}
	for _, c := range ColorPalette {
		if c.Value == val {
			return c.Name
		}
	}
	return val
}
