package editor

import (
	"testing"

	"gowrite/internal/services/document"
	"gowrite/internal/utils"
)

func TestApplyDocumentDefaults(t *testing.T) {
	settings := utils.DefaultSettings()
	settings.DefaultPageSize = "Letter"
	settings.DefaultOrientation = "Landscape"
	settings.MarginLeft = 8
	settings.MarginRight = 9
	settings.MarginTop = 3
	settings.MarginBottom = 4

	doc := document.New()
	applyDocumentDefaults(doc, settings)

	page := doc.GetPageSettings()
	width, height := document.GetDimensions(document.SizeLetter, document.Landscape)
	if page.Size != document.SizeLetter || page.Orientation != document.Landscape ||
		page.Width != width || page.Height != height ||
		page.MarginLeft != 8 || page.MarginRight != 9 ||
		page.MarginTop != 3 || page.MarginBottom != 4 {
		t.Fatalf("page settings = %+v, want Letter landscape with configured margins", page)
	}
}
