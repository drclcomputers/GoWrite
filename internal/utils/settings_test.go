package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	defaults.AutosaveMinutes = 12
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
	settings = DefaultSettings()
	settings.AutosaveMinutes = 16
	if err := settings.Validate(); err == nil {
		t.Fatal("out-of-range autosave interval was accepted")
	}
}

func TestSettingsDefaultAutosaveForOlderConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	data, err := json.Marshal(DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		t.Fatal(err)
	}
	delete(values, "autosave_minutes")
	data, err = json.Marshal(values)
	if err != nil {
		t.Fatal(err)
	}
	path, err := settingsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	settings, err := LoadSettings()
	if err != nil {
		t.Fatal(err)
	}
	if settings.AutosaveMinutes != DefaultSettings().AutosaveMinutes {
		t.Fatalf("autosave interval = %d, want default %d", settings.AutosaveMinutes, DefaultSettings().AutosaveMinutes)
	}
}

func TestRecoveryPathUsesGoWriteConfigDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path, err := RecoveryPath()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, ".gowrite", "~temp~.godoc")
	if path != want {
		t.Fatalf("recovery path = %q, want %q", path, want)
	}
}
