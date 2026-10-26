package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/means/internal/collection"
)

func TestSearchCommandFiltersAndErrors(t *testing.T) {
	dir := t.TempDir()
	if _, err := collection.Init(dir, []byte(`{"type":"object"}`)); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{
		"a.md": "---\nid: alpha\ntitle: Feature\ndata:\n  type: feature\n  priority: normal\n  count: 42\n  at: 2026-10-26T10:00:00+01:00\n---\nSearch the body",
		"b.md": "---\nid: beta\ntitle: Feature\ndata:\n  type: feature\n  priority: high\n---\nOther body",
		"c.md": "broken",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	args := []string{"search", "--path", dir, "--filter", "priority=normal", "--filter", "type=feature", "--json"}
	if err := Execute(args, nil, &stdout, &stderr, "test"); err != nil || stderr.Len() != 0 {
		t.Fatalf("search failed: %v, stderr=%s", err, stderr.String())
	}
	var result collection.SearchResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || len(result.Documents) != 1 || result.Documents[0].ID != "alpha" || len(result.Diagnostics) != 1 {
		t.Fatalf("search response = %s, %v", stdout.String(), err)
	}
	stdout.Reset()
	if err := Execute([]string{"search", "Search", "--path", dir, "--filter", "count=42", "--json"}, nil, &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || len(result.Documents) != 1 || result.Documents[0].Match == nil || result.Documents[0].Match.Field != "body" {
		t.Fatalf("typed text search = %s, %v", stdout.String(), err)
	}
	for _, flag := range []string{"before", "after", "on-or-before", "on-or-after"} {
		stdout.Reset()
		boundary := "2026-10-26T09:00:00Z"
		if flag == "before" {
			boundary = "2026-10-26T09:00:01Z"
		}
		if flag == "after" {
			boundary = "2026-10-26T08:59:59Z"
		}
		if err := Execute([]string{"search", "--path", dir, "--" + flag, "at=" + boundary, "--json"}, nil, &stdout, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil || len(result.Documents) != 1 || result.Documents[0].ID != "alpha" {
			t.Fatalf("datetime %s: %s %v", flag, stdout.String(), err)
		}
	}
	for _, args := range [][]string{
		{"search", "--before", "at=not-a-date"},
		{"search", "--after", "at"},
		{"search", "--on-or-before", "=2026-10-26T09:00:00Z"},
		{"search", "--filter", "broken"},
		{"search", "--filter", "=value"},
		{"search", "--filter", "type=feature", "--filter", "type=task"},
		{"search", "--filter", `tags=[broken`},
		{"search", "--limit", "-1"},
		{"search", "one", "two"},
	} {
		stdout.Reset()
		stderr.Reset()
		args = append(args, "--path", dir, "--json")
		if err := Execute(args, nil, &stdout, &stderr, "test"); err == nil || stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
			t.Errorf("invalid args %q: stdout=%s stderr=%s err=%v", args, stdout.String(), stderr.String(), err)
		}
	}
	for _, literal := range []bool{false, true} {
		stdout.Reset()
		stderr.Reset()
		args := []string{"search", "body search", "--path", dir, "--json"}
		if literal {
			args = append(args, "--literal")
		}
		if err := Execute(args, nil, &stdout, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		want := 1
		if literal {
			want = 0
		}
		if len(result.Documents) != want {
			t.Fatalf("literal=%v: %s", literal, stdout.String())
		}
	}
	stdout.Reset()
	stderr.Reset()
	if err := Execute([]string{"search", "--help", "--json"}, nil, &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	var help struct{ Description, Usage string }
	if err := json.Unmarshal(stdout.Bytes(), &help); err != nil {
		t.Fatal(err)
	}
	for _, topic := range []string{"--filter", "priority=normal", "case-insensitive", "diagnostics", "--limit", "exit 1"} {
		if !strings.Contains(help.Description+help.Usage, topic) {
			t.Errorf("search help omits %q", topic)
		}
	}
}
