package jujutsu

import (
	"claude-squad/config"
	"claude-squad/log"
	"claude-squad/session/git"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// JujutsuWorkspace manages Jujutsu workspace operations for a session.
type JujutsuWorkspace struct {
	// Path to the repository
	repoPath string
	// Name of the workspace
	workspaceName string
	// Path to the workspace
	workspacePath string
	// Name of the session
	sessionName string
	// Bookmark name for the workspace
	bookmarkName string
	// Base commit change ID for the workspace
	baseChangeID string
}

func getWorkspaceDirectory() (string, error) {
	configDir, err := config.GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "worktrees"), nil
}

// NewJujutsuWorkspace creates a new JujutsuWorkspace instance.
func NewJujutsuWorkspace(repoPath string, sessionName string) (*JujutsuWorkspace, error) {
	cfg := config.LoadConfig()
	bookmarkName := fmt.Sprintf("%s%s", cfg.BranchPrefix, sessionName)
	// Sanitize the final branch name to handle invalid characters from any source
	// (e.g., backslashes from Windows domain usernames like DOMAIN\user)
	bookmarkName = git.SanitizeBranchName(bookmarkName)

	// Convert repoPath to absolute path
	absPath, err := filepath.Abs(repoPath)
	if err != nil {
		log.ErrorLog.Printf("git worktree path abs error, falling back to repoPath %s: %s", repoPath, err)
		// If we can't get absolute path, use original path as fallback
		absPath = repoPath
	}

	repoPath, err = runJJ("root", "-R", absPath)
	if err != nil {
		return nil, err
	}

	worktreeDir, err := getWorkspaceDirectory()
	if err != nil {
		return nil, err
	}

	// Use sanitized branch name for the worktree directory name
	worktreePath := filepath.Join(worktreeDir, bookmarkName)
	worktreePath = worktreePath + "_" + fmt.Sprintf("%x", time.Now().UnixNano())
	return &JujutsuWorkspace{
		repoPath:      repoPath,
		workspacePath: worktreePath,
		workspaceName: bookmarkName,
		sessionName:   sessionName,
		bookmarkName:  bookmarkName,
	}, nil
}

// NewJujutsuWorkspaceFromStorage reconstructs a JujutsuWorkspace from persisted data.
func NewJujutsuWorkspaceFromStorage(repoPath string, workspacePath string, sessionName string, bookmarkName string, baseChangeID string) *JujutsuWorkspace {
	return &JujutsuWorkspace{
		repoPath:      repoPath,
		workspacePath: workspacePath,
		workspaceName: git.SanitizeBranchName(bookmarkName),
		sessionName:   sessionName,
		bookmarkName:  bookmarkName,
		baseChangeID:  baseChangeID,
	}
}

func runJJ(args ...string) (string, error) {
	cmd := exec.Command("jj", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("jj command failed: %s (%w)", output, err)
	}
	return string(output), nil
}

// runJJCommand executes a jj command in the given directory and returns its output.
func (j *JujutsuWorkspace) runJJCommand(args ...string) (string, error) {
	return runJJ(append([]string{"-R", j.workspacePath}, args...)...)
}

func (j *JujutsuWorkspace) runJJCommandInRepo(args ...string) (string, error) {
	return runJJ(append([]string{"-R", j.repoPath}, args...)...)
}

func (j *JujutsuWorkspace) GetWorktreePath() string {
	return j.workspacePath
}

func (j *JujutsuWorkspace) GetBranchName() string {
	return j.bookmarkName
}

func (j *JujutsuWorkspace) GetRepoPath() string {
	return j.repoPath
}

func (j *JujutsuWorkspace) GetRepoName() string {
	return filepath.Base(j.repoPath)
}

func (j *JujutsuWorkspace) GetBaseCommitSHA() string {
	return j.baseChangeID
}

func (j *JujutsuWorkspace) Setup() error {
	name := git.SanitizeBranchName(j.bookmarkName)
	_, err := j.runJJCommandInRepo("workspace", "add", "--name", name, j.workspacePath)
	return err
}

// Cleanup removes both the workspace and the bookmark
func (j *JujutsuWorkspace) Cleanup() error {
	err := j.Remove()
	if err != nil {
		return err
	}
	// TODO: rm -rf
	return nil
}

// Remove removes the workspace but keeps the branch
func (j *JujutsuWorkspace) Remove() error {
	// TODO: implement jj workspace forget (keep bookmark)
	_, err := j.runJJCommandInRepo("workspace", "forget", j.workspaceName)
	return err
}

func (j *JujutsuWorkspace) Prune() error {
	// Jujutsu does not have a prune equivalent; this is a no-op.
	return nil
}

func (j *JujutsuWorkspace) PushChanges(commitMessage string, open bool) error {
	// TODO: implement jj describe + jj git push
	return fmt.Errorf("jujutsu PushChanges not implemented")
}

func (j *JujutsuWorkspace) CommitChanges(commitMessage string) error {
	// TODO: implement jj describe + jj new
	return fmt.Errorf("jujutsu CommitChanges not implemented")
}

func (j *JujutsuWorkspace) IsDirty() (bool, error) {
	output, err := j.runJJCommand(j.workspacePath, "diff", "--stat")
	if err != nil {
		return false, fmt.Errorf("failed to check workspace status: %w", err)
	}
	return strings.TrimSpace(output) != "", nil
}

func (j *JujutsuWorkspace) IsBranchCheckedOut() (bool, error) {
	// TODO: determine if the bookmark is active in the main workspace
	return false, fmt.Errorf("jujutsu IsBranchCheckedOut not implemented")
}

func (j *JujutsuWorkspace) OpenBranchURL() error {
	// TODO: open bookmark URL in browser (requires git remote integration)
	return fmt.Errorf("jujutsu OpenBranchURL not implemented")
}

func (j *JujutsuWorkspace) Diff() *git.DiffStats {
	stats := &git.DiffStats{}

	content, err := j.runJJCommand(j.workspacePath, "diff", "--git")
	if err != nil {
		stats.Error = err
		return stats
	}

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			stats.Added++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			stats.Removed++
		}
	}
	stats.Content = content

	return stats
}
