// Package testutil holds helpers shared by the tests.
package testutil

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// Update regenerates the golden files: go test ./... -update.
var Update = flag.Bool("update", false, "update golden files")

// Golden compares got with the golden file at path, or rewrites it with -update.
func Golden(t *testing.T, path, got string) {
	t.Helper()
	if *Update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden file (run go test ./... -update): %v", err)
	}
	if string(want) != got {
		t.Errorf("output differs from %s (run go test ./... -update to accept):\n%s", path, got)
	}
}
