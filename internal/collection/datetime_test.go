package collection

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDateTimeComparisons(t *testing.T) {
	for _, tc := range []struct {
		value      string
		comparison int
	}{
		{"2026-10-26T09:00:00Z", 0},
		{"2026-10-26t10:00:00+01:00", 0},
		{"2026-10-26T04:00:00-05:00", 0},
		{"2026-10-26T09:00:00.000000001Z", 1},
		{"2026-10-26T08:59:59.999999999z", -1},
	} {
		for _, op := range []string{"before", "after", "on_or_before", "on_or_after"} {
			t.Run(tc.value+op, func(t *testing.T) {
				filters, err := compileDateTimeFilters([]DateTimeFilter{{"at", op, "2026-10-26T09:00:00Z"}})
				if err != nil {
					t.Fatal(err)
				}
				opts := SearchOptions{dateTimeFilters: filters}
				want := (op == "before" && tc.comparison < 0) || (op == "after" && tc.comparison > 0) || (op == "on_or_before" && tc.comparison <= 0) || (op == "on_or_after" && tc.comparison >= 0)
				if got := opts.matchesData(map[string]any{"at": tc.value}); got != want {
					t.Fatalf("got %v want %v", got, want)
				}
			})
		}
	}
	for _, value := range []string{"", "2026-10-26", "2026-10-26T09:00:00", "2026-02-30T09:00:00Z", "2026-10-26T24:00:00Z", "2026-10-26T09:00:60Z", "2026-10-26T09:00:00+24:00", "2026-10-26T09:00:00+01:60", "2026-10-26T09:00:00.1234567891Z", "2026-10-26T09:00:00,1Z"} {
		if _, err := parseDateTime(value); err == nil {
			t.Errorf("accepted %q", value)
		}
	}
	for _, filter := range []DateTimeFilter{{"", "before", "2026-10-26T09:00:00Z"}, {"at", "typo", "2026-10-26T09:00:00Z"}, {"at", "before", "bad"}} {
		// Validate even with no collection and before semantic configuration/API work.
		if _, err := (&Collection{}).Search(SearchOptions{DateTime: []DateTimeFilter{filter}, Semantic: true, Query: "test"}); err == nil {
			t.Errorf("accepted %+v", filter)
		}
	}
}

func TestDateTimeSearchModes(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"a": "2026-10-25T23:00:00Z", "b": "2026-10-26T01:00:00+01:00", "c": "2026-10-26T00:00:00.000000001Z", "d": "2026-10-27T00:00:00Z", "e": "null", "f": "42", "g": "invalid", "h": "2026-10-26",
	} {
		if err := os.WriteFile(filepath.Join(c.Path, name+".md"), []byte("---\nid: "+name+"\ntitle: Event\ndata:\n  at: "+value+"\n  status: open\n---\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(c.Path, "broken.md"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"data", "literal", "fulltext", "warm"} {
		opts := SearchOptions{Limit: 1, DateTime: []DateTimeFilter{{"at", "on_or_after", "2026-10-26T00:00:00Z"}, {"at", "before", "2026-10-27T00:00:00Z"}}, Filter: map[string]any{"status": "open"}}
		if mode != "data" {
			opts.Query = "event"
		}
		opts.Literal = mode == "literal"
		result, err := c.Search(opts)
		if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != "b" || !result.HasMore || len(result.Diagnostics) != 1 {
			t.Fatalf("%s: %+v %v", mode, result, err)
		}
		opts.DateTime[0].Op = "after"
		result, err = c.Search(opts)
		if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != "c" || result.HasMore {
			t.Fatalf("%s exclusive: %+v %v", mode, result, err)
		}
		opts.Filter["status"] = "closed"
		result, err = c.Search(opts)
		if err != nil || len(result.Documents) != 0 {
			t.Fatalf("%s AND: %+v %v", mode, result, err)
		}
	}
}

func TestDateTimeSemanticSearch(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	if _, err := c.Create(map[string]any{"title": "apple", "data": map[string]any{"at": "2026-10-25T00:00:00Z"}}, "orchard"); err != nil {
		t.Fatal(err)
	}
	later, err := c.Create(map[string]any{"title": "engine", "data": map[string]any{"at": "2026-10-26T00:00:00Z"}}, "machine")
	if err != nil {
		t.Fatal(err)
	}
	all := -1.0
	opts := SearchOptions{Query: "fruit", Semantic: true, MinScore: &all, Limit: 1, DateTime: []DateTimeFilter{{"at", "on_or_after", "2026-10-26T00:00:00Z"}}}
	result, err := c.Search(opts)
	if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != later.ID || result.HasMore {
		t.Fatalf("semantic: %+v %v", result, err)
	}
	before := len(api.inputs)
	opts.DateTime[0].Op = "after"
	result, err = c.Search(opts)
	if err != nil || len(result.Documents) != 0 || len(api.inputs) != before {
		t.Fatalf("empty semantic: %+v %v", result, err)
	}
}
