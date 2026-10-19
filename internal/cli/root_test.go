package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestExecute(t *testing.T) {
	for _, tt := range []struct {
		name    string
		args    []string
		want    string
		jsonKey string
		wantErr bool
	}{
		{name: "root help", want: "--path"},
		{name: "version", args: []string{"version"}, want: "means test-version\n"},
		{name: "JSON before command", args: []string{"--json", "version"}, jsonKey: "version", want: "test-version"},
		{name: "JSON after command", args: []string{"version", "--json"}, jsonKey: "version", want: "test-version"},
		{name: "JSON disabled", args: []string{"version", "--json=false"}, want: "means test-version\n"},
		{name: "JSON root help", args: []string{"--json"}, jsonKey: "command", want: "means"},
		{name: "JSON help flag", args: []string{"version", "--help", "--json"}, jsonKey: "command", want: "means version"},
		{name: "JSON help command", args: []string{"help", "version", "--json"}, jsonKey: "command", want: "means version"},
		{name: "unknown command", args: []string{"missing"}, want: "unknown command", wantErr: true},
		{name: "JSON unknown command", args: []string{"missing", "--json"}, jsonKey: "error", want: "unknown command", wantErr: true},
		{name: "JSON invalid flag", args: []string{"version", "--json", "--missing"}, jsonKey: "error", want: "unknown flag", wantErr: true},
		{name: "JSON after invalid flag", args: []string{"version", "--missing", "--json"}, jsonKey: "error", want: "unknown flag", wantErr: true},
		{name: "JSON flag as path value", args: []string{"missing", "--path", "--json"}, want: "unknown command", wantErr: true},
		{name: "JSON invalid argument", args: []string{"version", "extra", "--json"}, jsonKey: "error", want: "unknown command", wantErr: true},
		{name: "JSON empty path", args: []string{"version", "--path=", "--json"}, jsonKey: "error", want: "--path must not be empty", wantErr: true},
		{name: "JSON unknown help topic", args: []string{"help", "missing", "--json"}, jsonKey: "error", want: "unknown", wantErr: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := Execute(tt.args, &stdout, &stderr, "test-version")
			if (err != nil) != tt.wantErr {
				t.Fatalf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
			output, empty := stdout.String(), stderr.String()
			if tt.wantErr {
				output, empty = stderr.String(), stdout.String()
			}
			if empty != "" {
				t.Fatalf("unexpected output on the other stream: %q", empty)
			}
			if tt.jsonKey != "" {
				var response map[string]string
				if err := json.Unmarshal([]byte(output), &response); err != nil {
					t.Fatalf("invalid JSON response %q: %v", output, err)
				}
				output = response[tt.jsonKey]
			}
			if !strings.Contains(output, tt.want) {
				t.Errorf("output = %q, want %q", output, tt.want)
			}
		})
	}
}

func TestInheritedPath(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for _, tt := range []struct {
		name string
		args []string
		want string
	}{
		{name: "default", args: []string{"probe"}, want: cwd},
		{name: "before command", args: []string{"--path", dir, "probe"}, want: dir},
		{name: "after command", args: []string{"probe", "--path", dir}, want: dir},
		{name: "relative", args: []string{"probe", "--path", "notes"}, want: filepath.Join(cwd, "notes")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root, opts := newRootCommand("test")
			var got string
			root.AddCommand(&cobra.Command{
				Use: "probe",
				Run: func(cmd *cobra.Command, args []string) { got = opts.path },
			})
			root.SetArgs(tt.args)
			if err := root.Execute(); err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("path = %q, want %q", got, tt.want)
			}
			after, err := os.Getwd()
			if err != nil || after != cwd {
				t.Fatalf("working directory changed: %q, %v", after, err)
			}
		})
	}
}

type failingWriter struct{ err error }

func (w failingWriter) Write(p []byte) (int, error) { return 0, w.err }

func TestOutputFailure(t *testing.T) {
	want := errors.New("output unavailable")
	for _, args := range [][]string{{"version"}, {"version", "--json"}, {"--help", "--json"}} {
		var stderr bytes.Buffer
		if err := Execute(args, failingWriter{want}, &stderr, "test"); !errors.Is(err, want) {
			t.Errorf("Execute(%q) = %v, want %v", args, err, want)
		}
	}
}
