package config

import (
	"os/exec"
	"regexp"
	"strings"
)

// Convention labels returned by DetectConvention.
const (
	ConventionConventional = "conventional"
	ConventionGitmoji      = "gitmoji"
	ConventionAngular      = "angular"
	ConventionUnknown      = "unknown"
)

// conventionPattern pairs a convention label with a regex that matches its commit subject format.
type conventionPattern struct {
	Name    string
	Pattern *regexp.Regexp
}

var conventionPatterns = []conventionPattern{
	{
		// Angular: type(scope): subject — scope is mandatory
		Name:    ConventionAngular,
		Pattern: regexp.MustCompile(`^(feat|fix|docs|style|refactor|perf|test|build|ci|chore)\(.+\):\s`),
	},
	{
		// Conventional Commits: type[(scope)][!]: subject — scope is optional
		Name:    ConventionConventional,
		Pattern: regexp.MustCompile(`^(feat|fix|chore|docs|style|refactor|test|ci|build|perf)(\(.+\))?!?:\s`),
	},
	{
		// Gitmoji: starts with an emoji character or :emoji_code:
		Name:    ConventionGitmoji,
		Pattern: regexp.MustCompile(`^(:[a-z0-9_]+:|` + emojiRange + `)`),
	},
}

// emojiRange matches common emoji Unicode ranges used in gitmoji.
const emojiRange = `[\x{1F300}-\x{1F9FF}\x{2600}-\x{26FF}\x{2700}-\x{27BF}\x{FE00}-\x{FE0F}\x{1F000}-\x{1F02F}\x{1F0A0}-\x{1F0FF}\x{200D}\x{20E3}\x{FE0F}]`

// DetectConvention analyzes the last n commits in the current repository
// and returns the dominant commit convention label.
// If cfg.Conventions.Style is explicitly set (not empty and not "auto"),
// it is returned directly without scanning.
func DetectConvention(cfg Config, n int) string {
	// If style is explicitly set, respect it
	if cfg.Conventions.Style != "" && cfg.Conventions.Style != "auto" {
		return cfg.Conventions.Style
	}

	if n <= 0 {
		n = 20
	}

	subjects := recentCommitSubjects(n)
	if len(subjects) == 0 {
		return ConventionUnknown
	}

	return classifySubjects(subjects)
}

// recentCommitSubjects returns the subject lines of the last n commits
// by shelling out to git log.
func recentCommitSubjects(n int) []string {
	cmd := exec.Command("git", "log", "--format=%s", "-n", strings.TrimSpace(itoa(n)))
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var subjects []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			subjects = append(subjects, line)
		}
	}
	return subjects
}

// classifySubjects counts pattern matches across all subjects and returns
// the convention label if any pattern matches >60% of subjects.
// Angular is checked before Conventional since Angular is a strict subset.
func classifySubjects(subjects []string) string {
	total := len(subjects)
	if total == 0 {
		return ConventionUnknown
	}

	counts := make(map[string]int)
	for _, subj := range subjects {
		for _, cp := range conventionPatterns {
			if cp.Pattern.MatchString(subj) {
				counts[cp.Name]++
				break // first match wins (Angular before Conventional)
			}
		}
	}

	threshold := float64(total) * 0.6

	// Check in priority order: angular → conventional → gitmoji
	for _, cp := range conventionPatterns {
		if float64(counts[cp.Name]) > threshold {
			return cp.Name
		}
	}

	return ConventionUnknown
}

// itoa converts an int to a string (avoiding strconv import for this small helper).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	neg := false
	if n < 0 {
		neg = true
		n = -n
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	if neg {
		s = "-" + s
	}
	return s
}
