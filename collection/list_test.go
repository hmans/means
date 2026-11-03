package collection

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestListOffset(t *testing.T) {
	c, err := Init(t.TempDir(), []byte(`false`))
	if err != nil {
		t.Fatal(err)
	}
	for filename, source := range map[string]string{
		"a.md": "broken before",
		"b.md": "---\nid: zulu\ntitle: First\ndata: {status: todo, at: '2026-10-01T00:00:00Z'}\n---\nfirst body",
		"c.md": "---\nid: duplicate\ntitle: Duplicate\ndata: {}\n---\n",
		"d.md": "---\nid: duplicate\ntitle: Duplicate\ndata: {}\n---\n",
		"e.md": "---\nid: excluded\ntitle: Excluded\ndata: {status: done}\n---\n",
		"f.md": "---\nid: alpha\ntitle: Second\ndata: {status: todo, at: '2026-10-03T00:00:00Z'}\n---\nsecond body",
		"g.md": "---\nid: bravo\ntitle: Third\ndata: {status: todo, at: '2026-10-04T00:00:00Z'}\n---\nthird body",
		"z.md": "broken after",
	} {
		if err := os.WriteFile(filepath.Join(c.Path(), filename), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		offset, limit int
		ids           []string
		more          bool
	}{
		{0, 1, []string{"zulu"}, true},
		{1, 1, []string{"alpha"}, true},
		{2, 1, []string{"bravo"}, false},
		{1, 2, []string{"alpha", "bravo"}, false},
		{1, 0, []string{"alpha", "bravo"}, false},
		{3, 1, []string{}, false},
		{4, 0, []string{}, false},
		{int(^uint(0) >> 1), 1, []string{}, false},
	} {
		t.Run(fmt.Sprintf("offset=%d/limit=%d", tc.offset, tc.limit), func(t *testing.T) {
			result, err := c.List(context.Background(), ListOptions{
				Offset: tc.offset, Limit: tc.limit, WithBody: true,
				Filter:   map[string]any{"status": "todo"},
				DateTime: []DateTimeFilter{{Field: "at", Op: "on_or_after", Value: "2026-10-01T00:00:00Z"}},
			})
			if err != nil {
				t.Fatal(err)
			}
			ids := []string{}
			for _, doc := range result.Documents {
				ids = append(ids, doc.ID)
				if doc.Body == nil || *doc.Body == "" {
					t.Fatalf("missing body: %+v", doc)
				}
			}
			if !reflect.DeepEqual(ids, tc.ids) || result.HasMore != tc.more || len(result.Diagnostics) != 4 || result.Documents == nil {
				t.Fatalf("unexpected page: %+v, IDs %v", result, ids)
			}
		})
	}
	result, err := c.List(context.Background(), ListOptions{
		Offset: 1, Limit: 1, Filter: map[string]any{"status": "todo"},
		DateTime: []DateTimeFilter{{Field: "at", Op: "after", Value: "2026-10-02T00:00:00Z"}},
	})
	if err != nil || len(result.Documents) != 1 || result.Documents[0].ID != "bravo" || result.HasMore || result.Documents[0].Body != nil {
		t.Fatalf("datetime filter must apply before offset: %+v, %v", result, err)
	}
	for _, opts := range []ListOptions{{Offset: -1}, {Limit: -1}, {DateTime: []DateTimeFilter{{Field: "at", Op: "after", Value: "invalid"}}}} {
		if _, err := c.List(context.Background(), opts); err == nil {
			t.Fatalf("invalid options accepted: %+v", opts)
		}
	}
}
