// Copyright (c) 2025 @drclcomputers. All rights reserved.
// Licensed under the MIT license. See https://opensource.org/licenses/MIT.

package document

// PageSize represents standard paper sizes.
type PageSize string

const (
	SizeA3     PageSize = "A3"
	SizeA4     PageSize = "A4"
	SizeA5     PageSize = "A5"
	SizeLetter PageSize = "Letter"
	SizeLegal  PageSize = "Legal"
)

// Orientation is either portrait or landscape.
type Orientation string

const (
	Portrait  Orientation = "Portrait"
	Landscape Orientation = "Landscape"
)

// PageSettings holds the dimensions of the on-screen paper.
type PageSettings struct {
	Size        PageSize
	Orientation Orientation
	Width       int // in terminal cells (excluding borders)
	Height      int // in terminal cells
}

// DefaultA4Portrait returns the default settings.
func DefaultA4Portrait() PageSettings {
	return PageSettings{
		Size:        SizeA4,
		Orientation: Portrait,
		Width:       80,
		Height:      45,
	}
}

// GetDimensions returns the width and height based on size and orientation.
func GetDimensions(size PageSize, orient Orientation) (int, int) {
	var w, h int
	switch size {
	case SizeA3:
		w, h = 120, 70
	case SizeA4:
		w, h = 80, 50
	case SizeA5:
		w, h = 60, 35
	case SizeLetter:
		w, h = 80, 50
	case SizeLegal:
		w, h = 80, 55
	default:
		w, h = 80, 50
	}

	if orient == Landscape {
		return h, w
	}
	return w, h
}
