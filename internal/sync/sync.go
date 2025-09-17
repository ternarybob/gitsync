package sync

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ternarybob/gitsync/internal/config"
	"github.com/ternarybob/gitsync/internal/logger"
	"github.com/ternarybob/gitsync/internal/store"
)

type Syncer struct {
	config  *config.JobConfig
	store   *store.Store
	tempDir string
}

func NewSyncer(jobConfig *config.JobConfig, s *store.Store) (*Syncer, error) {
	tempDir := filepath.Join(os.TempDir(), "gitsync", jobConfig.Name)
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &Syncer{
		config:  jobConfig,
		store:   s,
		tempDir: tempDir,
	}, nil
}

func (s *Syncer) SyncAll(ctx context.Context) error {
	logger.WithField("job", s.config.Name).Info("Starting sync job")

	for _, repo := range s.config.Repos {
		if err := s.syncRepo(ctx, repo); err != nil {
			logger.WithFields(map[string]interface{}{
				"job":   s.config.Name,
				"repo":  repo.Name,
				"error": err,
			}).Error("Failed to sync repository")
			continue
		}
	}

	return nil
}

func (s *Syncer) syncRepo(ctx context.Context, repo config.RepoConfig) error {
	logger.WithFields(map[string]interface{}{
		"job":    s.config.Name,
		"repo":   repo.Name,
		"source": repo.Source,
	}).Info("Syncing repository")

	repoDir := filepath.Join(s.tempDir, sanitizeName(repo.Name))

	if err := s.setupGitAuth(); err != nil {
		return fmt.Errorf("failed to setup git auth: %w", err)
	}

	exists, err := dirExists(repoDir)
	if err != nil {
		return err
	}

	var commitHash string
	if exists {
		commitHash, err = s.updateRepository(ctx, repoDir, repo)
	} else {
		commitHash, err = s.cloneRepository(ctx, repoDir, repo)
	}

	if err != nil {
		return fmt.Errorf("failed to get repository: %w", err)
	}

	for _, target := range repo.Targets {
		tx := &store.Transaction{
			JobName:    s.config.Name,
			RepoName:   repo.Name,
			Source:     repo.Source,
			Target:     target,
			Status:     store.StatusRunning,
			CommitHash: commitHash,
			StartTime:  time.Now(),
		}

		if err := s.store.SaveTransaction(tx); err != nil {
			logger.WithField("error", err).Error("Failed to save transaction")
		}

		if err := s.pushToTarget(ctx, repoDir, target, repo.Branch); err != nil {
			tx.Status = store.StatusFailed
			tx.Error = err.Error()
			tx.EndTime = time.Now()
			s.store.SaveTransaction(tx)

			logger.WithFields(map[string]interface{}{
				"repo":   repo.Name,
				"target": target,
				"error":  err,
			}).Error("Failed to push to target")
			continue
		}

		tx.Status = store.StatusSuccess
		tx.EndTime = time.Now()
		s.store.SaveTransaction(tx)

		logger.WithFields(map[string]interface{}{
			"repo":   repo.Name,
			"target": target,
			"commit": commitHash,
		}).Info("Successfully synced to target")
	}

	return nil
}

func (s *Syncer) cloneRepository(ctx context.Context, repoDir string, repo config.RepoConfig) (string, error) {
	logger.WithField("repo", repo.Name).Debug("Cloning repository")

	cmd := exec.CommandContext(ctx, "git", "clone", repo.Source, repoDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to clone: %w\n%s", err, output)
	}

	if repo.Branch != "" && repo.Branch != "main" && repo.Branch != "master" {
		cmd = exec.CommandContext(ctx, "git", "checkout", repo.Branch)
		cmd.Dir = repoDir
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to checkout branch %s: %w\n%s", repo.Branch, err, output)
		}
	}

	return s.getLatestCommit(ctx, repoDir)
}

func (s *Syncer) updateRepository(ctx context.Context, repoDir string, repo config.RepoConfig) (string, error) {
	logger.WithField("repo", repo.Name).Debug("Updating repository")

	cmd := exec.CommandContext(ctx, "git", "fetch", "origin")
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to fetch: %w\n%s", err, output)
	}

	branch := repo.Branch
	if branch == "" {
		branch = "main"
	}

	cmd = exec.CommandContext(ctx, "git", "reset", "--hard", fmt.Sprintf("origin/%s", branch))
	cmd.Dir = repoDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to reset: %w\n%s", err, output)
	}

	return s.getLatestCommit(ctx, repoDir)
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

	if branch == "" {
		branch = "main"
	}

	cmd = exec.CommandContext(ctx, "git", "push", targetName, branch, "--force")
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
	if s.config.Git.Token != "" && s.config.Git.Username != "" {
		gitAskPass := filepath.Join(s.tempDir, "git-askpass.sh")
		content := fmt.Sprintf("#!/bin/sh\necho '%s'", s.config.Git.Token)

		if err := os.WriteFile(gitAskPass, []byte(content), 0755); err != nil {
			return fmt.Errorf("failed to create askpass script: %w", err)
		}

		os.Setenv("GIT_ASKPASS", gitAskPass)
	}

	if s.config.Git.SSHKeyPath != "" {
		os.Setenv("GIT_SSH_COMMAND", fmt.Sprintf("ssh -i %s -o StrictHostKeyChecking=no", s.config.Git.SSHKeyPath))
	}

	if s.config.Git.CommitAuthor != "" {
		exec.Command("git", "config", "--global", "user.name", s.config.Git.CommitAuthor).Run()
	}
	if s.config.Git.CommitEmail != "" {
		exec.Command("git", "config", "--global", "user.email", s.config.Git.CommitEmail).Run()
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
