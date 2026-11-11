# GoWrite

<div align="center">

![Version](https://img.shields.io/badge/version-0.8.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.24.2-00ADD8.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey.svg)

**A keyboard-first word processor that runs in your terminal**

</div>

---

## 📋 Overview

GoWrite is a terminal-native word processor built with Go and [tview](https://github.com/rivo/tview). It combines flowing text editing with page-aware layout, paragraph and run formatting, tables, and import/export for common document formats.

### Highlights

- **📝 Writing and editing**: wrapped paragraphs, text selection, clipboard operations, undo and redo
- **🎨 Formatting**: bold, italic, underline, strikethrough, text and background colors, paragraph styles, alignment, and lists
- **📄 Page layout**: paper size, orientation, margins, page background, headers, footers, and page numbers
- **📊 Tables**: structured tables with per-cell formatting
- **🔎 Navigation**: find and replace, go to line, heading outline, and page breaks
- **💾 File formats**: GoWrite native documents, Word DOCX, plain text, Markdown import, and RTF export

---

## ✨ Features

### Editing

- Paragraphs reflow to the terminal width; left, center, and right alignment apply to wrapped lines.
- Insert, replace, delete, and select text using the keyboard.
- Copy, cut, paste, undo, and redo.
- Lists can continue when you press Enter and exit when the current item is empty.
- Insert dates, times, and common Unicode symbols.
- Use formatting marks and a heading outline while editing.

### Formatting

- Apply bold, italic, underline, and strikethrough to a selection or use an active style while typing.
- Choose text and background colors.
- Set paragraph style to Normal, Title, Heading 1–3, Quote, or Code.
- Set paragraph alignment and spacing.
- Insert numbered, lettered, Roman numeral, bullet, dash, and checkbox lists.

### Page setup

- Choose A3, A4, A5, Letter, or Legal paper in portrait or landscape orientation.
- Set margins and page background.
- Configure headers, footers, and automatic page numbers.
- Insert page breaks and navigate between pages.

---

## 🚀 Installation

### Prerequisites

- Go 1.24.2 or newer
- A terminal emulator; color support is recommended

### Build from source

```bash
cd gowrite
go build -o gowrite .
./gowrite
```

### Quick start

Launch GoWrite to open the start screen, then choose **New Document**, **Open**, or **Options**. From the editor, press `Alt+O` to open a document.

The command-line `-file` option and positional filename open a file as plain text:

```bash
./gowrite -file notes.txt
# or
./gowrite notes.txt
```

Use the in-app Open dialog to load `.gdoc`, `.docx`, `.txt`, or `.md` files.

---

## 📖 Usage

On macOS, configure your terminal to use **Option as Meta** for Alt shortcuts.

### Navigation and editing

| Key | Action |
| --- | --- |
| Arrow keys | Move the cursor |
| Shift + Arrow keys | Extend the selection |
| Alt + Shift + Left / Right | Select by word |
| Home / End | Start / end of the wrapped line |
| Ctrl + Home / End or Alt + Home / End | Start / end of the document |
| Page Up / Page Down | Move by a screen |
| Type | Insert text or replace the selection |
| Enter | Start a new paragraph |
| Backspace / Delete | Delete text |
| Tab | Insert a tab; inside a table, open its cell editor |
| Esc | Open the file menu |

### Formatting and document actions

| Key | Action |
| --- | --- |
| Alt + Enter | Open the Format screen |
| Alt + B / I / U / K | Toggle bold / italic / underline / strikethrough |
| Alt + A | Apply the active style to the selection or paragraph |
| Alt + C / Y | Choose text / background color |
| Alt + L / E / R | Align left / center / right |
| Alt + P | Page setup |
| Alt + D | Header, footer, and page-number settings |
| Alt + Shift + P | Insert a page break |
| Alt + S | Save |
| Alt + Shift + S | Save As |
| Alt + Shift + E | Export |
| Alt + N / O | New / open document |
| Alt + G | Go to line |
| Alt + F / H | Find / find and replace |
| F3 / F4 | Find next / previous |
| Alt + Shift + C / X / V | Copy / cut / paste |
| Alt + Shift + A | Select all |
| Alt + Z / Shift + Alt + Z | Undo / redo |
| `?` or Alt + `/` | Show help |

The Format screen also groups style, list, view, insert, paragraph, and page-setup actions. Use `Alt + Shift + Left / Right` there to move between sections.

---

## 📁 File Formats

| Format | Extension | Open | Save / Export | Notes |
| --- | --- | --- | --- | --- |
| GoWrite | `.gdoc` | ✅ | ✅ | Native compressed document format; preserves GoWrite document and editor state |
| Word | `.docx` | ✅ | ✅ | Imports and exports supported text, tables, formatting, colors, and page settings; advanced Word features may not round-trip |
| Plain text | `.txt` | ✅ | ✅ | Formatting is not retained |
| Markdown | `.md` | ✅ | — | Opened as plain text; Markdown syntax is not rendered |
| Rich Text Format | `.rtf` | — | ✅ | Write/export only; RTF import is not supported |

The Open dialog accepts `.gdoc`, `.docx`, `.txt`, and `.md`. Save supports `.gdoc`, `.docx`, `.txt`, and `.rtf`; Export supports `.docx`, `.txt`, and `.rtf`.

Native `.gdoc` files are gzip-compressed JSON. For safety, GoWrite rejects files that expand beyond 500 MiB when opened.

---

## ⚙️ Options and appearance

The start-screen **Options** form controls defaults for new documents:

- Paper size and orientation
- Page margins
- Default save format
- Whether formatting marks are shown
- Autosave interval (1–15 minutes; defaults to 5)

Settings are stored at `~/.gowrite/settings.json`. Unsaved work is periodically written to `~/.gowrite/~temp~.godoc`; on the next launch, GoWrite asks whether to reopen or discard the recovery copy. Page background and other page settings belong to each document and can be changed from **Page Setup**.

The shared UI palette is defined in `internal/services/uitheme/theme.go`.

---

## 🏗️ Project structure

```text
gowrite/
├── main.go
├── internal/
│   ├── services/
│   │   ├── document/       # Document model, editing, pagination, tables, history
│   │   ├── fileop/         # Native, DOCX, RTF, and text I/O
│   │   ├── ui/
│   │   │   ├── editor/     # Editor, formatting, and file dialogs
│   │   │   ├── file/       # File-format operations
│   │   │   ├── navigation/ # Find, replace, and go-to forms
│   │   │   └── startmenu/  # Start screen and options
│   │   └── uitheme/        # Shared colors and widget styling
│   └── utils/              # Settings and utilities
├── go.mod
└── README.md
```

### Development

```bash
go test ./...
go build ./...
```

---

## 🤝 Contributing

Contributions and bug reports are welcome. For bug reports, include your operating system, terminal emulator, steps to reproduce, and expected versus actual behavior. For changes, add focused tests and update documentation when user-facing behavior changes.

---

## 📄 License

GoWrite is licensed under the MIT License. See [LICENSE](LICENSE) for details.

---

## 🙏 Acknowledgments

- [tview](https://github.com/rivo/tview) for the terminal UI framework
- [tcell](https://github.com/gdamore/tcell) for terminal handling
- [go-runewidth](https://github.com/mattn/go-runewidth) for terminal cell-width calculations
- [GoSheet](https://github.com/drclcomputers/GoSheet), a related terminal application
