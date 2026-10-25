package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hmans/means/internal/collection"
)

func TestDoctorCommandExitAndOutput(t *testing.T) {
	dir := t.TempDir()
	if _, err := collection.Init(dir, []byte(`{"type":"object","required":["value"]}`)); err != nil {
		t.Fatal(err)
	}
	for _, jsonMode := range []bool{false, true} {
		args := []string{"doctor", "--path", dir}
		if jsonMode {
			args = append(args, "--json")
		}
		var stdout, stderr bytes.Buffer
		if err := Execute(args, nil, &stdout, &stderr, "test"); err != nil || stderr.Len() != 0 || !strings.Contains(stdout.String(), "healthy") {
			t.Fatalf("healthy doctor = %v, stdout=%s stderr=%s", err, stdout.String(), stderr.String())
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "bad.md"), []byte("---\nid: bad\ntitle: Bad\ndata: {}\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, jsonMode := range []bool{false, true} {
		args := []string{"doctor", "--path", dir}
		if jsonMode {
			args = append(args, "--json")
		}
		var stdout, stderr bytes.Buffer
		err := Execute(args, nil, &stdout, &stderr, "test")
		if !errors.Is(err, ErrUnhealthy) || stderr.Len() != 0 {
			t.Fatalf("unhealthy doctor = %v, stdout=%s stderr=%s", err, stdout.String(), stderr.String())
		}
		if jsonMode {
			var report collection.DoctorResult
			if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.Healthy || report.Checked != 1 || len(report.Diagnostics) != 1 || report.Diagnostics[0].ID != "bad" {
				t.Fatalf("unexpected JSON report: %s, %v", stdout.String(), err)
			}
		} else if !strings.Contains(stdout.String(), "bad.md") || !strings.Contains(stdout.String(), "ERROR:") {
			t.Fatalf("unexpected text report: %s", stdout.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if err := Execute([]string{"doctor", "--path", filepath.Join(dir, "missing"), "--json"}, nil, &stdout, &stderr, "test"); err == nil || stdout.Len() != 0 || !json.Valid(stderr.Bytes()) {
		t.Fatalf("collection failure: stdout=%s stderr=%s err=%v", stdout.String(), stderr.String(), err)
	}
}

func TestDoctorHelp(t *testing.T) {
	for _, jsonMode := range []bool{false, true} {
		args := []string{"doctor", "--help"}
		if jsonMode {
			args = append(args, "--json")
		}
		var stdout, stderr bytes.Buffer
		if err := Execute(args, nil, &stdout, &stderr, "test"); err != nil || stderr.Len() != 0 {
			t.Fatalf("help failed: %v, stderr=%s", err, stderr.String())
		}
		output := stdout.String()
		if jsonMode {
			var help struct{ Description, Usage string }
			if err := json.Unmarshal(stdout.Bytes(), &help); err != nil {
				t.Fatal(err)
			}
			output = help.Description + help.Usage
		}
		for _, topic := range []string{"MEANS.md", "--path", "--json", "diagnostic", "exit 1", "never repairs"} {
			if !strings.Contains(output, topic) {
				t.Errorf("doctor help omits %q", topic)
			}
		}
	}
}

func TestDoctorMissingEmbeddingsOutput(t *testing.T) {
	c, err := collection.InitConfig(t.TempDir(), []byte("---\nschema: true\nembeddings: {provider: openai}\n---\n"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(c.Path, "document.md"), []byte("---\nid: doc01\ntitle: Document\ndata: {}\n---\nbody"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, jsonMode := range []bool{false, true} {
		args := []string{"doctor", "--path", c.Path}
		if jsonMode {
			args = append(args, "--json")
		}
		var out, stderr bytes.Buffer
		err := Execute(args, nil, &out, &stderr, "test")
		if err != nil || stderr.Len() != 0 || !strings.Contains(out.String(), "no usable embedding") {
			t.Fatalf("report: %s stderr: %s error: %v", out.String(), stderr.String(), err)
		}
		if jsonMode {
			var report collection.DoctorResult
			if err := json.Unmarshal(out.Bytes(), &report); err != nil || !report.Healthy || len(report.Diagnostics) != 0 || len(report.Warnings) != 1 {
				t.Fatalf("warning report: %+v %v", report, err)
			}
		} else if !strings.Contains(out.String(), "WARNING:") {
			t.Fatal("warning not labeled")
		}
	}
}
