package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hmans/means/internal/collection"
)

func TestHumanDocumentWorkflow(t *testing.T) {
	c, err := collection.Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	run := func(input string, args ...string) (string, error) {
		t.Helper()
		var out, stderr bytes.Buffer
		err := Execute(append(args, "--path", c.Path), strings.NewReader(input), &out, &stderr, "test")
		if err != nil && out.Len() != 0 {
			t.Fatalf("failure wrote stdout: %s", &out)
		}
		return out.String(), err
	}
	created, err := run("# Notes\n", "create", "--title", "Notes", "--body-file", "-", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var doc collection.Document
	if err := json.Unmarshal([]byte(created), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Body != "# Notes\n" || doc.Frontmatter["title"] != "Notes" {
		t.Fatal(created)
	}
	raw, err := run("", "show", doc.ID, "--raw")
	if err != nil {
		t.Fatal(err)
	}
	read, err := run("", "read", doc.ID)
	if err != nil || read != raw {
		t.Fatalf("raw/read mismatch: %v", err)
	}
	if _, err := run("", "show", doc.ID, "--raw", "--json"); err == nil {
		t.Fatal("accepted raw and JSON")
	}
	for _, args := range [][]string{
		{"create", "--title", "Notes", "--input", "-"},
		{"create", "--body-file", "-"},
		{"create", "--title", ""},
		{"delete", doc.ID, "--hash", doc.Hash, "--input", "-"},
		{"delete", doc.ID, "--hash", strings.Repeat("0", 64)},
	} {
		if _, err := run("{}", args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if _, err := run("", "delete", doc.ID, "--hash", doc.Hash); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Read(doc.ID); err == nil {
		t.Fatal("document still exists")
	}
}

func TestInfoAndFilteredList(t *testing.T) {
	c, err := collection.InitConfig(t.TempDir(), []byte("---\ntitle: Test collection\ndescription: Test workflow\nschema: true\n---\nExact instructions\n"))
	if err != nil {
		t.Fatal(err)
	}
	for _, data := range []map[string]any{
		{"status": "todo", "created": "2026-10-01T00:00:00Z"},
		{"status": "done", "created": "2026-10-02T00:00:00Z"},
		{"status": "todo", "created": "2026-10-03T00:00:00Z"},
	} {
		if _, err := c.Create(map[string]any{"title": "Task", "data": data}, "body"); err != nil {
			t.Fatal(err)
		}
	}
	run := func(args ...string) string {
		t.Helper()
		var out, stderr bytes.Buffer
		if err := Execute(append(args, "--path", c.Path, "--json"), nil, &out, &stderr, "test"); err != nil {
			t.Fatalf("%v: %v %s", args, err, &stderr)
		}
		return out.String()
	}
	var info struct {
		Title, Path, Instructions string
		Schema                    bool
	}
	if err := json.Unmarshal([]byte(run("info")), &info); err != nil {
		t.Fatal(err)
	}
	if info.Title != "Test collection" || info.Path != c.Path || info.Instructions != "Exact instructions\n" || !info.Schema {
		t.Fatalf("info: %+v", info)
	}
	filters := []string{"--filter", "status=todo", "--after", "created=2026-10-02T00:00:00Z", "--limit", "1", "--with-body"}
	listed := run(append([]string{"list"}, filters...)...)
	searched := run(append([]string{"search"}, filters...)...)
	if listed != searched {
		t.Fatalf("list/search mismatch: %s %s", listed, searched)
	}
	var result collection.ListResult
	if err := json.Unmarshal([]byte(listed), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Documents) != 1 || result.HasMore {
		t.Fatal(listed)
	}
}

func TestInformationalCommandsDoNotCreateConfig(t *testing.T) {
	for _, args := range [][]string{nil, {"help"}, {"help", "humans"}, {"help", "agents"}, {"help", "agents", "search"}, {"agents"}, {"version"}, {"create", "--help"}, {"search", "--help", "--json"}} {
		dir := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", dir)
		var out, stderr bytes.Buffer
		if err := Execute(args, nil, &out, &stderr, "test"); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 0 {
			t.Fatalf("%v created config: %v %v", args, entries, err)
		}
	}
}

func TestHumanAndAgentCommandHelp(t *testing.T) {
	var human, agent, structured, stderr bytes.Buffer
	if err := Execute([]string{"search", "--help"}, nil, &human, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if err := Execute([]string{"help", "agents", "search"}, nil, &agent, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	if err := Execute([]string{"search", "--help", "--json"}, nil, &structured, &stderr, "test"); err != nil {
		t.Fatal(err)
	}
	var detail struct{ Description, Usage string }
	if err := json.Unmarshal(structured.Bytes(), &detail); err != nil {
		t.Fatal(err)
	}
	if agent.String() != detail.Description+"\n\n"+detail.Usage {
		t.Fatal("agent/JSON reference mismatch")
	}
	if human.String() != agent.String() || strings.Contains(human.String(), ".means/documents.jsonl") {
		t.Fatalf("help differs or exposes cache implementation: %s", &human)
	}
}

func TestEditorWorkflow(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	for _, mode := range []string{"success", "invalid", "conflict", "schema-change", "editor-failure"} {
		t.Run(mode, func(t *testing.T) {
			c, err := collection.Init(t.TempDir(), []byte(`true`))
			if err != nil {
				t.Fatal(err)
			}
			doc, err := c.Create(map[string]any{"title": "Before", "data": map[string]any{"old": true}}, "before\n")
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			t.Setenv("TMPDIR", dir)
			content := "---\nid: " + doc.ID + "\ntitle: After\ndata: {nullable: null}\n---\nafter\n"
			if mode == "invalid" {
				content = "not a document"
			}
			edited := filepath.Join(dir, "content")
			if err := os.WriteFile(edited, []byte(content), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("MEANS_TEST_EDIT_CONTENT", edited)
			script := "#!/bin/sh\ncp \"$MEANS_TEST_EDIT_CONTENT\" \"$1\"\n"
			if mode == "conflict" {
				files, err := filepath.Glob(filepath.Join(c.Path, doc.ID+"-*.md"))
				if err != nil || len(files) != 1 {
					t.Fatal(files, err)
				}
				t.Setenv("MEANS_TEST_ORIGINAL", files[0])
				script += "printf '\\nexternal change\\n' >> \"$MEANS_TEST_ORIGINAL\"\n"
			}
			if mode == "editor-failure" {
				script += "exit 1\n"
			}
			if mode == "schema-change" {
				proposal := filepath.Join(dir, "new-config")
				if err := os.WriteFile(proposal, []byte("---\nschema: {type: object, required: [new_field]}\n---\n"), 0600); err != nil {
					t.Fatal(err)
				}
				t.Setenv("MEANS_TEST_CONFIG", filepath.Join(c.Path, "MEANS.md"))
				t.Setenv("MEANS_TEST_CONFIG_PROPOSAL", proposal)
				script += "cp \"$MEANS_TEST_CONFIG_PROPOSAL\" \"$MEANS_TEST_CONFIG\"\n"
			}
			editor := filepath.Join(dir, "test editor")
			if err := os.WriteFile(editor, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("VISUAL", "")
			t.Setenv("EDITOR", editor)
			var out, stderr bytes.Buffer
			err = Execute([]string{"edit", doc.ID, "--path", c.Path, "--json"}, nil, &out, &stderr, "test")
			after, readErr := c.Read(doc.ID)
			if readErr != nil {
				t.Fatal(readErr)
			}
			copies, _ := filepath.Glob(filepath.Join(dir, "means-edit-*.md"))
			if mode == "success" {
				if err != nil || !json.Valid(out.Bytes()) || after.Body != "after\n" || len(copies) != 0 {
					t.Fatalf("save: %v %s %+v", err, &stderr, copies)
				}
				data := after.Frontmatter["data"].(map[string]any)
				if value, ok := data["nullable"]; !ok || value != nil || len(data) != 1 {
					t.Fatal(data)
				}
			} else {
				if err == nil || out.Len() != 0 || len(copies) != 1 || !strings.Contains(stderr.String(), copies[0]) {
					t.Fatalf("failure recovery: %v %s %v", err, &stderr, copies)
				}
				if mode == "conflict" {
					if !strings.Contains(after.Body, "external change") {
						t.Fatal("external write lost")
					}
				} else if after.Hash != doc.Hash {
					t.Fatal("failed edit changed source")
				}
			}
		})
	}
}
