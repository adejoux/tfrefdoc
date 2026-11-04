# tfrefdoc

`tfrefdoc` makes the documentation that
[tfplugindocs](https://github.com/hashicorp/terraform-plugin-docs) generates for
a Terraform provider easier to read. It works alongside tfplugindocs, which
keeps generating the pages, their front matter and the `docs/` layout
expected by the Terraform Registry:

- **`tfrefdoc examples`** creates the example files tfplugindocs includes in
  the pages (`examples/resources/<name>/resource.tf`, ...) when they are
  missing, from the real schema of each resource. No page ends up without an
  "Example Usage" section.
- **`tfrefdoc reference`** replaces the schema section of each page by an
  **Argument Reference** and an **Attribute Reference** in which nested
  attributes and blocks are listed under their parent, instead of in
  separate "Nested Schema for ..." sections at the bottom of the page.
- **`tfrefdoc schema`** builds the provider and exports its schema, without
  registry access or `terraform init`.

## Installation

```sh
go install github.com/adejoux/tfrefdoc@latest
```

## Workflow

From the root of the provider, with `terraform` in the `PATH`:

```sh
tfrefdoc schema -o build/schema.json
tfrefdoc examples --schema build/schema.json
tfplugindocs generate --providers-schema build/schema.json
tfrefdoc reference --schema build/schema.json
tfplugindocs validate
```

or, with `go generate ./...`:

```go
//go:generate go run github.com/adejoux/tfrefdoc schema -o build/schema.json
//go:generate go run github.com/adejoux/tfrefdoc examples --schema build/schema.json
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --providers-schema build/schema.json
//go:generate go run github.com/adejoux/tfrefdoc reference --schema build/schema.json
//go:generate go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs validate
```

When a new resource is added, `tfrefdoc examples` creates a skeleton
starting with a `# TODO(tfrefdoc):` line. Replace the placeholder values by a
real use case, remove the TODO line, run `go generate ./...` again and commit
the code, the examples and `docs/` together.

In CI:

```sh
go generate ./... && git diff --exit-code   # docs/ is up to date
tfrefdoc examples --schema build/schema.json --check   # no missing or unreviewed example
```

To go back to the default tfplugindocs layout, remove the `tfrefdoc
reference` step: the next `tfplugindocs generate` rewrites the pages.

## Commands

### `tfrefdoc schema`

Builds the provider with `go build` and runs `terraform providers schema
-json` with a temporary `dev_overrides` CLI configuration.

| Flag | Description |
| :--- | :--- |
| `-o FILE` | Output file (default: stdout). |
| `--provider-dir DIR` | Provider source directory (default `.`). |
| `--name NAME` | Provider short name (default: from `go.mod`, `terraform-provider-<name>`). |
| `--namespace NS` | Namespace of the exported address (default `hashicorp`, the address tfplugindocs looks up with `--providers-schema`). |
| `--terraform BIN` | Terraform or OpenTofu binary (default `terraform`). |

### `tfrefdoc examples`

| Flag | Description |
| :--- | :--- |
| `--schema FILE` | Provider schema JSON (required). |
| `--examples-dir DIR` | Examples directory (default `examples`). |
| `--required-only` | Do not add the optional arguments as comments. |
| `--import` | Also create `examples/resources/<name>/import.sh`. |
| `--check` | Write nothing; exit 1 if an example is missing or still has its TODO line. |
| `--provider ADDR` | Provider to use when the schema contains several. |

Existing files are never modified. Deprecated resources and data sources are
skipped.

The skeleton uses the real attribute and block names and types of the
schema:

- required arguments and blocks are set;
- optional arguments and blocks are present but commented out, so readers
  see everything they can configure;
- read-only and deprecated attributes are left out;
- when a description says "exactly one of `a` or `b` must be set", the
  first one is set.

The schema does not contain default values, validators or examples, so the
values are guessed, in this order:

1. from the description: "e.g. `30m`", "Defaults to `true`", "One of
   `small`, `medium`", "The minimum value is 1";
2. a reference to another resource of the provider: `network_id` →
   `acme_network.example.id`, `private_key_pem` →
   `tls_private_key.example.private_key_pem` when `tls_private_key` exports
   that attribute;
3. `file("<name>.pem")` for PEM content and `var.<name>` for sensitive values;
4. a plausible value from the attribute name (URLs, e-mails, CIDRs, ports,
   durations...), else a placeholder of the right type.

Example, generated from the schema of `hashicorp/tls`:

```hcl
# TODO(tfrefdoc): generated skeleton: replace the placeholder values and remove this line.
resource "tls_locally_signed_cert" "example" {
  allowed_uses          = ["any_extended"]
  ca_cert_pem           = file("ca_cert.pem")
  ca_private_key_pem    = file("ca_private_key.pem")
  cert_request_pem      = tls_cert_request.example.cert_request_pem
  validity_period_hours = 24

  # Optional arguments:
  # ca_private_key_pem_wo         = var.ca_private_key_pem_wo
  # ca_private_key_pem_wo_version = 1
  # early_renewal_hours           = 0
  # is_ca_certificate             = false
  # max_path_length               = 1
  # set_subject_key_id            = false
}
```

### `tfrefdoc reference`

Run after `tfplugindocs generate`. The section between the
`<!-- schema generated by tfplugindocs -->` comment and the next level-2
heading (or the end of the page) is replaced; the rest of the page,
including "Example Usage" and "Import", is kept. Running it twice is
harmless. Pages built from a template that does not use `.SchemaMarkdown`
have no such section and are left untouched.

| Flag | Description |
| :--- | :--- |
| `--schema FILE` | Provider schema JSON (required). |
| `--docs-dir DIR` | Rendered docs directory (default `docs`). |
| `--max-depth N` | Nesting levels rendered under their parent (default 3); deeper levels get their own "Nested schema" sub-section. |
| `--check` | Write nothing; exit 1 if a page is not up to date. |
| `--provider ADDR` | Provider to use when the schema contains several. |
| `-v` | Also list unchanged and skipped pages. |

Example (`tls_cert_request`, excerpt):

```markdown
## Argument Reference

The following arguments are supported:

- `private_key_pem` (String, Optional, Sensitive) Private key in PEM (RFC 1421) format, that the certificate will belong to. ...
- `subject` (Block List, Optional) The subject for which a certificate is being requested. ...
  - `common_name` (String, Optional) Distinguished name: `CN`
  - `country` (String, Optional) Distinguished name: `C`
  - `organization` (String, Optional) Distinguished name: `O`

## Attribute Reference

In addition to the arguments above, the following attributes are exported:

- `cert_request_pem` (String) The certificate request data in PEM (RFC 1421) format. ...
- `id` (String) Unique identifier for this resource: ...
```

Arguments (what goes in the configuration) are separated from exported
attributes (what can be referenced), required first. Read-only values typed
as objects, such as a list of objects returned by a data source, are
expanded too.

## Development

```sh
go test ./...
go test ./... -update   # regenerate the golden files in testdata/golden
```

`testdata/tls.json` is the schema of `hashicorp/tls`, exported with
`tfrefdoc schema`; `testdata/acme.json` is a synthetic Plugin Framework
schema covering nested attributes, deep nesting and references.

## License

[MIT](LICENSE)
