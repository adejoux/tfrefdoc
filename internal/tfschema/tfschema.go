// Package tfschema loads the output of `terraform providers schema -json` and
// exposes the objects of one provider in a uniform way.
package tfschema

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
)

// Kind is the kind of provider object.
type Kind string

const (
	KindProvider          Kind = "provider"
	KindResource          Kind = "resource"
	KindDataSource        Kind = "data_source"
	KindEphemeralResource Kind = "ephemeral_resource"
	KindAction            Kind = "action"
)

// HCLKeyword is the top-level HCL block keyword of the kind.
func (k Kind) HCLKeyword() string {
	switch k {
	case KindProvider:
		return "provider"
	case KindDataSource:
		return "data"
	case KindEphemeralResource:
		return "ephemeral"
	case KindAction:
		return "action"
	}
	return "resource"
}

// Dir is the directory used by tfplugindocs for the kind, both in docs/ and
// examples/.
func (k Kind) Dir() string {
	switch k {
	case KindProvider:
		return "provider"
	case KindDataSource:
		return "data-sources"
	case KindEphemeralResource:
		return "ephemeral-resources"
	case KindAction:
		return "actions"
	}
	return "resources"
}

// ExampleFile is the conventional example file name used by tfplugindocs.
func (k Kind) ExampleFile() string {
	switch k {
	case KindProvider:
		return "provider.tf"
	case KindDataSource:
		return "data-source.tf"
	case KindEphemeralResource:
		return "ephemeral-resource.tf"
	case KindAction:
		return "action.tf"
	}
	return "resource.tf"
}

// Entity is a provider configuration, resource, data source, ephemeral
// resource or action.
type Entity struct {
	Kind  Kind
	Name  string // full name, e.g. tls_cert_request; provider short name for KindProvider
	Block *tfjson.SchemaBlock
}

// Provider is the schema of a single provider.
type Provider struct {
	// Address is the provider address, e.g. registry.terraform.io/hashicorp/tls.
	Address string
	// Name is the provider short name, e.g. tls.
	Name   string
	Schema *tfjson.ProviderSchema
}

// Load reads a schema file and returns the selected provider. filter may be
// empty when the file contains a single provider, otherwise it is the full
// provider address or a suffix of it ("hashicorp/tls", "tls").
func Load(path, filter string) (*Provider, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var schemas tfjson.ProviderSchemas
	if err := json.Unmarshal(b, &schemas); err != nil {
		return nil, fmt.Errorf("%s: decoding provider schema JSON: %w", path, err)
	}
	var names []string
	for addr := range schemas.Schemas {
		if filter == "" || addr == filter || strings.HasSuffix(addr, "/"+filter) {
			names = append(names, addr)
		}
	}
	sort.Strings(names)
	switch {
	case len(names) == 0 && filter != "":
		return nil, fmt.Errorf("%s: no provider matching %q", path, filter)
	case len(names) == 0:
		return nil, fmt.Errorf("%s: no provider schema found", path)
	case len(names) > 1:
		return nil, fmt.Errorf("%s: several providers found (%s): select one with --provider", path, strings.Join(names, ", "))
	}
	addr := names[0]
	return &Provider{Address: addr, Name: addr[strings.LastIndex(addr, "/")+1:], Schema: schemas.Schemas[addr]}, nil
}

// Entities returns the provider configuration followed by every resource,
// data source, ephemeral resource and action, sorted by kind and name.
func (p *Provider) Entities() []Entity {
	var out []Entity
	if p.Schema.ConfigSchema != nil && p.Schema.ConfigSchema.Block != nil {
		out = append(out, Entity{Kind: KindProvider, Name: p.Name, Block: p.Schema.ConfigSchema.Block})
	}
	add := func(kind Kind, m map[string]*tfjson.Schema) {
		for _, name := range sortedKeys(m) {
			if s := m[name]; s != nil && s.Block != nil {
				out = append(out, Entity{Kind: kind, Name: name, Block: s.Block})
			}
		}
	}
	add(KindResource, p.Schema.ResourceSchemas)
	add(KindDataSource, p.Schema.DataSourceSchemas)
	add(KindEphemeralResource, p.Schema.EphemeralResourceSchemas)
	for _, name := range sortedKeys(p.Schema.ActionSchemas) {
		if s := p.Schema.ActionSchemas[name]; s != nil && s.Block != nil {
			out = append(out, Entity{Kind: KindAction, Name: name, Block: s.Block})
		}
	}
	return out
}

// ShortName strips the provider prefix from an entity name, as tfplugindocs
// does for page file names (tls_cert_request → cert_request).
func (p *Provider) ShortName(name string) string {
	return strings.TrimPrefix(name, p.Name+"_")
}

// HasResource reports whether the provider has a managed resource.
func (p *Provider) HasResource(name string) bool {
	_, ok := p.Schema.ResourceSchemas[name]
	return ok
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// SortedKeys returns the keys of a map in alphabetical order.
func SortedKeys[V any](m map[string]V) []string { return sortedKeys(m) }

// IsArgument reports whether an attribute can be set in configuration.
func IsArgument(a *tfjson.SchemaAttribute) bool { return a.Required || a.Optional }

// IsReadOnly reports whether an attribute is only computed by the provider.
func IsReadOnly(a *tfjson.SchemaAttribute) bool { return a.Computed && !a.Required && !a.Optional }

// BlockHasArguments reports whether a block contains at least one attribute
// or nested block that can be set in configuration.
func BlockHasArguments(b *tfjson.SchemaBlock) bool {
	if b == nil {
		return false
	}
	for _, a := range b.Attributes {
		if IsArgument(a) {
			return true
		}
	}
	return len(b.NestedBlocks) > 0
}
