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

func TestDocumentWorkflow(t *testing.T) {
	t.Chdir(t.TempDir())
	dir := filepath.Join(t.TempDir(), "memory")
	var stdout, stderr bytes.Buffer
	schema := `{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"],"additionalProperties":false}`
	if err := Execute([]string{"init", "--path", dir, "--schema", "-", "--json"}, strings.NewReader(schema), &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	envelope := `{"frontmatter":{"summary":"Use Go for means"},"body":"# Decision\n\nSingle-binary distribution.\n"}`
	if err := os.WriteFile("input.json", []byte(envelope), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"-", "input.json"} {
		stdout.Reset()
		if err := Execute([]string{"--path", dir, "create", "--input", source, "--json"}, strings.NewReader(envelope), &stdout, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		var created collection.Document
		if err := json.Unmarshal(stdout.Bytes(), &created); err != nil {
			t.Fatal(err)
		}
		if len(created.ID) != 32 || created.Frontmatter["summary"] != "Use Go for means" {
			t.Fatalf("unexpected creation: %s", stdout.String())
		}
		stdout.Reset()
		if err := Execute([]string{"read", created.ID, "--path", dir, "--json"}, unreadableInput{t}, &stdout, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		var read collection.Document
		if err := json.Unmarshal(stdout.Bytes(), &read); err != nil {
			t.Fatal(err)
		}
		if read.ID != created.ID || read.Body != created.Body || read.Frontmatter["summary"] != created.Frontmatter["summary"] {
			t.Fatalf("read changed document: %#v", read)
		}
		stdout.Reset()
		if err := Execute([]string{"read", created.ID, "--path", dir}, unreadableInput{t}, &stdout, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		stored, err := os.ReadFile(filepath.Join(dir, created.ID+".md"))
		if err != nil || !bytes.Equal(stdout.Bytes(), stored) {
			t.Fatalf("text read differs from file: %v", err)
		}
	}
	stdout.Reset()
	if err := Execute([]string{"list", "--path", dir, "--limit", "1", "--json"}, unreadableInput{t}, &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	var listed collection.ListResult
	if err := json.Unmarshal(stdout.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Documents) != 1 || !listed.HasMore || strings.Contains(stdout.String(), `"body"`) || stderr.Len() != 0 {
		t.Fatalf("unexpected listing: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
	stdout.Reset()
	if err := Execute([]string{"list", "--path", dir, "--limit", "0", "--json"}, nil, &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(stdout.Bytes(), &listed); err != nil || len(listed.Documents) != 2 || listed.HasMore {
		t.Fatalf("unexpected unlimited listing: %s, %v", stdout.String(), err)
	}
}

func TestCreateInvalidInputDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	if _, err := collection.Init(dir, nil); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{
		``, `{`, `null`, `[]`, `{}`, `{"frontmatter":null}`, `{"frontmatter":[]}`,
		`{"frontmatter":{},"body":null}`, `{"frontmatter":{},"body":1}`,
		`{"frontmatter":{"title":"ok"},"id":"not-allowed"}`, `{"frontmatter":{}}`,
		`{"frontmatter":{"title":false}}`, `{"frontmatter":{"title":"ok","unknown":1}}`,
		`{"frontmatter":{"title":"ok"}} {}`,
	} {
		t.Run(input, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Execute([]string{"create", "--input", "-", "--path", dir, "--json"}, strings.NewReader(input), &stdout, &stderr, "test")
			if err == nil || stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
				t.Fatalf("unexpected error response: %v, %s, %s", err, stdout.String(), stderr.String())
			}
		})
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Fatalf("invalid creates wrote files: %v, %v", entries, err)
	}
}

func TestDocumentCommandErrorsAndHelp(t *testing.T) {
	dir := t.TempDir()
	if _, err := collection.Init(dir, nil); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"create"}, {"create", "--input="}, {"create", "--input", "missing.json"},
		{"read"}, {"read", "missing"}, {"read", "../outside"}, {"list", "--limit", "-1"},
	} {
		var stdout, stderr bytes.Buffer
		args = append(args, "--path", dir, "--json")
		if err := Execute(args, unreadableInput{t}, &stdout, &stderr, "test"); err == nil || stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
			t.Fatalf("unexpected response for %q: %v, %s, %s", args, err, stdout.String(), stderr.String())
		}
	}
	for command, topics := range map[string][]string{
		"create": {"--input -", "frontmatter", "schema.json", "32-character", "exit 1", "<<'JSON'"},
		"read":   {"filename", "--json", "schema", "exit 1"},
		"list":   {"--limit", "has_more", "Malformed", "exit 1"},
	} {
		for _, jsonMode := range []bool{false, true} {
			args := []string{command, "--help"}
			if jsonMode {
				args = append(args, "--json")
			}
			var stdout, stderr bytes.Buffer
			if err := Execute(args, unreadableInput{t}, &stdout, &stderr, "test"); err != nil {
				t.Fatal(err)
			}
			output := stdout.String()
			if jsonMode {
				var help struct{ Description, Usage string }
				if err := json.Unmarshal(stdout.Bytes(), &help); err != nil {
					t.Fatal(err)
				}
				output = help.Description + help.Usage
			}
			for _, topic := range topics {
				if !strings.Contains(output, topic) {
					t.Errorf("%s help omits %q", command, topic)
				}
			}
		}
	}
}

func TestCreateTextIDAndOptionalBody(t *testing.T) {
	dir := t.TempDir()
	if _, err := collection.Init(dir, nil); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := Execute([]string{"create", "--path", dir, "--input", "-"}, strings.NewReader(`{"frontmatter":{"title":"Remember"}}`), &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	id := strings.TrimSpace(stdout.String())
	stdout.Reset()
	if err := Execute([]string{"read", id, "--path", dir, "--json"}, nil, &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	var doc collection.Document
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil || doc.Body != "" {
		t.Fatalf("unexpected optional body: %s, %v", stdout.String(), err)
	}
}
