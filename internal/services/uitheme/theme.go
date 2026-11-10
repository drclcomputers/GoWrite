// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

// Package uitheme defines GoWrite's colors and shared widget styles.
package uitheme

import (
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	HexInk   = "#0e111a"
	HexField = "#0c0e16"
	HexPanel = "#121622"
)

const (
	HexEditorBackground = "#0e111a"
	HexPageBackground   = "#000000"
)

const (
	HexText  = "#e8ecf4"
	HexLabel = "#a8bcd6"
	HexMuted = "#8aa0b8"
	HexDim   = "#5c6b7a"
	HexWhite = "#ffffff"
)

const (
	HexLine      = "#46a0aa"
	HexAmber     = "#f0d090"
	HexAmberDim  = "#a08246"
	HexCyan      = "#5ec8c8"
	HexSky       = "#8ecae6"
	HexMint      = "#7dcea0"
	HexBtn       = "#2463a6"
	HexSelection = "#3e5c7a"
)

const (
	HexOk   = "#5ab48c"
	HexWarn = "#ff4a00"
)

const HexCustomStart = "#000000"

// Default colors for paragraph styles.
const (
	HexTitle    = HexAmber
	HexHeading1 = HexCyan
	HexHeading2 = HexSky
	HexCode     = HexSky
)

const (
	TagReset       = "[" + HexText + ":" + HexInk + "]"
	TagEditorReset = "[" + HexText + ":" + HexEditorBackground + "]"
	TagPageReset   = "[" + HexText + ":" + HexPageBackground + "]"
	TagCaret       = TagPageReset + "[" + HexInk + ":" + HexAmber + "]"
)

var (
	Ink              = ParseHex(HexInk)
	EditorBackground = ParseHex(HexEditorBackground)
	Field            = ParseHex(HexField)
	Panel            = ParseHex(HexPanel)
	Text             = ParseHex(HexText)
	Label            = ParseHex(HexLabel)
	Muted            = ParseHex(HexMuted)
	Line             = ParseHex(HexLine)
	Amber            = ParseHex(HexAmber)
	AmberDim         = ParseHex(HexAmberDim)
	Btn              = ParseHex(HexBtn)
	Ok               = ParseHex(HexOk)
	Warn             = ParseHex(HexWarn)
)

var (
	BtnText     = Text
	BtnNewText  = ParseHex(HexMint)
	BtnOpenText = ParseHex(HexSky)
)

func init() {
	tview.Styles = tview.Theme{
		PrimitiveBackgroundColor:    Ink,
		ContrastBackgroundColor:     Btn,
		MoreContrastBackgroundColor: Amber,
		BorderColor:                 Line,
		TitleColor:                  Amber,
		GraphicsColor:               Line,
		PrimaryTextColor:            Text,
		SecondaryTextColor:          Label,
		TertiaryTextColor:           Muted,
		InverseTextColor:            Text,
		ContrastSecondaryTextColor:  Text,
	}
}

// Markup replaces placeholders in constant UI templates with tview color tags.
func Markup(s string) string { return markup.Replace(s) }

var markup = strings.NewReplacer(
	"{key}", "["+HexCyan+"]",
	"{val}", "["+HexAmber+"]",
	"{ok}", "["+HexMint+"]",
	"{plain}", "["+HexText+"]",
	"{bright}", "["+HexWhite+"]",
	"{h1}", "["+HexAmber+"::b]",
	"{h2}", "["+HexMint+"::b]",
	"{h3}", "["+HexSky+"::b]",
	"{muted}", "["+HexMuted+"]",
	"{dim}", "["+HexDim+"]",
	"{version}", "["+HexAmber+"]",
	"{end}", "[::-]",
	"{clear}", "["+HexText+"]",
)

// Swatch is an entry in the document color picker.
type Swatch struct {
	Name  string
	Value string
}

// Palette contains the saved-document color picker swatches.
var Palette = []Swatch{
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
	for _, c := range Palette {
		if c.Value == val {
			return c.Name
		}
	}
	return val
}

// ValidHex reports whether s is in #RRGGBB form.
func ValidHex(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	_, err := strconv.ParseUint(s[1:], 16, 24)
	return err == nil
}

// ParseHex converts #RRGGBB to a tcell color, defaulting to Ink if invalid.
func ParseHex(s string) tcell.Color {
	if !ValidHex(s) {
		s = HexInk
	}
	v, _ := strconv.ParseUint(s[1:], 16, 24)
	return tcell.NewRGBColor(int32((v>>16)&0xff), int32((v>>8)&0xff), int32(v&0xff))
}

// Contrast selects a readable text color for c.
func Contrast(c tcell.Color) tcell.Color {
	r, g, b := c.RGB()
	lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	if lum > 140 {
		return Ink
	}
	return Text
}

type boxed interface {
	SetBorder(bool) *tview.Box
	SetTitle(string) *tview.Box
	SetTitleAlign(int) *tview.Box
	SetTitleColor(tcell.Color) *tview.Box
	SetBorderColor(tcell.Color) *tview.Box
	SetBackgroundColor(tcell.Color) *tview.Box
}

func activatedButton() tcell.Style {
	return tcell.StyleDefault.Background(Amber).Foreground(Ink)
}

// Paint applies the standard panel style.
func Paint(b boxed, title string) {
	b.SetBorder(true)
	if t := strings.TrimSpace(title); t != "" {
		b.SetTitle(" " + t + " ")
	}
	b.SetTitleAlign(tview.AlignCenter)
	b.SetTitleColor(Amber)
	b.SetBorderColor(Line)
	b.SetBackgroundColor(Panel)
}

// StyleForm applies the standard form style.
func StyleForm(f *tview.Form, title string) *tview.Form {
	Paint(f, title)
	f.SetLabelColor(Label)
	f.SetFieldBackgroundColor(Field)
	f.SetFieldTextColor(Text)
	f.SetButtonBackgroundColor(Btn)
	f.SetButtonTextColor(BtnText)
	f.SetButtonActivatedStyle(activatedButton())
	f.SetBackgroundColor(Panel)
	for i := 0; i < f.GetFormItemCount(); i++ {
		if item, ok := f.GetFormItem(i).(interface {
			SetBackgroundColor(tcell.Color) *tview.Box
		}); ok {
			item.SetBackgroundColor(Panel)
		}
	}
	return f
}

// StyleButton applies the standard button style.
func StyleButton(b *tview.Button) *tview.Button {
	b.SetBackgroundColor(Btn)
	b.SetLabelColor(BtnText)
	b.SetBackgroundColorActivated(Amber)
	b.SetLabelColorActivated(Ink)
	return b
}

// StyleModal applies the standard modal style.
func StyleModal(m *tview.Modal, title string) *tview.Modal {
	return tone(m, title, Btn)
}

func StyleModalWarn(m *tview.Modal, title string) *tview.Modal {
	return tone(m, title, Warn)
}

func StyleModalOK(m *tview.Modal, title string) *tview.Modal {
	return tone(m, title, Ok)
}

func tone(m *tview.Modal, title string, borderColor tcell.Color) *tview.Modal {
	m.SetBorder(true)
	if t := strings.TrimSpace(title); t != "" {
		m.SetTitle(" " + t + " ")
	}
	m.SetTitleAlign(tview.AlignCenter)
	m.SetTitleColor(Text)
	m.SetBorderColor(borderColor)
	m.SetBackgroundColor(Panel)
	m.SetTextColor(Text)
	m.SetButtonBackgroundColor(Btn)
	m.SetButtonTextColor(BtnText)
	m.SetButtonActivatedStyle(activatedButton())
	return m
}

// StyleList applies the standard list style.
func StyleList(l *tview.List, title string) *tview.List {
	Paint(l, title)
	l.SetSelectedBackgroundColor(Btn)
	l.SetSelectedTextColor(BtnText)
	l.SetMainTextStyle(tcell.StyleDefault.Foreground(Text).Background(Panel))
	l.SetSecondaryTextStyle(tcell.StyleDefault.Foreground(Label).Background(Panel))
	l.SetShortcutStyle(tcell.StyleDefault.Foreground(Label).Background(Panel))
	l.SetBackgroundColor(Panel)
	return l
}
