package collection

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Increment when parsing rules or the record format change. This cache is
// derived data, never a source of schema validation results.
const documentCacheVersion = 1

type documentCacheHeader struct {
	Version int    `json:"version"`
	Hash    string `json:"records_sha256"`
}

type documentCacheRecord struct {
	Path        string         `json:"path"`
	Hash        string         `json:"sha256"`
	Frontmatter map[string]any `json:"frontmatter"`
	BodyOffset  int            `json:"body_offset"`
}

func loadDocumentCache(filename string) (map[string]documentCacheRecord, error) {
	info, err := os.Lstat(filename)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("document cache must be a regular file")
	}
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	header, records, ok := bytes.Cut(data, []byte("\n"))
	var h documentCacheHeader
	if !ok || json.Unmarshal(header, &h) != nil || h.Version != documentCacheVersion || h.Hash != contentHash(string(records)) {
		return nil, nil
	}
	result := map[string]documentCacheRecord{}
	decoder := json.NewDecoder(bytes.NewReader(records))
	decoder.UseNumber()
	for {
		var record documentCacheRecord
		err := decoder.Decode(&record)
		if err == io.EOF {
			return result, nil
		}
		if err != nil || !filepath.IsLocal(record.Path) || len(record.Hash) != 64 || record.BodyOffset < 0 || validateEnvelope(record.Frontmatter) != nil {
			return nil, nil
		}
		if _, exists := result[record.Path]; exists {
			return nil, nil
		}
		result[record.Path] = record
	}
}

// Called under search.lock. Always discover paths and hash their full contents;
// timestamps cannot prove freshness after external edits. Invalid documents are
// parsed again so diagnostics retain the same behavior as an uncached scan.
func scanCachedDocuments(root *os.Root, filename string) ([]storedDocument, error) {
	previous, err := loadDocumentCache(filename)
	if err != nil {
		return nil, err
	}
	changed := previous == nil
	reused := 0
	entries, err := scanDocumentsWithReader(root, func(root *os.Root, name string) (*Document, error) {
		data, err := readDocumentBytes(root, name)
		if err != nil {
			return nil, err
		}
		hash := contentHash(string(data))
		if record, ok := previous[name]; ok && record.Hash == hash && record.BodyOffset <= len(data) {
			reused++
			return &Document{
				ID: record.Frontmatter["id"].(string), Hash: hash,
				Frontmatter: record.Frontmatter, Body: string(data[record.BodyOffset:]),
				Markdown: string(data), filename: name,
			}, nil
		}
		doc, err := parseDocumentFile(name, data, hash)
		if err == nil {
			changed = true
		}
		return doc, err
	})
	if err != nil {
		return nil, err
	}
	// A missing old record also covers deletions, new collection boundaries,
	// and formerly valid files that now fail to read or parse.
	if !changed && reused == len(previous) {
		return entries, nil
	}
	var records bytes.Buffer
	encoder := json.NewEncoder(&records)
	for _, entry := range entries {
		if entry.err != nil {
			continue
		}
		doc := entry.doc
		if err := encoder.Encode(documentCacheRecord{
			Path: entry.filename, Hash: doc.Hash, Frontmatter: doc.Frontmatter,
			BodyOffset: len(doc.Markdown) - len(doc.Body),
		}); err != nil {
			return nil, err
		}
	}
	var output bytes.Buffer
	if err := json.NewEncoder(&output).Encode(documentCacheHeader{documentCacheVersion, contentHash(records.String())}); err != nil {
		return nil, err
	}
	output.Write(records.Bytes())
	if err := replaceDocumentCache(filename, output.Bytes()); err != nil {
		return nil, err
	}
	return entries, nil
}

func replaceDocumentCache(filename string, data []byte) error {
	temp, err := os.CreateTemp(filepath.Dir(filename), ".documents-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), filename)
}
