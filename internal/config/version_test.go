package config

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadWriteCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nuntius", "version-check.json")

	_, ok := readCache(path)
	if ok {
		t.Error("expected readCache to return false for non-existent file")
	}

	now := time.Now().Truncate(time.Second)
	published := now.Add(-1 * time.Hour)
	writeCache(path, versionCache{
		CheckedAt:   now,
		LatestTag:   "v1.2.3",
		PublishedAt: published,
	})

	got, ok := readCache(path)
	if !ok {
		t.Fatal("expected readCache to return true after write")
	}
	if got.LatestTag != "v1.2.3" {
		t.Errorf("cache.LatestTag = %q, want %q", got.LatestTag, "v1.2.3")
	}
	if !got.CheckedAt.Equal(now) {
		t.Errorf("cache.CheckedAt = %v, want %v", got.CheckedAt, now)
	}
	if !got.PublishedAt.Equal(published) {
		t.Errorf("cache.PublishedAt = %v, want %v", got.PublishedAt, published)
	}
}

func TestReadCacheEmptyPath(t *testing.T) {
	_, ok := readCache("")
	if ok {
		t.Error("expected readCache to return false for empty path")
	}
}

func TestReadCacheInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	_ = os.WriteFile(path, []byte("not json"), 0o644)

	_, ok := readCache(path)
	if ok {
		t.Error("expected readCache to return false for invalid JSON")
	}
}

func TestWriteCacheEmptyPath(t *testing.T) {
	writeCache("", versionCache{})
}

func TestCheckForUpdateDevVersion(t *testing.T) {
	result := CheckForUpdate("dev", "2026-01-01T00:00:00Z")
	if result != nil {
		t.Error("expected nil result for dev version")
	}
}

func TestCheckForUpdateEmptyVersion(t *testing.T) {
	result := CheckForUpdate("", "2026-01-01T00:00:00Z")
	if result != nil {
		t.Error("expected nil result for empty version")
	}
}

func TestCheckForUpdateUnknownDate(t *testing.T) {
	result := CheckForUpdate("v1.0.0", "unknown")
	if result != nil {
		t.Error("expected nil result for unknown build date")
	}
}

func TestCheckForUpdateInvalidDate(t *testing.T) {
	result := CheckForUpdate("v1.0.0", "not-a-date")
	if result != nil {
		t.Error("expected nil result for invalid build date")
	}
}

func TestCheckForUpdateWithFreshCacheUpdateAvailable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nuntius", "version-check.json")

	// Build date is Jan 1; latest release published Jan 15 — update available.
	writeCache(path, versionCache{
		CheckedAt:   time.Now(),
		LatestTag:   "v2.0.0",
		PublishedAt: time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
	})

	origFunc := versionCachePathFunc
	versionCachePathFunc = func() string { return path }
	defer func() { versionCachePathFunc = origFunc }()

	result := CheckForUpdate("v1.0.0", "2026-01-01T00:00:00Z")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.UpdateAvailable {
		t.Error("expected UpdateAvailable to be true")
	}
	if result.LatestTag != "v2.0.0" {
		t.Errorf("LatestTag = %q, want %q", result.LatestTag, "v2.0.0")
	}
}

func TestCheckForUpdateWithFreshCacheNoUpdate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nuntius", "version-check.json")

	// Build date is Jan 15; latest release published Jan 1 — no update.
	writeCache(path, versionCache{
		CheckedAt:   time.Now(),
		LatestTag:   "v1.0.0",
		PublishedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	origFunc := versionCachePathFunc
	versionCachePathFunc = func() string { return path }
	defer func() { versionCachePathFunc = origFunc }()

	result := CheckForUpdate("v1.0.0", "2026-01-15T00:00:00Z")
	if result != nil {
		t.Error("expected nil result when already up-to-date")
	}
}

func TestCheckForUpdateWithStaleCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nuntius", "version-check.json")

	writeCache(path, versionCache{
		CheckedAt:   time.Now().Add(-25 * time.Hour),
		LatestTag:   "v1.0.0",
		PublishedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(githubRelease{
			TagName:     "v2.0.0",
			PublishedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		})
	}))
	defer server.Close()

	origCachePath := versionCachePathFunc
	versionCachePathFunc = func() string { return path }
	defer func() { versionCachePathFunc = origCachePath }()

	origURL := releaseURL
	releaseURL = server.URL
	defer func() { releaseURL = origURL }()

	result := CheckForUpdate("v1.0.0", "2026-01-01T00:00:00Z")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.UpdateAvailable {
		t.Error("expected UpdateAvailable to be true")
	}
	if result.LatestTag != "v2.0.0" {
		t.Errorf("LatestTag = %q, want %q", result.LatestTag, "v2.0.0")
	}

	cached, ok := readCache(path)
	if !ok {
		t.Fatal("expected cache to be written")
	}
	if cached.LatestTag != "v2.0.0" {
		t.Errorf("cached.LatestTag = %q, want %q", cached.LatestTag, "v2.0.0")
	}
}

// TestCheckForUpdateSameVersionWithVPrefix ensures that a cached release whose
// tag is "v0.0.1-alpha" does not trigger an update notice when the running
// binary reports version "0.0.1-alpha" (no 'v' prefix), even if the release
// publish date is after the build date.
func TestCheckForUpdateSameVersionWithVPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nuntius", "version-check.json")

	writeCache(path, versionCache{
		CheckedAt:   time.Now(),
		LatestTag:   "v0.0.1-alpha",
		PublishedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	})

	origFunc := versionCachePathFunc
	versionCachePathFunc = func() string { return path }
	defer func() { versionCachePathFunc = origFunc }()

	// Build date is before the release publish date, but it is the same version.
	result := CheckForUpdate("0.0.1-alpha", "2026-02-01T00:00:00Z")
	if result != nil {
		t.Errorf("expected nil result for same version with differing v-prefix, got %+v", result)
	}
}

// TestCheckForUpdateSameVersionNoVPrefix is the mirror: cached tag has no 'v',
// but the running binary's version string includes the 'v'.
func TestCheckForUpdateSameVersionNoVPrefix(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nuntius", "version-check.json")

	writeCache(path, versionCache{
		CheckedAt:   time.Now(),
		LatestTag:   "0.0.1-alpha",
		PublishedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
	})

	origFunc := versionCachePathFunc
	versionCachePathFunc = func() string { return path }
	defer func() { versionCachePathFunc = origFunc }()

	result := CheckForUpdate("v0.0.1-alpha", "2026-02-01T00:00:00Z")
	if result != nil {
		t.Errorf("expected nil result for same version with differing v-prefix, got %+v", result)
	}
}

func TestCheckForUpdateNetworkFailure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nuntius", "version-check.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	origCachePath := versionCachePathFunc
	versionCachePathFunc = func() string { return path }
	defer func() { versionCachePathFunc = origCachePath }()

	origURL := releaseURL
	releaseURL = server.URL
	defer func() { releaseURL = origURL }()

	result := CheckForUpdate("v1.0.0", "2026-01-01T00:00:00Z")
	if result != nil {
		t.Error("expected nil result on network failure")
	}
}
