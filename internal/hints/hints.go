// Package hints extracts example values from attribute descriptions:
// default values, allowed values and minimums written in plain English, as
// commonly found in provider documentation.
package hints

import (
	"regexp"
	"strings"
)

var (
	// "Default value is `true`.", "Defaults to `30s`.", "Default: `x`".
	defaultRe = regexp.MustCompile("(?i)\\bdefaults?(?:\\s+value)?(?:\\s+(?:is|to))?\\s*:?\\s*`([^`]+)`")
	// "One of `a`, `b`.", "Valid values are `a` and `b`.", "Must be one of: `a`".
	enumRe = regexp.MustCompile("(?i)\\b(?:one of|(?:valid|possible|allowed|supported|accepted) values(?: are| include)?|must be(?: one of)?|can be(?: one of)?)\\s*:?\\s*\\[?`([^`]+)`")
	// "The minimum value for length is 1", "minimum of 3", "at least 2".
	minRe = regexp.MustCompile(`(?i)\b(?:minimum(?: value)?(?: for \S+)?(?: is| of)?|at least|must be (?:greater than or equal to|>=))\s*:?\s*` + "`?" + `(-?\d+(?:\.\d+)?)`)
)

var exampleRe = regexp.MustCompile("(?i)(?:\\be\\.g\\.|\\bfor example|\\bexample:|\\bsuch as)\\s*,?\\s*:?\\s*`([^`]+)`")

// Example returns an example value given in a description ("e.g. `30m`").
func Example(desc string) (string, bool) { return first(exampleRe, desc) }

// Default returns the default value mentioned in a description.
func Default(desc string) (string, bool) { return first(defaultRe, desc) }

// Enum returns the first allowed value listed in a description. Phrases
// such as "exactly one of `a` or `b` must be set" express a constraint
// between attributes, not allowed values, and are ignored.
func Enum(desc string) (string, bool) {
	for _, m := range enumRe.FindAllStringSubmatchIndex(desc, -1) {
		before := strings.ToLower(desc[:m[0]])
		if constraintRe.MatchString(before) {
			continue
		}
		v := strings.Trim(strings.TrimSpace(desc[m[2]:m[3]]), `"`)
		if v != "" {
			return v, true
		}
	}
	return "", false
}

var constraintRe = regexp.MustCompile(`(exactly|at least|at most|only|any|none)\s*$`)

// Minimum returns the minimum numeric value mentioned in a description.
func Minimum(desc string) (string, bool) { return first(minRe, desc) }

func first(re *regexp.Regexp, s string) (string, bool) {
	m := re.FindStringSubmatch(s)
	if m == nil {
		return "", false
	}
	v := strings.TrimSpace(m[1])
	return strings.Trim(v, `"`), v != ""
}

var (
	exactlyOneRe = regexp.MustCompile("(?i)\\b(?:exactly one|one \\(and only one\\)) of\\s*:?\\s*([^.;]*)")
	backtickRe   = regexp.MustCompile("`([^`]+)`")
)

// ExactlyOneOf returns the attributes of an "exactly one of `a` or `b` must
// be set" constraint mentioned in a description.
func ExactlyOneOf(desc string) []string {
	m := exactlyOneRe.FindStringSubmatch(desc)
	if m == nil {
		return nil
	}
	var names []string
	for _, b := range backtickRe.FindAllStringSubmatch(m[1], -1) {
		names = append(names, b[1])
	}
	if len(names) < 2 {
		return nil
	}
	return names
}
