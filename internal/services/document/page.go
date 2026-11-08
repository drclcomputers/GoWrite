// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

import "math"

// PageSize represents standard paper sizes.
type PageSize string

const (
	SizeA3     PageSize = "A3"
	SizeA4     PageSize = "A4"
	SizeA5     PageSize = "A5"
	SizeLetter PageSize = "Letter"
	SizeLegal  PageSize = "Legal"

	colsPerMM = 100.0 / 210.0
	rowsPerMM = 59.0 / 297.0
)

type Orientation string

const (
	Portrait  Orientation = "Portrait"
	Landscape Orientation = "Landscape"
)

// HeaderFooter is the text of a running header or footer. It may use the
// tokens {page}, {pages} and {title}.
type HeaderFooter struct {
	Left   string `json:"left"`
	Center string `json:"center"`
	Right  string `json:"right"`
}

func (h HeaderFooter) empty() bool {
	return h.Left == "" && h.Center == "" && h.Right == ""
}

// NumberWhere says where page numbers go.
type NumberWhere string

const (
	NumOff    NumberWhere = "Off"
	NumHeader NumberWhere = "Header"
	NumFooter NumberWhere = "Footer"
)

// NumberFormat is how a page number reads.
type NumberFormat string

const (
	FormatPlain  NumberFormat = "1"
	FormatPage   NumberFormat = "Page 1"
	FormatOf     NumberFormat = "1 of N"
	FormatPageOf NumberFormat = "Page 1 of N"
)

// PageNumbers places an automatic page number in the header or footer.
type PageNumbers struct {
	Where  NumberWhere  `json:"where"`
	Align  Align        `json:"alignment"`
	Format NumberFormat `json:"format"`
}

const (
	minBodyWidth = 10
	minBodyRows  = 3
)

// PageSettings holds the on-screen paper: size, margins, header and footer.
// Margins are in terminal cells (columns left/right, rows top/bottom).
type PageSettings struct {
	Size        PageSize    `json:"size"`
	Orientation Orientation `json:"orientation"`
	Width       int         `json:"width"`
	Height      int         `json:"height"`

	MarginLeft   int `json:"margin_left"`
	MarginRight  int `json:"margin_right"`
	MarginTop    int `json:"margin_top"`
	MarginBottom int `json:"margin_bottom"`

	Header      HeaderFooter `json:"header"`
	Footer      HeaderFooter `json:"footer"`
	PageNumbers PageNumbers  `json:"page_numbers"`
}

func (p PageSettings) hasHeader() bool {
	return !p.Header.empty() || p.PageNumbers.Where == NumHeader
}

func (p PageSettings) hasFooter() bool {
	return !p.Footer.empty() || p.PageNumbers.Where == NumFooter
}

// Normalized clamps the margins so the text area keeps a usable size.
func (p PageSettings) Normalized() PageSettings {
	p.MarginLeft, p.MarginRight = max(p.MarginLeft, 0), max(p.MarginRight, 0)
	p.MarginTop, p.MarginBottom = max(p.MarginTop, 0), max(p.MarginBottom, 0)

	if sum, room := p.MarginLeft+p.MarginRight, max(p.Width-minBodyWidth, 0); sum > room {
		p.MarginLeft, p.MarginRight = p.MarginLeft*room/sum, p.MarginRight*room/sum
	}

	chrome := 0
	if p.hasHeader() {
		chrome++
	}
	if p.hasFooter() {
		chrome++
	}
	if sum, room := p.MarginTop+p.MarginBottom, max(p.Height-chrome-minBodyRows, 0); sum > room {
		p.MarginTop, p.MarginBottom = p.MarginTop*room/sum, p.MarginBottom*room/sum
	}
	return p
}

// DefaultA4Portrait returns the default settings.
func DefaultA4Portrait() PageSettings {
	w, h := GetDimensions(SizeA4, Portrait)
	return PageSettings{
		Size:         SizeA4,
		Orientation:  Portrait,
		Width:        w,
		Height:       h,
		MarginLeft:   6,
		MarginRight:  6,
		MarginTop:    2,
		MarginBottom: 2,
		PageNumbers:  PageNumbers{Where: NumOff, Align: AlignCenter, Format: FormatPage},
	}
}

// GetDimensions returns the width and height based on size and orientation.
func GetDimensions(size PageSize, orient Orientation) (int, int) {
	var wmm, hmm float64
	switch size {
	case SizeA3:
		wmm, hmm = 297, 420
	case SizeA4:
		wmm, hmm = 210, 297
	case SizeA5:
		wmm, hmm = 148, 210
	case SizeLetter:
		wmm, hmm = 215.9, 279.4
	case SizeLegal:
		wmm, hmm = 215.9, 355.6
	default: // A4
		wmm, hmm = 210, 297
	}

	if orient == Landscape {
		wmm, hmm = hmm, wmm
	}

	return int(math.Round(wmm * colsPerMM)), int(math.Round(hmm * rowsPerMM))
}
