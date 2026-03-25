package onboarding

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func sendKey(t *testing.T, w Wizard, key string) Wizard {
	t.Helper()
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
	switch key {
	case "enter":
		msg = tea.KeyMsg{Type: tea.KeyEnter}
	case "up":
		msg = tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		msg = tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		msg = tea.KeyMsg{Type: tea.KeyLeft}
	case "backspace":
		msg = tea.KeyMsg{Type: tea.KeyBackspace}
	case "esc":
		msg = tea.KeyMsg{Type: tea.KeyEsc}
	case "ctrl+c":
		msg = tea.KeyMsg{Type: tea.KeyCtrlC}
	}
	model, _ := w.Update(msg)
	return model.(Wizard)
}

func TestWizardInitialStep(t *testing.T) {
	w := NewWizard()
	if w.currentStep != 0 {
		t.Errorf("expected initial step 0, got %d", w.currentStep)
	}
	if w.Done() {
		t.Error("expected Done() = false initially")
	}
	if w.Skipped() {
		t.Error("expected Skipped() = false initially")
	}
}

func TestWizardStepForward(t *testing.T) {
	w := NewWizard()
	for i := 0; i < totalSteps-1; i++ {
		w = sendKey(t, w, "enter")
		if w.currentStep != i+1 {
			t.Errorf("after enter on step %d: expected step %d, got %d", i, i+1, w.currentStep)
		}
	}
}

func TestWizardStepBackward(t *testing.T) {
	w := NewWizard()
	w = sendKey(t, w, "enter")
	w = sendKey(t, w, "enter")
	if w.currentStep != 2 {
		t.Fatalf("expected step 2, got %d", w.currentStep)
	}
	w = sendKey(t, w, "backspace")
	if w.currentStep != 1 {
		t.Errorf("expected step 1 after backspace, got %d", w.currentStep)
	}
	w = sendKey(t, w, "left")
	if w.currentStep != 0 {
		t.Errorf("expected step 0 after left, got %d", w.currentStep)
	}
	w = sendKey(t, w, "backspace")
	if w.currentStep != 0 {
		t.Errorf("expected step 0 stays at 0, got %d", w.currentStep)
	}
}

func TestWizardSkipEsc(t *testing.T) {
	w := NewWizard()
	w = sendKey(t, w, "esc")
	if !w.Skipped() {
		t.Error("expected Skipped() = true after esc")
	}
	if w.Done() {
		t.Error("expected Done() = false after esc")
	}
}

func TestWizardSkipS(t *testing.T) {
	w := NewWizard()
	w = sendKey(t, w, "s")
	if !w.Skipped() {
		t.Error("expected Skipped() = true after 's'")
	}
}

func TestWizardSkipCtrlC(t *testing.T) {
	w := NewWizard()
	w = sendKey(t, w, "ctrl+c")
	if !w.Skipped() {
		t.Error("expected Skipped() = true after ctrl+c")
	}
}

func TestWizardSkipAtAnyStep(t *testing.T) {
	for step := 0; step < totalSteps; step++ {
		w := NewWizard()
		for i := 0; i < step; i++ {
			w = sendKey(t, w, "enter")
		}
		w = sendKey(t, w, "esc")
		if !w.Skipped() {
			t.Errorf("expected Skipped() = true at step %d", step)
		}
	}
}

func TestWizardCompletionDone(t *testing.T) {
	w := NewWizard()
	for i := 0; i < totalSteps; i++ {
		w = sendKey(t, w, "enter")
	}
	if !w.Done() {
		t.Error("expected Done() = true after all steps")
	}
	if w.Skipped() {
		t.Error("expected Skipped() = false after completion")
	}
}

func TestWizardDefaultResult(t *testing.T) {
	w := NewWizard()
	for i := 0; i < totalSteps; i++ {
		w = sendKey(t, w, "enter")
	}
	r := w.Result()
	if r.Provider != "claude" {
		t.Errorf("expected provider %q, got %q", "claude", r.Provider)
	}
	if r.Mode != "api" {
		t.Errorf("expected mode %q, got %q", "api", r.Mode)
	}
	if r.Model != "claude-haiku-4-5" {
		t.Errorf("expected model %q, got %q", "claude-haiku-4-5", r.Model)
	}
	if r.AutoCommit {
		t.Error("expected AutoCommit = false (default)")
	}
	if r.AutoPush {
		t.Error("expected AutoPush = false (default)")
	}
	if !r.AutoUpdateCheck {
		t.Error("expected AutoUpdateCheck = true (default)")
	}
}

func TestWizardCLIModeResult(t *testing.T) {
	w := NewWizard()
	w = sendKey(t, w, "enter") // step 0: select claude
	w = sendKey(t, w, "enter") // step 1: select model
	// step 2: cursor is at 0 (api); press down to select cli (index 1)
	w = sendKey(t, w, "down")
	w = sendKey(t, w, "enter") // confirm cli
	for i := 3; i <= 5; i++ {
		w = sendKey(t, w, "enter")
	}
	r := w.Result()
	if r.Provider != "claude" {
		t.Errorf("expected provider %q, got %q", "claude", r.Provider)
	}
	if r.Mode != "cli" {
		t.Errorf("expected mode %q, got %q", "cli", r.Mode)
	}
}

func TestWizardAPIModeResult(t *testing.T) {
	w := NewWizard()
	w = sendKey(t, w, "enter") // step 0: select claude
	w = sendKey(t, w, "enter") // step 1: select model
	// step 2: cursor is already at 0 (api) — just confirm
	w = sendKey(t, w, "enter") // confirm api
	for i := 3; i <= 5; i++ {
		w = sendKey(t, w, "enter")
	}
	r := w.Result()
	if r.Provider != "claude" {
		t.Errorf("expected provider %q, got %q", "claude", r.Provider)
	}
	if r.Mode != "api" {
		t.Errorf("expected mode %q, got %q", "api", r.Mode)
	}
}

func TestWizardProviderChangesModelList(t *testing.T) {
	w := NewWizard()
	w = sendKey(t, w, "down")
	w = sendKey(t, w, "down")
	w = sendKey(t, w, "down")
	w = sendKey(t, w, "enter")
	opts := w.currentOptions()
	if len(opts) != 4 {
		t.Fatalf("expected 4 gemini models, got %d", len(opts))
	}
	if opts[0].Value != "gemini-3.1-flash-lite-preview" {
		t.Errorf("expected first model %q, got %q", "gemini-3.1-flash-lite-preview", opts[0].Value)
	}
	if opts[1].Value != "gemini-3-flash-preview" {
		t.Errorf("expected second model %q, got %q", "gemini-3-flash-preview", opts[1].Value)
	}
}

func TestWizardOllamaUsesTextInput(t *testing.T) {
	w := NewWizard()
	for i := 0; i < 4; i++ {
		w = sendKey(t, w, "down")
	}
	w = sendKey(t, w, "enter")
	if !w.isOllamaModelStep() {
		t.Error("expected ollama model step")
	}
	if opts := w.currentOptions(); opts != nil {
		t.Errorf("expected nil options on ollama model step, got %v", opts)
	}
}

func TestWizardOllamaResultIncludesTypedModel(t *testing.T) {
	w := NewWizard()
	for i := 0; i < 4; i++ {
		w = sendKey(t, w, "down")
	}
	w = sendKey(t, w, "enter")
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("llama3.2")}
	model, _ := w.Update(msg)
	w = model.(Wizard)
	w = sendKey(t, w, "enter")
	w = sendKey(t, w, "enter")
	w = sendKey(t, w, "enter")
	w = sendKey(t, w, "enter")

	r := w.Result()
	if r.Provider != "ollama" {
		t.Errorf("expected provider %q, got %q", "ollama", r.Provider)
	}
	if r.Model != "llama3.2" {
		t.Errorf("expected model %q, got %q", "llama3.2", r.Model)
	}
}

func TestWizardAllOptionsSortedAlphabetically(t *testing.T) {
	t.Run("providers", func(t *testing.T) {
		labels := make([]string, len(ProviderOptions))
		for i, o := range ProviderOptions {
			labels[i] = o.Label
		}
		if !sort.StringsAreSorted(labels) {
			t.Errorf("provider options not sorted: %v", labels)
		}
	})
}

func TestWriteConfigToPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	result := WizardResult{
		Provider:        "claude",
		Mode:            "cli",
		Model:           "claude-haiku-4.5",
		AutoCommit:      false,
		AutoPush:        false,
		AutoUpdateCheck: true,
	}
	if err := writeConfigToPath(path, result); err != nil {
		t.Fatalf("writeConfigToPath() error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	content := string(data)
	checks := []string{
		`provider = "claude"`,
		`mode = "cli"`,
		`model = "claude-haiku-4.5"`,
		`auto_commit = false`,
		`auto_push = false`,
		`auto_update_check = true`,
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("config missing %q; got:\n%s", check, content)
		}
	}
}

func TestWriteConfigWithMode(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	result := WizardResult{
		Provider:        "copilot",
		Mode:            "cli",
		Model:           "gpt-4o-mini",
		AutoCommit:      false,
		AutoPush:        false,
		AutoUpdateCheck: true,
	}
	if err := writeConfigToPath(path, result); err != nil {
		t.Fatalf("writeConfigToPath() error: %v", err)
	}

	data, _ := os.ReadFile(path)
	content := string(data)
	if !strings.Contains(content, `provider = "copilot"`) {
		t.Errorf("expected provider = copilot in config, got:\n%s", content)
	}
	if !strings.Contains(content, `mode = "cli"`) {
		t.Errorf("expected mode = cli in config, got:\n%s", content)
	}
}
