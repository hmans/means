package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/hmans/means/internal/collection"
)

func TestListSearchWithBody(t *testing.T) {
	c, err := collection.Init(t.TempDir(), []byte(`true`))
	if err != nil {
		t.Fatal(err)
	}
	bodies := map[string]string{}
	for _, body := range []string{"", "## Heading\n\nUnicode Über **body**.\n"} {
		doc, err := c.Create(map[string]any{"title": "Matching title"}, body)
		if err != nil {
			t.Fatal(err)
		}
		bodies[doc.ID] = body
	}
	if err := os.WriteFile(filepath.Join(c.Path, "broken.md"), []byte("broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"list"}, {"search"}, {"search", "matching"}, {"search", "matching", "--literal"}, {"search", "matching", "--limit", "1"}} {
		t.Run(args[0]+"/"+filepath.Join(args[1:]...), func(t *testing.T) {
			run := func(withBody bool) map[string]any {
				t.Helper()
				var out, errs bytes.Buffer
				flags := append(append([]string{}, args...), "--path", c.Path, "--json")
				if withBody {
					flags = append(flags, "--with-body")
				}
				if err := Execute(flags, nil, &out, &errs, "test"); err != nil {
					t.Fatalf("%v: %s", err, &errs)
				}
				var result map[string]any
				if err := json.Unmarshal(out.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				return result
			}
			plain, full := run(false), run(true)
			for _, item := range full["documents"].([]any) {
				doc := item.(map[string]any)
				if _, invalid := doc["error"]; invalid {
					if _, ok := doc["body"]; ok {
						t.Fatal("diagnostic has body")
					}
					continue
				}
				if body, ok := doc["body"]; !ok || body != bodies[doc["id"].(string)] {
					t.Fatalf("incorrect body: %+v", doc)
				}
				delete(doc, "body")
			}
			if !reflect.DeepEqual(plain, full) {
				t.Fatalf("other output changed: %+v vs %+v", plain, full)
			}
		})
	}
	for _, command := range []string{"list", "search"} {
		var out, errs bytes.Buffer
		if err := Execute([]string{command, "--path", c.Path, "--with-body"}, nil, &out, &errs, "test"); err == nil || out.Len() != 0 {
			t.Fatal("accepted --with-body without --json")
		}
	}
}
