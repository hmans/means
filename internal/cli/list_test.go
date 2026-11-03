package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/means/collection"
)

func TestListOffset(t *testing.T) {
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for i := range 5 {
		status := "todo"
		if i == 2 {
			status = "done"
		}
		source := fmt.Sprintf("---\nid: doc%d\ntitle: Document %d\ndata: {status: %s, at: '2026-10-0%dT00:00:00Z'}\n---\nbody %d", i, i, status, i+1, i)
		if err := os.WriteFile(filepath.Join(c.Path(), fmt.Sprintf("%d.md", i)), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(c.Path(), "broken.md"), []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		offset, limit string
		count         int
		more          bool
	}{
		{"0", "1", 1, true}, {"1", "1", 1, true}, {"1", "0", 2, false},
		{"2", "1", 1, false}, {"3", "1", 0, false}, {"100", "0", 0, false},
	} {
		args := []string{"list", "--path", c.Path(), "--filter", "status=todo", "--after", "at=2026-10-01T00:00:00Z", "--offset", tc.offset, "--limit", tc.limit}
		var out, stderr bytes.Buffer
		if err := Execute(append(args, "--json", "--with-body"), nil, &out, &stderr, "test"); err != nil {
			t.Fatalf("%v: %v %s", args, err, &stderr)
		}
		var result collection.ListResult
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Documents) != tc.count || result.HasMore != tc.more || len(result.Diagnostics) != 1 || strings.Contains(out.String(), `"hash"`) {
			t.Fatalf("%v: %s", args, &out)
		}
		if tc.offset == "1" && tc.count > 0 {
			doc := result.Documents[0]
			if doc.ID != "doc3" || doc.Body == nil || *doc.Body != "body 3" {
				t.Fatalf("incorrect filtered page: %+v", doc)
			}
		}
		out.Reset()
		if err := Execute(args, nil, &out, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		if tc.offset == "1" && (!strings.Contains(out.String(), "doc3") || strings.Contains(out.String(), "doc1")) {
			t.Fatalf("incorrect text page: %s", &out)
		}
	}
}
