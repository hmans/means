package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/means/internal/collection"
)

func TestInitSuppliedSchema(t *testing.T) {
	t.Chdir(t.TempDir())
	schema := `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "$defs": {"kind": {"enum": ["fact", "decision"]}},
  "properties": {
    "summary": {"type": "string", "description": "A fact worth remembering."},
    "kind": {"$ref": "#/$defs/kind"}
  },
  "required": ["summary", "kind"],
  "additionalProperties": false
}
`
	if err := os.WriteFile("authored.json", []byte(schema), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"authored.json", "-"} {
		t.Run(source, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), "memory")
			var input io.Reader = unreadableInput{t}
			if source == "-" {
				input = strings.NewReader(schema)
			}
			var stdout, stderr bytes.Buffer
			args := []string{"init", "--path", dir, "--schema", source, "--json"}
			if err := Execute(args, input, &stdout, &stderr, "test"); err != nil {
				t.Fatal(err)
			}
			if stderr.Len() != 0 || !json.Valid(stdout.Bytes()) {
				t.Fatalf("unexpected response: stdout=%s stderr=%s", stdout.String(), stderr.String())
			}
			stored, err := os.ReadFile(filepath.Join(dir, "schema.json"))
			if err != nil || string(stored) != schema {
				t.Fatalf("schema was not preserved: %s, %v", stored, err)
			}
			c, err := collection.Load(dir)
			if err != nil {
				t.Fatal(err)
			}
			if err := c.ValidateData(map[string]any{"summary": "Remember this", "kind": "fact"}); err != nil {
				t.Fatal(err)
			}
			if err := c.ValidateData(map[string]any{"summary": "Remember this", "kind": "invalid"}); err == nil {
				t.Fatal("custom schema constraints were not applied")
			}
			stdout.Reset()
			if err := Execute(args, strings.NewReader(`{"type":"object"}`), &stdout, &stderr, "test"); err == nil {
				t.Fatal("reinitialization overwrote the schema")
			}
			after, err := os.ReadFile(filepath.Join(dir, "schema.json"))
			if err != nil || !bytes.Equal(after, stored) {
				t.Fatalf("reinitialization changed the schema: %s, %v", after, err)
			}
		})
	}
}

func TestInitInvalidSchemaInput(t *testing.T) {
	for _, schema := range []string{
		"", "  ", "{", "null", `{} {}`, `{"type":"invalid"}`,
		`{"properties":{"summary":{"type":42}}}`, `{"$ref":"external.json"}`,
	} {
		for _, source := range []string{"file", "stdin"} {
			t.Run(source+"/"+schema, func(t *testing.T) {
				base := t.TempDir()
				dir := filepath.Join(base, "memory")
				arg := "-"
				if source == "file" {
					arg = filepath.Join(base, "input.json")
					if err := os.WriteFile(arg, []byte(schema), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				var stdout, stderr bytes.Buffer
				args := []string{"init", "--path", dir, "--schema", arg, "--json"}
				if err := Execute(args, strings.NewReader(schema), &stdout, &stderr, "test"); err == nil {
					t.Fatal("invalid schema was accepted")
				}
				var response struct{ Error string }
				if err := json.Unmarshal(stderr.Bytes(), &response); err != nil || response.Error == "" || stdout.Len() != 0 {
					t.Fatalf("expected a JSON error: stdout=%s stderr=%s", stdout.String(), stderr.String())
				}
				if _, err := os.Stat(dir); !os.IsNotExist(err) {
					t.Fatalf("invalid schema left a directory behind: %v", err)
				}
			})
		}
	}
}

type unreadableInput struct{ t *testing.T }

func (r unreadableInput) Read([]byte) (int, error) {
	r.t.Fatal("stdin must not be read")
	return 0, io.EOF
}

type failingReader struct{ err error }

func (r failingReader) Read([]byte) (int, error) { return 0, r.err }

func TestInitStdinFailure(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "memory")
	want := errors.New("stdin unavailable")
	var stdout, stderr bytes.Buffer
	err := Execute([]string{"init", "--path", dir, "--schema", "-", "--json"}, failingReader{want}, &stdout, &stderr, "test")
	if !errors.Is(err, want) || stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
		t.Fatalf("unexpected read failure: %v, stdout=%s stderr=%s", err, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("read failure left a directory behind: %v", err)
	}
}

func TestInitHelp(t *testing.T) {
	for _, args := range [][]string{
		{"init", "--help", "--schema", "-"},
		{"help", "init"},
		{"init", "--help", "--json", "--schema", "-"},
		{"help", "init", "--json"},
	} {
		var stdout, stderr bytes.Buffer
		if err := Execute(args, unreadableInput{t}, &stdout, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		output := stdout.String()
		if strings.Contains(strings.Join(args, " "), "--json") {
			var help struct{ Description, Usage string }
			if err := json.Unmarshal(stdout.Bytes(), &help); err != nil {
				t.Fatal(err)
			}
			output = help.Description + help.Usage
		}
		for _, topic := range []string{"--schema -", "2020-12", "#/$defs/status", "required", "additionalProperties", "Refuse to overwrite", "exit 1", "EOF", "<<'JSON'"} {
			if !strings.Contains(output, topic) {
				t.Errorf("help for %q omits %q", args, topic)
			}
		}
		if stderr.Len() != 0 {
			t.Errorf("help wrote to stderr: %s", stderr.String())
		}
	}
}
