// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package fileop

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"math"
	"path"
	"strconv"
	"strings"

	"gowrite/internal/services/document"
)

const (
	wordNS    = "http://schemas.openxmlformats.org/wordprocessingml/2006/main"
	officeRel = "http://schemas.openxmlformats.org/officeDocument/2006/relationships"
	docxLimit = 64 << 20
)

const (
	rowTwips   = 1440 * 297 / 25.4 / 59
	columnTwip = 1440 * 210 / 25.4 / 100
)

var docxHighlightColors = map[string]string{
	"black": "#000000", "blue": "#0000ff", "cyan": "#00ffff",
	"darkblue": "#00008b", "darkcyan": "#008b8b", "darkgray": "#a9a9a9",
	"darkgreen": "#006400", "darkmagenta": "#8b008b", "darkred": "#8b0000",
	"darkyellow": "#9b870c", "green": "#008000", "lightgray": "#d3d3d3",
	"magenta": "#ff00ff", "red": "#ff0000", "white": "#ffffff", "yellow": "#ffff00",
}

func SaveDOCX(filename string, state document.State) error {
	return writeAtomic(filename, func(w io.Writer) error {
		return writeDOCX(w, state)
	})
}

func ExportDOCX(filename string, state document.State) error {
	return SaveDOCX(filename, state)
}

func writeDOCX(w io.Writer, state document.State) error {
	page := state.Page.Normalized()
	needHeader := page.Header.Left != "" || page.Header.Center != "" || page.Header.Right != "" || page.PageNumbers.Where == document.NumHeader
	needFooter := page.Footer.Left != "" || page.Footer.Center != "" || page.Footer.Right != "" || page.PageNumbers.Where == document.NumFooter
	headerID, footerID := "", ""
	if needHeader {
		headerID = "rId2"
	}
	if needFooter {
		footerID = "rId3"
	}

	zw := zip.NewWriter(w)
	files := []struct {
		name string
		data string
	}{
		{"[Content_Types].xml", docxContentTypes(needHeader, needFooter)},
		{"_rels/.rels", docxRootRelationships()},
		{"word/document.xml", docxDocumentXML(state, headerID, footerID)},
		{"word/styles.xml", docxStylesXML()},
		{"word/_rels/document.xml.rels", docxDocumentRelationships(headerID, footerID)},
		{"docProps/core.xml", docxCoreProperties(state.Title)},
	}
	if needHeader {
		files = append(files, struct {
			name string
			data string
		}{"word/header1.xml", docxHeaderFooterXML(true, page.Header, page.PageNumbers, page.PageNumbers.Where == document.NumHeader, state.Title, page)})
	}
	if needFooter {
		files = append(files, struct {
			name string
			data string
		}{"word/footer1.xml", docxHeaderFooterXML(false, page.Footer, page.PageNumbers, page.PageNumbers.Where == document.NumFooter, state.Title, page)})
	}
	for _, file := range files {
		entry, err := zw.Create(file.name)
		if err != nil {
			_ = zw.Close()
			return fmt.Errorf("create DOCX part %s: %w", file.name, err)
		}
		if _, err := io.WriteString(entry, file.data); err != nil {
			_ = zw.Close()
			return fmt.Errorf("write DOCX part %s: %w", file.name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return fmt.Errorf("finish DOCX archive: %w", err)
	}
	return nil
}

func docxContentTypes(header, footer bool) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">`)
	b.WriteString(`<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>`)
	b.WriteString(`<Default Extension="xml" ContentType="application/xml"/>`)
	b.WriteString(`<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>`)
	b.WriteString(`<Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>`)
	b.WriteString(`<Override PartName="/docProps/core.xml" ContentType="application/vnd.openxmlformats-package.core-properties+xml"/>`)
	if header {
		b.WriteString(`<Override PartName="/word/header1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.header+xml"/>`)
	}
	if footer {
		b.WriteString(`<Override PartName="/word/footer1.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.footer+xml"/>`)
	}
	b.WriteString(`</Types>`)
	return b.String()
}

func docxRootRelationships() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/><Relationship Id="rId2" Type="http://schemas.openxmlformats.org/package/2006/relationships/metadata/core-properties" Target="docProps/core.xml"/></Relationships>`
}

func docxDocumentRelationships(header, footer string) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">`)
	b.WriteString(`<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/styles" Target="styles.xml"/>`)
	if header != "" {
		b.WriteString(`<Relationship Id="rId2" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/header" Target="header1.xml"/>`)
	}
	if footer != "" {
		b.WriteString(`<Relationship Id="rId3" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/footer" Target="footer1.xml"/>`)
	}
	b.WriteString(`</Relationships>`)
	return b.String()
}

func docxCoreProperties(title string) string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><cp:coreProperties xmlns:cp="http://schemas.openxmlformats.org/package/2006/metadata/core-properties" xmlns:dc="http://purl.org/dc/elements/1.1/" xmlns:dcterms="http://purl.org/dc/terms/" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"><dc:title>` +
		xmlText(title) + `</dc:title></cp:coreProperties>`
}

func docxStylesXML() string {
	return `<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:styles xmlns:w="` + wordNS + `">` +
		`<w:docDefaults><w:rPrDefault><w:rPr><w:lang w:val="en-US"/></w:rPr></w:rPrDefault></w:docDefaults>` +
		`<w:style w:type="paragraph" w:default="1" w:styleId="Normal"><w:name w:val="Normal"/></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Title"><w:name w:val="Title"/><w:basedOn w:val="Normal"/><w:pPr><w:keepNext/></w:pPr><w:rPr><w:b/><w:sz w:val="36"/></w:rPr></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Heading1"><w:name w:val="heading 1"/><w:basedOn w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:b/><w:sz w:val="32"/></w:rPr></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Heading2"><w:name w:val="heading 2"/><w:basedOn w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:outlineLvl w:val="1"/></w:pPr><w:rPr><w:b/><w:sz w:val="26"/></w:rPr></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Heading3"><w:name w:val="heading 3"/><w:basedOn w:val="Normal"/><w:qFormat/><w:pPr><w:keepNext/><w:outlineLvl w:val="2"/></w:pPr><w:rPr><w:b/><w:sz w:val="24"/></w:rPr></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Quote"><w:name w:val="Quote"/><w:basedOn w:val="Normal"/></w:style>` +
		`<w:style w:type="paragraph" w:styleId="Code"><w:name w:val="Code"/><w:basedOn w:val="Normal"/></w:style>` +
		`</w:styles>`
}

func docxDocumentXML(state document.State, headerID, footerID string) string {
	page := state.Page.Normalized()
	widthMM, heightMM := rtfPageDimensions(page)
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>`)
	b.WriteString(`<w:document xmlns:w="` + wordNS + `" xmlns:r="` + officeRel + `"><w:body>`)
	for _, paragraph := range state.Paragraphs {
		if paragraph.TablePart {
			continue
		}
		if paragraph.Table != nil {
			writeDOCXTable(&b, *paragraph.Table, page)
			continue
		}
		writeDOCXParagraph(&b, paragraph)
	}
	fmt.Fprintf(&b, `<w:sectPr>`)
	if headerID != "" {
		fmt.Fprintf(&b, `<w:headerReference w:type="default" r:id="%s"/>`, headerID)
	}
	if footerID != "" {
		fmt.Fprintf(&b, `<w:footerReference w:type="default" r:id="%s"/>`, footerID)
	}
	fmt.Fprintf(&b, `<w:pgSz w:w="%d" w:h="%d"`, mmToTwips(widthMM), mmToTwips(heightMM))
	if page.Orientation == document.Landscape {
		b.WriteString(` w:orient="landscape"`)
	}
	b.WriteString(`/>`)
	fmt.Fprintf(&b, `<w:pgMar w:top="%d" w:right="%d" w:bottom="%d" w:left="%d" w:header="720" w:footer="720" w:gutter="0"/>`,
		mmToTwips(float64(page.MarginTop)*297/59),
		mmToTwips(float64(page.MarginRight)*210/100),
		mmToTwips(float64(page.MarginBottom)*297/59),
		mmToTwips(float64(page.MarginLeft)*210/100))
	b.WriteString(`</w:sectPr></w:body></w:document>`)
	return b.String()
}

func writeDOCXTable(b *strings.Builder, table document.Table, page document.PageSettings) {
	page = page.Normalized()
	widthMM, _ := rtfPageDimensions(page)
	totalWidth := mmToTwips(widthMM) -
		mmToTwips(float64(page.MarginLeft)*210/100) -
		mmToTwips(float64(page.MarginRight)*210/100)
	columns := len(table.Rows[0])
	columnWidth := max(totalWidth/columns, 1)

	b.WriteString(`<w:tbl><w:tblPr><w:tblBorders>`)
	for _, edge := range []string{"top", "left", "bottom", "right", "insideH", "insideV"} {
		fmt.Fprintf(b, `<w:%s w:val="single" w:sz="4" w:space="0" w:color="808080"/>`, edge)
	}
	b.WriteString(`</w:tblBorders></w:tblPr><w:tblGrid>`)
	for range columns {
		fmt.Fprintf(b, `<w:gridCol w:w="%d"/>`, columnWidth)
	}
	b.WriteString(`</w:tblGrid>`)
	for ri, row := range table.Rows {
		b.WriteString(`<w:tr>`)
		for ci, cell := range row {
			format := document.TableCellFormat{}
			if ri < len(table.CellFormats) && ci < len(table.CellFormats[ri]) {
				format = table.CellFormats[ri][ci]
			}
			fmt.Fprintf(b, `<w:tc><w:tcPr><w:tcW w:w="%d" w:type="dxa"/></w:tcPr>`, columnWidth)
			writeDOCXParagraph(b, document.Paragraph{
				Align:     format.Alignment,
				ParaStyle: format.ParagraphStyle,
				Runs: []document.Run{{
					Text:  cell,
					Style: format.Style,
					Fg:    format.Foreground,
					Bg:    format.Background,
				}},
			})
			b.WriteString(`</w:tc>`)
		}
		b.WriteString(`</w:tr>`)
	}
	b.WriteString(`</w:tbl>`)
}

func writeDOCXParagraph(b *strings.Builder, paragraph document.Paragraph) {
	b.WriteString(`<w:p><w:pPr>`)
	if style := docxParagraphStyle(paragraph.ParaStyle); style != "" {
		fmt.Fprintf(b, `<w:pStyle w:val="%s"/>`, style)
	}
	if paragraph.PageBreak {
		b.WriteString(`<w:pageBreakBefore/>`)
	}
	if paragraph.SpaceAfter > 0 {
		fmt.Fprintf(b, `<w:spacing w:after="%d"/>`, rowsToTwips(paragraph.SpaceAfter))
	}
	if paragraph.ParaStyle == document.ParagraphQuote {
		b.WriteString(`<w:ind w:left="360" w:right="360"/>`)
	}
	switch paragraph.Align {
	case document.AlignCenter:
		b.WriteString(`<w:jc w:val="center"/>`)
	case document.AlignRight:
		b.WriteString(`<w:jc w:val="right"/>`)
	}
	b.WriteString(`</w:pPr>`)
	for _, run := range paragraph.Runs {
		writeDOCXRun(b, run, paragraph.ParaStyle)
	}
	b.WriteString(`</w:p>`)
}

func docxParagraphStyle(style document.ParagraphStyle) string {
	switch style {
	case document.ParagraphTitle:
		return "Title"
	case document.ParagraphHeading1:
		return "Heading1"
	case document.ParagraphHeading2:
		return "Heading2"
	case document.ParagraphHeading3:
		return "Heading3"
	case document.ParagraphQuote:
		return "Quote"
	case document.ParagraphCode:
		return "Code"
	default:
		return ""
	}
}

func writeDOCXRun(b *strings.Builder, run document.Run, paragraphStyle document.ParagraphStyle) {
	style, fg, _, font := effectiveRTFRun(paragraphStyle, run)
	b.WriteString(`<w:r>`)
	if style != document.StyleNone || fg != "" || run.Bg != "" || font != 0 {
		b.WriteString(`<w:rPr>`)
		if font == 1 {
			b.WriteString(`<w:rFonts w:ascii="Courier New" w:hAnsi="Courier New"/>`)
		}
		if style&document.StyleBold != 0 {
			b.WriteString(`<w:b/>`)
		}
		if style&document.StyleItalic != 0 {
			b.WriteString(`<w:i/>`)
		}
		if style&document.StyleUnderline != 0 {
			b.WriteString(`<w:u w:val="single"/>`)
		}
		if style&document.StyleStrike != 0 {
			b.WriteString(`<w:strike/>`)
		}
		if color, ok := docxColor(fg); ok {
			fmt.Fprintf(b, `<w:color w:val="%s"/>`, color)
		}
		if color, ok := docxColor(run.Bg); ok {
			fmt.Fprintf(b, `<w:shd w:val="clear" w:color="auto" w:fill="%s"/>`, color)
		}
		b.WriteString(`</w:rPr>`)
	}
	writeDOCXText(b, run.Text)
	b.WriteString(`</w:r>`)
}

func writeDOCXText(b *strings.Builder, text string) {
	for len(text) > 0 {
		index := strings.IndexAny(text, "\t\n\r")
		if index < 0 {
			fmt.Fprintf(b, `<w:t xml:space="preserve">%s</w:t>`, xmlText(text))
			return
		}
		if index > 0 {
			fmt.Fprintf(b, `<w:t xml:space="preserve">%s</w:t>`, xmlText(text[:index]))
		}
		switch text[index] {
		case '\t':
			b.WriteString(`<w:tab/>`)
		case '\n':
			b.WriteString(`<w:br/>`)
		}
		text = text[index+1:]
		if len(text) == 0 {
			return
		}
	}
}

func docxHeaderFooterXML(isHeader bool, value document.HeaderFooter, numbers document.PageNumbers, showNumber bool, title string, page document.PageSettings) string {
	slots := []string{value.Left, value.Center, value.Right}
	if showNumber {
		index := 0
		switch numbers.Align {
		case document.AlignCenter:
			index = 1
		case document.AlignRight:
			index = 2
		}
		number := docxPageNumber(numbers.Format)
		if !strings.Contains(slots[index], "{page}") {
			if slots[index] != "" {
				slots[index] += " "
			}
			slots[index] += number
		}
	}
	widthMM, _ := rtfPageDimensions(page)
	contentWidth := mmToTwips(widthMM) -
		mmToTwips(float64(page.MarginLeft)*210/100) -
		mmToTwips(float64(page.MarginRight)*210/100)
	var b strings.Builder
	if isHeader {
		b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:hdr xmlns:w="` + wordNS + `">`)
	} else {
		b.WriteString(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?><w:ftr xmlns:w="` + wordNS + `">`)
	}
	fmt.Fprintf(&b, `<w:p><w:pPr><w:tabs><w:tab w:val="center" w:pos="%d"/><w:tab w:val="right" w:pos="%d"/></w:tabs></w:pPr>`, contentWidth/2, contentWidth)
	writeDOCXHeaderText(&b, slots[0], title)
	b.WriteString(`<w:r><w:tab/></w:r>`)
	writeDOCXHeaderText(&b, slots[1], title)
	b.WriteString(`<w:r><w:tab/></w:r>`)
	writeDOCXHeaderText(&b, slots[2], title)
	b.WriteString(`</w:p>`)
	if !isHeader {
		b.WriteString(`</w:ftr>`)
	} else {
		b.WriteString(`</w:hdr>`)
	}
	return b.String()
}

func docxPageNumber(format document.NumberFormat) string {
	switch format {
	case document.FormatPage:
		return "Page {page}"
	case document.FormatOf:
		return "{page} of {pages}"
	case document.FormatPageOf:
		return "Page {page} of {pages}"
	default:
		return "{page}"
	}
}

func writeDOCXHeaderText(b *strings.Builder, text, title string) {
	for len(text) > 0 {
		tokenAt, token := len(text), ""
		for _, candidate := range []string{"{page}", "{pages}", "{title}"} {
			if i := strings.Index(text, candidate); i >= 0 && i < tokenAt {
				tokenAt, token = i, candidate
			}
		}
		writeDOCXRunText(b, text[:tokenAt])
		if token == "" {
			break
		}
		switch token {
		case "{page}":
			writeDOCXField(b, "PAGE")
		case "{pages}":
			writeDOCXField(b, "NUMPAGES")
		case "{title}":
			writeDOCXRunText(b, title)
		}
		text = text[tokenAt+len(token):]
	}
}

func writeDOCXRunText(b *strings.Builder, text string) {
	if text == "" {
		return
	}
	b.WriteString(`<w:r><w:t xml:space="preserve">`)
	b.WriteString(xmlText(text))
	b.WriteString(`</w:t></w:r>`)
}

func writeDOCXField(b *strings.Builder, instruction string) {
	fmt.Fprintf(b, `<w:fldSimple w:instr="%s"><w:r><w:t>1</w:t></w:r></w:fldSimple>`, instruction)
}

func docxColor(value string) (string, bool) {
	if len(value) != 7 || value[0] != '#' {
		return "", false
	}
	if _, err := strconv.ParseUint(value[1:], 16, 24); err != nil {
		return "", false
	}
	return strings.ToUpper(value[1:]), true
}

func xmlText(value string) string {
	return html.EscapeString(value)
}

func LoadDOCX(filename string) (document.State, error) {
	var state document.State
	archive, err := zip.OpenReader(filename)
	if err != nil {
		return state, fmt.Errorf("open DOCX archive: %w", err)
	}
	defer archive.Close()

	parts := make(map[string]*zip.File, len(archive.File))
	for _, file := range archive.File {
		parts[file.Name] = file
	}
	documentXML, err := readDOCXPart(parts, "word/document.xml")
	if err != nil {
		return state, err
	}
	parsed, err := parseDOCXDocument(documentXML)
	if err != nil {
		return state, fmt.Errorf("parse DOCX document: %w", err)
	}
	state = document.State{
		Title:      "",
		Paragraphs: parsed.paragraphs,
		Page:       parsed.page,
	}
	if len(state.Paragraphs) == 0 {
		state.Paragraphs = []document.Paragraph{{Runs: []document.Run{{Text: ""}}}}
	}

	if _, exists := parts["docProps/core.xml"]; exists {
		core, err := readDOCXPart(parts, "docProps/core.xml")
		if err != nil {
			return document.State{}, err
		}
		state.Title, err = parseDOCXTitle(core)
		if err != nil {
			return document.State{}, fmt.Errorf("parse DOCX properties: %w", err)
		}
	}
	if _, exists := parts["word/_rels/document.xml.rels"]; exists {
		relationships, err := readDOCXPart(parts, "word/_rels/document.xml.rels")
		if err != nil {
			return document.State{}, err
		}
		rels, parseErr := parseDOCXRelationships(relationships)
		if parseErr != nil {
			return document.State{}, fmt.Errorf("parse DOCX relationships: %w", parseErr)
		}
		if parsed.headerID != "" {
			target := rels[parsed.headerID]
			if target == "" {
				return document.State{}, fmt.Errorf("DOCX header relationship %q is missing", parsed.headerID)
			}
			headerPath, pathErr := docxRelationshipPath(target)
			if pathErr != nil {
				return document.State{}, pathErr
			}
			headerData, readErr := readDOCXPart(parts, headerPath)
			if readErr != nil {
				return document.State{}, readErr
			}
			state.Page.Header, err = parseDOCXHeaderFooter(headerData)
			if err != nil {
				return document.State{}, fmt.Errorf("parse DOCX header: %w", err)
			}
		}
		if parsed.footerID != "" {
			target := rels[parsed.footerID]
			if target == "" {
				return document.State{}, fmt.Errorf("DOCX footer relationship %q is missing", parsed.footerID)
			}
			footerPath, pathErr := docxRelationshipPath(target)
			if pathErr != nil {
				return document.State{}, pathErr
			}
			footerData, readErr := readDOCXPart(parts, footerPath)
			if readErr != nil {
				return document.State{}, readErr
			}
			state.Page.Footer, err = parseDOCXHeaderFooter(footerData)
			if err != nil {
				return document.State{}, fmt.Errorf("parse DOCX footer: %w", err)
			}
		}
	} else if parsed.headerID != "" || parsed.footerID != "" {
		return document.State{}, fmt.Errorf("DOCX references headers or footers but has no document relationships")
	}
	inferDOCXPageNumbers(&state.Page)
	return state, nil
}

func plainDOCXParagraph(paragraph document.Paragraph) string {
	var text strings.Builder
	for _, run := range paragraph.Runs {
		text.WriteString(run.Text)
	}
	return text.String()
}

func readDOCXPart(parts map[string]*zip.File, name string) ([]byte, error) {
	file := parts[name]
	if file == nil {
		return nil, fmt.Errorf("DOCX is missing %s", name)
	}
	if file.UncompressedSize64 > docxLimit {
		return nil, fmt.Errorf("DOCX part %s is too large", name)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open DOCX part %s: %w", name, err)
	}
	data, readErr := io.ReadAll(io.LimitReader(reader, docxLimit+1))
	closeErr := reader.Close()
	if readErr != nil {
		return nil, fmt.Errorf("read DOCX part %s: %w", name, readErr)
	}
	if closeErr != nil {
		return nil, fmt.Errorf("close DOCX part %s: %w", name, closeErr)
	}
	if len(data) > docxLimit {
		return nil, fmt.Errorf("DOCX part %s is too large", name)
	}
	return data, nil
}

type parsedDOCXDocument struct {
	paragraphs []document.Paragraph
	page       document.PageSettings
	headerID   string
	footerID   string
}

func parseDOCXDocument(data []byte) (parsedDOCXDocument, error) {
	result := parsedDOCXDocument{page: document.DefaultA4Portrait()}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	var paragraph *document.Paragraph
	var run *document.Run
	inPPr, inRPr, inText := false, false, false
	inSection := false
	var width, height int
	orientation := document.Portrait
	marginTop, marginRight, marginBottom, marginLeft := -1, -1, -1, -1
	tableDepth := 0
	var table document.Table
	var tableRow []string
	var tableRowFormats []document.TableCellFormat
	var cellParagraphs []document.Paragraph
	inTableCell := false

	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return result, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			name := value.Name.Local
			switch name {
			case "tbl":
				if tableDepth == 0 {
					table = document.Table{}
				}
				tableDepth++
			case "tr":
				if tableDepth == 1 {
					tableRow = nil
					tableRowFormats = nil
				}
			case "tc":
				if tableDepth == 1 {
					inTableCell = true
					cellParagraphs = nil
				}
			case "p":
				paragraph = &document.Paragraph{Align: document.AlignLeft}
			case "r":
				if paragraph != nil {
					run = &document.Run{}
				}
			case "pPr":
				inPPr = true
			case "rPr":
				inRPr = true
			case "t", "delText":
				inText = true
			case "pStyle":
				if paragraph != nil && inPPr {
					if style := attrValue(value.Attr, "val"); style != "" {
						paragraph.ParaStyle = parseDOCXParagraphStyle(style)
					}
				}
			case "jc":
				if paragraph != nil && inPPr {
					switch attrValue(value.Attr, "val") {
					case "center":
						paragraph.Align = document.AlignCenter
					case "right":
						paragraph.Align = document.AlignRight
					}
				}
			case "spacing":
				if paragraph != nil && inPPr {
					if after, err := strconv.Atoi(attrValue(value.Attr, "after")); err == nil && after > 0 {
						paragraph.SpaceAfter = min(10, max(0, int(math.Round(float64(after)/rowTwips))))
					}
				}
			case "pageBreakBefore":
				if paragraph != nil && inPPr && wordBoolEnabled(attrValue(value.Attr, "val")) {
					paragraph.PageBreak = true
				}
			case "b":
				if run != nil && inRPr && wordBoolEnabled(attrValue(value.Attr, "val")) {
					run.Style |= document.StyleBold
				}
			case "i":
				if run != nil && inRPr && wordBoolEnabled(attrValue(value.Attr, "val")) {
					run.Style |= document.StyleItalic
				}
			case "u":
				if run != nil && inRPr && attrValue(value.Attr, "val") != "none" {
					run.Style |= document.StyleUnderline
				}
			case "strike", "dstrike":
				if run != nil && inRPr && wordBoolEnabled(attrValue(value.Attr, "val")) {
					run.Style |= document.StyleStrike
				}
			case "color":
				if run != nil && inRPr {
					if color := attrValue(value.Attr, "val"); len(color) == 6 {
						run.Fg = "#" + strings.ToLower(color)
					}
				}
			case "shd":
				if run != nil && inRPr {
					if color := attrValue(value.Attr, "fill"); len(color) == 6 && color != "auto" {
						run.Bg = "#" + strings.ToLower(color)
					}
				}
			case "highlight":
				if run != nil && inRPr {
					if color := docxHighlightColor(attrValue(value.Attr, "val")); color != "" {
						run.Bg = color
					}
				}
			case "tab":
				if run != nil {
					run.Text += "\t"
				}
			case "br", "cr":
				if run != nil {
					if attrValue(value.Attr, "type") == "page" && paragraph != nil {
						paragraph.PageBreak = true
					} else {
						run.Text += "\n"
					}
				}
			case "sectPr":
				inSection = true
			case "pgSz":
				if inSection {
					width, _ = strconv.Atoi(attrValue(value.Attr, "w"))
					height, _ = strconv.Atoi(attrValue(value.Attr, "h"))
					if attrValue(value.Attr, "orient") == "landscape" {
						orientation = document.Landscape
					}
				}
			case "pgMar":
				if inSection {
					marginTop, _ = strconv.Atoi(attrValue(value.Attr, "top"))
					marginRight, _ = strconv.Atoi(attrValue(value.Attr, "right"))
					marginBottom, _ = strconv.Atoi(attrValue(value.Attr, "bottom"))
					marginLeft, _ = strconv.Atoi(attrValue(value.Attr, "left"))
				}
			case "headerReference":
				if inSection {
					result.headerID = attrValue(value.Attr, "id")
				}
			case "footerReference":
				if inSection {
					result.footerID = attrValue(value.Attr, "id")
				}
			}
		case xml.CharData:
			if inText && run != nil {
				run.Text += string(value)
			}
		case xml.EndElement:
			switch value.Name.Local {
			case "t", "delText":
				inText = false
			case "pPr":
				inPPr = false
			case "rPr":
				inRPr = false
			case "r":
				if paragraph != nil && run != nil {
					paragraph.Runs = append(paragraph.Runs, *run)
				}
				run = nil
			case "p":
				if paragraph != nil {
					if len(paragraph.Runs) == 0 {
						paragraph.Runs = []document.Run{{Text: ""}}
					}
					if tableDepth > 0 && inTableCell {
						cellParagraphs = append(cellParagraphs, *paragraph)
					} else {
						result.paragraphs = append(result.paragraphs, *paragraph)
					}
				}
				paragraph = nil
			case "tc":
				if tableDepth == 1 {
					var cell strings.Builder
					for i, paragraph := range cellParagraphs {
						if i > 0 {
							cell.WriteByte('\n')
						}
						cell.WriteString(plainDOCXParagraph(paragraph))
					}
					tableRow = append(tableRow, cell.String())
					format := document.TableCellFormat{}
					if len(cellParagraphs) > 0 {
						first := cellParagraphs[0]
						format.Alignment = first.Align
						format.ParagraphStyle = first.ParaStyle
						if len(first.Runs) > 0 {
							format.Style = first.Runs[0].Style
							format.Foreground = first.Runs[0].Fg
							format.Background = first.Runs[0].Bg
						}
					}
					tableRowFormats = append(tableRowFormats, format)
					inTableCell = false
					cellParagraphs = nil
				}
			case "tr":
				if tableDepth == 1 && len(tableRow) > 0 {
					table.Rows = append(table.Rows, tableRow)
					table.CellFormats = append(table.CellFormats, tableRowFormats)
				}
			case "tbl":
				if tableDepth == 1 && len(table.Rows) > 0 {
					columns := 0
					for _, row := range table.Rows {
						columns = max(columns, len(row))
					}
					for i := range table.Rows {
						for len(table.Rows[i]) < columns {
							table.Rows[i] = append(table.Rows[i], "")
						}
						for len(table.CellFormats[i]) < columns {
							table.CellFormats[i] = append(table.CellFormats[i], document.TableCellFormat{})
						}
					}
					tableCopy := table
					result.paragraphs = append(result.paragraphs, document.Paragraph{
						Runs:  []document.Run{{Text: ""}},
						Table: &tableCopy,
					})
				}
				tableDepth--
			case "sectPr":
				inSection = false
			}
		}
	}
	if width > 0 && height > 0 {
		result.page = pageFromDOCXDimensions(width, height, orientation)
	}
	if marginLeft >= 0 {
		result.page.MarginLeft = twipsToColumns(marginLeft)
	}
	if marginRight >= 0 {
		result.page.MarginRight = twipsToColumns(marginRight)
	}
	if marginTop >= 0 {
		result.page.MarginTop = twipsToRows(marginTop)
	}
	if marginBottom >= 0 {
		result.page.MarginBottom = twipsToRows(marginBottom)
	}
	result.page = result.page.Normalized()
	return result, nil
}

func parseDOCXParagraphStyle(style string) document.ParagraphStyle {
	switch strings.ToLower(strings.ReplaceAll(style, " ", "")) {
	case "title":
		return document.ParagraphTitle
	case "heading1":
		return document.ParagraphHeading1
	case "heading2":
		return document.ParagraphHeading2
	case "heading3":
		return document.ParagraphHeading3
	case "quote", "intensequote":
		return document.ParagraphQuote
	case "code":
		return document.ParagraphCode
	default:
		return document.ParagraphNormal
	}
}

func pageFromDOCXDimensions(widthTwips, heightTwips int, orientation document.Orientation) document.PageSettings {
	widthMM := float64(widthTwips) * 25.4 / 1440
	heightMM := float64(heightTwips) * 25.4 / 1440
	if orientation == document.Landscape {
		widthMM, heightMM = maxFloat(widthMM, heightMM), minFloat(widthMM, heightMM)
	} else {
		widthMM, heightMM = minFloat(widthMM, heightMM), maxFloat(widthMM, heightMM)
	}
	sizes := []struct {
		size document.PageSize
		w, h float64
	}{
		{document.SizeA3, 297, 420},
		{document.SizeA4, 210, 297},
		{document.SizeA5, 148, 210},
		{document.SizeLetter, 215.9, 279.4},
		{document.SizeLegal, 215.9, 355.6},
	}
	for _, size := range sizes {
		expectedWidth, expectedHeight := size.w, size.h
		if orientation == document.Landscape {
			expectedWidth, expectedHeight = expectedHeight, expectedWidth
		}
		if math.Abs(widthMM-expectedWidth) < 1 && math.Abs(heightMM-expectedHeight) < 1 {
			width, height := document.GetDimensions(size.size, orientation)
			return document.PageSettings{
				Size: size.size, Orientation: orientation, Width: width, Height: height,
				PageNumbers: document.PageNumbers{Where: document.NumOff, Align: document.AlignCenter, Format: document.FormatPage},
			}
		}
	}
	width := max(1, int(math.Round(widthMM*100/210)))
	height := max(1, int(math.Round(heightMM*59/297)))
	return document.PageSettings{
		Size: document.SizeA4, Orientation: orientation, Width: width, Height: height,
		PageNumbers: document.PageNumbers{Where: document.NumOff, Align: document.AlignCenter, Format: document.FormatPage},
	}
}

func twipsToColumns(value int) int {
	return max(0, int(math.Round(float64(value)/columnTwip)))
}

func twipsToRows(value int) int {
	return max(0, int(math.Round(float64(value)/rowTwips)))
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func attrValue(attributes []xml.Attr, name string) string {
	for _, attribute := range attributes {
		if attribute.Name.Local == name {
			return attribute.Value
		}
	}
	return ""
}

func wordBoolEnabled(value string) bool {
	switch strings.ToLower(value) {
	case "0", "false", "off":
		return false
	default:
		return true
	}
}

func parseDOCXRelationships(data []byte) (map[string]string, error) {
	result := make(map[string]string)
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return result, nil
		}
		if err != nil {
			return nil, err
		}
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "Relationship" {
			id, target := attrValue(start.Attr, "Id"), attrValue(start.Attr, "Target")
			if id != "" && target != "" {
				result[id] = target
			}
		}
	}
}

func docxRelationshipPath(target string) (string, error) {
	if strings.HasPrefix(target, "/") {
		target = strings.TrimPrefix(target, "/")
	} else {
		target = path.Join("word", target)
	}
	target = path.Clean(target)
	if target == "." || target == ".." || strings.HasPrefix(target, "../") {
		return "", fmt.Errorf("invalid DOCX relationship target %q", target)
	}
	return target, nil
}

func parseDOCXTitle(data []byte) (string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	inTitle := false
	var title strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			return title.String(), nil
		}
		if err != nil {
			return "", err
		}
		switch value := token.(type) {
		case xml.StartElement:
			if value.Name.Local == "title" {
				inTitle = true
			}
		case xml.CharData:
			if inTitle {
				title.Write(value)
			}
		case xml.EndElement:
			if value.Name.Local == "title" {
				inTitle = false
			}
		}
	}
}

func parseDOCXHeaderFooter(data []byte) (document.HeaderFooter, error) {
	var result document.HeaderFooter
	var text strings.Builder
	inText, inSimpleField := false, false
	var fieldCode strings.Builder
	var complexField, fieldInstruction, fieldResult bool
	var complexCode strings.Builder
	inRun := false
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return result, err
		}
		switch value := token.(type) {
		case xml.StartElement:
			switch value.Name.Local {
			case "r":
				inRun = true
			case "fldSimple":
				inSimpleField = true
				fieldCode.Reset()
				fieldCode.WriteString(attrValue(value.Attr, "instr"))
			case "fldChar":
				switch attrValue(value.Attr, "fldCharType") {
				case "begin":
					complexField, fieldInstruction, fieldResult = true, true, false
					complexCode.Reset()
				case "separate":
					fieldInstruction, fieldResult = false, true
				case "end":
					if complexField {
						text.WriteString(docxFieldToken(complexCode.String()))
					}
					complexField, fieldInstruction, fieldResult = false, false, false
				}
			case "instrText":
				inText = true
			case "t", "delText":
				inText = true
			case "tab":
				if inRun {
					text.WriteByte('\t')
				}
			}
		case xml.CharData:
			if inText && complexField && fieldInstruction {
				complexCode.Write(value)
			} else if inText && !inSimpleField && !(complexField && fieldResult) {
				text.Write(value)
			}
		case xml.EndElement:
			switch value.Name.Local {
			case "fldSimple":
				if inSimpleField {
					text.WriteString(docxFieldToken(fieldCode.String()))
				}
				inSimpleField = false
			case "instrText", "t", "delText":
				inText = false
			case "p":
				text.WriteByte('\n')
			case "r":
				inRun = false
			}
		}
	}
	lines := strings.Split(strings.TrimRight(text.String(), "\n"), "\n")
	if len(lines) == 0 {
		return result, nil
	}
	parts := strings.Split(lines[0], "\t")
	switch len(parts) {
	case 1:
		result.Left = parts[0]
	case 2:
		result.Left, result.Right = parts[0], parts[1]
	default:
		result.Left, result.Center, result.Right = parts[0], parts[1], strings.Join(parts[2:], "\t")
	}
	return result, nil
}

func docxFieldToken(instruction string) string {
	instruction = strings.ToUpper(strings.TrimSpace(instruction))
	switch {
	case strings.HasPrefix(instruction, "NUMPAGES"):
		return "{pages}"
	case strings.HasPrefix(instruction, "PAGE"):
		return "{page}"
	default:
		return ""
	}
}

func docxHighlightColor(value string) string {
	return docxHighlightColors[strings.ToLower(value)]
}

func inferDOCXPageNumbers(page *document.PageSettings) {
	for _, location := range []struct {
		where document.NumberWhere
		value *document.HeaderFooter
	}{
		{document.NumHeader, &page.Header},
		{document.NumFooter, &page.Footer},
	} {
		slots := []*string{&location.value.Left, &location.value.Center, &location.value.Right}
		for index, slot := range slots {
			for _, pattern := range []struct {
				text   string
				format document.NumberFormat
			}{
				{"Page {page} of {pages}", document.FormatPageOf},
				{"{page} of {pages}", document.FormatOf},
				{"Page {page}", document.FormatPage},
				{"{page}", document.FormatPlain},
			} {
				if strings.HasSuffix(*slot, pattern.text) {
					*slot = strings.TrimRight(strings.TrimSuffix(*slot, pattern.text), " ")
					align := document.AlignLeft
					if index == 1 {
						align = document.AlignCenter
					} else if index == 2 {
						align = document.AlignRight
					}
					page.PageNumbers = document.PageNumbers{
						Where: location.where, Align: align, Format: pattern.format,
					}
					return
				}
			}
		}
	}
}
