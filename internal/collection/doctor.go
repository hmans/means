package collection

import (
	"fmt"
	"os"
	"path"
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
		}
	}
	return result, nil
}
