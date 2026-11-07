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

// PageSettings holds the dimensions of the on-screen paper.
type PageSettings struct {
	Size        PageSize
	Orientation Orientation
	Width       int
	Height      int
}

// DefaultA4Portrait returns the default settings.
func DefaultA4Portrait() PageSettings {
	w, h := GetDimensions(SizeA4, Portrait)
	return PageSettings{
		Size:        SizeA4,
		Orientation: Portrait,
		Width:       w,
		Height:      h,
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
