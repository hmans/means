package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeEmbeddingBatchDocuments(t *testing.T, c *Collection, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		title := "machine"
		if i%2 == 0 {
			title = "fruit"
		}
		content := fmt.Sprintf("---\nid: doc%03d\ntitle: %s\ndata: {}\n---\nbody %d", i, title, i)
		if err := os.WriteFile(filepath.Join(c.Path, fmt.Sprintf("doc%03d.md", i)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestEmbeddingBatchesMapResponseIndices(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	writeEmbeddingBatchDocuments(t, c, 65)
	var progress []string
	c = c.WithEmbeddingProgress(func(message string) { progress = append(progress, message) })
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	if len(api.batches) != 3 {
		t.Fatalf("65 documents made %d requests; want 3", len(api.batches))
	}
	for i, want := range []int{32, 32, 1} {
		if len(api.batches[i]) != want {
			t.Fatalf("batch %d size: %d", i, len(api.batches[i]))
		}
	}
	records := embeddingRecords(t, c)
	if len(records) != 65 {
		t.Fatalf("stored %d records", len(records))
	}
	for i := 0; i < 65; i++ {
		r := records[fmt.Sprintf("doc%03d", i)]
		want := float32(0)
		if i%2 == 0 {
			want = 1
		}
		if r.Vector[0] != want {
			t.Fatalf("response index mapped to wrong document: %d", i)
		}
	}
	if !strings.Contains(strings.Join(progress, "\n"), "Embedded 65/65 documents") {
		t.Fatalf("missing batch progress: %v", progress)
	}
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	if len(api.batches) != 3 {
		t.Fatal("unchanged documents were embedded again")
	}
}

func TestForcedEmbeddingRefresh(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	writeEmbeddingBatchDocuments(t, c, 65)
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	before, err := c.Read("doc000")
	if err != nil {
		t.Fatal(err)
	}
	// A failed first batch must not retain any vectors from the old cache.
	api.failOnRequest = len(api.batches) + 1
	if _, err := c.EmbedWithOptions(EmbedOptions{Force: true}); err == nil {
		t.Fatal("expected failure")
	}
	if len(embeddingRecords(t, c)) != 0 {
		t.Fatal("force retained old vectors")
	}
	api.failOnRequest = 0
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	// A failed second batch retains only newly completed work.
	api.failOnRequest = len(api.batches) + 2
	if _, err := c.EmbedWithOptions(EmbedOptions{Force: true}); err == nil {
		t.Fatal("expected failure")
	}
	if len(embeddingRecords(t, c)) != 32 {
		t.Fatal("force did not checkpoint only completed work")
	}
	api.failOnRequest = 0
	requests := len(api.batches)
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	if len(api.batches) != requests+2 || len(embeddingRecords(t, c)) != 65 {
		t.Fatal("resume did not reuse completed work")
	}
	requests = len(api.batches)
	if _, err := c.EmbedWithOptions(EmbedOptions{Force: true}); err != nil {
		t.Fatal(err)
	}
	if len(api.batches) != requests+3 || len(embeddingRecords(t, c)) != 65 {
		t.Fatal("force did not regenerate every document")
	}
	after, err := c.Read("doc000")
	if err != nil || before.Hash != after.Hash {
		t.Fatal("force changed Markdown")
	}
}

func TestEmbeddingBatchFailureRetainsCompletedBatches(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	c := embeddingCollection(t)
	writeEmbeddingBatchDocuments(t, c, 65)
	api.failOnRequest = 2
	if _, err := c.Embed(); err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("expected failed second batch: %v", err)
	}
	if len(embeddingRecords(t, c)) != 32 {
		t.Fatal("completed batch was not checkpointed")
	}
	api.failOnRequest = 0
	if _, err := c.Embed(); err != nil {
		t.Fatal(err)
	}
	if len(api.batches) != 4 || len(api.batches[2]) != 32 || len(api.batches[3]) != 1 {
		t.Fatalf("retry regenerated completed work: %v", api.batches)
	}
	if len(embeddingRecords(t, c)) != 65 {
		t.Fatal("retry did not complete cache")
	}
}

func TestEmbeddingBatchRejectsIncompleteOrAmbiguousResponses(t *testing.T) {
	api := fakeEmbeddingAPI(t)
	for _, response := range []string{
		`{"data":[{"index":0,"embedding":[1,0]}]}`,
		`{"data":[{"index":0,"embedding":[1,0]},{"index":0,"embedding":[0,1]}]}`,
		`{"data":[{"index":0,"embedding":[1,0]},{"index":2,"embedding":[0,1]}]}`,
		`{"data":[{"index":0,"embedding":[1,0]},{"embedding":[0,1]}]}`,
		`{"data":[{"index":0,"embedding":[1,0]},{"index":1,"embedding":[0]}]}`,
	} {
		c := embeddingCollection(t)
		writeEmbeddingBatchDocuments(t, c, 2)
		api.response = response
		if _, err := c.Embed(); err == nil {
			t.Fatalf("accepted invalid batch: %s", response)
		}
		if len(embeddingRecords(t, c)) != 0 {
			t.Fatal("partially applied invalid batch")
		}
	}
}
