// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package document

import "fmt"

const (
	maxRestoredPageWidth  = 1000
	maxRestoredPageHeight = 600
)

// State contains the persistent document data, excluding transient selection
// and undo history.
type State struct {
	Title           string       `json:"title"`
	Paragraphs      []Paragraph  `json:"paragraphs"`
	Page            PageSettings `json:"page"`
	CursorPara      int          `json:"cursor_paragraph"`
	CursorOff       int          `json:"cursor_offset"`
	ActiveStyle     Style        `json:"active_style"`
	ActiveFg        string       `json:"active_foreground,omitempty"`
	ActiveBg        string       `json:"active_background,omitempty"`
	LastSearch      string       `json:"last_search,omitempty"`
	LastSearchCase  bool         `json:"last_search_case,omitempty"`
	LastSearchWhole bool         `json:"last_search_whole,omitempty"`
}

// Snapshot returns a deep copy of the document's persistent state.
func (d *Document) Snapshot() State {
	d.mu.RLock()
	defer d.mu.RUnlock()

	paragraphs := make([]Paragraph, len(d.Paragraphs))
	for i, paragraph := range d.Paragraphs {
		paragraphs[i] = cloneParagraph(paragraph)
	}
	return State{
		Title:           d.Title,
		Paragraphs:      paragraphs,
		Page:            d.Page,
		CursorPara:      d.CursorPara,
		CursorOff:       d.CursorOff,
		ActiveStyle:     d.ActiveStyle,
		ActiveFg:        d.ActiveFg,
		ActiveBg:        d.ActiveBg,
		LastSearch:      d.LastSearch,
		LastSearchCase:  d.LastSearchCase,
		LastSearchWhole: d.LastSearchWhole,
	}
}

// Restore replaces document data from a validated persistent state.
func (d *Document) Restore(state State) error {
	if len(state.Paragraphs) == 0 {
		return fmt.Errorf("document has no paragraphs")
	}
	if state.Page.Width <= 0 || state.Page.Height <= 0 {
		return fmt.Errorf("page dimensions must be positive")
	}
	if state.Page.Width > maxRestoredPageWidth || state.Page.Height > maxRestoredPageHeight {
		state.Page = state.Page.FitToWidth(maxRestoredPageWidth)
	}
	const validStyles = StyleBold | StyleItalic | StyleUnderline | StyleStrike
	if state.ActiveStyle&^validStyles != 0 {
		return fmt.Errorf("document has invalid active text style")
	}
	for pi, paragraph := range state.Paragraphs {
		if paragraph.Align < AlignLeft || paragraph.Align > AlignRight {
			return fmt.Errorf("paragraph %d has invalid alignment", pi+1)
		}
		if paragraph.ParaStyle > ParagraphCode {
			return fmt.Errorf("paragraph %d has invalid style", pi+1)
		}
		if paragraph.SpaceAfter < 0 || paragraph.SpaceAfter > maxParagraphSpaceAfter {
			return fmt.Errorf("paragraph %d has invalid spacing", pi+1)
		}
		if paragraph.Table != nil {
			if err := validateTable(paragraph.Table); err != nil {
				return fmt.Errorf("paragraph %d has invalid table: %w", pi+1, err)
			}
		}
		for _, run := range paragraph.Runs {
			if run.Style&^validStyles != 0 {
				return fmt.Errorf("paragraph %d has invalid text style", pi+1)
			}
		}
	}
	if state.Page.PageNumbers.Align < AlignLeft || state.Page.PageNumbers.Align > AlignRight {
		return fmt.Errorf("page numbers have invalid alignment")
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	d.Title = state.Title
	d.Paragraphs = make([]Paragraph, len(state.Paragraphs))
	for i, paragraph := range state.Paragraphs {
		d.Paragraphs[i] = cloneParagraph(paragraph)
		if len(d.Paragraphs[i].Runs) == 0 {
			d.Paragraphs[i].Runs = []Run{{Text: "", Style: StyleNone}}
		}
	}
	d.Page = state.Page.Normalized()
	d.CursorPara = state.CursorPara
	d.CursorOff = state.CursorOff
	d.ensureLeadingPageBreakAnchor()
	d.reflowTablesLocked()
	d.ActiveStyle = state.ActiveStyle
	d.ActiveFg = state.ActiveFg
	d.ActiveBg = state.ActiveBg
	d.LastSearch = state.LastSearch
	d.LastSearchCase = state.LastSearchCase
	d.LastSearchWhole = state.LastSearchWhole
	d.hasAnchor = false
	d.goalSet = false
	d.history = nil
	d.redo = nil
	d.lastEdit = editOther
	d.Dirty = false
	d.ensureCursor()
	return nil
}
