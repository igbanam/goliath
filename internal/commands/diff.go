package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/igbanam/goliath/internal/workspace"
)

// DiffCmd shows changed files since last stable point
var DiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show changed files since last stable point in time",
	Long: `Show files that have changed since the most recent stable git tag.

Stable points are git tags matching the pattern in goliath.yaml
(default: ^stable-.*).

Examples:
  goliath diff
  goliath diff --since HEAD~1
  goliath diff --since stable`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ws, err := loadWorkspaceFromArgs(cmd)
		if err != nil {
			return err
		}

		since, _ := cmd.Flags().GetString("since")

		files, err := getDiffFiles(ws, since)
		if err != nil {
			return err
		}

		if len(files) == 0 {
			fmt.Println("No changes since last stable point.")
			return nil
		}

		for _, f := range files {
			fmt.Println(f)
		}
		return nil
	},
}

func init() {
	DiffCmd.Flags().String("ws-dir", "", "Path to workspace directory")
	DiffCmd.Flags().String("since", "", "Git ref to diff against (default: latest stable tag)")
}

// getDiffFiles returns the list of changed files since the given ref
func getDiffFiles(ws *workspace.Workspace, since string) ([]string, error) {
	// Check if it's a git repo
	if !isGitRepo(ws.Dir) {
		return nil, fmt.Errorf("not a git repository")
	}

	// Determine the SHA to diff against
	sha, err := findStableSHA(ws, since)
	if err != nil {
		return nil, err
	}

	// Run git diff
	cmd := exec.Command("git", "diff", sha, "--name-only")
	cmd.Dir = ws.Dir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	// Also include untracked files
	cmd2 := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	cmd2.Dir = ws.Dir
	untracked, _ := cmd2.Output()

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}
	for _, line := range strings.Split(strings.TrimSpace(string(untracked)), "\n") {
		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// findStableSHA finds the SHA of the most recent stable tag
func findStableSHA(ws *workspace.Workspace, since string) (string, error) {
	if since != "" {
		// Resolve the given ref
		cmd := exec.Command("git", "rev-parse", since)
		cmd.Dir = ws.Dir
		output, err := cmd.Output()
		if err != nil {
			// Try as a tag pattern key
			pattern := ws.Config.TagPatterns.Stable
			if since == "stable" {
				if tag, err := findLatestTag(ws.Dir, pattern); err == nil {
					return getTagSHA(ws.Dir, tag)
				}
			}
			if since == "release" {
				if tag, err := findLatestTag(ws.Dir, ws.Config.TagPatterns.Release); err == nil {
					return getTagSHA(ws.Dir, tag)
				}
			}
			return "", fmt.Errorf("could not resolve ref %q: %w", since, err)
		}
		return strings.TrimSpace(string(output)), nil
	}

	// Find latest stable tag
	pattern := ws.Config.TagPatterns.Stable
	if pattern == "" {
		pattern = "^stable-.*"
	}

	tag, err := findLatestTag(ws.Dir, pattern)
	if err != nil {
		// No stable tag found, use first commit
		return getFirstCommit(ws.Dir)
	}

	return getTagSHA(ws.Dir, tag)
}

func isGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	return cmd.Run() == nil
}

func findLatestTag(dir, pattern string) (string, error) {
	// Get all tags
	cmd := exec.Command("git", "tag", "--list")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	tags := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Find the most recent tag matching the pattern
	// Git tags are listed in reverse chronological order by default with --sort
	cmd2 := exec.Command("git", "tag", "--list", "--sort=-creatordate", pattern)
	cmd2.Dir = dir
	output2, err := cmd2.Output()
	if err == nil && len(output2) > 0 {
		lines := strings.Split(strings.TrimSpace(string(output2)), "\n")
		if len(lines) > 0 && lines[0] != "" {
			return lines[0], nil
		}
	}

	// Fallback: filter manually
	for _, tag := range tags {
		if tag == "" {
			continue
		}
		matched, _ := matchPattern(pattern, tag)
		if matched {
			return tag, nil
		}
	}

	return "", fmt.Errorf("no tag found matching pattern %q", pattern)
}

func getTagSHA(dir, tag string) (string, error) {
	cmd := exec.Command("git", "rev-list", "-n", "1", tag)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getFirstCommit(dir string) (string, error) {
	cmd := exec.Command("git", "rev-list", "--max-parents=0", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// matchPattern does simple glob matching for tag patterns
func matchPattern(pattern, tag string) (bool, error) {
	// Convert regex-like pattern to simple matching
	// For ^stable-.* -> tag starts with "stable-"
	if strings.HasPrefix(pattern, "^") && strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(strings.TrimPrefix(pattern, "^"), ".*")
		return strings.HasPrefix(tag, prefix), nil
	}
	if strings.HasPrefix(pattern, "^v[0-9].*") {
		// Release pattern: ^v[0-9].*
		if len(tag) < 2 {
			return false, nil
		}
		return tag[0] == 'v' && tag[1] >= '0' && tag[1] <= '9', nil
	}
	// Exact match fallback
	return tag == pattern, nil
}
