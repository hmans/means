package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/means/collection"
)

func TestAppendCLIWorkflow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(context.Background(), collection.CreateOptions{Title: "Note", Body: "Original"})
	if err != nil {
		t.Fatal(err)
	}
	text := "\n\n## Addition\nExact text.\n"
	bodyFile := filepath.Join(t.TempDir(), "addition.md")
	if err := os.WriteFile(bodyFile, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"inline", "body file", "body stdin", "JSON file", "JSON stdin", "empty"} {
		for _, withHash := range []bool{false, true} {
			name := mode
			if withHash {
				name += " with hash"
			}
			t.Run(name, func(t *testing.T) {
				addition := text
				args := []string{"append", doc.ID, "--path", c.Path()}
				input := ""
				switch mode {
				case "inline":
					args = append(args, text)
				case "body file":
					args = append(args, "--body-file", bodyFile)
				case "body stdin":
					args = append(args, "--body-file", "-")
					input = text
				case "JSON file", "JSON stdin":
					envelope := map[string]string{"body": text}
					if withHash {
						envelope["hash"] = doc.Hash
					}
					data, err := json.Marshal(envelope)
					if err != nil {
						t.Fatal(err)
					}
					input = string(data)
					source := "-"
					if mode == "JSON file" {
						source = filepath.Join(t.TempDir(), "append.json")
						if err := os.WriteFile(source, data, 0o600); err != nil {
							t.Fatal(err)
						}
					}
					args = append(args, "--input", source)
				case "empty":
					addition = ""
					args = append(args, "")
				}
				if withHash && !strings.HasPrefix(mode, "JSON") {
					args = append(args, "--hash", doc.Hash)
				}
				jsonOutput := mode != "body stdin"
				if jsonOutput {
					args = append(args, "--json")
				}
				var stdout, stderr bytes.Buffer
				if err := Execute(args, strings.NewReader(input), &stdout, &stderr, "test"); err != nil || stderr.Len() != 0 {
					t.Fatalf("append failed: %v %s", err, &stderr)
				}
				stored, err := c.Read(context.Background(), doc.ID)
				if err != nil || stored.Body != doc.Body+addition || stored.ID != doc.ID || stored.Frontmatter["title"] != "Note" {
					t.Fatalf("incorrect append: %+v %v", stored, err)
				}
				if jsonOutput {
					var response collection.Document
					if err := json.Unmarshal(stdout.Bytes(), &response); err != nil || response.Hash != stored.Hash || response.Body != stored.Body || response.ID != stored.ID {
						t.Fatalf("incorrect JSON: %s %v", &stdout, err)
					}
				} else if stdout.String() != doc.ID+"\n" {
					t.Fatalf("incorrect text output: %q", &stdout)
				}
				if addition != "" && withHash {
					stdout.Reset()
					stderr.Reset()
					if err := Execute(append(args, "--json"), strings.NewReader(input), &stdout, &stderr, "test"); err == nil || stdout.Len() != 0 || !json.Valid(stderr.Bytes()) || !strings.Contains(stderr.String(), "content has changed") {
						t.Fatalf("replayed append: %v %s %s", err, &stdout, &stderr)
					}
				}
				doc = stored
			})
		}
	}
}

func TestAppendCLIInvalidInput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(context.Background(), collection.CreateOptions{Title: "Keep", Body: "Original"})
	if err != nil {
		t.Fatal(err)
	}
	check := func(args []string, input string) {
		t.Helper()
		var stdout, stderr bytes.Buffer
		args = append(args, "--path", c.Path(), "--json")
		if err := Execute(args, strings.NewReader(input), &stdout, &stderr, "test"); err == nil || stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
			t.Fatalf("invalid input response for %v: %v %s %s", args, err, &stdout, &stderr)
		}
		stored, err := c.Read(context.Background(), doc.ID)
		if err != nil || stored.Hash != doc.Hash {
			t.Fatalf("failed append changed source: %+v %v", stored, err)
		}
	}
	for _, input := range []string{
		``, `{`, `null`, `[]`, `{}`, `{"hash":42,"body":"x"}`, `{"hash":null,"body":"x"}`,
		`{"hash":"","body":"x"}`, `{"hash":"bad","body":"x"}`,
		`{"hash":"HASH"}`, `{"hash":"HASH","body":null}`,
		`{"hash":"HASH","body":42}`, `{"hash":"HASH","body":"x","frontmatter":{}}`,
		`{"hash":"HASH","body":"x"} {}`,
	} {
		check([]string{"append", doc.ID, "--input", "-"}, strings.ReplaceAll(input, "HASH", doc.Hash))
	}
	for _, args := range [][]string{
		{"append"}, {"append", doc.ID}, {"append", doc.ID, "text", "--hash", ""},
		{"append", doc.ID, "--hash", doc.Hash},
		{"append", doc.ID, "text", "extra", "--hash", doc.Hash},
		{"append", doc.ID, "text", "--input", "-"},
		{"append", doc.ID, "text", "--hash", doc.Hash, "--body-file", "-"},
		{"append", doc.ID, "--hash", doc.Hash, "--input", "-"},
		{"append", doc.ID, "--body-file", "-", "--input", "-"},
		{"append", doc.ID, "--body-file", "-"},
		{"append", doc.ID, "--input", "missing.json"},
		{"append", doc.ID, "--hash", doc.Hash, "--body-file", "missing.md"},
		{"append", doc.ID, "--hash", doc.Hash, "--body-file", "-"},
	} {
		check(args, "\xff")
	}
}
