package git

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/crazzyghost/nuntius/internal/events"
)

// Status returns a structured list of changed files in the current
// repository by parsing `git status --porcelain=v2` output.
// Returns an error if the current directory is not inside a git repository.
func Status() ([]events.FileStatus, error) {
	cmd := exec.Command("git", "status", "--porcelain=v2")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git status failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, fmt.Errorf("git status failed: %w", err)
	}

	return parsePorcelainV2(string(out))
}

// parsePorcelainV2 parses the output of `git status --porcelain=v2`
// into a slice of FileStatus entries.
//
// Porcelain v2 line formats:
//
//	Ordinary changed entries:
//	  1 XY <sub> <mH> <mI> <mW> <hH> <hI> <path>
//
//	Renamed/copied entries:
//	  2 XY <sub> <mH> <mI> <mW> <hH> <hI> <X><score> <path><sep><origPath>
//
//	Unmerged entries:
//	  u XY <sub> <m1> <m2> <m3> <mW> <h1> <h2> <h3> <path>
//
//	Untracked entries:
//	  ? <path>
//
//	Ignored entries:
//	  ! <path>
//
// XY is a two-character field where X = index (staged) status and
// Y = worktree (unstaged) status. '.' means no change.
func parsePorcelainV2(output string) ([]events.FileStatus, error) {
	var files []events.FileStatus
	lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		switch {
		case strings.HasPrefix(line, "1 "):
			// Ordinary changed entry
			fs, err := parseOrdinaryEntry(line)
			if err != nil {
				continue
			}
			files = append(files, fs...)

		case strings.HasPrefix(line, "2 "):
			// Renamed/copied entry
			fs, err := parseRenamedEntry(line)
			if err != nil {
				continue
			}
			files = append(files, fs...)

		case strings.HasPrefix(line, "u "):
			// Unmerged entry
			fs, err := parseUnmergedEntry(line)
			if err != nil {
				continue
			}
			files = append(files, fs...)

		case strings.HasPrefix(line, "? "):
			// Untracked
			path := line[2:]
			files = append(files, events.FileStatus{
				Path:   path,
				Status: "untracked",
				Staged: false,
			})

		default:
			// Ignored or unrecognized — skip
			continue
		}
	}

	return files, nil
}

// parseOrdinaryEntry parses a "1 XY ..." line.
// It may produce up to 2 FileStatus entries: one staged, one unstaged.
func parseOrdinaryEntry(line string) ([]events.FileStatus, error) {
	// Format: 1 XY <sub> <mH> <mI> <mW> <hH> <hI> <path>
	// Fields are space-separated; path is the 9th field onwards (may contain spaces).
	parts := strings.SplitN(line, " ", 9)
	if len(parts) < 9 {
		return nil, fmt.Errorf("malformed ordinary entry: %s", line)
	}

	xy := parts[1]
	path := parts[8]

	return xyToFileStatus(xy, path), nil
}

// parseRenamedEntry parses a "2 XY ... <path>\t<origPath>" line.
func parseRenamedEntry(line string) ([]events.FileStatus, error) {
	// Format: 2 XY <sub> <mH> <mI> <mW> <hH> <hI> <Xscore> <path>\t<origPath>
	parts := strings.SplitN(line, " ", 10)
	if len(parts) < 10 {
		return nil, fmt.Errorf("malformed renamed entry: %s", line)
	}

	xy := parts[1]
	pathPart := parts[9]

	// path and origPath are separated by a tab
	paths := strings.SplitN(pathPart, "\t", 2)
	newPath := paths[0]

	var files []events.FileStatus

	indexCode := xy[0]
	worktreeCode := xy[1]

	switch indexCode {
	case 'R':
		files = append(files, events.FileStatus{
			Path:   newPath,
			Status: "renamed",
			Staged: true,
		})
	case 'C':
		files = append(files, events.FileStatus{
			Path:   newPath,
			Status: "copied",
			Staged: true,
		})
	}

	if worktreeCode != '.' {
		files = append(files, events.FileStatus{
			Path:   newPath,
			Status: statusLabel(worktreeCode),
			Staged: false,
		})
	}

	return files, nil
}

// parseUnmergedEntry parses a "u XY ..." line.
func parseUnmergedEntry(line string) ([]events.FileStatus, error) {
	// Format: u XY <sub> <m1> <m2> <m3> <mW> <h1> <h2> <h3> <path>
	parts := strings.SplitN(line, " ", 11)
	if len(parts) < 11 {
		return nil, fmt.Errorf("malformed unmerged entry: %s", line)
	}

	path := parts[10]

	return []events.FileStatus{
		{Path: path, Status: "unmerged", Staged: false},
	}, nil
}

// xyToFileStatus converts an XY status pair and path into FileStatus entries.
// X = index (staged) status, Y = worktree (unstaged) status.
// '.' means no change in that area.
func xyToFileStatus(xy string, path string) []events.FileStatus {
	if len(xy) < 2 {
		return nil
	}

	var files []events.FileStatus
	indexCode := xy[0]
	worktreeCode := xy[1]

	// Staged change (index)
	if indexCode != '.' {
		files = append(files, events.FileStatus{
			Path:   path,
			Status: statusLabel(indexCode),
			Staged: true,
		})
	}

	// Unstaged change (worktree)
	if worktreeCode != '.' {
		files = append(files, events.FileStatus{
			Path:   path,
			Status: statusLabel(worktreeCode),
			Staged: false,
		})
	}

	return files
}

// statusLabel maps a single-character porcelain code to a human-readable label.
func statusLabel(code byte) string {
	switch code {
	case 'M':
		return "modified"
	case 'A':
		return "added"
	case 'D':
		return "deleted"
	case 'R':
		return "renamed"
	case 'C':
		return "copied"
	case 'T':
		return "type-changed"
	case 'U':
		return "unmerged"
	default:
		return "unknown"
	}
}
