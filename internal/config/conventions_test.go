package config

import "testing"

func TestClassifySubjectsConventional(t *testing.T) {
	subjects := []string{
		"feat: add user authentication",
		"fix: resolve null pointer in handler",
		"chore: update dependencies",
		"docs: add API documentation",
		"feat(auth): implement OAuth2 flow",
		"test: add unit tests for config",
		"refactor: extract helper functions",
		"ci: add GitHub Actions workflow",
		"some random commit message",
		"another non-conventional message",
	}
	result := classifySubjects(subjects)
	if result != ConventionConventional {
		t.Errorf("expected %q, got %q", ConventionConventional, result)
	}
}

func TestClassifySubjectsAngular(t *testing.T) {
	subjects := []string{
		"feat(auth): add user authentication",
		"fix(api): resolve null pointer in handler",
		"chore(deps): update dependencies",
		"docs(readme): add API documentation",
		"feat(auth): implement OAuth2 flow",
		"test(config): add unit tests for config",
		"refactor(utils): extract helper functions",
		"ci(actions): add GitHub Actions workflow",
		"some random commit message",
		"another non-angular message",
	}
	result := classifySubjects(subjects)
	if result != ConventionAngular {
		t.Errorf("expected %q, got %q", ConventionAngular, result)
	}
}

func TestClassifySubjectsGitmoji(t *testing.T) {
	subjects := []string{
		":sparkles: add new feature",
		":bug: fix login issue",
		":memo: update documentation",
		":art: improve code structure",
		":fire: remove dead code",
		":white_check_mark: add tests",
		":rocket: deploy to production",
		":recycle: refactor auth module",
		"some random commit",
		"another random thing",
	}
	result := classifySubjects(subjects)
	if result != ConventionGitmoji {
		t.Errorf("expected %q, got %q", ConventionGitmoji, result)
	}
}

func TestClassifySubjectsUnknown(t *testing.T) {
	subjects := []string{
		"updated the thing",
		"fixed stuff",
		"work in progress",
		"misc changes",
		"another commit",
	}
	result := classifySubjects(subjects)
	if result != ConventionUnknown {
		t.Errorf("expected %q, got %q", ConventionUnknown, result)
	}
}

func TestClassifySubjectsEmpty(t *testing.T) {
	result := classifySubjects(nil)
	if result != ConventionUnknown {
		t.Errorf("expected %q, got %q", ConventionUnknown, result)
	}
}

func TestDetectConventionExplicitStyle(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Conventions.Style = "gitmoji"

	result := DetectConvention(cfg, 20)
	if result != "gitmoji" {
		t.Errorf("expected explicit style %q, got %q", "gitmoji", result)
	}
}

func TestDetectConventionAutoMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Conventions.Style = "auto"

	// No git repo in temp dir, so should return unknown
	result := DetectConvention(cfg, 20)
	if result != ConventionUnknown {
		t.Errorf("expected %q for auto with no repo, got %q", ConventionUnknown, result)
	}
}

func TestClassifySubjectsGitmojiUnicode(t *testing.T) {
	subjects := []string{
		"\U0001F525 remove dead code",
		"\U0001F41B fix login bug",
		"\U0001F4DD update docs",
		"\U00002728 add feature",
		"\U0001F680 deploy changes",
		"\U0001F3A8 improve styles",
		"\U0001F527 fix config",
		"some normal commit",
		"another random commit",
		"not gitmoji",
	}
	result := classifySubjects(subjects)
	if result != ConventionGitmoji {
		t.Errorf("expected %q, got %q", ConventionGitmoji, result)
	}
}
