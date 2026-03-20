package git

import "os/exec"

// ApplyEnvForTest exposes applyEnv for unit testing.
func ApplyEnvForTest(cmd *exec.Cmd) {
	applyEnv(cmd)
}
