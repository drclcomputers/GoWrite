// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package file

import (
	"fmt"
	"path/filepath"
	"strings"
)

type FormatOption struct {
	Extension   string
	Description string
}

func DocumentFormats() []FormatOption {
	return []FormatOption{
		{Extension: ".gdoc", Description: "GoWrite native document"},
		{Extension: ".docx", Description: "Microsoft Word document"},
		{Extension: ".txt", Description: "Plain text"},
		{Extension: ".rtf", Description: "Rich text"},
	}
}

func Extension(formats []FormatOption, index int) string {
	if index < 0 || index >= len(formats) {
		return ".gdoc"
	}
	return formats[index].Extension
}

func FormatIndex(extension string, formats []FormatOption) int {
	for i, format := range formats {
		if strings.EqualFold(format.Extension, extension) {
			return i
		}
	}
	return 0
}

func WithExtension(filename, extension string, formats []FormatOption) string {
	filename = strings.TrimSpace(filename)
	ext := filepath.Ext(filename)
	if strings.EqualFold(ext, extension) {
		return filename
	}
	for _, format := range formats {
		if strings.EqualFold(ext, format.Extension) {
			filename = strings.TrimSuffix(filename, ext)
			break
		}
	}
	return filename + extension
}

func FormatSize(size int64) string {
	switch {
	case size < 1024:
		return fmt.Sprintf("%d B", size)
	case size < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(size)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
	}
}
