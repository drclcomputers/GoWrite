// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gowrite/internal/services/document"
	"gowrite/internal/services/fileop"
)

func SaveDocument(filename string, state document.State, showFormattingMarks bool) error {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".gdoc":
		return fileop.SaveGDoc(filename, state, showFormattingMarks)
	case ".docx":
		return fileop.SaveDOCX(filename, state)
	case ".txt":
		return fileop.ExportTXT(filename, state)
	case ".rtf":
		return fileop.ExportRTF(filename, state)
	default:
		return fmt.Errorf("unsupported save format %q", filepath.Ext(filename))
	}
}

func ExportDocument(filename string, state document.State) error {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".txt":
		return fileop.ExportTXT(filename, state)
	case ".rtf":
		return fileop.ExportRTF(filename, state)
	case ".docx":
		return fileop.ExportDOCX(filename, state)
	default:
		return fmt.Errorf("unsupported export format %q", filepath.Ext(filename))
	}
}

func LoadNative(filename string) (fileop.GDoc, error) {
	return fileop.LoadGDoc(filename)
}

func LoadWord(filename string) (document.State, error) {
	return fileop.LoadDOCX(filename)
}

func ReadText(filename string) (string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	return strings.ReplaceAll(text, "\r", "\n"), nil
}
