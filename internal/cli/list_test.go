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

func TestListAndSearchWhere(t *testing.T) {
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, status := range []string{"todo", "done", "in_progress"} {
		source := fmt.Sprintf("---\nid: doc%d\ntitle: Document %d\ndata: {status: %s}\n---\nbody %d", i, i, status, i)
		if err := os.WriteFile(filepath.Join(c.Path(), fmt.Sprintf("%d.md", i)), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	where := `data.status in ["todo", "in_progress"]`
	for _, args := range [][]string{
		{"list", "--where", where, "--offset", "1"},
		{"search", "document", "--where", where, "--filter", "status=in_progress"},
		{"search", "--where", where + ` and id != "doc0"`},
	} {
		var out, stderr bytes.Buffer
		if err := Execute(append(args, "--path", c.Path(), "--json"), nil, &out, &stderr, "test"); err != nil {
			t.Fatalf("%v: %v %s", args, err, &stderr)
		}
		var result collection.SearchResult
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Documents) != 1 || result.Documents[0].ID != "doc2" {
			t.Fatalf("%v: %s", args, &out)
		}
	}
	for _, where := range []string{"data.status ==", "data.status"} {
		var out, stderr bytes.Buffer
		if err := Execute([]string{"list", "--path", c.Path(), "--where", where, "--json"}, nil, &out, &stderr, "test"); err == nil || !strings.Contains(stderr.String(), "where expression") {
			t.Errorf("%q: %v %s", where, err, &stderr)
		}
	}
}

func TestListAndSearchView(t *testing.T) {
	path := t.TempDir()
	config := "---\nschema: {type: object}\napp:\n  views:\n    - title: Open work\n      filter: data.status in [\"todo\", \"in_progress\"]\n---\n"
	c, err := collection.InitConfig(path, []byte(config))
	if err != nil {
		t.Fatal(err)
	}
	for i, status := range []string{"todo", "done", "in_progress"} {
		source := fmt.Sprintf("---\nid: doc%d\ntitle: Document %d\ndata: {status: %s}\n---\nbody %d", i, i, status, i)
		if err := os.WriteFile(filepath.Join(c.Path(), fmt.Sprintf("%d.md", i)), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"list", "--view", "Open work", "--offset", "1"},
		{"search", "document", "--view", "Open work", "--filter", "status=in_progress"},
		{"search", "--view", "Open work", "--where", `id != "doc0"`},
	} {
		var out, stderr bytes.Buffer
		if err := Execute(append(args, "--path", c.Path(), "--json"), nil, &out, &stderr, "test"); err != nil {
			t.Fatalf("%v: %v %s", args, err, &stderr)
		}
		var result collection.SearchResult
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		if len(result.Documents) != 1 || result.Documents[0].ID != "doc2" {
			t.Fatalf("%v: %s", args, &out)
		}
	}
	var out, stderr bytes.Buffer
	if err := Execute([]string{"list", "--path", c.Path(), "--view", "Closed", "--json"}, nil, &out, &stderr, "test"); err == nil || !strings.Contains(stderr.String(), `available views: \"Open work\"`) {
		t.Errorf("unknown view: %v %s", err, &stderr)
	}
	out.Reset()
	if err := Execute([]string{"info", "--path", c.Path(), "--json"}, nil, &out, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	var info struct{ Views []collection.AppView }
	if err := json.Unmarshal(out.Bytes(), &info); err != nil || len(info.Views) != 1 || info.Views[0].Title != "Open work" {
		t.Fatalf("info views: %s %v", &out, err)
	}
}

func TestListAndSearchSort(t *testing.T) {
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, p := range []string{"b", "c", "a", "b"} {
		source := fmt.Sprintf("---\nid: doc%d\ntitle: Document %d\ndata: {p: %s}\n---\nbody", i, i, p)
		if err := os.WriteFile(filepath.Join(c.Path(), fmt.Sprintf("%d.md", i)), []byte(source), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"list", "--sort", "-data.p,-id"},
		{"list", "--sort", "-data.p", "--sort", "-id"},
		{"search", "body", "--sort", "-data.p,-id"},
	} {
		var out, stderr bytes.Buffer
		if err := Execute(append(args, "--path", c.Path(), "--json"), nil, &out, &stderr, "test"); err != nil {
			t.Fatalf("%v: %v %s", args, err, &stderr)
		}
		var result collection.SearchResult
		if err := json.Unmarshal(out.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		var ids []string
		for _, doc := range result.Documents {
			ids = append(ids, doc.ID)
		}
		if strings.Join(ids, ",") != "doc1,doc3,doc0,doc2" {
			t.Fatalf("%v: %v", args, ids)
		}
	}
	var out, stderr bytes.Buffer
	if err := Execute([]string{"list", "--path", c.Path(), "--sort", "body"}, nil, &out, &stderr, "test"); err == nil || !strings.Contains(err.Error(), "invalid sort key") {
		t.Fatalf("invalid key: %v", err)
	}
}
