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

func TestUpdateCLIWorkflow(t *testing.T) {
	t.Chdir(t.TempDir())
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	created, err := c.Create(map[string]any{"title": "Before", "tags": []string{"keep"}}, "Body\n")
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if err := Execute([]string{"read", created.ID, "--path", c.Path, "--json"}, nil, &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	var doc collection.Document
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"-", "patch.json"} {
		input := `{"hash":"` + doc.Hash + `","frontmatter":{"title":"` + source + `"}}`
		if err := os.WriteFile("patch.json", []byte(input), 0o644); err != nil {
			t.Fatal(err)
		}
		stdout.Reset()
		if err := Execute([]string{"update", doc.ID, "--input", source, "--path", c.Path, "--json"}, strings.NewReader(input), &stdout, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		previousHash := doc.Hash
		if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		if doc.ID != created.ID || doc.Hash == previousHash || doc.Frontmatter["title"] != source || doc.Body != "Body\n" || doc.Frontmatter["tags"].([]any)[0] != "keep" {
			t.Fatalf("update result: %s", stdout.String())
		}
	}
	stdout.Reset()
	input := `{"hash":"` + doc.Hash + `","body":""}`
	if err := Execute([]string{"update", doc.ID, "--path", c.Path, "--input", "-"}, strings.NewReader(input), &stdout, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != doc.ID+"\n" || stderr.Len() != 0 {
		t.Fatalf("unexpected streams: %q %q", stdout.String(), stderr.String())
	}
	read, err := c.Read(doc.ID)
	if err != nil || read.Body != "" {
		t.Fatalf("body not cleared: %+v %v", read, err)
	}
	// A successful content-changing request cannot be replayed with its old hash.
	stdout.Reset()
	if err := Execute([]string{"update", doc.ID, "--path", c.Path, "--input", "-", "--json"}, strings.NewReader(input), &stdout, &stderr, "test"); err == nil || stdout.Len() != 0 || !json.Valid(stderr.Bytes()) || !strings.Contains(stderr.String(), "read it again") {
		t.Fatalf("missing conflict response: %v %s", err, stderr.String())
	}
}

func TestUpdateCLIInvalidInput(t *testing.T) {
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(map[string]any{"title": "Keep"}, "Body")
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{
		``, `{`, `null`, `[]`, `{}`, `{"hash":42,"body":"x"}`, `{"hash":null,"body":"x"}`,
		`{"hash":"","body":"x"}`, `{"hash":"wrong","body":"x"}`,
		`{"hash":"HASH"}`, `{"hash":"HASH","frontmatter":null}`, `{"hash":"HASH","frontmatter":[]}`,
		`{"hash":"HASH","body":null}`, `{"hash":"HASH","body":42}`,
		`{"hash":"HASH","frontmatter":{"title":null}}`, `{"hash":"HASH","body":"x","force":true}`,
		`{"hash":"HASH","body":"x"} {}`,
	} {
		t.Run(input, func(t *testing.T) {
			input = strings.ReplaceAll(input, "HASH", doc.Hash)
			var stdout, stderr bytes.Buffer
			if err := Execute([]string{"update", doc.ID, "--path", c.Path, "--input", "-", "--json"}, strings.NewReader(input), &stdout, &stderr, "test"); err == nil || stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
				t.Fatalf("invalid input response: %v %s %s", err, stdout.String(), stderr.String())
			}
			data, err := os.ReadFile(filepath.Join(c.Path, doc.ID+".md"))
			if err != nil || string(data) != doc.Markdown {
				t.Fatalf("failed patch changed file: %q %v", data, err)
			}
		})
	}
	for _, args := range [][]string{{"update"}, {"update", doc.ID}, {"update", doc.ID, "--input", "missing.json"}} {
		var stdout, stderr bytes.Buffer
		if err := Execute(append(args, "--path", c.Path, "--json"), unreadableInput{t}, &stdout, &stderr, "test"); err == nil || stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
			t.Fatalf("invalid command response: %v %s", err, stderr.String())
		}
	}
}
