package collection

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/analyzer/custom"
	"github.com/blevesearch/bleve/v2/analysis/token/lowercase"
	"github.com/blevesearch/bleve/v2/analysis/tokenizer/unicode"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
)

const searchIndexVersion = 1
const searchAnalyzer = "means"

var searchManifestKey = []byte("means-manifest")

type searchManifest struct {
	Version int               `json:"version"`
	Hashes  map[string]string `json:"hashes"`
}

func searchMapping() (*mapping.IndexMappingImpl, error) {
	m := bleve.NewIndexMapping()
	if err := m.AddCustomAnalyzer(searchAnalyzer, map[string]any{
		"type": custom.Name, "tokenizer": unicode.Name, "token_filters": []string{lowercase.Name},
	}); err != nil {
		return nil, err
	}
	m.DefaultAnalyzer = searchAnalyzer
	m.DefaultMapping = bleve.NewDocumentStaticMapping()
	for _, name := range []string{"title", "body"} {
		field := bleve.NewTextFieldMapping()
		field.Analyzer = searchAnalyzer
		field.Store = false
		field.IncludeInAll = false
		m.DefaultMapping.AddFieldMappingsAt(name, field)
	}
	return m, nil
}

// Cache paths are reserved for means, but never follow symlinks into user data.
func checkCachePaths(path string) error {
	return filepath.WalkDir(path, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("search cache contains symlink %q", name)
		}
		if !entry.IsDir() && !entry.Type().IsRegular() {
			return fmt.Errorf("search cache contains non-regular file %q", name)
		}
		return nil
	})
}

func openSearchIndex(path string, m *mapping.IndexMappingImpl) (bleve.Index, searchManifest, error) {
	empty := searchManifest{Version: searchIndexVersion, Hashes: map[string]string{}}
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		index, err := bleve.New(path, m)
		return index, empty, err
	} else if err != nil {
		return nil, empty, err
	}
	if err := checkCachePaths(path); err != nil {
		return nil, empty, err
	}
	index, err := bleve.Open(path)
	if errors.Is(err, os.ErrPermission) {
		return nil, empty, err
	}
	if err == nil {
		data, readErr := index.GetInternal(searchManifestKey)
		var manifest searchManifest
		if readErr == nil && json.Unmarshal(data, &manifest) == nil && manifest.Version == searchIndexVersion && manifest.Hashes != nil {
			return index, manifest, nil
		}
		if err := index.Close(); err != nil {
			return nil, empty, err
		}
		if readErr != nil {
			return nil, empty, readErr
		}
	}
	// Missing/corrupt metadata or an obsolete format is rebuilt under the lock.
	if err := os.RemoveAll(path); err != nil {
		return nil, empty, err
	}
	index, err = bleve.New(path, m)
	return index, empty, err
}

// Reconcile against the index's own committed hashes, independently of the
// metadata cache. Updating that cache does not imply an index batch succeeded.
func reconcileSearchIndex(index bleve.Index, manifest searchManifest, docs map[string]*Document) error {
	batch := index.NewBatch()
	next := searchManifest{Version: searchIndexVersion, Hashes: map[string]string{}}
	changed := false
	for id, doc := range docs {
		next.Hashes[id] = doc.Hash
		if manifest.Hashes[id] != doc.Hash {
			if err := batch.Index(id, map[string]any{"title": doc.Frontmatter["title"], "body": doc.Body}); err != nil {
				return err
			}
			changed = true
		}
	}
	for id := range manifest.Hashes {
		if docs[id] == nil {
			batch.Delete(id)
			changed = true
		}
	}
	if changed || len(manifest.Hashes) == 0 {
		data, err := json.Marshal(next)
		if err != nil {
			return err
		}
		// Hashes and indexed documents commit together; a crash cannot advance one alone.
		batch.SetInternal(searchManifestKey, data)
		return index.Batch(batch)
	}
	return nil
}

func (c *Collection) searchFullText(opts SearchOptions) (result *SearchResult, err error) {
	m, err := searchMapping()
	if err != nil {
		return nil, err
	}
	tokens, err := m.AnalyzeText(searchAnalyzer, []byte(opts.Query))
	if err != nil {
		return nil, err
	}
	// The lock is outside search/ so deleting/rebuilding the index cannot replace it.
	cache := filepath.Join(c.Path, ".means")
	if stat, e := os.Lstat(cache); e == nil && (!stat.IsDir() || stat.Mode()&os.ModeSymlink != 0) {
		return nil, fmt.Errorf(".means must be a directory, not a symlink")
	} else if e != nil && !errors.Is(e, os.ErrNotExist) {
		return nil, e
	}
	if err := os.MkdirAll(cache, 0o700); err != nil {
		return nil, err
	}
	lockPath := filepath.Join(cache, "search.lock")
	if stat, e := os.Lstat(lockPath); e == nil && !stat.Mode().IsRegular() {
		return nil, fmt.Errorf("search lock must be a regular file")
	} else if e != nil && !errors.Is(e, os.ErrNotExist) {
		return nil, e
	}
	lock, err := tryLock(context.Background(), lockPath, "search cache", "search cache is busy; retry after the current search finishes")
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, lock.Unlock()) }()
	root, err := os.OpenRoot(c.Path)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	entries, err := scanCachedDocuments(root, filepath.Join(cache, "documents.jsonl"))
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, entry := range entries {
		if entry.doc != nil && entry.doc.ID != "" {
			counts[entry.doc.ID]++
		}
	}
	result = &SearchResult{Documents: []SearchHit{}, Diagnostics: []Diagnostic{}}
	docs := map[string]*Document{}
	for _, entry := range entries {
		id := ""
		if entry.doc != nil {
			id = entry.doc.ID
		}
		if id != "" && counts[id] > 1 {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{ID: id, Path: entry.filename, Error: fmt.Sprintf("%s %q; resolve the duplicate files", ErrDuplicateID, id)})
			continue
		}
		if entry.err != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{ID: id, Path: entry.filename, Error: entry.err.Error()})
			continue
		}
		docs[id] = entry.doc
	}
	index, manifest, err := openSearchIndex(filepath.Join(cache, "search"), m)
	if err != nil {
		return nil, fmt.Errorf("open search cache: %w", err)
	}
	defer func() { err = errors.Join(err, index.Close()) }()
	if err := reconcileSearchIndex(index, manifest, docs); err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return result, nil
	}
	terms := map[string]bool{}
	var clauses []query.Query
	for _, token := range tokens {
		term := string(token.Term)
		if terms[term] {
			continue
		}
		terms[term] = true
		title := bleve.NewTermQuery(term)
		title.SetField("title")
		title.SetBoost(3)
		body := bleve.NewTermQuery(term)
		body.SetField("body")
		clauses = append(clauses, bleve.NewDisjunctionQuery(title, body))
	}
	q := bleve.NewConjunctionQuery(clauses...)
	for offset := 0; ; offset += 100 {
		request := bleve.NewSearchRequestOptions(q, 100, offset, false)
		request.SortBy([]string{"-_score", "_id"})
		found, err := index.Search(request)
		if err != nil {
			return nil, err
		}
		for _, hit := range found.Hits {
			doc := docs[hit.ID]
			if doc == nil || !opts.matchesData(doc.Frontmatter["data"].(map[string]any)) {
				continue
			}
			if opts.Limit > 0 && len(result.Documents) == opts.Limit {
				result.HasMore = true
				return result, nil
			}
			match, err := fullTextExcerpt(m, doc, terms)
			if err != nil {
				return nil, err
			}
			hit := SearchHit{ID: doc.ID, Frontmatter: doc.Frontmatter, Match: match}
			if opts.WithBody {
				hit.Body = &doc.Body
			}
			result.Documents = append(result.Documents, hit)
		}
		if len(found.Hits) < 100 {
			break
		}
	}
	return result, nil
}

func fullTextExcerpt(m *mapping.IndexMappingImpl, doc *Document, terms map[string]bool) (*SearchMatch, error) {
	var best *SearchMatch
	bestCoverage := 0
	for _, field := range []struct{ name, text string }{{"title", doc.Frontmatter["title"].(string)}, {"body", doc.Body}} {
		tokens, err := m.AnalyzeText(searchAnalyzer, []byte(field.text))
		if err != nil {
			return nil, err
		}
		// Map tokenizer byte boundaries to rune positions without repeatedly
		// scanning prefixes. Excerpts must not split a UTF-8 character.
		positions := make([]int, len(field.text)+1)
		runes := []rune(field.text)
		n := 0
		for offset := range field.text {
			positions[offset] = n
			n++
		}
		positions[len(field.text)] = n
		type occurrence struct {
			term       string
			start, end int
		}
		var matches []occurrence
		for _, token := range tokens {
			term := string(token.Term)
			if terms[term] {
				matches = append(matches, occurrence{term, positions[token.Start], positions[token.End]})
			}
		}
		// Score bounded passages by distinct query terms, not repetitions.
		// Sliding pointers keep long documents with many matches linear.
		counts := map[string]int{}
		first, last := 0, 0
		for _, anchor := range matches {
			left := anchor.start
			right := min(len(runes), left+180)
			for last < len(matches) && matches[last].end <= right {
				counts[matches[last].term]++
				last++
			}
			for first < last && matches[first].start < left {
				term := matches[first].term
				counts[term]--
				if counts[term] == 0 {
					delete(counts, term)
				}
				first++
			}
			if len(counts) > bestCoverage || best == nil {
				// Spend unused passage space on leading context without dropping
				// any complete query term from the selected window.
				if last > first {
					left = max(0, left-min(40, max(0, 180-(matches[last-1].end-left))))
				}
				right = min(len(runes), left+180)
				text := strings.Join(strings.Fields(string(runes[left:right])), " ")
				if left > 0 {
					text = "…" + text
				}
				if right < len(runes) {
					text += "…"
				}
				best = &SearchMatch{Field: field.name, Excerpt: text}
				bestCoverage = len(counts)
			}
		}
	}
	// Equal coverage retains the title, then the earliest passage in that field.
	return best, nil
}
