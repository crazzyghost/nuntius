package git

import (
	"testing"
)

func TestIsRelevantGitEvent(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{"index file", "/repo/.git/index", true},
		{"HEAD file", "/repo/.git/HEAD", true},
		{"COMMIT_EDITMSG", "/repo/.git/COMMIT_EDITMSG", true},
		{"MERGE_HEAD", "/repo/.git/MERGE_HEAD", true},
		{"FETCH_HEAD", "/repo/.git/FETCH_HEAD", true},
		{"refs/heads/main", "/repo/.git/refs/heads/main", true},
		{"refs/tags/v1", "/repo/.git/refs/tags/v1.0.0", true},
		{"random file in .git", "/repo/.git/description", false},
		{"logs file", "/repo/.git/logs/HEAD", false},
		{"objects file", "/repo/.git/objects/pack/pack-abc.idx", false},
		{"config file", "/repo/.git/config", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRelevantGitEvent(tt.path)
			if got != tt.want {
				t.Errorf("isRelevantGitEvent(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestContainsDir(t *testing.T) {
	tests := []struct {
		path string
		dir  string
		want bool
	}{
		{"/repo/.git/refs/heads/main", "refs", true},
		{"/repo/.git/refs/tags/v1", "refs", true},
		{"/repo/.git/objects/ab/cdef", "refs", false},
		{"/repo/.git/index", "refs", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := containsDir(tt.path, tt.dir)
			if got != tt.want {
				t.Errorf("containsDir(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
			}
		})
	}
}
