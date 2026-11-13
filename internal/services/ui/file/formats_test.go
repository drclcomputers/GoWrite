package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gowrite/internal/services/document"
)

func TestFormatListsExposeSupportedOpenSaveAndExportTypes(t *testing.T) {
	extensions := func(formats []FormatOption) map[string]bool {
		result := make(map[string]bool, len(formats))
		for _, format := range formats {
			result[format.Extension] = true
		}
		return result
	}
	save, export, dialog, open := extensions(SaveFormats()), extensions(ExportFormats()),
		extensions(SaveExportFormats()), extensions(OpenFormats())

	for _, extension := range []string{".gdoc", ".docx", ".txt", ".rtf"} {
		if !save[extension] {
			t.Errorf("save formats omit %s", extension)
		}
	}
	for _, extension := range []string{".docx", ".txt", ".rtf", ".pdf", ".html", ".md"} {
		if !export[extension] {
			t.Errorf("export formats omit %s", extension)
		}
	}
	for _, extension := range []string{".gdoc", ".docx", ".txt", ".rtf", ".md", ".pdf", ".html"} {
		if !dialog[extension] {
			t.Errorf("save/export dialog formats omit %s", extension)
		}
	}
	for _, extension := range []string{".gdoc", ".docx", ".txt", ".md", ".rtf"} {
		if !open[extension] {
			t.Errorf("open formats omit %s", extension)
		}
	}
	if save[".pdf"] || save[".html"] || save[".md"] || open[".pdf"] || open[".html"] {
		t.Fatal("Markdown, PDF, and HTML should be export-only formats")
	}
}

func TestExportDocumentDispatchesPDFHTMLAndMarkdown(t *testing.T) {
	doc := document.New()
	doc.LoadText("export")
	state := doc.Snapshot()
	dir := t.TempDir()
	for _, test := range []struct {
		extension string
		signature string
	}{
		{extension: ".pdf", signature: "%PDF-"},
		{extension: ".html", signature: "<!doctype html>"},
		{extension: ".md", signature: "export"},
	} {
		filename := filepath.Join(dir, "export"+test.extension)
		if err := ExportDocument(filename, state); err != nil {
			t.Fatalf("export %s: %v", test.extension, err)
		}
		data, err := os.ReadFile(filename)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(data), test.signature) {
			t.Errorf("%s export has invalid signature", test.extension)
		}
	}
}
