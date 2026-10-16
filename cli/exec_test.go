package cli

import "testing"

func TestParseProfileChoice(t *testing.T) {
	names := []string{"default", "dev"}
	tests := []struct {
		answer  string
		want    string
		wantErr bool
	}{
		{"1", "default", false},
		{"2", "dev", false},
		{"0", "", true},
		{"3", "", true},
		{"abc", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		got, err := parseProfileChoice(names, tt.answer)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("parseProfileChoice(%q) = %q, %v; want %q, wantErr %v", tt.answer, got, err, tt.want, tt.wantErr)
		}
	}
}

func TestNextIndex(t *testing.T) {
	tests := []struct {
		cur          int
		key          string
		idx          int
		done, cancel bool
	}{
		{0, "\x1b[B", 1, false, false},
		{2, "\x1b[B", 0, false, false},
		{0, "\x1b[A", 2, false, false},
		{1, "k", 0, false, false},
		{1, "j", 2, false, false},
		{1, "\r", 1, true, false},
		{1, "\n", 1, true, false},
		{1, "q", 1, false, true},
		{1, "\x1b", 1, false, false},
		{1, "\x1bOA", 0, false, false},
		{1, "\x04", 1, false, true},
		{1, "\x03", 1, false, true},
		{1, "x", 1, false, false},
	}
	for _, tt := range tests {
		idx, done, cancel := nextIndex(tt.cur, 3, []byte(tt.key))
		if idx != tt.idx || done != tt.done || cancel != tt.cancel {
			t.Errorf("nextIndex(%d, %q) = %d, %v, %v; want %d, %v, %v", tt.cur, tt.key, idx, done, cancel, tt.idx, tt.done, tt.cancel)
		}
	}
}

func TestResolveURL(t *testing.T) {
	tests := []struct {
		base, target, want string
		wantErr            bool
	}{
		{"", "http://example.com/a", "http://example.com/a", false},
		{"https://api.example.com", "https://other.com/a", "https://other.com/a", false},
		{"", "HTTPS://example.com/a", "HTTPS://example.com/a", false},
		{"https://api.example.com", "/v2/me", "https://api.example.com/v2/me", false},
		{"https://api.example.com", "v2/me", "https://api.example.com/v2/me", false},
		{"https://api.example.com/", "/v2/me", "https://api.example.com/v2/me", false},
		{"", "/v2/me", "", true},
	}
	for _, tt := range tests {
		got, err := resolveURL(tt.base, tt.target)
		if (err != nil) != tt.wantErr || got != tt.want {
			t.Errorf("resolveURL(%q, %q) = %q, %v; want %q, wantErr %v", tt.base, tt.target, got, err, tt.want, tt.wantErr)
		}
	}
}
