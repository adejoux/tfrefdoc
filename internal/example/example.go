// Package example generates skeleton Terraform configurations from a
// provider schema: real attribute and block names and types, with
// placeholder values guessed from the attribute names and descriptions.
package example

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	tfjson "github.com/hashicorp/terraform-json"
	"github.com/zclconf/go-cty/cty"

	"github.com/adejoux/tfrefdoc/internal/hints"
	"github.com/adejoux/tfrefdoc/internal/tfschema"
)

// TodoMarker starts the first line of every generated file. Its presence
// means the example still has to be reviewed by a human.
const TodoMarker = "# TODO(tfrefdoc):"

// Options tunes the generation.
type Options struct {
	// RequiredOnly omits the commented optional arguments.
	RequiredOnly bool
}

// Generator builds examples for the entities of a provider.
type Generator struct {
	p    *tfschema.Provider
	opts Options
	// current entity name, never referenced by itself.
	current string
}

// New returns a generator for a provider.
func New(p *tfschema.Provider, opts Options) *Generator {
	return &Generator{p: p, opts: opts}
}

// Example returns the skeleton configuration of an entity.
func (g *Generator) Example(e tfschema.Entity) string {
	g.current = e.Name
	var header string
	if e.Kind == tfschema.KindProvider {
		header = fmt.Sprintf("provider %q {", e.Name)
	} else {
		header = fmt.Sprintf("%s %q \"example\" {", e.Kind.HCLKeyword(), e.Name)
	}
	body := g.body(e.Block.Attributes, e.Block.NestedBlocks, 1, -1, true)

	var b strings.Builder
	b.WriteString(TodoMarker + " generated skeleton: replace the placeholder values and remove this line.\n")
	if len(body) == 0 {
		b.WriteString(strings.TrimSuffix(header, "{") + "{}\n")
		return b.String()
	}
	b.WriteString(header + "\n")
	b.WriteString(render(body))
	b.WriteString("}\n")
	return b.String()
}

// ImportExample returns a skeleton import.sh for a managed resource.
func ImportExample(name string) string {
	return TodoMarker + " generated skeleton: replace the placeholder ID and remove this line.\n" +
		fmt.Sprintf("terraform import %s.example \"<id>\"\n", name)
}

// --- configuration tree --------------------------------------------------

// line is one line of generated HCL. Attribute lines (key != "") with a
// single-line value are aligned on "=" like `terraform fmt` does.
type line struct {
	indent int
	// commentAt is the level at which the line is commented out ("# "), or
	// -1. Lines of a commented block share the level of the block so their
	// indentation is kept after the "#".
	commentAt int
	key, val  string // attribute line
	raw       string // any other line (block header, brace, comment, blank)
	multi     bool   // the value continues on the next lines
}

// body generates the content of a block. In a commented block (inComment)
// every argument is shown; otherwise optional arguments are commented out.
func (g *Generator) body(attrs map[string]*tfjson.SchemaAttribute, blocks map[string]*tfjson.SchemaBlockType, indent, commentAt int, top bool) []line {
	inComment := commentAt >= 0
	var req, opt []string
	for _, name := range tfschema.SortedKeys(attrs) {
		a := attrs[name]
		if !tfschema.IsArgument(a) || a.Deprecated {
			continue
		}
		if a.Required {
			req = append(req, name)
		} else {
			opt = append(opt, name)
		}
	}
	req, opt = promoteExactlyOne(attrs, req, opt)
	var reqBlocks, optBlocks []string
	for _, name := range tfschema.SortedKeys(blocks) {
		bt := blocks[name]
		if bt.Block == nil || bt.Block.Deprecated || !tfschema.BlockHasArguments(bt.Block) {
			continue
		}
		if bt.MinItems > 0 {
			reqBlocks = append(reqBlocks, name)
		} else {
			optBlocks = append(optBlocks, name)
		}
	}

	// Single-line attributes first, then multi-line nested attributes, then
	// blocks, as in hand-written configurations.
	req, opt = simpleFirst(attrs, req), simpleFirst(attrs, opt)
	var out []line
	for _, name := range req {
		out = append(out, g.attribute(name, attrs[name], indent, commentAt)...)
	}
	for _, name := range reqBlocks {
		out = appendBlank(out)
		out = append(out, g.block(name, blocks[name], indent, commentAt)...)
	}

	showOptional := !g.opts.RequiredOnly || inComment
	if !showOptional || len(opt)+len(optBlocks) == 0 {
		return out
	}
	at := indent // optional arguments are commented out at this level
	if inComment {
		at = commentAt
	} else if top {
		out = appendBlank(out)
		out = append(out, line{indent: indent, commentAt: -1, raw: optionalHeader})
	} else if len(reqBlocks) > 0 {
		out = appendBlank(out)
	}
	for _, name := range opt {
		out = append(out, g.attribute(name, attrs[name], indent, at)...)
	}
	for _, name := range optBlocks {
		out = appendBlank(out)
		out = append(out, g.block(name, blocks[name], indent, at)...)
	}
	return out
}

// simpleFirst moves nested attributes after the other attributes, keeping
// the alphabetical order inside each group.
func simpleFirst(attrs map[string]*tfjson.SchemaAttribute, names []string) []string {
	sort.SliceStable(names, func(i, j int) bool {
		return attrs[names[i]].AttributeNestedType == nil && attrs[names[j]].AttributeNestedType != nil
	})
	return names
}

const optionalHeader = "# Optional arguments:"

// promoteExactlyOne moves to the required arguments the first attribute of
// each "exactly one of `a` or `b` must be set" group found in descriptions,
// since the configuration is invalid without one of them. Write-only
// attributes are avoided when possible: they need a recent Terraform.
func promoteExactlyOne(attrs map[string]*tfjson.SchemaAttribute, req, opt []string) ([]string, []string) {
	isOpt := map[string]bool{}
	for _, n := range opt {
		isOpt[n] = true
	}
	promoted := map[string]bool{}
	for _, n := range opt {
		group := hints.ExactlyOneOf(attrs[n].Description)
		if len(group) == 0 {
			continue
		}
		satisfied := false
		for _, m := range group {
			if a, ok := attrs[m]; ok && (a.Required || promoted[m]) {
				satisfied = true
			}
		}
		if satisfied {
			continue
		}
		pick := ""
		for _, m := range group {
			if isOpt[m] && !attrs[m].WriteOnly {
				pick = m
				break
			}
		}
		if pick == "" {
			for _, m := range group {
				if isOpt[m] {
					pick = m
					break
				}
			}
		}
		if pick != "" {
			promoted[pick] = true
		}
	}
	if len(promoted) == 0 {
		return req, opt
	}
	var rest []string
	for _, n := range opt {
		if promoted[n] {
			req = append(req, n)
		} else {
			rest = append(rest, n)
		}
	}
	sort.Strings(req)
	return req, rest
}

func appendBlank(ls []line) []line {
	if n := len(ls); n == 0 || (ls[n-1].raw == "" && ls[n-1].key == "") || ls[n-1].raw == optionalHeader {
		return ls
	}
	return append(ls, line{})
}

func (g *Generator) block(name string, bt *tfjson.SchemaBlockType, indent, commentAt int) []line {
	header := name + " {"
	if bt.NestingMode == tfjson.SchemaNestingModeMap {
		header = name + " \"key\" {"
	}
	out := []line{{indent: indent, commentAt: commentAt, raw: header}}
	out = append(out, g.body(bt.Block.Attributes, bt.Block.NestedBlocks, indent+1, commentAt, false)...)
	return append(out, line{indent: indent, commentAt: commentAt, raw: "}"})
}

func (g *Generator) attribute(name string, a *tfjson.SchemaAttribute, indent, commentAt int) []line {
	if nt := a.AttributeNestedType; nt != nil {
		return g.nestedAttribute(name, nt, indent, commentAt)
	}
	v := g.value(name, a.AttributeType, a.Description, a.Sensitive || a.WriteOnly)
	return []line{{indent: indent, commentAt: commentAt, key: name, val: v}}
}

// nestedAttribute renders a Plugin Framework nested attribute as an object
// (or a collection of objects) literal.
func (g *Generator) nestedAttribute(name string, nt *tfjson.SchemaNestedAttributeType, indent, commentAt int) []line {
	open, close := "{", "}"
	inner := indent + 1
	var out []line
	switch nt.NestingMode {
	case tfjson.SchemaNestingModeList, tfjson.SchemaNestingModeSet:
		out = append(out, line{indent: indent, commentAt: commentAt, key: name, val: "[", multi: true},
			line{indent: indent + 1, commentAt: commentAt, raw: "{"})
		open, close, inner = "", "},", indent+2
	case tfjson.SchemaNestingModeMap:
		out = append(out, line{indent: indent, commentAt: commentAt, key: name, val: "{", multi: true},
			line{indent: indent + 1, commentAt: commentAt, raw: "key = {"})
		open, close, inner = "", "}", indent+2
	}
	if open != "" {
		out = append(out, line{indent: indent, commentAt: commentAt, key: name, val: open, multi: true})
	}
	out = append(out, g.body(nt.Attributes, nil, inner, commentAt, false)...)
	switch nt.NestingMode {
	case tfjson.SchemaNestingModeList, tfjson.SchemaNestingModeSet:
		out = append(out, line{indent: indent + 1, commentAt: commentAt, raw: close},
			line{indent: indent, commentAt: commentAt, raw: "]"})
	case tfjson.SchemaNestingModeMap:
		out = append(out, line{indent: indent + 1, commentAt: commentAt, raw: close},
			line{indent: indent, commentAt: commentAt, raw: "}"})
	default:
		out = append(out, line{indent: indent, commentAt: commentAt, raw: close})
	}
	return out
}

// render prints lines, aligning "=" of consecutive single-line attributes
// at the same level.
func render(ls []line) string {
	width := make([]int, len(ls))
	for i := 0; i < len(ls); {
		if ls[i].key == "" || ls[i].multi {
			i++
			continue
		}
		j, max := i, 0
		for j < len(ls) && ls[j].key != "" && !ls[j].multi && ls[j].indent == ls[i].indent && ls[j].commentAt == ls[i].commentAt {
			if len(ls[j].key) > max {
				max = len(ls[j].key)
			}
			j++
		}
		for k := i; k < j; k++ {
			width[k] = max
		}
		i = j
	}

	var b strings.Builder
	for i, l := range ls {
		text := l.raw
		if l.key != "" {
			key := l.key
			if width[i] > 0 {
				key += strings.Repeat(" ", width[i]-len(l.key))
			}
			text = key + " = " + l.val
		}
		if text == "" {
			b.WriteString("\n")
			continue
		}
		prefix := strings.Repeat("  ", l.indent)
		if l.commentAt >= 0 {
			prefix = strings.Repeat("  ", l.commentAt) + "# " + strings.Repeat("  ", l.indent-l.commentAt)
		}
		b.WriteString(prefix + text + "\n")
	}
	return b.String()
}

// --- values --------------------------------------------------------------

var numberRe = regexp.MustCompile(`^-?\d+(\.\d+)?$`)

// value returns a single-line HCL expression for an attribute.
func (g *Generator) value(name string, t cty.Type, desc string, sensitive bool) string {
	switch {
	case t == cty.NilType || t == cty.DynamicPseudoType:
		return `"example"`
	case t.IsPrimitiveType():
		return g.primitive(name, t, desc, sensitive)
	case t.IsListType() || t.IsSetType():
		if ref, ok := g.reference(name, true); ok {
			return "[" + ref + "]"
		}
		return "[" + g.value(singular(name), t.ElementType(), desc, false) + "]"
	case t.IsMapType():
		if (name == "tags" || name == "labels") && t.ElementType() == cty.String {
			return `{ environment = "dev" }`
		}
		return "{ key = " + g.value("value", t.ElementType(), "", false) + " }"
	case t.IsObjectType():
		return objectLiteral(g, t)
	case t.IsTupleType():
		var parts []string
		for i, et := range t.TupleElementTypes() {
			parts = append(parts, g.value(fmt.Sprintf("%s_%d", name, i), et, "", false))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	return `"example"`
}

func objectLiteral(g *Generator, t cty.Type) string {
	at := t.AttributeTypes()
	keys := make([]string, 0, len(at))
	for k := range at {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+" = "+g.value(k, at[k], "", false))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

func (g *Generator) primitive(name string, t cty.Type, desc string, sensitive bool) string {
	// 1. Values documented in the description.
	for _, hint := range []func(string) (string, bool){hints.Example, hints.Default, hints.Enum, hints.Minimum} {
		if v, ok := hint(desc); ok {
			if lit, ok := literal(t, v); ok {
				return lit
			}
		}
	}
	if t != cty.String {
		return typedFallback(name, t)
	}
	// 2. A reference to another resource of the provider.
	if ref, ok := g.reference(name, false); ok {
		return ref
	}
	// 3. Secrets come from variables, PEM content from files.
	if strings.HasSuffix(name, "_pem") {
		return fmt.Sprintf("file(%q)", strings.TrimSuffix(name, "_pem")+".pem")
	}
	if sensitive {
		return "var." + name
	}
	// 4. Plausible values from the attribute name.
	return stringByName(name)
}

func literal(t cty.Type, v string) (string, bool) {
	switch t {
	case cty.String:
		return strconv.Quote(v), true
	case cty.Number:
		return v, numberRe.MatchString(v)
	case cty.Bool:
		return v, v == "true" || v == "false"
	}
	return "", false
}

func typedFallback(name string, t cty.Type) string {
	if t == cty.Bool {
		return "true"
	}
	switch {
	case name == "port" || strings.HasSuffix(name, "_port"):
		return "443"
	case strings.HasSuffix(name, "_hours"):
		return "24"
	case strings.HasSuffix(name, "_days"):
		return "30"
	case strings.HasSuffix(name, "_minutes"):
		return "10"
	case strings.HasSuffix(name, "_seconds"):
		return "60"
	}
	return "1"
}

func stringByName(name string) string {
	n := strings.ToLower(name)
	switch {
	case n == "description":
		return `"Managed by Terraform"`
	case strings.Contains(n, "email"):
		return `"user@example.com"`
	case n == "url" || n == "uri" || n == "endpoint" || strings.HasSuffix(n, "_url") || strings.HasSuffix(n, "_uri") || strings.HasSuffix(n, "_endpoint"):
		return `"https://example.com"`
	case strings.Contains(n, "cidr"):
		return `"10.0.0.0/16"`
	case n == "ip" || strings.HasSuffix(n, "ip_address") || strings.HasSuffix(n, "_ip"):
		return `"192.0.2.10"`
	case n == "hostname" || n == "domain" || strings.HasSuffix(n, "dns_name") || strings.HasSuffix(n, "_domain") || strings.HasSuffix(n, "hostname"):
		return `"example.com"`
	case strings.HasSuffix(n, "path"):
		return `"./example"`
	}
	return `"example"`
}

// reference looks for another managed resource the attribute most likely
// points to: `network_id` → acme_network.example.id, or `private_key_pem`
// → tls_private_key.example.private_key_pem when tls_private_key exports
// that attribute.
func (g *Generator) reference(name string, list bool) (string, bool) {
	best, bestLen := "", 0
	for _, rname := range tfschema.SortedKeys(g.p.Schema.ResourceSchemas) {
		if rname == g.current {
			continue
		}
		short := g.p.ShortName(rname)
		var ref string
		switch {
		case list && name == short+"_ids":
			ref = rname + ".example.id"
		case !list && name == short+"_id":
			ref = rname + ".example.id"
		case !list && strings.HasPrefix(name, short+"_") && exports(g.p.Schema.ResourceSchemas[rname], name):
			ref = rname + ".example." + name
		default:
			continue
		}
		if len(short) > bestLen {
			best, bestLen = ref, len(short)
		}
	}
	return best, best != ""
}

// exports reports whether a resource has a computed attribute name.
func exports(s *tfjson.Schema, name string) bool {
	if s == nil || s.Block == nil {
		return false
	}
	a, ok := s.Block.Attributes[name]
	return ok && a.Computed
}

// singular is a naive singular used to name list elements
// (dns_names → dns_name, ip_addresses → ip_address).
func singular(s string) string {
	switch {
	case strings.HasSuffix(s, "sses"), strings.HasSuffix(s, "ches"), strings.HasSuffix(s, "shes"), strings.HasSuffix(s, "xes"):
		return strings.TrimSuffix(s, "es")
	case strings.HasSuffix(s, "ies"):
		return strings.TrimSuffix(s, "ies") + "y"
	case strings.HasSuffix(s, "s") && !strings.HasSuffix(s, "ss"):
		return strings.TrimSuffix(s, "s")
	}
	return s
}
