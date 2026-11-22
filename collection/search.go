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
	"time"
	"unicode/utf8"
)

// SearchOptions selects a search mode and filters its results. The zero value
// lists every valid document in filename order. Filter, DateTime, View, and
// Where conditions are ANDed and apply before Sort and Limit in every mode.
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
	// Where is an Expr boolean expression over id, title, created_at, updated_at,
	// body, data, and now();
	// see Expression. Blank means no condition. Compile errors, runtime errors,
	// and non-boolean results fail the operation.
	Where string
	where *Expression
	// View names a MEANS.md app.views entry by exact title and ANDs its filter,
	// following Where rules. Unknown titles fail and list the configured views.
	// Blank means no view. Only the filter applies; columns are TUI-only.
	View string
	view *Expression
	now  time.Time
	// Literal selects a case-insensitive substring match in title or body,
	// ordered by filename. It cannot be combined with Semantic.
	Literal bool
	// Sort orders matches by metadata keys before Limit, replacing relevance
	// and filename order. Each key is id, title, created_at, updated_at (compared
	// as instants), or data.FIELD (dots address
	// nested objects), prefixed with - for descending. Later keys break ties.
	// Missing and null values sort last in either direction. Mixed types order
	// booleans (false first), numbers (exact), strings (Unicode code points),
	// arrays, then objects; arrays and objects tie. A data field whose schema
	// declares an enum (reached through properties and local $ref) orders
	// enum values by position, before other values. Remaining ties keep
	// relevance order for ranked searches and use ID otherwise. Nil or empty
	// keeps the mode's default order.
	Sort     []string
	sortKeys []sortKey
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
	// Documents contains matches in rank order, filename order for scans, or
	// Sort order when requested.
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
// Full-text and semantic searches sort rank ties by ID. Sort replaces these
// orders and applies before Limit and HasMore. Full-text search writes
// a disposable .means/ cache; literal and data-only searches remain offline.
func (c *Collection) Search(ctx context.Context, opts SearchOptions) (*SearchResult, error) {
	c, err := c.operation(ctx)
	if err != nil {
		return nil, err
	}
	if err := opts.prepare(c.schema, c.app); err != nil {
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

// prepare normalizes Filter, resolves View in app, and compiles DateTime,
// View, and Where conditions.
func (opts *SearchOptions) prepare(schema json.RawMessage, app AppConfig) error {
	var err error
	if opts.Filter != nil {
		if opts.Filter, err = normalizeObject(opts.Filter); err != nil {
			return fmt.Errorf("invalid filter: %w", err)
		}
	}
	if opts.dateTimeFilters, err = compileDateTimeFilters(opts.DateTime); err != nil {
		return err
	}
	if strings.TrimSpace(opts.Where) != "" {
		if opts.where, err = CompileExpression(opts.Where); err != nil {
			return fmt.Errorf("invalid where expression: %w", err)
		}
	}
	if opts.View != "" {
		view, err := app.View(opts.View)
		if err != nil {
			return err
		}
		if strings.TrimSpace(view.Filter) != "" {
			if opts.view, err = CompileExpression(view.Filter); err != nil {
				return fmt.Errorf("invalid filter in view %q: %w", view.Title, err)
			}
		}
	}
	if opts.sortKeys, err = compileSortKeys(opts.Sort, schema); err != nil {
		return err
	}
	opts.now = time.Now()
	return nil
}

// page sorts matches when requested, then applies offset and Limit.
func (opts SearchOptions) page(result *SearchResult, offset int, ranked bool) {
	sortHits(result.Documents, opts.sortKeys, ranked)
	result.Documents = result.Documents[min(offset, len(result.Documents)):]
	if opts.Limit > 0 && len(result.Documents) > opts.Limit {
		result.HasMore = true
		result.Documents = result.Documents[:opts.Limit]
	}
}

// matches applies Filter, DateTime, View, and Where to one valid document.
func (opts SearchOptions) matches(frontmatter map[string]any, body string) (bool, error) {
	if !opts.matchesData(frontmatter["data"].(map[string]any)) {
		return false, nil
	}
	if opts.view != nil {
		match, err := opts.view.Match(frontmatter, body, opts.now)
		if err != nil {
			return false, fmt.Errorf("view %q filter failed for document %v: %w", opts.View, frontmatter["id"], err)
		}
		if !match {
			return false, nil
		}
	}
	if opts.where == nil {
		return true, nil
	}
	match, err := opts.where.Match(frontmatter, body, opts.now)
	if err != nil {
		return false, fmt.Errorf("where expression failed for document %v: %w", frontmatter["id"], err)
	}
	return match, nil
}

// searchScan returns matches in filename order, or Sort order, and diagnostics
// for the whole collection.
// Callers prepare options before scanning.
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
		if ok, err := opts.matches(entry.doc.Frontmatter, entry.doc.Body); err != nil {
			return nil, err
		} else if !ok {
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
		hit := SearchHit{ID: id, Frontmatter: entry.doc.Frontmatter, Match: match}
		if opts.WithBody {
			hit.Body = &entry.doc.Body
		}
		result.Documents = append(result.Documents, hit)
	}
	opts.page(result, offset, false)
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
