package collection

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorEmbeddingDiagnostics(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	writeEmbeddingBatchDocuments(t, c, 4)
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	if report, err := c.Doctor(); err != nil || !report.Healthy {
		t.Fatalf("healthy: %+v %v", report, err)
	}
	cache := filepath.Join(c.Path, ".means", "embeddings.jsonl")
	before, err := os.ReadFile(cache)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(c.Path, "doc000.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.Path, "doc001.md"), []byte("---\nid: doc001\ntitle: changed\ndata: {}\n---\nnew body"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.Path, "new.md"), []byte("---\nid: fresh\ntitle: new\ndata: {}\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Moving a document under a new collection boundary leaves an orphan.
	nested := filepath.Join(c.Path, "nested")
	if err := os.Mkdir(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(c.Path, "doc002.md"), filepath.Join(nested, "doc002.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, ConfigFilename), []byte("invalid config still establishes a boundary"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MEANS_EMBEDDING_API_KEY", "")
	requests := len(api.inputs)
	report, err := c.Doctor()
	if err != nil || !report.Healthy || report.Checked != 3 || len(report.Diagnostics) != 0 || len(report.Warnings) != 4 {
		t.Fatalf("report: %+v %v", report, err)
	}
	want := map[string]string{"doc000": "orphaned", "doc001": "stale", "doc002": "orphaned", "fresh": "no usable embedding"}
	for _, d := range report.Warnings {
		if want[d.ID] == "" || !strings.Contains(d.Error, want[d.ID]) || !strings.Contains(d.Error, "means embed") {
			t.Fatalf("diagnostic: %+v", d)
		}
		delete(want, d.ID)
	}
	if len(want) != 0 {
		t.Fatal(want)
	}
	after, err := os.ReadFile(cache)
	if err != nil || string(before) != string(after) || len(api.inputs) != requests {
		t.Fatal("doctor changed cache or called API")
	}
}

func TestDoctorMissingAndInvalidEmbeddingCache(t *testing.T) {
	for _, state := range []string{"missing", "corrupt", "disabled"} {
		t.Run(state, func(t *testing.T) {
			c := embeddingCollection(t)
			writeEmbeddingBatchDocuments(t, c, 1)
			cacheDir := filepath.Join(c.Path, ".means")
			if state == "corrupt" {
				if err := os.Mkdir(cacheDir, 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(cacheDir, "embeddings.jsonl"), []byte("broken"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			if state == "disabled" {
				c.config["embeddings"] = false
			}
			report, err := c.Doctor()
			if err != nil {
				t.Fatal(err)
			}
			count := 1
			if state == "corrupt" {
				count = 2
			}
			if state == "disabled" {
				count = 0
			}
			if len(report.Warnings) != count || len(report.Diagnostics) != 0 || !report.Healthy {
				t.Fatalf("report: %+v", report)
			}
			if state != "corrupt" {
				if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
					t.Fatal("doctor created cache files")
				}
			}
		})
	}
}

func TestDoctorEmbeddingOwnership(t *testing.T) {
	fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	writeEmbeddingBatchDocuments(t, c, 2)
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	// Data schema changes do not invalidate title/body vectors or ownership.
	if _, err := SetSchema(c.Path, []byte(`{"required":["missing"]}`), true); err != nil {
		t.Fatal(err)
	}
	c, err := Load(c.Path)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(c.Path, "doc000.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.Path, "duplicate.md"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := c.Doctor()
	if err != nil {
		t.Fatal(err)
	}
	orphans := 0
	if report.Healthy || len(report.Diagnostics) == 0 {
		t.Fatal("document errors must remain unhealthy alongside warnings")
	}
	for _, d := range report.Warnings {
		if strings.Contains(d.Error, "orphaned") {
			orphans++
			if d.ID != "doc000" {
				t.Fatalf("schema-invalid document lost ownership: %+v", d)
			}
		}
		if strings.Contains(d.Error, "no usable embedding") {
			t.Fatalf("duplicate should not get a missing-vector diagnostic: %+v", d)
		}
	}
	if orphans != 1 {
		t.Fatalf("orphans: %d", orphans)
	}
}
