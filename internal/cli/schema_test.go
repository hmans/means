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

func TestSchemaSetCLI(t *testing.T) {
	for _, source := range []string{"-", "file"} {
		c, err := collection.Init(t.TempDir(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := c.Create(map[string]any{"title": "Keep"}, ""); err != nil {
			t.Fatal(err)
		}
		input := source
		if source == "file" {
			input = filepath.Join(t.TempDir(), "proposal.json")
			if err := os.WriteFile(input, []byte(`false`), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		for _, force := range []bool{false, true} {
			args := []string{"schema", "set", "--path", c.Path, "--input", input, "--json"}
			if force {
				args = append(args, "--force")
			}
			var out, stderr bytes.Buffer
			err := Execute(args, strings.NewReader(`false`), &out, &stderr, "test")
			var response struct {
				Error       string
				Installed   bool
				Healthy     bool
				Checked     int
				Diagnostics []collection.Diagnostic
			}
			data := out.Bytes()
			if !force {
				data = stderr.Bytes()
			}
			if e := json.Unmarshal(data, &response); e != nil {
				t.Fatalf("output: %s %v", data, e)
			}
			if (err == nil) != force || response.Installed != force || response.Healthy || response.Checked != 1 || len(response.Diagnostics) != 1 {
				t.Fatalf("result: %+v %v", response, err)
			}
			if force && stderr.Len() != 0 || !force && (out.Len() != 0 || response.Error == "") {
				t.Fatalf("streams: %s %s", out.String(), stderr.String())
			}
		}
	}
}

func TestSchemaSetCLIValidationAndHelp(t *testing.T) {
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"schema", "set"}, {"schema", "set", "--input", "missing.json"},
		{"schema", "set", "--input", "-", "--force"}, {"schema", "set", "extra", "--input", "-"},
	} {
		var out, stderr bytes.Buffer
		if err := Execute(append(args, "--path", c.Path, "--json"), strings.NewReader(`{"type":42}`), &out, &stderr, "test"); err == nil || out.Len() != 0 || !json.Valid(stderr.Bytes()) {
			t.Fatalf("response: %v %s %s", err, out.String(), stderr.String())
		}
	}
	for _, jsonMode := range []bool{false, true} {
		args := []string{"help", "agents", "schema", "set"}
		if jsonMode {
			args = append(args, "--json")
		}
		var out, stderr bytes.Buffer
		if err := Execute(args, nil, &out, &stderr, "test"); err != nil {
			t.Fatal(err)
		}
		for _, topic := range []string{"--force", "--input", "installed", "diagnostics", "concurrent", "stderr"} {
			if !strings.Contains(out.String(), topic) {
				t.Fatalf("help missing %s", topic)
			}
		}
	}
	var out, stderr bytes.Buffer
	if err := Execute([]string{"schema", "set", "--path", c.Path, "--input", "-"}, strings.NewReader(`true`), &out, &stderr, "test"); err != nil || stderr.Len() != 0 || !strings.Contains(out.String(), "Installed schema") {
		t.Fatalf("text success: %v %s %s", err, out.String(), stderr.String())
	}
}
