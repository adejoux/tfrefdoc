package hints

import "testing"

func TestHints(t *testing.T) {
	tests := []struct {
		fn   func(string) (string, bool)
		name string
		in   string
		want string
	}{
		{Example, "e.g.", "Timeout for creation, e.g. `30m`.", "30m"},
		{Example, "for example", "Image name, for example: `ubuntu-24.04`.", "ubuntu-24.04"},
		{Example, "none", "Image name.", ""},
		{Default, "default value is", "Include lowercase characters. Default value is `true`.", "true"},
		{Default, "defaults to", "Timeout of the operation. Defaults to `30s`.", "30s"},
		{Default, "default colon", "Region. Default: `\"eu-de\"`", "eu-de"},
		{Default, "none", "The name of the server.", ""},
		{Enum, "one of", "Size of the server. One of `small`, `medium`, `large`.", "small"},
		{Enum, "valid values", "Algorithm. Valid values are `RSA`, `ECDSA` and `ED25519`.", "RSA"},
		{Enum, "must be one of", "Must be one of: `a`, `b`.", "a"},
		{Enum, "none", "Free-form text.", ""},
		{Enum, "constraint", "Exactly one of `private_key_pem` or `private_key_pem_wo` must be set.", ""},
		{Enum, "constraint then enum", "Only one of `a` or `b`. Valid values are `x`, `y`.", "x"},
		{Minimum, "minimum value", "The minimum value for length is 1 and must be...", "1"},
		{Minimum, "at least", "Number of replicas, at least 3.", "3"},
		{Minimum, "none", "Number of replicas.", ""},
	}
	for _, tt := range tests {
		got, _ := tt.fn(tt.in)
		if got != tt.want {
			t.Errorf("%s: got %q, want %q", tt.name, got, tt.want)
		}
	}
}

func TestExactlyOneOf(t *testing.T) {
	got := ExactlyOneOf("Private key. Exactly one of `private_key_pem` or `private_key_pem_wo` must be set.")
	if len(got) != 2 || got[0] != "private_key_pem" || got[1] != "private_key_pem_wo" {
		t.Errorf("ExactlyOneOf = %v", got)
	}
	if got := ExactlyOneOf("One of `a`, `b`."); got != nil {
		t.Errorf("an enum is not a constraint, got %v", got)
	}
}
