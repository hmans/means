package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/hmans/means/collection"
)

func TestSetCLI(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	c, err := collection.Init(t.TempDir(), []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := c.Create(context.Background(), collection.CreateOptions{Title: "Note", Data: map[string]any{"due": "soon"}})
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := []string{"set", doc.ID, "status=done", "tags+=a", "tags+=b", "tags-=a", "owner.name=Ada", "n=3", `code="042"`, "empty=", "due=null", "--title", "Renamed", "--path", c.Path(), "--json"}
	if err := Execute(args, strings.NewReader(""), &stdout, &stderr, "test"); err != nil || stderr.Len() != 0 {
		t.Fatalf("set failed: %v %s", err, &stderr)
	}
	stored, err := c.Read(context.Background(), doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"status": "done", "tags": []any{"b"}, "owner": map[string]any{"name": "Ada"}, "n": json.Number("3"), "code": "042", "empty": ""}
	if stored.Frontmatter["title"] != "Renamed" || !reflect.DeepEqual(stored.Frontmatter["data"], want) {
		t.Fatalf("incorrect set: %#v", stored.Frontmatter)
	}
	var response collection.Document
	if err := json.Unmarshal(stdout.Bytes(), &response); err != nil || response.Hash != stored.Hash {
		t.Fatalf("incorrect JSON: %s %v", &stdout, err)
	}

	stdout.Reset()
	if err := Execute([]string{"set", doc.ID, "status=open", "--hash", stored.Hash, "--path", c.Path()}, strings.NewReader(""), &stdout, &stderr, "test"); err != nil || stdout.String() != doc.ID+"\n" {
		t.Fatalf("set with hash: %v %q %s", err, &stdout, &stderr)
	}
	for _, args := range [][]string{
		{"set", doc.ID},
		{"set", doc.ID, "status"},
		{"set", doc.ID, "=x"},
		{"set", doc.ID, "a..b=1"},
		{"set", doc.ID, "a={"},
		{"set", doc.ID, "a=1", "--hash", ""},
		{"set", doc.ID, "a=1", "--hash", stored.Hash},
	} {
		stdout.Reset()
		stderr.Reset()
		if err := Execute(append(args, "--path", c.Path(), "--json"), strings.NewReader(""), &stdout, &stderr, "test"); err == nil || stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
			t.Fatalf("%v: expected JSON error, got %v %s %s", args, err, &stdout, &stderr)
		}
	}
}
