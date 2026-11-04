package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExamplesCommand(t *testing.T) {
	dir := t.TempDir()
	var out, errOut bytes.Buffer
	args := []string{"examples", "--schema", "testdata/acme.json", "--examples-dir", dir}

	if err := run(append(args, "--check"), &out, &errOut); !errors.Is(err, errFindings) {
		t.Fatalf("check before generation: err = %v, want findings", err)
	}
	if !strings.Contains(errOut.String(), "missing example") {
		t.Errorf("check must list missing examples:\n%s", errOut.String())
	}

	out.Reset()
	if err := run(args, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "created "+filepath.Join(dir, "resources", "acme_server", "resource.tf")) {
		t.Errorf("unexpected output:\n%s", out.String())
	}

	// Review every file: remove the TODO line.
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, _ := os.ReadFile(path)
		_, rest, _ := strings.Cut(string(b), "\n")
		return os.WriteFile(path, []byte(rest), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	errOut.Reset()
	if err := run(append(args, "--check"), &out, &errOut); err != nil {
		t.Errorf("check after review: %v\n%s", err, errOut.String())
	}
}

func TestReferenceCommand(t *testing.T) {
	dir := t.TempDir()
	page := filepath.Join(dir, "resources", "cert_request.md")
	if err := os.MkdirAll(filepath.Dir(page), 0o755); err != nil {
		t.Fatal(err)
	}
	src, err := os.ReadFile("testdata/tfplugindocs-cert_request.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(page, src, 0o644); err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	args := []string{"reference", "--schema", "testdata/tls.json", "--docs-dir", dir}
	if err := run(append(args, "--check"), &out, &errOut); !errors.Is(err, errFindings) {
		t.Fatalf("check before update: err = %v, want findings", err)
	}
	if err := run(args, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if err := run(append(args, "--check"), &out, &errOut); err != nil {
		t.Errorf("check after update: %v", err)
	}
	b, _ := os.ReadFile(page)
	if !strings.Contains(string(b), "## Argument Reference") {
		t.Errorf("page not updated:\n%s", b)
	}
}

func TestBadUsage(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run(nil, &out, &errOut); err == nil {
		t.Error("no command must fail")
	}
	if err := run([]string{"nope"}, &out, &errOut); err == nil {
		t.Error("unknown command must fail")
	}
	if err := run([]string{"examples"}, &out, &errOut); err == nil {
		t.Error("missing --schema must fail")
	}
}
