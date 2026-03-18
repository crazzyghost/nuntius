package git

import (
	"testing"
)

func TestParseCommitHash(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "standard commit output",
			output: "[main abc1234] feat: add new feature\n 1 file changed, 10 insertions(+)\n",
			want:   "abc1234",
		},
		{
			name:   "detached HEAD",
			output: "[detached HEAD 9f8e7d6] fix: resolve issue\n 2 files changed\n",
			want:   "9f8e7d6",
		},
		{
			name:   "branch with slash",
			output: "[feature/auth a1b2c3d] chore: update deps\n",
			want:   "a1b2c3d",
		},
		{
			name:   "root commit",
			output: "[main (root-commit) 1234567] Initial commit\n",
			want:   "1234567",
		},
		{
			name:   "no hash found",
			output: "some unexpected output\n",
			want:   "",
		},
		{
			name:   "empty output",
			output: "",
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCommitHash(tt.output)
			if got != tt.want {
				t.Errorf("parseCommitHash() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFirstLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"single line", "hello", "hello"},
		{"multi line", "first\nsecond\nthird", "first"},
		{"leading blank", "\n\nthird", "third"},
		{"empty", "", ""},
		{"whitespace only", "  \n  \n", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := firstLine(tt.input)
			if got != tt.want {
				t.Errorf("firstLine() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCommit_EmptyMessage(t *testing.T) {
	_, err := Commit("")
	if err == nil {
		t.Fatal("expected error for empty message")
	}
	if err.Error() != "commit message cannot be empty" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCommit_WhitespaceOnlyMessage(t *testing.T) {
	_, err := Commit("   \n  \t  ")
	if err == nil {
		t.Fatal("expected error for whitespace-only message")
	}
}
