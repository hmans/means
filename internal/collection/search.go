package collection

import (
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

type SearchOptions struct {
	MinScore *float64 // Semantic cosine threshold; nil uses DefaultSemanticMinScore.
	WithBody bool
	Query    string
	Filter   map[string]any // Partial match against frontmatter.data.
	Limit    int            // Zero returns every match.
	Literal  bool           // Use the original case-insensitive substring search.
	Semantic bool           // Use opt-in OpenAI embeddings and cosine ranking.
}

const DefaultSemanticMinScore = 0.3

type SearchMatch struct {
	Field   string `json:"field"` // title or body
	Excerpt string `json:"excerpt"`
}

type SearchHit struct {
	Body        *string        `json:"body,omitempty"`
	ID          string         `json:"id"`
	Frontmatter map[string]any `json:"frontmatter"`
	Match       *SearchMatch   `json:"match,omitempty"`
	Score       *float64       `json:"score,omitempty"`
}

type SearchResult struct {
	Documents   []SearchHit  `json:"documents"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	HasMore     bool         `json:"has_more"`
}

// Search ranks full-text queries using a refreshed collection-local index.
// Literal and data-only searches scan in filename order. Filter is a recursive partial match of
// application data; arrays must match completely. Schema-invalid data stays
// searchable, while malformed and duplicate documents become diagnostics.
func (c *Collection) Search(opts SearchOptions) (*SearchResult, error) {
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
	if opts.Semantic {
		if opts.Literal || strings.TrimSpace(opts.Query) == "" {
			return nil, fmt.Errorf("semantic search requires a nonempty query and cannot be combined with literal search")
		}
		return c.searchSemantic(opts)
	}
	if opts.Query != "" && !opts.Literal {
		return c.searchFullText(opts)
	}
	var pattern *regexp.Regexp
	if opts.Query != "" {
		var err error
		pattern, err = regexp.Compile("(?i)" + regexp.QuoteMeta(opts.Query))
		if err != nil {
			return nil, fmt.Errorf("compile query: %w", err)
		}
	}
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
		if !matchesFilter(data, opts.Filter) {
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

func matchesFilter(data, filter map[string]any) bool {
	for key, expected := range filter {
		actual, exists := data[key]
		if !exists || !matchesValue(actual, expected) {
			return false
		}
	}
	return true
}

func matchesValue(actual, expected any) bool {
	switch expected := expected.(type) {
	case map[string]any:
		object, ok := actual.(map[string]any)
		return ok && matchesFilter(object, expected)
	case []any:
		array, ok := actual.([]any)
		if !ok || len(array) != len(expected) {
			return false
		}
		for i := range expected {
			if !matchesValue(array[i], expected[i]) {
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
