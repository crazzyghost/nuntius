package config

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const (
	// cacheTTL is how long a version check result is considered fresh.
	cacheTTL = 24 * time.Hour
	// httpTimeout is the maximum time for the version check HTTP request.
	httpTimeout = 5 * time.Second
)

// Overridable for testing.
var (
	releaseURL           = "https://api.github.com/repos/crazzyghost/nuntius/releases/latest"
	versionCachePathFunc = defaultVersionCachePath
)

// VersionCheckResult holds the outcome of a version check.
type VersionCheckResult struct {
	// LatestTag is the tag name of the newest release (e.g. "v0.2.0").
	LatestTag string
	// Current is the version string of the running binary.
	Current string
	// UpdateAvailable is true when the latest release is newer than the running build.
	UpdateAvailable bool
}

type versionCache struct {
	CheckedAt   time.Time `json:"checked_at"`
	LatestTag   string    `json:"latest_tag"`
	PublishedAt time.Time `json:"published_at"`
}

type githubRelease struct {
	TagName     string    `json:"tag_name"`
	PublishedAt time.Time `json:"published_at"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
}

// normalizeVersion strips a leading 'v' or 'V' so that "v1.0.0" and "1.0.0"
// compare as equal regardless of how the build was tagged.
func normalizeVersion(v string) string {
	if len(v) > 0 && (v[0] == 'v' || v[0] == 'V') {
		return v[1:]
	}
	return v
}

// CheckForUpdate checks whether a newer version of nuntius is available by
// comparing the build date of the running binary against the publish date of
// the latest GitHub release. Returns nil when up-to-date, on dev builds, or
// if anything goes wrong (graceful degradation).
func CheckForUpdate(currentVersion, buildDate string) *VersionCheckResult {
	if currentVersion == "" || currentVersion == "dev" || buildDate == "" || buildDate == "unknown" {
		return nil
	}

	buildTime, err := time.Parse(time.RFC3339, buildDate)
	if err != nil {
		return nil
	}

	cachePath := versionCachePathFunc()

	if cached, ok := readCache(cachePath); ok {
		if time.Since(cached.CheckedAt) < cacheTTL {
			if cached.PublishedAt.After(buildTime) &&
				normalizeVersion(cached.LatestTag) != normalizeVersion(currentVersion) {
				return &VersionCheckResult{
					LatestTag:       cached.LatestTag,
					Current:         currentVersion,
					UpdateAvailable: true,
				}
			}
			return nil
		}
	}

	release, err := fetchLatestRelease()
	if err != nil {
		return nil
	}

	writeCache(cachePath, versionCache{
		CheckedAt:   time.Now(),
		LatestTag:   release.TagName,
		PublishedAt: release.PublishedAt,
	})

	if release.PublishedAt.After(buildTime) &&
		normalizeVersion(release.TagName) != normalizeVersion(currentVersion) {
		return &VersionCheckResult{
			LatestTag:       release.TagName,
			Current:         currentVersion,
			UpdateAvailable: true,
		}
	}
	return nil
}

func defaultVersionCachePath() string {
	dir := NuntiusDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "version-check.json")
}

func readCache(path string) (versionCache, bool) {
	if path == "" {
		return versionCache{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return versionCache{}, false
	}
	var c versionCache
	if err := json.Unmarshal(data, &c); err != nil {
		return versionCache{}, false
	}
	return c, true
}

func writeCache(path string, c versionCache) {
	if path == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0o644)
}

func fetchLatestRelease() (githubRelease, error) {
	client := &http.Client{Timeout: httpTimeout}
	req, err := http.NewRequest("GET", releaseURL, nil)
	if err != nil {
		return githubRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return githubRelease{}, err
	}

	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		return githubRelease{}, err
	}

	if release.TagName == "" {
		return githubRelease{}, fmt.Errorf("empty tag_name in release")
	}

	// Skip drafts and prereleases. The /releases/latest endpoint already
	// excludes these, but we check explicitly for safety.
	if release.Draft || release.Prerelease {
		return githubRelease{}, fmt.Errorf("latest release is a draft or prerelease")
	}

	return release, nil
}
