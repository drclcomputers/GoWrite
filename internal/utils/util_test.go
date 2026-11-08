package utils

import (
	"path/filepath"
	"testing"
)

func TestPrettyPathFormatsBreadcrumbs(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	t.Setenv("HOME", home)

	tests := []struct {
		name    string
		path    string
		context string
		want    string
	}{
		{
			name: "home path",
			path: filepath.Join(home, "Documents", "Drafts"),
			want: "Documents > Drafts",
		},
		{
			name:    "recent file path",
			path:    filepath.Join(home, "Documents", "Drafts", "notes.txt"),
			context: "recentfiles",
			want:    "Documents > Drafts",
		},
		{
			name: "home directory",
			path: home,
			want: "~",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PrettyPath(tt.path, tt.context); got != tt.want {
				t.Errorf("PrettyPath(%q, %q) = %q, want %q", tt.path, tt.context, got, tt.want)
			}
		})
	}
}
