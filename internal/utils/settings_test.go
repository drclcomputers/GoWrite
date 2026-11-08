package utils

import (
	"testing"
)

func TestSettingsDefaultAndPersistence(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	defaults, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if defaults != DefaultSettings() {
		t.Fatalf("defaults = %+v, want %+v", defaults, DefaultSettings())
	}

	defaults.DefaultPageSize = "Letter"
	defaults.DefaultOrientation = "Landscape"
	defaults.MarginLeft = 8
	defaults.DefaultSaveFormat = ".docx"
	defaults.ShowFormattingMarks = true
	if err := SaveSettings(defaults); err != nil {
		t.Fatal(err)
	}
	got, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if got != defaults {
		t.Fatalf("loaded settings = %+v, want %+v", got, defaults)
	}
}

func TestSettingsRejectInvalidValues(t *testing.T) {
	settings := DefaultSettings()
	settings.DefaultPageSize = "B7"
	if err := settings.Validate(); err == nil {
		t.Fatal("invalid page size was accepted")
	}
}
