package collection

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gofrs/flock"
)

func TestSearchLockRespectsOperationContext(t *testing.T) {
	c, err := Init(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(c.Path(), ".means")
	if err := os.Mkdir(cache, 0700); err != nil {
		t.Fatal(err)
	}
	lock := flock.New(filepath.Join(cache, "search.lock"))
	if err := lock.Lock(); err != nil {
		t.Fatal(err)
	}
	defer lock.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := c.Search(ctx, SearchOptions{Query: "note"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("lock wait ignored operation deadline: %v", err)
	}
}
