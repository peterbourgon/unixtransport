package unixproxy

import "testing"

func TestNormalizeHost(t *testing.T) {
	for _, tc := range []struct {
		input string
		want  string
	}{
		{"example.com", "example.com"},
		{"Example.COM", "example.com"},
		{"example.com:8080", "example.com"},
		{"EXAMPLE.COM:80", "example.com"},
		{"localhost", "localhost"},
		{"localhost:12345", "localhost"},
		{"192.168.0.1", "192.168.0.1"},
		{"192.168.0.1:8000", "192.168.0.1"},
		{"[::1]", "[::1]"},
		{"[::1]:8080", "[::1]"},
		{"[FE80::2]", "[fe80::2]"},
		{"[FE80::2]:53", "[fe80::2]"},
		{"", ""},
		{":8080", ""},
	} {
		if want, have := tc.want, normalizeHost(tc.input); want != have {
			t.Errorf("normalizeHost(%q): want %q, have %q", tc.input, want, have)
		}
	}
}
