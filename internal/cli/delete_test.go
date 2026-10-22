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

func TestDeleteCLI(t *testing.T) {
	for _, source := range []string{"-", "file"} {
		for _, asJSON := range []bool{false, true} {
			c, err := collection.Init(t.TempDir(), nil)
			if err != nil {
				t.Fatal(err)
			}
			doc, err := c.Create(map[string]any{"title": "Remove"}, "Body")
			if err != nil {
				t.Fatal(err)
			}
			input := `{"hash":"` + doc.Hash + `"}`
			inputPath := source
			if source == "file" {
				inputPath = filepath.Join(t.TempDir(), "delete.json")
				if err := os.WriteFile(inputPath, []byte(input), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			args := []string{"delete", doc.ID, "--path", c.Path, "--input", inputPath}
			if asJSON {
				args = append(args, "--json")
			}
			var out, stderr bytes.Buffer
			if err := Execute(args, strings.NewReader(input), &out, &stderr, "test"); err != nil {
				t.Fatal(err)
			}
			if stderr.Len() != 0 {
				t.Fatal(stderr.String())
			}
			if asJSON {
				var result map[string]any
				if err := json.Unmarshal(out.Bytes(), &result); err != nil || len(result) != 1 || result["id"] != doc.ID {
					t.Fatalf("output: %s %v", out.String(), err)
				}
			} else if out.String() != doc.ID+"\n" {
				t.Fatal(out.String())
			}
			if _, err := c.Read(doc.ID); err == nil {
				t.Fatal("document remains")
			}
			out.Reset()
			if err := Execute(args, strings.NewReader(input), &out, &stderr, "test"); err == nil || out.Len() != 0 {
				t.Fatalf("repeated delete: %v %s", err, out.String())
			}
		}
	}
}

func TestDeleteCLIInvalidInput(t *testing.T) {
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(map[string]any{"title": "Keep"}, "Body")
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{"", "{", "null", "[]", "{}", `{"hash":null}`, `{"hash":1}`, `{"hash":"bad"}`, `{"hash":""}`, `{"hash":"HASH","force":true}`, `{"hash":"HASH"} {}`, `{"hash":"` + strings.Repeat("0", 64) + `"}`} {
		input = strings.ReplaceAll(input, "HASH", doc.Hash)
		var out, stderr bytes.Buffer
		if err := Execute([]string{"delete", doc.ID, "--path", c.Path, "--input", "-", "--json"}, strings.NewReader(input), &out, &stderr, "test"); err == nil || out.Len() != 0 || !json.Valid(stderr.Bytes()) {
			t.Fatalf("input %s: %v %s %s", input, err, out.String(), stderr.String())
		}
		current, err := c.Read(doc.ID)
		if err != nil || current.Hash != doc.Hash {
			t.Fatalf("changed document: %+v %v", current, err)
		}
	}
	for _, args := range [][]string{{"delete"}, {"delete", doc.ID}, {"delete", doc.ID, "--input", "missing.json"}, {"delete", doc.ID, "--input", "-"}} {
		var out, stderr bytes.Buffer
		if err := Execute(append(args, "--path", c.Path, "--json"), strings.NewReader(""), &out, &stderr, "test"); err == nil || out.Len() != 0 || !json.Valid(stderr.Bytes()) {
			t.Fatalf("args: %v %s", err, stderr.String())
		}
	}
}
