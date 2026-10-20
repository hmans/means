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
		"a.md": "---\nid: alpha\ntitle: Feature\ndata:\n  type: feature\n  priority: normal\n  count: 42\n---\nSearch the body",
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
	for _, args := range [][]string{
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
