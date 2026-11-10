// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

// Package uitheme is the single place where every colour used by GoWrite lives.
//
// To restyle the app, edit this file and nothing else. No other file should
// contain a colour literal (hex string, RGB triple, tcell.Color* constant or a
// named tview colour tag); they should reference the names defined here.
//
// The file is laid out top to bottom:
//
//  1. Palette          raw hex values, one line per colour (start here to retheme)
//  2. Roles            names for where palette colours are used in the document
//  3. tcell colours    the palette as tcell.Color, for widget setters
//  4. Markup           {key}-style shorthand for tview dynamic-colour tags
//  5. Document swatches the colours offered by the Text/Background colour picker
//  6. Helpers          hex parsing and contrast
//  7. Widget stylers   Paint, StyleForm, StyleModal*, StyleList, StyleButton
package uitheme

import (
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// ---------------------------------------------------------------------------
// 1. Palette
// ---------------------------------------------------------------------------

// Surfaces, darkest to lightest.
const (
	HexInk   = "#0e111a" // window and canvas background
	HexField = "#0c0e16" // text-input background
	HexPanel = "#121622" // dialog, form and list background
)

// Keep the editor surround and document page independently configurable.
const (
	HexEditorBackground = "#0e111a"
	HexPageBackground   = "#000000"
)

// Text.
const (
	HexText  = "#e8ecf4" // primary text
	HexLabel = "#a8bcd6" // form labels and secondary text
	HexMuted = "#8aa0b8" // hints
	HexDim   = "#5c6b7a" // footnotes
	HexWhite = "#ffffff" // pure white
)

// Accents.
const (
	HexLine      = "#46a0aa" // borders
	HexAmber     = "#f0d090" // titles, values, hotkeys, text cursor, active border
	HexAmberDim  = "#a08246" // quieter amber border
	HexCyan      = "#5ec8c8" // top-bar labels, Heading 1
	HexSky       = "#8ecae6" // Heading 2, code, help headings
	HexMint      = "#7dcea0" // status-bar label, help headings
	HexBtn       = "#2463a6" // button and selected-row background
	HexSelection = "#3e5c7a" // selected text in the document
)

// Feedback.
const (
	HexOk   = "#5ab48c" // success
	HexWarn = "#ff4a00" // warning
)

// Starting value of the colour picker's "Custom..." form.
const HexCustomStart = "#000000"

// ---------------------------------------------------------------------------
// 2. Roles
// ---------------------------------------------------------------------------

// Default colour of each paragraph style when a run has none of its own.
// The on-screen renderer and the RTF writer both read these, so they always agree.
const (
	HexTitle    = HexAmber
	HexHeading1 = HexCyan
	HexHeading2 = HexSky
	HexCode     = HexSky
)

// TagCaret draws the text cursor using the page background and amber colors.
const (
	TagReset       = "[" + HexText + ":" + HexInk + "]"
	TagEditorReset = "[" + HexText + ":" + HexEditorBackground + "]"
	TagPageReset   = "[" + HexText + ":" + HexPageBackground + "]"
	TagCaret       = TagPageReset + "[" + HexInk + ":" + HexAmber + "]"
)

// ---------------------------------------------------------------------------
// 3. tcell colours
// ---------------------------------------------------------------------------

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

// These use the terminal's own ANSI palette instead of RGB, so they follow the
// user's terminal theme. Swap them for ParseHex(...) to pin them to the palette.
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

// ---------------------------------------------------------------------------
// 4. Markup
// ---------------------------------------------------------------------------

// Markup expands the {name} shorthands below into tview colour tags, so
// strings never hold a colour themselves:
//
//	uitheme.Markup("{key}Ln{end} {val}%d{end}")
//
// Call it on constant templates only, never on user text.
//
//	{key}     label                  {h1}  help heading, amber
//	{val}     value / hotkey         {h2}  help heading, mint
//	{ok}      success label          {h3}  help heading, sky
//	{plain}   primary text            {muted} hint     {dim} footnote
//	{bright}  pure white             {version} version number
//	{end}     reset attributes       {clear} reset colour
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

// ---------------------------------------------------------------------------
// 5. Document swatches
// ---------------------------------------------------------------------------

// Swatch is one entry in the colour picker.
type Swatch struct {
	Name  string
	Value string
}

// Palette lists the swatches offered by the Text and Background colour pickers.
//
// These values are written into saved documents, so they are deliberately
// literal and not derived from the UI palette above: restyling the app must not
// change what an existing document contains. Add new swatches freely, but do
// not edit existing values.
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

// ---------------------------------------------------------------------------
// 6. Helpers
// ---------------------------------------------------------------------------

// ValidHex reports whether s is a colour in #RRGGBB form.
func ValidHex(s string) bool {
	if len(s) != 7 || s[0] != '#' {
		return false
	}
	_, err := strconv.ParseUint(s[1:], 16, 24)
	return err == nil
}

// ParseHex converts #RRGGBB to a tcell colour, falling back to Ink if s is invalid.
func ParseHex(s string) tcell.Color {
	if !ValidHex(s) {
		s = HexInk
	}
	v, _ := strconv.ParseUint(s[1:], 16, 24)
	return tcell.NewRGBColor(int32((v>>16)&0xff), int32((v>>8)&0xff), int32(v&0xff))
}

// Contrast returns Ink or Text, whichever is easier to read on c.
func Contrast(c tcell.Color) tcell.Color {
	r, g, b := c.RGB()
	lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	if lum > 140 {
		return Ink
	}
	return Text
}

// ---------------------------------------------------------------------------
// 7. Widget stylers
// ---------------------------------------------------------------------------

type boxed interface {
	SetBorder(bool) *tview.Box
	SetTitle(string) *tview.Box
	SetTitleAlign(int) *tview.Box
	SetTitleColor(tcell.Color) *tview.Box
	SetBorderColor(tcell.Color) *tview.Box
	SetBackgroundColor(tcell.Color) *tview.Box
}

// activatedButton is how a focused button looks.
func activatedButton() tcell.Style {
	return tcell.StyleDefault.Background(Amber).Foreground(Ink)
}

// Paint gives a panel the standard border and title.
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

// StyleForm paints a form and its fields and buttons.
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

// StyleButton paints a standalone button like the buttons inside a form.
func StyleButton(b *tview.Button) *tview.Button {
	b.SetBackgroundColor(Btn)
	b.SetLabelColor(BtnText)
	b.SetBackgroundColorActivated(Amber)
	b.SetLabelColorActivated(Ink)
	return b
}

// StyleModal paints a message on a dark background; warnings get a rust border, successes green.
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

// StyleList paints a list the same way as a form.
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
