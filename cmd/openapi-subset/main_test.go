package main

import "testing"

func TestMatchesAnyPrefix(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		path     string
		prefixes []string
		want     bool
	}{
		{name: "single match", path: "/todos/123", prefixes: []string{"/todos"}, want: true},
		{name: "shared attachments match", path: "/attachments/123/links", prefixes: []string{"/todos", "/attachments"}, want: true},
		{name: "shared resources match", path: "/resources/memo/123/attachments", prefixes: []string{"/memos", "/resources"}, want: true},
		{name: "no match", path: "/counter", prefixes: []string{"/todos", "/attachments"}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := matchesAnyPrefix(tt.path, tt.prefixes)
			if got != tt.want {
				t.Fatalf("matchesAnyPrefix(%q, %q) = %v, want %v", tt.path, tt.prefixes, got, tt.want)
			}
		})
	}
}
