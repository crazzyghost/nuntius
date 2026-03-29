package git

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestTruncateDiff_NoTruncation(t *testing.T) {
	diff := "short diff content"
	result := truncateDiff(diff, 100)
	if result != diff {
		t.Errorf("expected no truncation, got %q", result)
	}
}

func TestTruncateDiff_ExactSize(t *testing.T) {
	diff := "exactly right"
	result := truncateDiff(diff, len(diff))
	if result != diff {
		t.Errorf("expected no truncation at exact size, got %q", result)
	}
}

func TestTruncateDiff_Truncated(t *testing.T) {
	diff := strings.Repeat("a", 100)
	maxBytes := 50

	result := truncateDiff(diff, maxBytes)

	if !strings.HasSuffix(result, TruncationMarker) {
		t.Errorf("expected truncation marker, got %q", result)
	}

	if len(result) > maxBytes {
		t.Errorf("result length %d exceeds maxBytes %d", len(result), maxBytes)
	}
}

func TestTruncateDiff_VerySmallMax(t *testing.T) {
	diff := strings.Repeat("x", 100)
	result := truncateDiff(diff, 5)

	if !strings.HasSuffix(result, TruncationMarker) {
		t.Errorf("expected truncation marker, got %q", result)
	}
}

func TestTruncateDiff_EmptyDiff(t *testing.T) {
	result := truncateDiff("", 100)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestDefaultMaxDiffBytes(t *testing.T) {
	if DefaultMaxDiffBytes != 32768 {
		t.Errorf("expected DefaultMaxDiffBytes = 32768, got %d", DefaultMaxDiffBytes)
	}
}

func TestParseDiffFileHeaders(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		diff string
		want []string
	}{
		{
			name: "empty input",
			diff: "",
			want: nil,
		},
		{
			name: "git format - single file",
			diff: "diff --git a/foo.go b/foo.go\nindex abc..def 100644\n--- a/foo.go\n+++ b/foo.go\n+added\n",
			want: []string{"foo.go"},
		},
		{
			name: "git format - multiple files",
			diff: "diff --git a/a.go b/a.go\nindex 000..111 100644\ndiff --git a/b.go b/b.go\nindex 222..333 100644\n",
			want: []string{"a.go", "b.go"},
		},
		{
			name: "git format - rename takes b/ side",
			diff: "diff --git a/old.go b/new.go\nsimilarity index 80%\nrename from old.go\nrename to new.go\n--- a/old.go\n+++ b/new.go\n",
			want: []string{"new.go"},
		},
		{
			name: "git format - deduplicated",
			diff: "diff --git a/foo.go b/foo.go\ndiff --git a/foo.go b/foo.go\n",
			want: []string{"foo.go"},
		},
		{
			name: "git format - new file",
			diff: "diff --git a/new.go b/new.go\nnew file mode 100644\n--- /dev/null\n+++ b/new.go\n+line\n",
			want: []string{"new.go"},
		},
		{
			name: "git format - binary file",
			diff: "diff --git a/img.png b/img.png\nindex abc..def 100644\nBinary files a/img.png and b/img.png differ\n",
			want: []string{"img.png"},
		},
		{
			name: "unified diff fallback - single file",
			diff: "--- a/foo.go\n+++ b/foo.go\n+changed\n",
			want: []string{"foo.go"},
		},
		{
			name: "unified diff fallback - multiple files",
			diff: "--- a/a.go\n+++ b/a.go\n+x\n--- a/b.go\n+++ b/b.go\n+y\n",
			want: []string{"a.go", "b.go"},
		},
		{
			name: "unified diff fallback - /dev/null ignored",
			diff: "--- /dev/null\n+++ b/new.go\n+line\n",
			want: []string{"new.go"},
		},
		{
			name: "unified diff fallback - deduplicated",
			diff: "--- a/a.go\n+++ b/a.go\n--- a/a.go\n+++ b/a.go\n",
			want: []string{"a.go"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := ParseDiffFileHeaders(tc.diff)
			if len(got) != len(tc.want) {
				t.Fatalf("ParseDiffFileHeaders(%q)\n  got  %v\n  want %v", tc.diff, got, tc.want)
			}
			for i, f := range got {
				if f != tc.want[i] {
					t.Errorf("files[%d] = %q, want %q", i, f, tc.want[i])
				}
			}
		})
	}
}

func TestUntrackedDiff_UntrackedFileIncluded(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")

	if err := os.WriteFile(repo+"/new_file.txt", []byte("hello\nworld\n"), 0o600); err != nil {
		t.Fatalf("write untracked file: %v", err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir repo: %v", err)
	}

	diff, err := UntrackedDiff(DefaultMaxDiffBytes)
	if err != nil {
		t.Fatalf("UntrackedDiff: %v", err)
	}
	if diff == "" {
		t.Fatal("expected untracked diff, got empty")
	}
	if !strings.Contains(diff, "+++ b/new_file.txt") {
		t.Fatalf("expected diff header for untracked file, got: %q", diff)
	}
}

func TestUntrackedDiff_NoUntrackedFiles(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init")

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(orig)
	})
	if err := os.Chdir(repo); err != nil {
		t.Fatalf("chdir repo: %v", err)
	}

	diff, err := UntrackedDiff(DefaultMaxDiffBytes)
	if err != nil {
		t.Fatalf("UntrackedDiff: %v", err)
	}
	if diff != "" {
		t.Fatalf("expected empty diff with no untracked files, got %q", diff)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
}
