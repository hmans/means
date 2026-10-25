package collection

import (
	"fmt"
	"os"
	"path"
	"sort"
)

// Diagnostic identifies one problem with a stored document. Path is relative
// to the collection so files without a readable ID can still be located.
type Diagnostic struct {
	ID    string `json:"id"`
	Path  string `json:"path"`
	Error string `json:"error"`
}

type DoctorResult struct {
	Healthy     bool         `json:"healthy"`
	Checked     int          `json:"checked"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Warnings    []Diagnostic `json:"warnings,omitempty"`
}

// Doctor inspects every visible document against the current schema without
// changing any files. Malformed envelopes and duplicate IDs are also reported.
func (c *Collection) Doctor() (*DoctorResult, error) {
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	entries, err := scanDocuments(root)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, entry := range entries {
		if entry.doc != nil && entry.doc.ID != "" {
			counts[entry.doc.ID]++
		}
	}
	result := &DoctorResult{Healthy: true, Checked: len(entries), Diagnostics: []Diagnostic{}}
	// Read the atomically replaced cache without locking or creating files.
	embeddings, err := c.embeddingStore()
	if err != nil {
		return nil, err
	}
	if embeddings != nil {
		if err := embeddings.load(); err != nil {
			return nil, err
		}
		if embeddings.dirty {
			result.Warnings = append(result.Warnings, Diagnostic{Path: ".means/embeddings.jsonl", Error: "embedding cache is corrupt or incompatible with the current configuration; run means embed"})
		}
	}
	eligible := map[string]bool{}
	for _, entry := range entries {
		id := ""
		if entry.doc != nil {
			id = entry.doc.ID
		}
		add := func(err error) {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{ID: id, Path: path.Clean(entry.filename), Error: err.Error()})
			result.Healthy = false
		}
		if entry.err != nil {
			add(entry.err)
		}
		if id != "" && counts[id] > 1 {
			add(fmt.Errorf("%w %q; resolve the duplicate files", ErrDuplicateID, id))
		}
		if entry.err == nil {
			if err := c.ValidateData(entry.doc.Frontmatter["data"].(map[string]any)); err != nil {
				add(err)
			}
			if embeddings != nil && counts[id] == 1 {
				eligible[id] = true
				record, exists := embeddings.records[id]
				warn := func(message string) {
					result.Warnings = append(result.Warnings, Diagnostic{ID: id, Path: path.Clean(entry.filename), Error: message})
				}
				if !exists {
					warn("document has no usable embedding; run means embed")
				} else if record.Hash != contentHash(embeddingInput(entry.doc.Frontmatter, entry.doc.Body)) {
					warn("document embedding is stale; run means embed")
				}
			}
		}
	}
	if embeddings != nil {
		var orphaned []string
		for id := range embeddings.records {
			if !eligible[id] {
				orphaned = append(orphaned, id)
			}
		}
		sort.Strings(orphaned)
		for _, id := range orphaned {
			result.Warnings = append(result.Warnings, Diagnostic{ID: id, Path: ".means/embeddings.jsonl", Error: "orphaned embedding has no unique valid document in this collection; run means embed"})
		}
	}
	return result, nil
}
