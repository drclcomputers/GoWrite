// Copyright (c) 2026 @drclcomputers. All rights reserved.
//
// This work is licensed under the terms of the MIT license.
// For a copy, see <https://opensource.org/licenses/MIT>.

package fileop

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gowrite/internal/services/document"
)

const (
	gdocVersion           = 1
	maxNativeDocumentSize = int64(500 << 20)
)

type GDoc struct {
	Format   string         `json:"format"`
	Version  int            `json:"version"`
	Document document.State `json:"document"`
	Editor   EditorState    `json:"editor"`
}

type EditorState struct {
	ShowFormattingMarks bool `json:"show_formatting_marks"`
}

func SaveGDoc(filename string, state document.State, showFormattingMarks bool) error {
	file := GDoc{
		Format:   "gowrite",
		Version:  gdocVersion,
		Document: state,
		Editor:   EditorState{ShowFormattingMarks: showFormattingMarks},
	}
	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode native document: %w", err)
	}
	return writeAtomic(filename, func(w io.Writer) error {
		gz := gzip.NewWriter(w)
		if _, err := gz.Write(data); err != nil {
			_ = gz.Close()
			return fmt.Errorf("compress native document: %w", err)
		}
		if err := gz.Close(); err != nil {
			return fmt.Errorf("finish compressed document: %w", err)
		}
		return nil
	})
}

func LoadGDoc(filename string) (GDoc, error) {
	var result GDoc
	file, err := os.Open(filename)
	if err != nil {
		return result, err
	}
	defer file.Close()

	gz, err := gzip.NewReader(file)
	if err != nil {
		return result, fmt.Errorf("decompress native document: %w", err)
	}
	data, readErr := readNativePayload(gz, maxNativeDocumentSize)
	closeErr := gz.Close()
	if readErr != nil {
		return result, fmt.Errorf("read compressed document: %w", readErr)
	}
	if closeErr != nil {
		return result, fmt.Errorf("finish decompression: %w", closeErr)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return result, fmt.Errorf("decode native document: %w", err)
	}
	if result.Format != "gowrite" {
		return result, fmt.Errorf("file is not a GoWrite document")
	}
	if result.Version != gdocVersion {
		return result, fmt.Errorf("unsupported GoWrite document version %d", result.Version)
	}
	return result, nil
}

func readNativePayload(reader io.Reader, limit int64) ([]byte, error) {
	if limit < 0 {
		return nil, fmt.Errorf("native document size limit cannot be negative")
	}
	data, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("decompressed native document exceeds the %d-byte limit", limit)
	}
	return data, nil
}

func writeAtomic(filename string, write func(io.Writer) error) error {
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}
	file, err := os.CreateTemp(dir, "."+filepath.Base(filename)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tempName := file.Name()
	defer os.Remove(tempName)

	if err := file.Chmod(0o644); err != nil {
		_ = file.Close()
		return fmt.Errorf("set file permissions: %w", err)
	}
	if err := write(file); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return fmt.Errorf("flush file: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	if err := os.Rename(tempName, filename); err != nil {
		return fmt.Errorf("replace %s: %w", filename, err)
	}
	return nil
}
