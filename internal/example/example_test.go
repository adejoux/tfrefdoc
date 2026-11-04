package example

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"

	"github.com/adejoux/tfrefdoc/internal/testutil"
	"github.com/adejoux/tfrefdoc/internal/tfschema"
)

var fixtures = []string{"tls", "acme"}

// commentedCode matches an argument or block line commented out by the
// generator, to check that the commented code is valid HCL too.
var commentedCode = regexp.MustCompile(`(?m)^(\s*)# (\s*(?:[a-z_]+\s+=|[a-z_]+ (?:"key" )?\{|[\]\}],?|\{))`)

func TestExamples(t *testing.T) {
	for _, fx := range fixtures {
		p, err := tfschema.Load(filepath.Join("..", "..", "testdata", fx+".json"), "")
		if err != nil {
			t.Fatal(err)
		}
		g := New(p, Options{})
		for _, e := range p.Entities() {
			name := fx + "/" + Path(e)
			t.Run(name, func(t *testing.T) {
				src := g.Example(e)
				testutil.Golden(t, filepath.Join("..", "..", "testdata", "golden", fx, "examples", Path(e)), src)

				for label, code := range map[string]string{"as generated": src, "uncommented": commentedCode.ReplaceAllString(src, "$1$2")} {
					if _, diags := hclsyntax.ParseConfig([]byte(code), "example.tf", hcl.InitialPos); diags.HasErrors() {
						t.Errorf("%s: invalid HCL: %s\n%s", label, diags.Error(), code)
					}
				}
				if !strings.HasPrefix(src, TodoMarker) {
					t.Error("generated example must start with the TODO marker")
				}
			})
		}
	}
}

func TestRequiredOnly(t *testing.T) {
	p, err := tfschema.Load(filepath.Join("..", "..", "testdata", "acme.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	g := New(p, Options{RequiredOnly: true})
	for _, e := range p.Entities() {
		if e.Name == "acme_server" {
			if src := g.Example(e); strings.Contains(src, "# Optional arguments:") || strings.Contains(src, "# password") {
				t.Errorf("optional arguments must be omitted:\n%s", src)
			}
		}
	}
}

func TestMissingAndUnreviewed(t *testing.T) {
	p, err := tfschema.Load(filepath.Join("..", "..", "testdata", "acme.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	g := New(p, Options{})
	files, err := g.Missing(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	// provider, 2 resources + 2 import.sh, 1 data source.
	if len(files) != 6 {
		t.Fatalf("got %d missing files, want 6", len(files))
	}
	if err := Write(dir, files); err != nil {
		t.Fatal(err)
	}
	if files, _ := g.Missing(dir, true); len(files) != 0 {
		t.Errorf("existing files must not be generated again, got %d", len(files))
	}
	unreviewed, err := Unreviewed(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(unreviewed) != 6 {
		t.Errorf("got %d unreviewed files, want 6", len(unreviewed))
	}
	if got, err := Unreviewed(filepath.Join(dir, "missing")); err != nil || len(got) != 0 {
		t.Errorf("a missing directory has nothing to review: %v %v", got, err)
	}
}
