package provider

import (
	"testing"
)

func TestS3Provider_ImplementsProvider(t *testing.T) {
	var _ Provider = (*S3Provider)(nil)
}

func TestS3Provider_BuildKey(t *testing.T) {
	tests := []struct {
		prefix string
		path   string
		expect string
	}{
		{"", "test.txt", "test.txt"},
		{"", "/test.txt", "test.txt"},
		{"myprefix", "test.txt", "myprefix/test.txt"},
		{"myprefix/", "test.txt", "myprefix/test.txt"},
		{"myprefix", "/test.txt", "myprefix/test.txt"},
		{"myprefix/", "/test.txt", "myprefix/test.txt"},
		{"my/deep/prefix", "some/path.txt", "my/deep/prefix/some/path.txt"},
		{"my/deep/prefix/", "/some/path.txt", "my/deep/prefix/some/path.txt"},
		{"", "", ""},
		{"myprefix", "", "myprefix"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix+"+"+tt.path, func(t *testing.T) {
			p := &S3Provider{prefix: tt.prefix}
			actual := p.buildKey(tt.path)
			if actual != tt.expect {
				t.Errorf("buildKey(%q, %q) = %q; want %q", tt.prefix, tt.path, actual, tt.expect)
			}
		})
	}
}
