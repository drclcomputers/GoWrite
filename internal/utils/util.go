// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package utils

import (
	"os"
	"path/filepath"
	"strings"
)

const VER = "0.8.0"

// PrettyPath shortens paths under the home directory and separates components for display.
func PrettyPath(p, context string) string {
	full := filepath.ToSlash(filepath.Clean(p))
	home, err := os.UserHomeDir()
	if err == nil {
		home = filepath.ToSlash(filepath.Clean(home))
		switch {
		case full == home:
			full = "~"
		case strings.HasPrefix(full, home+"/"):
			full = strings.TrimPrefix(full, home+"/")
		}
	}

	if full == "~" || full == "/" || full == "." {
		return full
	}
	full = strings.TrimPrefix(full, "/")
	if full == "" {
		return "/"
	}
	parts := strings.Split(full, "/")
	if context == "recentfiles" && len(parts) > 1 {
		parts = parts[:len(parts)-1]
	}
	return strings.Join(parts, " > ")
}

// ConfigDir returns ~/.gowrite (creates it if needed).
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gowrite"
	}
	dir := filepath.Join(home, ".gowrite")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}
