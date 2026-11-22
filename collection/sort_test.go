package collection

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func sortTestCollection(t *testing.T) *Collection {
	t.Helper()
	c, err := Init(t.TempDir(), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	for filename, source := range map[string]string{
		"1.md": "---\nid: e\ntitle: Echo note\ndata: {rank: 10, meta: {at: b}}\n---\nnote",
		"2.md": "---\nid: d\ntitle: Delta\ndata: {rank: 9.5, meta: {at: a}}\n---\nnote",
		"3.md": "---\nid: c\ntitle: Charlie\ndata: {rank: 10}\n---\nnote",
		"4.md": "---\nid: b\ntitle: Bravo\ndata: {rank: two, meta: x}\n---\nnote",
		"5.md": "---\nid: a\ntitle: Alpha\ndata: {rank: true}\n---\nnote",
		"6.md": "---\nid: f\ntitle: Foxtrot\ndata: {rank: null}\n---\nnote",
		"7.md": "---\nid: g\ntitle: Golf\ndata: {rank: [1]}\n---\nnote",
	} {
		if err := os.WriteFile(filepath.Join(c.Path(), filename), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

func listIDs(t *testing.T, c *Collection, opts ListOptions) ([]string, bool) {
	t.Helper()
	result, err := c.List(context.Background(), opts)
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, doc := range result.Documents {
		ids = append(ids, doc.ID)
	}
	return ids, result.HasMore
}

func TestListSort(t *testing.T) {
	c := sortTestCollection(t)
	for _, tc := range []struct {
		sort []string
		ids  []string
	}{
		{nil, []string{"e", "d", "c", "b", "a", "f", "g"}},
		{[]string{"id"}, []string{"a", "b", "c", "d", "e", "f", "g"}},
		{[]string{"-title"}, []string{"g", "f", "e", "d", "c", "b", "a"}},
		// Mixed types: booleans, numbers (exact), strings, arrays; null sorts last.
		{[]string{"data.rank"}, []string{"a", "d", "c", "e", "b", "g", "f"}},
		// Descending reverses values but keeps missing/null last; ID breaks ties.
		{[]string{"-data.rank"}, []string{"g", "b", "c", "e", "d", "a", "f"}},
		{[]string{"-data.rank", "-id"}, []string{"g", "b", "e", "c", "d", "a", "f"}},
		// Nested keys; paths through non-objects count as missing.
		{[]string{"data.meta.at"}, []string{"d", "e", "a", "b", "c", "f", "g"}},
		{[]string{"-data.meta.at"}, []string{"e", "d", "a", "b", "c", "f", "g"}},
	} {
		t.Run(strings.Join(tc.sort, ","), func(t *testing.T) {
			if ids, _ := listIDs(t, c, ListOptions{Sort: tc.sort}); !reflect.DeepEqual(ids, tc.ids) {
				t.Fatalf("ids = %v, want %v", ids, tc.ids)
			}
		})
	}
	// Sort applies before offset and limit.
	ids, more := listIDs(t, c, ListOptions{Sort: []string{"id"}, Offset: 2, Limit: 2})
	if !reflect.DeepEqual(ids, []string{"c", "d"}) || !more {
		t.Fatalf("page = %v, more = %v", ids, more)
	}
	ids, more = listIDs(t, c, ListOptions{Sort: []string{"id"}, Offset: 5, Limit: 2})
	if !reflect.DeepEqual(ids, []string{"f", "g"}) || more {
		t.Fatalf("last page = %v, more = %v", ids, more)
	}
}

func TestSortKeyValidation(t *testing.T) {
	c := sortTestCollection(t)
	for _, key := range []string{"", "-", "body", "data", "data.", "data..x", "title.x", "frontmatter.data.x", "--id"} {
		if _, err := c.List(context.Background(), ListOptions{Sort: []string{key}}); err == nil || !strings.Contains(err.Error(), "invalid sort key") {
			t.Fatalf("%q: err = %v", key, err)
		}
	}
	if _, err := c.Search(context.Background(), SearchOptions{Sort: []string{"id", "-id"}}); err == nil || !strings.Contains(err.Error(), "duplicate sort key") {
		t.Fatalf("duplicate: err = %v", err)
	}
}

func TestSearchSort(t *testing.T) {
	c := sortTestCollection(t)
	search := func(opts SearchOptions) ([]string, bool) {
		t.Helper()
		result, err := c.Search(context.Background(), opts)
		if err != nil {
			t.Fatal(err)
		}
		ids := []string{}
		for _, hit := range result.Documents {
			ids = append(ids, hit.ID)
			if opts.Query != "" && hit.Match == nil {
				t.Fatalf("missing excerpt: %+v", hit)
			}
		}
		return ids, result.HasMore
	}
	// Full text ranks e first (boosted title match); sort replaces relevance order.
	if ids, _ := search(SearchOptions{Query: "note", Limit: 1}); !reflect.DeepEqual(ids, []string{"e"}) {
		t.Fatalf("relevance = %v", ids)
	}
	if ids, more := search(SearchOptions{Query: "note", Sort: []string{"-id"}, Limit: 2}); !reflect.DeepEqual(ids, []string{"g", "f"}) || !more {
		t.Fatalf("full-text sort = %v, %v", ids, more)
	}
	// Sort ties keep relevance order: c and e share rank 10, and e ranks higher.
	if ids, _ := search(SearchOptions{Query: "note", Sort: []string{"data.rank"}, Where: "type(data.rank) in ['int', 'float']"}); !reflect.DeepEqual(ids, []string{"d", "e", "c"}) {
		t.Fatalf("full-text ties = %v", ids)
	}
	if ids, _ := search(SearchOptions{Query: "o", Literal: true, Sort: []string{"-title"}, Limit: 3}); !reflect.DeepEqual(ids, []string{"g", "f", "e"}) {
		t.Fatalf("literal sort = %v", ids)
	}
	if ids, more := search(SearchOptions{Sort: []string{"data.meta.at"}, Limit: 2}); !reflect.DeepEqual(ids, []string{"d", "e"}) || !more {
		t.Fatalf("data-only sort = %v, %v", ids, more)
	}
}

func TestEnumSort(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`{
		"$defs": {"priority": {"enum": ["low", "normal", "high", "urgent"]}},
		"type": "object",
		"properties": {
			"priority": {"$ref": "#/$defs/priority"},
			"meta": {"properties": {"size": {"type": "number", "enum": [3, 1, 2]}}},
			"combined": {"allOf": [{"enum": ["b", "a"]}]}
		}
	}`))
	if err != nil {
		t.Fatal(err)
	}
	for filename, source := range map[string]string{
		"1.md": "---\nid: a\ntitle: A\ndata: {priority: high, meta: {size: 1}, combined: b}\n---\n",
		"2.md": "---\nid: b\ntitle: B\ndata: {priority: low, meta: {size: 2}, combined: a}\n---\n",
		"3.md": "---\nid: c\ntitle: C\ndata: {priority: urgent, meta: {size: 3}}\n---\n",
		"4.md": "---\nid: d\ntitle: D\ndata: {priority: aaa}\n---\n",
		"5.md": "---\nid: e\ntitle: E\ndata: {priority: normal, meta: {size: 9}}\n---\n",
		"6.md": "---\nid: f\ntitle: F\ndata: {}\n---\n",
	} {
		if err := os.WriteFile(filepath.Join(c.Path(), filename), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		sort []string
		ids  []string
	}{
		// Enum order through $ref; values outside the enum follow; missing last.
		{[]string{"data.priority"}, []string{"b", "e", "a", "c", "d", "f"}},
		{[]string{"-data.priority"}, []string{"d", "c", "a", "e", "b", "f"}},
		// Nested properties and non-string enums.
		{[]string{"data.meta.size"}, []string{"c", "a", "b", "e", "d", "f"}},
		// Combinators are not followed: literal order.
		{[]string{"data.combined"}, []string{"b", "a", "c", "d", "e", "f"}},
	} {
		t.Run(strings.Join(tc.sort, ","), func(t *testing.T) {
			if ids, _ := listIDs(t, c, ListOptions{Sort: tc.sort}); !reflect.DeepEqual(ids, tc.ids) {
				t.Fatalf("ids = %v, want %v", ids, tc.ids)
			}
		})
	}
}
