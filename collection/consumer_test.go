package collection_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// A separate module proves that a dependency can import and use the package.
// Build in a temporary directory so go.mod and go.sum changes stay disposable.
func TestExternalConsumerModule(t *testing.T) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate consumer fixture")
	}
	packageDir := filepath.Dir(source)
	root := filepath.Dir(packageDir)
	dir := t.TempDir()
	for dest, src := range map[string]string{
		"go.mod":           filepath.Join(packageDir, "testdata", "consumer", "go.mod"),
		"consumer_test.go": filepath.Join(packageDir, "testdata", "consumer", "consumer_test.go"),
		"go.sum":           filepath.Join(root, "go.sum"),
	} {
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, dest), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for _, args := range [][]string{
		{"mod", "edit", "-replace=github.com/hmans/means=" + root},
		{"test", "-mod=mod", "./..."},
	} {
		cmd := exec.CommandContext(ctx, "go", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOWORK=off", "GOPROXY=off")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("consumer %v: %v\n%s", args, err, out)
		}
	}
}
