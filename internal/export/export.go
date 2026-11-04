// Package export builds a provider from source and exports its schema with
// `terraform providers schema -json`, using a dev_overrides CLI
// configuration so no registry access or `terraform init` is needed.
package export

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Options configures the export.
type Options struct {
	// ProviderDir is the root of the provider source code.
	ProviderDir string
	// Name is the provider short name (e.g. tls). When empty it is derived
	// from the module path in go.mod (terraform-provider-tls → tls).
	Name string
	// Namespace used in the provider address. tfplugindocs looks the schema
	// up under registry.terraform.io/hashicorp/<name>, so keep the default
	// when the file is passed to `tfplugindocs generate --providers-schema`.
	Namespace string
	// Terraform is the Terraform (or OpenTofu) binary.
	Terraform string
}

// Schema builds the provider and writes its schema JSON to w.
func Schema(ctx context.Context, w io.Writer, opts Options) error {
	if opts.ProviderDir == "" {
		opts.ProviderDir = "."
	}
	if opts.Namespace == "" {
		opts.Namespace = "hashicorp"
	}
	if opts.Terraform == "" {
		opts.Terraform = "terraform"
	}
	if opts.Name == "" {
		name, err := nameFromGoMod(opts.ProviderDir)
		if err != nil {
			return err
		}
		opts.Name = name
	} else if err := ValidateName(opts.Name); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "tfrefdoc-schema-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	binDir := filepath.Join(tmp, "bin")
	var buildErr bytes.Buffer
	build := exec.CommandContext(ctx, "go", "build", "-o", filepath.Join(binDir, "terraform-provider-"+opts.Name), ".")
	build.Dir = opts.ProviderDir
	build.Stdout, build.Stderr = &buildErr, &buildErr
	if err := build.Run(); err != nil {
		return fmt.Errorf("building the provider: %w\n%s", err, buildErr.String())
	}

	source := opts.Namespace + "/" + opts.Name
	ws := filepath.Join(tmp, "ws")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		return err
	}
	rc := filepath.Join(tmp, "terraform.rc")
	files := map[string]string{
		rc:                           fmt.Sprintf("provider_installation {\n  dev_overrides {\n    %q = %q\n  }\n  direct {}\n}\n", source, binDir),
		filepath.Join(ws, "main.tf"): fmt.Sprintf("terraform {\n  required_providers {\n    %s = {\n      source = %q\n    }\n  }\n}\n", opts.Name, source),
	}
	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			return err
		}
	}

	var out, tfErr bytes.Buffer
	cmd := exec.CommandContext(ctx, opts.Terraform, "providers", "schema", "-json")
	cmd.Dir = ws
	cmd.Env = append(os.Environ(), "TF_CLI_CONFIG_FILE="+rc, "TF_IN_AUTOMATION=1", "CHECKPOINT_DISABLE=1")
	cmd.Stdout, cmd.Stderr = &out, &tfErr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running %s providers schema -json: %w\n%s", opts.Terraform, err, tfErr.String())
	}
	_, err = w.Write(out.Bytes())
	return err
}

// nameFromGoMod derives the provider short name from the module path.
func nameFromGoMod(dir string) (string, error) {
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", fmt.Errorf("cannot guess the provider name (use --name): %w", err)
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && fields[0] == "module" {
			return nameFromModule(strings.Trim(fields[1], `"`+"`"))
		}
	}
	return "", fmt.Errorf("no module directive in go.mod: use --name")
}

// nameFromModule derives the provider short name from a module path:
//
//	github.com/acme/terraform-provider-foo          → foo
//	github.com/acme/terraform-provider-foo/v2       → foo
//	gitlab.example.com/acme/terraform-provider-foo.git    → foo
//	gitlab.example.com/acme/terraform-provider-foo.git/v2 → foo
func nameFromModule(mod string) (string, error) {
	parts := strings.Split(mod, "/")
	base := parts[len(parts)-1]
	// Major version suffix.
	if len(parts) > 1 && len(base) >= 2 && base[0] == 'v' && strings.Trim(base[1:], "0123456789") == "" {
		base = parts[len(parts)-2]
	}
	// VCS qualifier required by some hosts (GitLab subgroups, private
	// servers): gitlab.example.com/group/terraform-provider-foo.git.
	base = strings.TrimSuffix(base, ".git")
	name, ok := strings.CutPrefix(base, "terraform-provider-")
	if !ok {
		return "", fmt.Errorf("module %q is not named terraform-provider-<name>: use --name", mod)
	}
	if err := ValidateName(name); err != nil {
		return "", fmt.Errorf("module %q: %w: use --name", mod, err)
	}
	return name, nil
}

var nameRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// ValidateName checks that a provider short name is usable in a provider
// address: lowercase letters, digits and dashes.
func ValidateName(name string) error {
	if !nameRe.MatchString(name) {
		return fmt.Errorf("invalid provider name %q (lowercase letters, digits and dashes only)", name)
	}
	return nil
}
