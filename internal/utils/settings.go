// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Settings struct {
	DefaultPageSize     string `json:"default_page_size"`
	DefaultOrientation  string `json:"default_orientation"`
	MarginLeft          int    `json:"margin_left"`
	MarginRight         int    `json:"margin_right"`
	MarginTop           int    `json:"margin_top"`
	MarginBottom        int    `json:"margin_bottom"`
	DefaultSaveFormat   string `json:"default_save_format"`
	ShowFormattingMarks bool   `json:"show_formatting_marks"`
}

func DefaultSettings() Settings {
	return Settings{
		DefaultPageSize:     "A4",
		DefaultOrientation:  "Portrait",
		MarginLeft:          6,
		MarginRight:         6,
		MarginTop:           2,
		MarginBottom:        2,
		DefaultSaveFormat:   ".gdoc",
		ShowFormattingMarks: false,
	}
}

func LoadSettings() (Settings, error) {
	defaults := DefaultSettings()
	path, err := settingsPath()
	if err != nil {
		return defaults, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaults, nil
	}
	if err != nil {
		return defaults, fmt.Errorf("read settings: %w", err)
	}
	var settings Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return defaults, fmt.Errorf("parse settings: %w", err)
	}
	if err := settings.Validate(); err != nil {
		return defaults, fmt.Errorf("invalid settings: %w", err)
	}
	return settings, nil
}

func SaveSettings(settings Settings) error {
	if err := settings.Validate(); err != nil {
		return fmt.Errorf("invalid settings: %w", err)
	}
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create settings directory: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode settings: %w", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}
	return nil
}

func (s Settings) Validate() error {
	switch s.DefaultPageSize {
	case "A3", "A4", "A5", "Letter", "Legal":
	default:
		return fmt.Errorf("unsupported default page size %q", s.DefaultPageSize)
	}
	switch s.DefaultOrientation {
	case "Portrait", "Landscape":
	default:
		return fmt.Errorf("unsupported default orientation %q", s.DefaultOrientation)
	}
	for _, margin := range []struct {
		name  string
		value int
	}{
		{"left", s.MarginLeft},
		{"right", s.MarginRight},
		{"top", s.MarginTop},
		{"bottom", s.MarginBottom},
	} {
		if margin.value < 0 || margin.value > 100 {
			return fmt.Errorf("%s margin must be from 0 to 100 rows/columns", margin.name)
		}
	}
	switch strings.ToLower(s.DefaultSaveFormat) {
	case ".gdoc", ".docx", ".txt", ".rtf":
	default:
		return fmt.Errorf("unsupported default save format %q", s.DefaultSaveFormat)
	}
	return nil
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	return filepath.Join(home, ".gowrite", "settings.json"), nil
}
