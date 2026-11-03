package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"os"
	"path"
	"reflect"
	"regexp"
	"strings"
	"unicode/utf8"
)

// SearchOptions selects a search mode and filters its results. The zero value
// lists every valid document in filename order. Filter and DateTime conditions
// are ANDed and apply before Limit in every mode.
type SearchOptions struct {
	// DateTime compares timestamp strings under data; all comparisons must match.
	DateTime        []DateTimeFilter
	dateTimeFilters []compiledDateTimeFilter
	// MinScore is an inclusive cosine threshold from -1 to 1. It requires
	// Semantic. Nil selects DefaultSemanticMinScore; a pointer to zero selects 0.
	MinScore *float64
	// WithBody includes exact Markdown bodies. False leaves SearchHit.Body nil.
	WithBody bool
	// Query searches title and body. By default, all lowercase Unicode tokens
	// must match; title matches rank higher. There is no query syntax, stemming,
	// or stop-word removal. Empty Query selects a data-only scan.
	Query string
	// Filter is a recursive partial match against frontmatter.data. Arrays match
	// completely, including objects inside arrays. Nil or empty means no filter.
	// JSON-compatible Go values are normalized without changing the input map.
	Filter map[string]any
	// Limit caps valid matches. Zero returns every match; negative values fail.
	Limit int
	// Literal selects a case-insensitive substring match in title or body,
	// ordered by filename. It cannot be combined with Semantic.
	Literal bool
	// Semantic ranks cached document vectors by cosine similarity. It requires
	// a nonblank Query and settings supplied through Collection.WithEmbeddings.
	// It can call the embedding provider for the query; missing or stale document
	// vectors are skipped. Use Collection.Embed to refresh those vectors.
	Semantic bool
}

// DefaultSemanticMinScore is the inclusive cosine threshold used when
// SearchOptions.MinScore is nil. It does not apply to other search modes.
const DefaultSemanticMinScore = 0.3

// SearchMatch contains a title or body excerpt for a text match.
// Data-only and semantic searches do not produce excerpts.
type SearchMatch struct {
	// Field is "title" or "body".
	Field string `json:"field"`
	// Excerpt is nearby text, with whitespace collapsed and ellipses when cut.
	Excerpt string `json:"excerpt"`
}

// SearchHit identifies one matching document without a content hash.
// Call Collection.Read with ID before updating or deleting the document.
type SearchHit struct {
	// Body is nil unless WithBody was set. A non-nil empty string is an empty body.
	Body *string `json:"body,omitempty"`
	// ID is the stored document ID, independent of its filename.
	ID string `json:"id"`
	// Frontmatter contains id, title, and data. Numbers use json.Number.
	Frontmatter map[string]any `json:"frontmatter"`
	// Match is an excerpt for a full-text or literal query; otherwise nil.
	Match *SearchMatch `json:"match,omitempty"`
	// Score is cosine similarity for semantic search, from -1 to 1; otherwise
	// nil. Full-text ranking scores are not exposed.
	Score *float64 `json:"score,omitempty"`
}

// SearchResult separates valid matches from document diagnostics. A nil error
// from Search does not imply that Diagnostics is empty.
type SearchResult struct {
	// Documents contains matches in rank order, or filename order for scans.
	Documents []SearchHit `json:"documents"`
	// Diagnostics reports malformed documents and duplicate IDs across the
	// collection, independent of filters and Limit.
	Diagnostics []Diagnostic `json:"diagnostics"`
	// HasMore reports additional matches after filtering, score threshold, and
	// Limit. It is false for an unlimited search.
	HasMore bool `json:"has_more"`
}

// Search ranks full-text queries using a refreshed collection-local index.
// Literal and data-only searches scan in filename order. Filter is a recursive partial match of
// application data; arrays must match completely. Schema-invalid data stays
// searchable, while malformed and duplicate documents become diagnostics.
// Full-text and semantic searches sort rank ties by ID. Full-text search writes
// a disposable .means/ cache; literal and data-only searches remain offline.
func (c *Collection) Search(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	c, err := c.operation(ctx)
	if err != nil {
		return nil, err
	}
	opts.dateTimeFilters, err = compileDateTimeFilters(opts.DateTime)
	if err != nil {
		return nil, err
	}
	if opts.MinScore != nil {
		if !opts.Semantic {
			return nil, fmt.Errorf("min_score requires semantic search")
		}
		if math.IsNaN(*opts.MinScore) || math.IsInf(*opts.MinScore, 0) || *opts.MinScore < -1 || *opts.MinScore > 1 {
			return nil, fmt.Errorf("min_score must be a finite number from -1 to 1")
		}
	}
	if opts.Limit < 0 {
		return nil, fmt.Errorf("limit must be zero or greater")
	}
	if !utf8.ValidString(opts.Query) {
		return nil, fmt.Errorf("query must be valid UTF-8")
	}
	if opts.Filter != nil {
		opts.Filter, err = normalizeObject(opts.Filter)
		if err != nil {
			return nil, fmt.Errorf("invalid filter: %w", err)
		}
	}
	if opts.Semantic {
		if opts.Literal || strings.TrimSpace(opts.Query) == "" {
			return nil, fmt.Errorf("semantic search requires a nonempty query and cannot be combined with literal search")
		}
		return c.searchSemantic(opts)
	}
	if opts.Query != "" && !opts.Literal {
		return c.searchFullText(opts)
	}
	return c.searchScan(opts, 0)
}

// searchScan returns matches in filename order and diagnostics for the whole collection.
// Callers validate options and compile datetime filters before scanning.
func (c *Collection) searchScan(opts SearchOptions, offset int) (*SearchResult, error) {
	var pattern *regexp.Regexp
	if opts.Query != "" {
		var err error
		pattern, err = regexp.Compile("(?i)" + regexp.QuoteMeta(opts.Query))
		if err != nil {
			return nil, fmt.Errorf("compile query: %w", err)
		}
	}
	root, err := os.OpenRoot(c.path)
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
	result := &SearchResult{Documents: []SearchHit{}, Diagnostics: []Diagnostic{}}
	for _, entry := range entries {
		id := ""
		if entry.doc != nil {
			id = entry.doc.ID
		}
		if id != "" && counts[id] > 1 {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{ID: id, Path: path.Clean(entry.filename), Error: fmt.Sprintf("%s %q; resolve the duplicate files", ErrDuplicateID, id)})
			continue
		}
		if entry.err != nil {
			result.Diagnostics = append(result.Diagnostics, Diagnostic{ID: id, Path: path.Clean(entry.filename), Error: entry.err.Error()})
			continue
		}
		data := entry.doc.Frontmatter["data"].(map[string]any)
		if !opts.matchesData(data) {
			continue
		}
		var match *SearchMatch
		if pattern != nil {
			title := entry.doc.Frontmatter["title"].(string)
			if indices := pattern.FindStringIndex(title); indices != nil {
				match = &SearchMatch{Field: "title", Excerpt: excerpt(title, indices)}
			} else if indices := pattern.FindStringIndex(entry.doc.Body); indices != nil {
				match = &SearchMatch{Field: "body", Excerpt: excerpt(entry.doc.Body, indices)}
			} else {
				continue
			}
		}
		if offset > 0 {
			offset--
			continue
		}
		if opts.Limit > 0 && len(result.Documents) == opts.Limit {
			result.HasMore = true
			continue
		}
		hit := SearchHit{ID: id, Frontmatter: entry.doc.Frontmatter, Match: match}
		if opts.WithBody {
			hit.Body = &entry.doc.Body
		}
		result.Documents = append(result.Documents, hit)
	}
	return result, nil
}

// MatchesFilter reports whether data matches a partial metadata filter using the
// same rules as List and Search. Inputs must already use map[string]any, []any,
// and json.Number for objects, arrays, and numbers. Unlike List and Search,
// this helper does not normalize native Go values such as int or []string.
// Nil or empty filter matches every object. It does not apply datetime filters.
func MatchesFilter(data, filter map[string]any) bool {
	for key, expected := range filter {
		actual, exists := data[key]
		if !exists || !matchesValue(actual, expected) {
			return false
		}
	}
	return true
}

func matchesValue(actual, expected any) bool {
	return matchesFilterValue(actual, expected, false)
}

// Objects are partial filters except inside arrays, where every value is exact.
func matchesFilterValue(actual, expected any, exact bool) bool {
	switch expected := expected.(type) {
	case map[string]any:
		object, ok := actual.(map[string]any)
		if !ok || (exact && len(object) != len(expected)) {
			return false
		}
		for key, value := range expected {
			actual, exists := object[key]
			if !exists || !matchesFilterValue(actual, value, exact) {
				return false
			}
		}
		return true
	case []any:
		array, ok := actual.([]any)
		if !ok || len(array) != len(expected) {
			return false
		}
		for i := range expected {
			if !matchesFilterValue(array[i], expected[i], true) {
				return false
			}
		}
		return true
	case json.Number:
		number, ok := actual.(json.Number)
		if !ok {
			return false
		}
		a, aOK := new(big.Rat).SetString(number.String())
		b, bOK := new(big.Rat).SetString(expected.String())
		return aOK && bOK && a.Cmp(b) == 0
	default:
		return reflect.DeepEqual(actual, expected)
	}
}

func excerpt(value string, indices []int) string {
	runes := []rune(value)
	start := utf8.RuneCountInString(value[:indices[0]])
	end := start + utf8.RuneCountInString(value[indices[0]:indices[1]])
	left, right := max(0, start-60), min(len(runes), end+60)
	text := strings.Join(strings.Fields(string(runes[left:right])), " ")
	if left > 0 {
		text = "…" + text
	}
	if right < len(runes) {
		text += "…"
	}
	return text
}
