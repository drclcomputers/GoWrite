// Copyright (c) 2025 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package utils

import (
	"os"
	"path/filepath"
	"strings"
)

const VER = "0.1.0"

// PrettyPath shortens a path for display (home → ~).
func PrettyPath(p, context string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
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
