# GoWrite

<div align="center">

![Version](https://img.shields.io/badge/version-0.1.0-blue.svg)
![Go Version](https://img.shields.io/badge/go-1.24.2-00ADD8.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)
![Platform](https://img.shields.io/badge/platform-linux%20%7C%20macos%20%7C%20windows-lightgrey.svg)

**A terminal-based word processor built with Go**

Same DNA as [GoSheet](https://github.com/drclcomputers/GoSheet) — tview, dark teal/amber UI, lightweight, keyboard-first.

</div>

---

## 📋 Overview

GoWrite is a lightweight, terminal-native word processor.  
It reuses the visual language, structure and conventions of GoSheet so the two apps feel like siblings.

### Current status (v0.1.0 – base)

- Start menu (New / Open / Quit) with the same look & feel as GoSheet
- Full-screen editor with word-wrap
- Live status bar (words · chars · paragraphs)
- Dirty-document tracking
- Keyboard shortcuts (Alt+N / Alt+O / Alt+S, ?, Esc)
- Clean document model ready for styles, find/replace, export, etc.

### Roadmap (next steps)

- Real file open/save (plain text + Markdown)
- Recent-files list
- Find & replace
- Basic formatting (bold / italic / underline via markup or attributes)
- Export to PDF / Markdown / HTML
- Multiple buffers / tabs
- Spell-check hooks

---

## 🚀 Quick start

```bash
# Prerequisites: Go 1.24.2+
git clone <this-repo>
cd gowrite

go mod tidy
go build -o gowrite

./gowrite
```

Open a file directly:

```bash
./gowrite -file notes.txt
# or
./gowrite notes.txt
```

---

## ⌨️ Keyboard

| Key                | Action               |
| ------------------ | -------------------- |
| Arrow keys         | Move cursor          |
| Enter              | New paragraph        |
| Backspace/Delete   | Edit                 |
| **Alt+N**          | New document         |
| **Alt+O**          | Open (placeholder)   |
| **Alt+S**          | Save (placeholder)   |
| **?** or **Alt+/** | Help                 |
| **Esc**            | Quit (asks if dirty) |

---

## 🏗️ Structure

```
gowrite/
├── main.go
├── go.mod
├── internal/
│   ├── services/
│   │   ├── document/     # in-memory model
│   │   ├── ui/           # editor + start menu
│   │   └── uitheme/      # shared colours & styling (identical to GoSheet)
│   └── utils/
└── README.md
```

The theme package (`uitheme`) is deliberately kept in sync with GoSheet so dialogs, lists and modals look identical.

---

## 📄 License

MIT – same as GoSheet.
