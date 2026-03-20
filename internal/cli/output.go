package cli

import (
	"encoding/json"
	"fmt"
	"io"
)

// WriteJSON serializes the result as a single JSON line to w.
// HTML escaping is disabled so URLs and symbols render cleanly.
func WriteJSON(w io.Writer, r Result) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(r)
}

// WritePlain writes human-readable output to stdout/stderr matching
// the headless plain-text behavior.
//
// The commit message is written to stdout (clean for piping).
// Diagnostic lines (committed hash, pushed remote/branch, errors) go to stderr.
func WritePlain(stdout, stderr io.Writer, r Result) {
	if r.Message != "" {
		_, _ = fmt.Fprint(stdout, r.Message)
	}
	if r.Committed {
		_, _ = fmt.Fprintf(stderr, "committed %s\n", r.CommitHash)
	}
	if r.Pushed {
		remote := r.PushRemote
		if remote == "" {
			remote = "origin"
		}
		if r.PushBranch != "" {
			_, _ = fmt.Fprintf(stderr, "pushed to %s/%s\n", remote, r.PushBranch)
		} else {
			_, _ = fmt.Fprintf(stderr, "pushed to %s\n", remote)
		}
	}
	if !r.OK {
		_, _ = fmt.Fprintf(stderr, "Error: %s\n", r.Error)
	}
}
