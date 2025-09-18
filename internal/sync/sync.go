package sync

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ternarybob/gitsync/internal/common"
)

type Syncer struct {
	jobName   string
	jobConfig *common.JobConfig
	tempDir   string
}

func NewSyncer(jobName string, jobConfig *common.JobConfig) (*Syncer, error) {
	tempDir := filepath.Join(os.TempDir(), "gitsync", jobName)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &Syncer{
		jobName:   jobName,
		jobConfig: jobConfig,
		tempDir:   tempDir,
	}, nil
}

func (s *Syncer) SyncAll(ctx context.Context) error {
	startTime := time.Now()

	// Use direct logging functions that work
	common.Infof("=== STARTING SYNC JOB: %s ===", s.jobName)
	common.Infof("Job: %s | Source: %s | Time: %s", s.jobName, s.jobConfig.Source, startTime.Format("2006-01-02 15:04:05"))

	if err := s.syncJob(ctx); err != nil {
		duration := time.Since(startTime)
		common.Errorf("=== FAILED SYNC JOB: %s === Duration: %v | Error: %v", s.jobName, duration, err)
		return err
	}

	duration := time.Since(startTime)
	common.Infof("=== COMPLETED SYNC JOB: %s === Duration: %v", s.jobName, duration)
	return nil
}

func (s *Syncer) syncJob(ctx context.Context) error {
	common.Infof("Syncing repository: %s from %s", s.jobName, s.jobConfig.Source)

	repoDir := filepath.Join(s.tempDir, sanitizeName(s.jobConfig.Source))

	// Set up authentication using job-level credentials
	if err := s.setupGitAuth(); err != nil {
		return fmt.Errorf("failed to setup git auth: %w", err)
	}

	exists, err := dirExists(repoDir)
	if err != nil {
		return err
	}

	if exists {
		if err := s.updateRepository(ctx, repoDir); err != nil {
			return fmt.Errorf("failed to update repository: %w", err)
		}
	} else {
		if err := s.cloneRepository(ctx, repoDir); err != nil {
			return fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	// Get branches to sync
	branchesToSync, err := s.getBranchesToSync(ctx, repoDir)
	if err != nil {
		return fmt.Errorf("failed to get branches to sync: %w", err)
	}

	if len(branchesToSync) == 0 {
		common.WithFields(map[string]interface{}{
			"job": s.jobName,
		}).Warn("No branches to sync")
		return nil
	}

	common.WithFields(map[string]interface{}{
		"job":      s.jobName,
		"branches": branchesToSync,
	}).Info("Found branches to sync")

	// Rewrite commit history if author replacement is configured
	if s.jobConfig.RewriteHistory && len(s.jobConfig.AuthorReplace) > 0 {
		if err := s.rewriteCommitAuthors(ctx, repoDir); err != nil {
			return fmt.Errorf("failed to rewrite commit authors: %w", err)
		}
	}

	// Sync each branch to all targets
	for _, branch := range branchesToSync {
		if err := s.syncBranchToTargets(ctx, repoDir, branch); err != nil {
			common.WithFields(map[string]interface{}{
				"job":    s.jobName,
				"branch": branch,
				"error":  err,
			}).Error("Failed to sync branch")
			continue
		}
	}

	return nil
}

func (s *Syncer) getBranchesToSync(ctx context.Context, repoDir string) ([]string, error) {
	// Fetch all remote branches and filter against configured patterns
	remoteBranches, err := s.getRemoteBranches(ctx, repoDir)
	if err != nil {
		return nil, err
	}

	var matchingBranches []string
	for _, remoteBranch := range remoteBranches {
		if s.jobConfig.ShouldSyncBranch(remoteBranch) {
			matchingBranches = append(matchingBranches, remoteBranch)
		}
	}

	return matchingBranches, nil
}

func (s *Syncer) getRemoteBranches(ctx context.Context, repoDir string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "branch", "-r", "--format=%(refname:short)")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote branches: %w", err)
	}

	var branches []string
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && strings.HasPrefix(line, "origin/") {
			branch := strings.TrimPrefix(line, "origin/")
			if branch != "HEAD" {
				branches = append(branches, branch)
			}
		}
	}

	return branches, nil
}

func (s *Syncer) syncBranchToTargets(ctx context.Context, repoDir string, branch string) error {
	// Checkout the branch
	if err := s.checkoutBranch(ctx, repoDir, branch); err != nil {
		return fmt.Errorf("failed to checkout branch %s: %w", branch, err)
	}

	// Get commit hash for this branch
	commitHash, err := s.getLatestCommit(ctx, repoDir)
	if err != nil {
		return fmt.Errorf("failed to get commit hash: %w", err)
	}

	// Authentication is already set up at job level, no need to change it

	// Sync to each target
	for _, target := range s.jobConfig.Targets {
		startTime := time.Now()

		common.WithFields(map[string]interface{}{
			"job":    s.jobName,
			"branch": branch,
			"target": target,
			"commit": commitHash,
		}).Info("Starting sync to target")

		if err := s.pushToTarget(ctx, repoDir, target, branch); err != nil {
			common.WithFields(map[string]interface{}{
				"job":      s.jobName,
				"branch":   branch,
				"target":   target,
				"error":    err,
				"duration": time.Since(startTime).Seconds(),
			}).Error("Failed to sync to target")
			continue
		}

		common.WithFields(map[string]interface{}{
			"job":      s.jobName,
			"branch":   branch,
			"target":   target,
			"commit":   commitHash,
			"duration": time.Since(startTime).Seconds(),
		}).Info("Successfully synced to target")
	}

	return nil
}

func (s *Syncer) cloneRepository(ctx context.Context, repoDir string) error {
	common.WithField("job", s.jobName).Debug("Cloning repository")

	cmd := exec.CommandContext(ctx, "git", "clone", s.jobConfig.Source, repoDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone: %w\n%s", err, output)
	}

	return nil
}

func (s *Syncer) updateRepository(ctx context.Context, repoDir string) error {
	common.WithField("job", s.jobName).Debug("Updating repository")

	cmd := exec.CommandContext(ctx, "git", "fetch", "origin", "--prune")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch: %w\n%s", err, output)
	}

	return nil
}

func (s *Syncer) checkoutBranch(ctx context.Context, repoDir, branch string) error {
	// Try to checkout local branch first
	cmd := exec.CommandContext(ctx, "git", "checkout", branch)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		// If local branch doesn't exist, create it from remote
		cmd = exec.CommandContext(ctx, "git", "checkout", "-b", branch, fmt.Sprintf("origin/%s", branch))
		cmd.Dir = repoDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to checkout branch %s: %w\n%s", branch, err, output)
		}
	} else {
		// Reset to match remote
		cmd = exec.CommandContext(ctx, "git", "reset", "--hard", fmt.Sprintf("origin/%s", branch))
		cmd.Dir = repoDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to reset branch %s: %w\n%s", branch, err, output)
		}
	}

	return nil
}

func (s *Syncer) pushToTarget(ctx context.Context, repoDir, target, branch string) error {
	targetName := sanitizeName(target)

	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", targetName)
	cmd.Dir = repoDir
	if _, err := cmd.CombinedOutput(); err != nil {
		cmd = exec.CommandContext(ctx, "git", "remote", "add", targetName, target)
		cmd.Dir = repoDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to add remote: %w\n%s", err, output)
		}
	} else {
		cmd = exec.CommandContext(ctx, "git", "remote", "set-url", targetName, target)
		cmd.Dir = repoDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to update remote: %w\n%s", err, output)
		}
	}

	// Use force push if override is enabled, otherwise regular push
	if s.jobConfig.Override {
		cmd = exec.CommandContext(ctx, "git", "push", targetName, fmt.Sprintf("%s:%s", branch, branch), "--force")
	} else {
		cmd = exec.CommandContext(ctx, "git", "push", targetName, fmt.Sprintf("%s:%s", branch, branch))
	}
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push: %w\n%s", err, output)
	}

	return nil
}

func (s *Syncer) getLatestCommit(ctx context.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get commit hash: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (s *Syncer) setupGitAuth() error {
	if s.jobConfig.GitToken != "" && s.jobConfig.GitUsername != "" {
		gitAskPass := filepath.Join(s.tempDir, "git-askpass.sh")
		content := fmt.Sprintf("#!/bin/sh\necho '%s'", s.jobConfig.GitToken)

		if err := os.WriteFile(gitAskPass, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to create askpass script: %w", err)
		}

		os.Setenv("GIT_ASKPASS", gitAskPass)
	}

	if s.jobConfig.SSHKeyPath != "" {
		os.Setenv("GIT_SSH_COMMAND", fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", s.jobConfig.SSHKeyPath))
	}

	return nil
}

func (s *Syncer) Cleanup() error {
	return os.RemoveAll(s.tempDir)
}

func dirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// rewriteCommitAuthors rewrites commit history to replace author information
func (s *Syncer) rewriteCommitAuthors(ctx context.Context, repoDir string) error {
	common.WithFields(map[string]interface{}{
		"job":          s.jobName,
		"replacements": len(s.jobConfig.AuthorReplace),
	}).Info("Rewriting commit authors")

	// Build the environment filter script for git filter-branch
	var filterScript strings.Builder
	for _, replacement := range s.jobConfig.AuthorReplace {
		if replacement.FromEmail != "" {
			filterScript.WriteString(fmt.Sprintf(`
if [ "$GIT_AUTHOR_EMAIL" = "%s" ]; then
    export GIT_AUTHOR_NAME="%s"
    export GIT_AUTHOR_EMAIL="%s"
    export GIT_COMMITTER_NAME="%s"
    export GIT_COMMITTER_EMAIL="%s"
fi`, replacement.FromEmail, replacement.ToName, replacement.ToEmail, replacement.ToName, replacement.ToEmail))
		}
		if replacement.FromName != "" && replacement.FromEmail == "" {
			filterScript.WriteString(fmt.Sprintf(`
if [ "$GIT_AUTHOR_NAME" = "%s" ]; then
    export GIT_AUTHOR_NAME="%s"
    export GIT_AUTHOR_EMAIL="%s"
    export GIT_COMMITTER_NAME="%s"
    export GIT_COMMITTER_EMAIL="%s"
fi`, replacement.FromName, replacement.ToName, replacement.ToEmail, replacement.ToName, replacement.ToEmail))
		}
	}

	if filterScript.Len() == 0 {
		return nil // No replacements to make
	}

	// Execute git filter-branch with the environment filter
	cmd := exec.CommandContext(ctx, "git", "filter-branch", "-f", "--env-filter", filterScript.String(), "--", "--all")
	cmd.Dir = repoDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git filter-branch failed: %w\nOutput: %s", err, string(output))
	}

	common.WithFields(map[string]interface{}{
		"job": s.jobName,
	}).Info("Successfully rewrote commit authors")

	return nil
}

func sanitizeName(name string) string {
	name = strings.ReplaceAll(name, "https://", "")
	name = strings.ReplaceAll(name, "http://", "")
	name = strings.ReplaceAll(name, "git@", "")
	name = strings.ReplaceAll(name, ".git", "")
	name = strings.ReplaceAll(name, "/", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, ".", "-")
	return name
}
