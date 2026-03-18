package git

import (
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

	if !strings.HasSuffix(result, truncationMarker) {
		t.Errorf("expected truncation marker, got %q", result)
	}

	if len(result) > maxBytes {
		t.Errorf("result length %d exceeds maxBytes %d", len(result), maxBytes)
	}
}

func TestTruncateDiff_VerySmallMax(t *testing.T) {
	diff := strings.Repeat("x", 100)
	result := truncateDiff(diff, 5)

	if !strings.HasSuffix(result, truncationMarker) {
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
