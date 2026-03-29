# nuntius

[![CI](https://github.com/crazzyghost/nuntius/actions/workflows/ci.yml/badge.svg)](https://github.com/crazzyghost/nuntius/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/crazzyghost/nuntius)](https://github.com/crazzyghost/nuntius/releases/latest)

AI-powered commit messages from your terminal. Works with Claude, Gemini, GPT, Ollama, and GitHub Copilot.

nuntius reads your git diff, sends it to your chosen AI provider, and returns a
commit message that matches your repo's conventions. You decide whether to
commit and push — or let it handle the whole pipeline for you.

### Highlights

- **Multi-provider AI** — Claude, Gemini, GPT, Ollama, and GitHub Copilot, via API or local CLI
- **Convention-aware** — auto-detects Conventional Commits, Angular, and Gitmoji from your commit history
- **Interactive TUI** — live file watcher with generate, commit, and push at a keypress
- **MCP server** — expose nuntius as a tool for AI-powered editors and agentic workflows
- **Headless mode** — one-liner commands with JSON output for scripts, aliases, and CI
- **Non-destructive by default** — generate a message without committing; opt in to the rest

## Why nuntius

**You live in the terminal.** You want AI-generated commit messages without
switching to an IDE or copy-pasting from a chat window.

**You use multiple AI providers.** Switch between Claude, Gemini, GPT, Ollama,
or Copilot with a single flag — no config file changes required.

**You care about commit conventions.** nuntius scans your recent commit history
and auto-detects whether you use Conventional Commits, Angular, or Gitmoji,
then generates messages that match.

**You want editor integration without lock-in.** The built-in MCP server turns
nuntius into a tool that any MCP-capable editor can call — generate messages,
commit, and push without leaving your editor's AI workflow.

**You script your Git workflows.** Headless flags (`-g`, `-gc`, `-gcp`) and
structured JSON output (`--json`) make nuntius composable with shell scripts,
aliases, and automation pipelines.

**You want to stay in control.** Nothing happens unless you ask for it.
Generate-only is the default. Auto-commit and auto-push are opt-in.

## Install

### Homebrew

Works on macOS, Linux, and WSL.

```bash
brew install crazzyghost/homebrew-tap/nuntius
```

### Go

```bash
go install github.com/crazzyghost/nuntius/cmd/nuntius@latest
```

### Build from source

```bash
git clone https://github.com/crazzyghost/nuntius.git
cd nuntius
go build ./cmd/nuntius
```

The examples below assume `nuntius` is on your `PATH` or you are running the
built binary from the repo root.

## Quickstart

Run nuntius inside any Git repository.

### First run

1. Export your API key (skip this for Ollama or CLI mode):

```bash
export NUNTIUS_AI_API_KEY=<your-api-key>
```

2. Start nuntius:

```bash
nuntius
```

On first launch, an onboarding wizard walks you through choosing a provider,
model, connection mode, and behavior preferences. The result is saved to
`~/.nuntius/config.toml`. Rerun the wizard anytime with `nuntius --setup`.

## Common Workflows

### Interactive TUI

The TUI watches your repo for changes and lets you generate, commit, and push
interactively. Press `g` to generate, `c` to commit, `p` to push, or `?` for
the full key reference.

```bash
nuntius
nuntius --auto-commit --auto-push
nuntius --agent gemini --model gemini-3-flash-preview
```

### One-liner commands

Headless flags let you skip the TUI entirely.

```bash
nuntius -g              # generate message, print to stdout
nuntius -gc             # generate and commit
nuntius -gcp            # generate, commit, and push
nuntius -p              # push existing unpushed commits
```

### Diff control

Choose what goes into the AI prompt.

```bash
nuntius -g --diff-from=staged                        # staged changes only
git diff HEAD~3..HEAD | nuntius -g --diff-from=stdin  # pipe in an external diff
```

### Automation and scripting

Compose nuntius with other tools using JSON output or pipes.

```bash
nuntius -gcp --json                # structured JSON for tool integrations
nuntius -g | git commit -F -       # pipe the message straight into git commit
```

`--json` works with action flags (`-g`, `-gc`, `-gcp`, `-p`).

## MCP Integration

nuntius includes a built-in MCP server that runs over stdio. This lets
MCP-capable editors and AI agents call nuntius as a tool — generating commit
messages, committing, and pushing without switching to a terminal.

```bash
nuntius mcp
```

The server exposes five tools:

| Tool | Description |
| --- | --- |
| `generate_message` | Generate an AI-powered commit message from git changes |
| `commit` | Stage all changes and commit with a given message |
| `push` | Push commits to the remote |
| `generate_and_commit` | Generate a message, then stage and commit in one step |
| `status` | List changed files in the working tree |

### VS Code example

Add this to your VS Code MCP settings:

```json
{
  "servers": {
    "nuntius": {
      "type": "stdio",
      "command": "nuntius",
      "args": [
        "mcp",
        "--agent",
        "gemini",
        "--model",
        "gemini-3-flash-preview"
      ],
      "env": {
        "NUNTIUS_AI_API_KEY": "<your-api-key>"
      }
    }
  }
}
```

Once configured, your editor's AI agent can generate convention-aware commit
messages and drive the full commit-and-push workflow through nuntius.

## AI Providers

nuntius supports multiple AI backends. Each can be used via direct API calls or
through a local CLI tool.

| Provider | API mode | CLI mode | Default model (API) |
| --- | --- | --- | --- |
| **Claude** | ✓ | ✓ | `claude-haiku-4-5` |
| **Gemini** | ✓ | ✓ | `gemini-2.0-flash` |
| **Codex (OpenAI)** | ✓ | ✓ | `gpt-5.4-mini` |
| **Ollama** | ✓ (no key needed) | ✓ | `llama3.2` |
| **GitHub Copilot** | — | ✓ | — |
| **Custom command** | — | ✓ | — |

Set the provider and mode from the command line:

```bash
nuntius --agent gemini --agent-mode api
nuntius --agent claude --agent-mode cli
nuntius --agent ollama                    # no API key required
```

Or pin them in your config:

```toml
[ai]
provider = "gemini"
mode = "api"
model = "gemini-2.0-flash"
```

API-backed providers read the key from the `NUNTIUS_AI_API_KEY` environment variable.

## Configuration

nuntius loads configuration with this precedence:

1. CLI flags
2. Environment variables
3. Repo-local `.nuntius.toml`
4. Global `~/.nuntius/config.toml`
5. Built-in defaults

### Config example

```toml
[ai]
provider = "gemini"
mode = "api"
model = ""

[behavior]
auto_commit = false
auto_push = false
auto_update_check = true

[conventions]
# auto-detected from commit history; override with: conventional, angular, gitmoji
style = "conventional"
```

### Environment variables

```bash
export NUNTIUS_AI_PROVIDER=gemini
export NUNTIUS_AI_MODE=api
export NUNTIUS_AI_MODEL=gemini-3-flash-preview
export NUNTIUS_AI_API_KEY=<your-api-key>
```

## Troubleshooting

| Problem | Likely cause | What to do |
| --- | --- | --- |
| `current directory is not a Git repository` | You started nuntius outside a repo root. | `cd` into a Git repository before running `nuntius` or `nuntius --setup`. |
| `API key not set` | You selected API mode for a provider that needs a key. | Export `NUNTIUS_AI_API_KEY` and rerun the command. |
| `provider only supports cli mode` | You chose API mode for `copilot` or `custom`. | Switch the provider to `claude`, `codex`, `gemini`, or `ollama`, or use `mode = "cli"`. |
| `--json requires at least one of -g, -c, or -p` | JSON output is only supported in headless action mode. | Add an action flag such as `-g` or remove `--json`. |
| `--diff-from requires --generate (-g)` | Diff source selection only applies to generation. | Use `--diff-from` together with `-g`. |
| `--commit (-c) requires --generate (-g)` | Commit mode depends on a generated message. | Use `-gc` instead of `-c` alone. |
| `--push (-p) requires --commit (-c) when used with --generate (-g)` | Push can join a generate+commit pipeline, but not generate alone. | Use `-gcp` for the full pipeline, or `-p` alone to push existing commits. |
| You want to redo setup | Your current global config no longer matches your preferred agent setup. | Run `nuntius --setup` again to rewrite `~/.nuntius/config.toml`. |

For the full CLI surface:

```bash
nuntius --help
nuntius mcp --help
```