package cli

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/alecthomas/kingpin/v2"
)

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

func TestHandleKey(t *testing.T) {
	tests := []struct {
		cur, n       int
		filter, key  string
		idx          int
		wantFilter   string
		done, cancel bool
	}{
		{0, 3, "", "\x1b[B", 1, "", false, false},
		{2, 3, "", "\x1b[B", 0, "", false, false},
		{0, 3, "", "\x1b[A", 2, "", false, false},
		{1, 3, "", "\x1bOA", 0, "", false, false},
		{1, 3, "", "\x1bOB", 2, "", false, false},
		{1, 3, "de", "\x10", 0, "de", false, false},
		{1, 3, "de", "\x0e", 2, "de", false, false},
		{0, 3, "de", "\x10", 2, "de", false, false},
		{2, 3, "de", "\x0e", 0, "de", false, false},
		{0, 0, "zz", "\x1b[B", 0, "zz", false, false},
		{0, 0, "zz", "\x1b[A", 0, "zz", false, false},
		{1, 3, "", "\r", 1, "", true, false},
		{1, 3, "", "\n", 1, "", true, false},
		{0, 0, "zz", "\r", 0, "zz", false, false},
		{1, 3, "", "\x03", 1, "", false, true},
		{1, 3, "", "\x04", 1, "", false, true},
		{1, 3, "dev", "\x7f", 0, "de", false, false},
		{1, 3, "dé", "\x08", 0, "d", false, false},
		{1, 3, "", "\x7f", 1, "", false, false},
		{2, 3, "de", "v", 0, "dev", false, false},
		{1, 3, "", "\x1b", 1, "", false, false},
		{1, 3, "", "j", 0, "j", false, false},
		{1, 3, "", "q", 0, "q", false, false},
	}
	for _, tt := range tests {
		idx, filter, done, cancel := handleKey(tt.cur, tt.n, tt.filter, []byte(tt.key))
		if idx != tt.idx || filter != tt.wantFilter || done != tt.done || cancel != tt.cancel {
			t.Errorf("handleKey(%d, %d, %q, %q) = %d, %q, %v, %v; want %d, %q, %v, %v", tt.cur, tt.n, tt.filter, tt.key, idx, filter, done, cancel, tt.idx, tt.wantFilter, tt.done, tt.cancel)
		}
	}
}

func TestApplyInput(t *testing.T) {
	items := []string{"default", "dev", "prod-dev", "staging"}
	tests := []struct {
		cur          int
		filter       string
		input        string
		idx          int
		wantFilter   string
		wantVisible  []string
		done, cancel bool
	}{
		{0, "", "dev", 0, "dev", []string{"dev", "prod-dev"}, false, false},
		{0, "de", "v\r", 0, "dev", []string{"dev", "prod-dev"}, true, false},
		{0, "", "d\x03ev", 0, "d", []string{"default", "dev", "prod-dev"}, false, true},
		{0, "", "\x1b[B", 1, "", items, false, false},
		{0, "", "\x0e\x0e", 2, "", items, false, false},
		{0, "", "\x1b[B\x1b[B", 2, "", items, false, false},
		{0, "", "\x1bOB\x1b[B\x1bOA", 1, "", items, false, false},
		{0, "", "\x1b[Bd", 0, "d", []string{"default", "dev", "prod-dev"}, false, false},
		{0, "", "\x1b[1;5Bs", 0, "s", []string{"staging"}, false, false},
		{0, "", "\x1b[B\r", 1, "", items, true, false},
	}
	for _, tt := range tests {
		visible := filterItems(items, tt.filter)
		idx, gotVisible, filter, done, cancel := applyInput(tt.cur, items, visible, tt.filter, []byte(tt.input))
		if idx != tt.idx || filter != tt.wantFilter || !reflect.DeepEqual(gotVisible, tt.wantVisible) || done != tt.done || cancel != tt.cancel {
			t.Errorf("applyInput(%d, %q, %q) = %d, %q, %q, %v, %v; want %d, %q, %q, %v, %v", tt.cur, tt.filter, tt.input, idx, gotVisible, filter, done, cancel, tt.idx, tt.wantVisible, tt.wantFilter, tt.done, tt.cancel)
		}
	}
}

func TestFilterItems(t *testing.T) {
	items := []string{"default", "Dev", "prod-dev"}
	tests := []struct {
		filter string
		want   []string
	}{
		{"", items},
		{"DEV", []string{"Dev", "prod-dev"}},
		{"xyz", nil},
	}
	for _, tt := range tests {
		if got := filterItems(items, tt.filter); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("filterItems(%q) = %q; want %q", tt.filter, got, tt.want)
		}
	}
}

func TestScrollOffset(t *testing.T) {
	tests := []struct {
		cur, offset, n, rows int
		want                 int
	}{
		{0, 0, 3, 5, 0},
		{2, 1, 3, 5, 0},
		{4, 0, 10, 5, 0},
		{5, 0, 10, 5, 1},
		{3, 4, 10, 5, 3},
		{9, 0, 10, 5, 5},
		{0, 5, 10, 5, 0},
		{0, 3, 4, 1, 0},
		{7, 6, 8, 3, 5},
	}
	for _, tt := range tests {
		if got := scrollOffset(tt.cur, tt.offset, tt.n, tt.rows); got != tt.want {
			t.Errorf("scrollOffset(%d, %d, %d, %d) = %d; want %d", tt.cur, tt.offset, tt.n, tt.rows, got, tt.want)
		}
	}
}

func TestTruncateRunes(t *testing.T) {
	tests := []struct {
		s     string
		width int
		want  string
	}{
		{"abcdef", 4, "abcd"},
		{"abc", 4, "abc"},
		{"abcd", 4, "abcd"},
		{"あいうえお", 3, "あいう"},
		{"abc", 0, "abc"},
	}
	for _, tt := range tests {
		if got := truncateRunes(tt.s, tt.width); got != tt.want {
			t.Errorf("truncateRunes(%q, %d) = %q; want %q", tt.s, tt.width, got, tt.want)
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

func TestProfileNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".aurl"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".aurl", "profiles"), []byte("[v1only]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name      string
		wantUsage bool
		wantV1    bool
		wantMsg   string
	}{
		{"https://example.com/a", false, true, ""},
		{"HTTP://example.com", false, true, ""},
		{"v1only", false, false, "aurl migrate"},
		{"typo", true, false, `unknown profile or command "typo"`},
	}
	for _, tt := range tests {
		usage, err := profileNotFound(tt.name, "config")
		if usage != tt.wantUsage || errors.Is(err, ErrV1Style) != tt.wantV1 || err == nil || !strings.Contains(err.Error(), tt.wantMsg) {
			t.Errorf("profileNotFound(%q) = %v, %v; want usage %v, v1 %v, message containing %q", tt.name, usage, err, tt.wantUsage, tt.wantV1, tt.wantMsg)
		}
	}
}

func TestCommandNamesAreReserved(t *testing.T) {
	app := kingpin.New("aurl", "")
	a := ConfigureGlobals(app)
	ConfigureAddCommand(app, a)
	ConfigureExecCommand(app, a)
	ConfigureMigrateCommand(app, a)
	for _, cmd := range app.Model().Commands {
		if !isReserved(cmd.Name) {
			t.Errorf("command %q is missing from reservedNames", cmd.Name)
		}
	}
}
