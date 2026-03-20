package cli_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/crazzyghost/nuntius/internal/cli"
)

// --- WriteJSON tests ---

func TestWriteJSON_Success_GenerateOnly(t *testing.T) {
	t.Parallel()
	r := cli.Result{
		OK:      true,
		Message: "feat: add new feature\n",
	}
	var buf bytes.Buffer
	if err := cli.WriteJSON(&buf, r); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, buf.String())
	}

	if got["ok"] != true {
		t.Errorf("expected ok=true, got %v", got["ok"])
	}
	if got["message"] != "feat: add new feature\n" {
		t.Errorf("expected message field, got %v", got["message"])
	}
	if got["committed"] != false {
		t.Errorf("expected committed=false, got %v", got["committed"])
	}
	if got["pushed"] != false {
		t.Errorf("expected pushed=false, got %v", got["pushed"])
	}
}

func TestWriteJSON_Success_GenerateCommitPush(t *testing.T) {
	t.Parallel()
	r := cli.Result{
		OK:          true,
		Message:     "feat: do something\n",
		Committed:   true,
		CommitHash:  "abc1234",
		Pushed:      true,
		PushRemote:  "origin",
		PushBranch:  "main",
		SetUpstream: false,
	}
	var buf bytes.Buffer
	if err := cli.WriteJSON(&buf, r); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, buf.String())
	}

	if got["ok"] != true {
		t.Errorf("expected ok=true, got %v", got["ok"])
	}
	if got["committed"] != true {
		t.Errorf("expected committed=true, got %v", got["committed"])
	}
	if got["commit_hash"] != "abc1234" {
		t.Errorf("expected commit_hash=abc1234, got %v", got["commit_hash"])
	}
	if got["pushed"] != true {
		t.Errorf("expected pushed=true, got %v", got["pushed"])
	}
	if got["push_remote"] != "origin" {
		t.Errorf("expected push_remote=origin, got %v", got["push_remote"])
	}
	if got["push_branch"] != "main" {
		t.Errorf("expected push_branch=main, got %v", got["push_branch"])
	}
}

func TestWriteJSON_Error(t *testing.T) {
	t.Parallel()
	r := cli.Result{
		OK:    false,
		Error: "AI provider init failed",
		Stage: "generate",
	}
	var buf bytes.Buffer
	if err := cli.WriteJSON(&buf, r); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, buf.String())
	}

	if got["ok"] != false {
		t.Errorf("expected ok=false, got %v", got["ok"])
	}
	if got["error"] != "AI provider init failed" {
		t.Errorf("expected error field, got %v", got["error"])
	}
	if got["stage"] != "generate" {
		t.Errorf("expected stage=generate, got %v", got["stage"])
	}
}

func TestWriteJSON_PartialFailure(t *testing.T) {
	t.Parallel()
	r := cli.Result{
		OK:         false,
		Committed:  true,
		CommitHash: "def5678",
		Error:      "push rejected by remote",
		Stage:      "push",
	}
	var buf bytes.Buffer
	if err := cli.WriteJSON(&buf, r); err != nil {
		t.Fatalf("WriteJSON returned error: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, buf.String())
	}

	if got["ok"] != false {
		t.Errorf("expected ok=false, got %v", got["ok"])
	}
	if got["committed"] != true {
		t.Errorf("expected committed=true, got %v", got["committed"])
	}
	if got["commit_hash"] != "def5678" {
		t.Errorf("expected commit_hash=def5678, got %v", got["commit_hash"])
	}
}

func TestWriteJSON_ValidJSON_AllCases(t *testing.T) {
	t.Parallel()
	cases := []cli.Result{
		{OK: true},
		{OK: true, Message: "fix: something\n", Committed: true, CommitHash: "aaa"},
		{OK: false, Error: "boom", Stage: "commit"},
		{OK: true, Pushed: true, PushRemote: "upstream", PushBranch: "feature"},
	}
	for _, r := range cases {
		var buf bytes.Buffer
		if err := cli.WriteJSON(&buf, r); err != nil {
			t.Errorf("WriteJSON returned error for %+v: %v", r, err)
			continue
		}
		// Round-trip: must unmarshal back to a valid Result.
		var decoded cli.Result
		if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
			t.Errorf("JSON round-trip failed for %+v: %v\nraw: %s", r, err, buf.String())
		}
		if decoded.OK != r.OK {
			t.Errorf("round-trip OK mismatch: want %v got %v", r.OK, decoded.OK)
		}
	}
}

// --- WritePlain tests ---

func TestWritePlain_MessageToStdout(t *testing.T) {
	t.Parallel()
	r := cli.Result{OK: true, Message: "feat: hello world\n"}
	var stdout, stderr bytes.Buffer
	cli.WritePlain(&stdout, &stderr, r)

	if stdout.String() != "feat: hello world\n" {
		t.Errorf("expected message on stdout, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Errorf("expected empty stderr, got %q", stderr.String())
	}
}

func TestWritePlain_CommittedToStderr(t *testing.T) {
	t.Parallel()
	r := cli.Result{OK: true, Committed: true, CommitHash: "abc1234"}
	var stdout, stderr bytes.Buffer
	cli.WritePlain(&stdout, &stderr, r)

	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "committed abc1234") {
		t.Errorf("expected 'committed abc1234' on stderr, got %q", stderr.String())
	}
}

func TestWritePlain_PushedToStderr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		remote     string
		branch     string
		wantStderr string
	}{
		{
			name:       "remote and branch",
			remote:     "origin",
			branch:     "main",
			wantStderr: "pushed to origin/main\n",
		},
		{
			name:       "remote only no branch",
			remote:     "upstream",
			branch:     "",
			wantStderr: "pushed to upstream\n",
		},
		{
			name:       "empty remote defaults to origin",
			remote:     "",
			branch:     "feat",
			wantStderr: "pushed to origin/feat\n",
		},
		{
			name:       "empty remote and no branch defaults to origin",
			remote:     "",
			branch:     "",
			wantStderr: "pushed to origin\n",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := cli.Result{OK: true, Pushed: true, PushRemote: tc.remote, PushBranch: tc.branch}
			var stdout, stderr bytes.Buffer
			cli.WritePlain(&stdout, &stderr, r)
			if stderr.String() != tc.wantStderr {
				t.Errorf("expected stderr %q, got %q", tc.wantStderr, stderr.String())
			}
		})
	}
}

func TestWritePlain_ErrorToStderr(t *testing.T) {
	t.Parallel()
	r := cli.Result{OK: false, Error: "something went wrong"}
	var stdout, stderr bytes.Buffer
	cli.WritePlain(&stdout, &stderr, r)

	if stdout.Len() != 0 {
		t.Errorf("expected empty stdout, got %q", stdout.String())
	}
	want := "Error: something went wrong\n"
	if stderr.String() != want {
		t.Errorf("expected stderr %q, got %q", want, stderr.String())
	}
}
