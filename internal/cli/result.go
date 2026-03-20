// Package cli implements the headless execution pipeline for nuntius.
// When action flags (-g, -c, -p) are passed, nuntius skips the TUI and
// runs this pipeline instead, writing only the commit message to stdout
// and all diagnostic output to stderr.
package cli

// Result holds the structured outcome of a headless pipeline run.
type Result struct {
	OK          bool     `json:"ok"`
	Message     string   `json:"message,omitempty"`
	Files       []string `json:"files,omitempty"`
	DiffSource  string   `json:"diff_source,omitempty"`
	Committed   bool     `json:"committed"`
	CommitHash  string   `json:"commit_hash,omitempty"`
	Pushed      bool     `json:"pushed"`
	PushRemote  string   `json:"push_remote,omitempty"`
	PushBranch  string   `json:"push_branch,omitempty"`
	SetUpstream bool     `json:"set_upstream,omitempty"`
	Error       string   `json:"error,omitempty"`
	// Stage identifies which pipeline stage failed.
	// Values: "generate", "stage", "commit", "push".
	Stage string `json:"stage,omitempty"`
}

// ExitCode maps the result to a process exit code.
//
//	0 — success
//	1 — general / validation error
//	2 — no changes to process
//	3 — AI provider error
//	4 — git operation failed
func (r Result) ExitCode() int {
	if r.OK {
		return 0
	}
	switch r.Stage {
	case "generate":
		if r.Error == "no changes to generate a message for" {
			return 2
		}
		return 3
	case "stage", "commit", "push":
		return 4
	default:
		return 1
	}
}
