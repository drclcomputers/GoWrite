# GoWrite

<div align="center">

![Version](https://img.shields.io/badge/version-0.2.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.24.2-00ADD8.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey.svg)

**A terminal-based word processor built with Go**

Same DNA as [GoSheet](https://github.com/drclcomputers/GoSheet) — tview, dark teal/amber UI, lightweight, keyboard-first.

</div>

---

## Overview

GoWrite is a lightweight, terminal-native word processor.

### Current status (v0.2.0)

- Start menu (New / Open / Quit) with the same look & feel as GoSheet
- Full-screen editor with **word wrap** and **flowing paragraphs**
- Live status bar (line/col · words · chars · paragraphs · active style)
- Dirty-document tracking
- **Page setup**: paper size, orientation and margins
- **Header & footer** with `{page}`, `{pages}`, `{title}` tokens and automatic page numbers
- **Basic styling**: Bold / Italic / Underline / Strikethrough (toggle + apply to paragraph)
- **Find / Find & Replace** with case-sensitive and whole-word options
- **Go to line** (paragraph)
- Find next / previous (F3 / F4)
- Keyboard shortcuts aligned with GoSheet habits

### Roadmap

- Real file open/save (plain text + Markdown)
- Recent-files list
- True rich-text runs rendered with style tags (model already supports runs)
- Export to PDF / Markdown / HTML
- Multiple buffers / tabs
- Spell-check hooks

---

## Quick start

```bash
# Prerequisites: Go 1.24.2+
cd gowrite

go mod tidy
go build -o gowrite .

./gowrite
```

Open a file directly:

```bash
./gowrite -file notes.txt
# or
./gowrite notes.txt
```

---

## Keyboard

| Key                            | Action                                                          |
| ------------------------------ | --------------------------------------------------------------- |
| Arrow keys                     | Move cursor (↑/↓ follow wrapped lines and remember the column)  |
| Home / End                     | Start / end of the wrapped line                                 |
| Ctrl+Home / Ctrl+End (or Alt+) | Start / end of the document                                     |
| PgUp / PgDn                    | Move one screen up / down                                       |
| Ctrl+Left / Ctrl+Right         | Word left / right                                               |
| Shift + any movement key       | Select (Shift+Ctrl+Left/Right selects by word)                  |
| **Ctrl+A**                     | Select all                                                      |
| **Esc** (with a selection)     | Clear the selection                                             |
| **Ctrl+C** / Ctrl+Insert       | Copy (Ctrl+C with nothing selected still opens the exit prompt) |
| **Ctrl+X** / Shift+Delete      | Cut                                                             |
| **Ctrl+V** / Shift+Insert      | Paste (keeps styles, colours and alignment)                     |
| Enter                          | New paragraph (flowing)                                         |
| **Alt+G** / **Ctrl+G**         | Go to line                                                      |
| **Alt+F** / **Ctrl+F**         | Find                                                            |
| **Alt+H** / **Ctrl+H**         | Find & Replace                                                  |
| **F3** / **F4**                | Find next / previous                                            |
| **Alt+B** / **Ctrl+B**         | Toggle Bold                                                     |
| **Alt+I** / **Ctrl+I**         | Toggle Italic                                                   |
| **Alt+U** / **Ctrl+U**         | Toggle Underline                                                |
| **Alt+K**                      | Toggle Strikethrough                                            |
| **Alt+A**                      | Apply active style to current paragraph                         |
| **Alt+N**                      | New document                                                    |
| **Alt+O**                      | Open document                                                   |
| **Alt+S** / **Ctrl+S**         | Save document                                                   |
| **Alt+P**                      | Page setup (size, orientation, margins)                         |
| **Alt+D**                      | Header, footer and page numbers                                 |
| **?** / **Alt+/**              | Help                                                            |
| **Esc**                        | Quit (asks if dirty)                                            |

Typing, Backspace and Delete replace the selection, and bold/italic/underline/strike, colours and alignment apply to it (alignment to every paragraph it touches).

Word wrap is always on. Paragraphs reflow with the terminal width, and left/center/right alignment applies to every wrapped line.

---

## Structure

```
gowrite/
├── main.go
├── go.mod
├── internal/
│   ├── services/
│   │   ├── document/     # paragraphs + styled runs + find/replace
│   │   ├── ui/           # editor, start menu, dialogs
│   │   └── uitheme/      # shared colours & styling
│   └── utils/
└── README.md
```
