package ai

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// maxPromptBytes is the approximate token budget (~4000 tokens ≈ 16 KB).
const maxPromptBytes = 16_000

// BuildPrompt assembles a complete prompt from the message request.
// It includes system instructions, convention rules, a file list,
// and the diff, staying within the configured token budget.
func BuildPrompt(req MessageRequest) string {
	var b strings.Builder

	// System instruction
	b.WriteString("You are a commit message generator. Write a concise commit message for the following changes.")

	// Convention rules
	rules := conventionRules(req.Conventions)
	if rules != "" {
		b.WriteString(" Follow the ")
		b.WriteString(req.Conventions)
		b.WriteString(" commit convention.\n\nRULES:\n")
		b.WriteString(rules)
	}

	b.WriteString("\n- Subject line max 72 characters\n")
	b.WriteString("- Use imperative mood\n")
	b.WriteString("- Include a body only if the change is non-trivial\n")
	b.WriteString("- Output ONLY the raw commit message text — no markdown fences, no backticks, no code blocks\n")

	// File list
	if len(req.FileList) > 0 {
		b.WriteString("\nCHANGED FILES:\n")
		for _, f := range req.FileList {
			b.WriteString("- ")
			b.WriteString(f)
			b.WriteString("\n")
		}
	}

	// Diff (with truncation)
	if req.Diff != "" {
		b.WriteString("\nDIFF:\n")
		remaining := maxPromptBytes - b.Len()
		if remaining <= 0 {
			b.WriteString("[diff omitted — prompt too large]\n")
		} else if len(req.Diff) > remaining {
			b.WriteString(req.Diff[:remaining])
			b.WriteString("\n\n[diff truncated — showing first ")
			b.WriteString(fmt.Sprintf("%d", remaining))
			b.WriteString(" bytes]\n")
		} else {
			b.WriteString(req.Diff)
			b.WriteString("\n")
		}
	}

	return b.String()
}

// conventionRules returns the convention-specific rules block.
func conventionRules(convention string) string {
	switch strings.ToLower(convention) {
	case "conventional":
		return "- Use conventional commit format: `type(scope): subject`\n" +
			"- Allowed types: feat, fix, chore, docs, style, refactor, test, ci, build, perf\n" +
			"- Scope is optional\n"
	case "gitmoji":
		return "- Start the subject with a relevant gitmoji emoji\n" +
			"- Common mappings: ✨ feat, 🐛 fix, 📝 docs, 🎨 style, ♻️ refactor, 🚀 perf, ✅ test, 🔧 chore\n"
	case "angular":
		return "- Use Angular commit format: `type(scope): subject`\n" +
			"- Allowed types: feat, fix, docs, style, refactor, perf, test, build, ci, chore\n" +
			"- Scope is mandatory\n"
	case "custom":
		return loadCustomTemplate()
	default:
		return ""
	}
}

// loadCustomTemplate reads a custom prompt template from the path indicated
// by the NUNTIUS_CUSTOM_TEMPLATE environment variable or returns empty.
func loadCustomTemplate() string {
	path := os.Getenv("NUNTIUS_CUSTOM_TEMPLATE")
	if path == "" {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("- [error loading custom template: %v]\n", err)
	}
	return string(data)
}

// fencePattern matches opening/closing markdown code fences like ```commit, ``` etc.
var fencePattern = regexp.MustCompile("(?m)^\\s*```[a-zA-Z]*\\s*$")

// CleanMessage strips markdown code fences and leading/trailing whitespace
// from an AI-generated commit message.
func CleanMessage(msg string) string {
	cleaned := fencePattern.ReplaceAllString(msg, "")
	return strings.TrimSpace(cleaned)
}
