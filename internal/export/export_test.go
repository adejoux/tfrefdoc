package export

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNameFromModule(t *testing.T) {
	tests := map[string]string{
		"github.com/hashicorp/terraform-provider-tls":             "tls",
		"github.com/acme/terraform-provider-foo/v2":               "foo",
		"gitlab.example.com/acme/terraform-provider-foo.git":      "foo",
		"gitlab.example.com/a/b/c/terraform-provider-foo.git":     "foo",
		"gitlab.example.com/acme/terraform-provider-foo.git/v3":   "foo",
		"gitlab.example.com/acme/terraform-provider-my-cloud.git": "my-cloud",
	}
	for mod, want := range tests {
		got, err := nameFromModule(mod)
		if err != nil || got != want {
			t.Errorf("nameFromModule(%q) = %q, %v; want %q", mod, got, err, want)
		}
	}
	for _, mod := range []string{"github.com/acme/foo", "github.com/acme/terraform-provider-Foo", "github.com/acme/terraform-provider-foo.bar"} {
		if _, err := nameFromModule(mod); err == nil {
			t.Errorf("nameFromModule(%q) must fail", mod)
		}
	}
}

func TestNameFromGoMod(t *testing.T) {
	dir := t.TempDir()
	content := "module gitlab.example.com/acme/terraform-provider-foo.git\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := nameFromGoMod(dir); err != nil || got != "foo" {
		t.Errorf("nameFromGoMod = %q, %v; want foo", got, err)
	}
}
