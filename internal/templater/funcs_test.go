package templater

import "testing"

func TestJoinUrl(t *testing.T) {
	for _, tt := range []struct {
		name string
		elem []string
		want string
	}{
		// The URL scheme's "//" must be preserved, not collapsed by path.Clean.
		{"scheme preserved", []string{"http://localhost", "path1", "path2"}, "http://localhost/path1/path2"},
		{"https with base path", []string{"https://example.com/api", "v1", "users"}, "https://example.com/api/v1/users"},
		{"trailing slash on base", []string{"http://localhost/", "path"}, "http://localhost/path"},
		{"single element", []string{"http://localhost"}, "http://localhost/"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := joinUrl(tt.elem...)
			if err != nil {
				t.Fatalf("joinUrl(%q) unexpected error: %v", tt.elem, err)
			}
			if got != tt.want {
				t.Errorf("joinUrl(%q) = %q; want %q", tt.elem, got, tt.want)
			}
		})
	}
}
