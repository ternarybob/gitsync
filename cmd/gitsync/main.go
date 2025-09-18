package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ternarybob/gitsync/internal/common"
	"github.com/ternarybob/gitsync/internal/services"
)

func main() {
	var (
		configPath     = flag.String("config", "", "Path to configuration file (defaults to gitsync.toml in executable directory)")
		validateConfig = flag.Bool("validate", false, "Validate configuration file and exit")
		showVersion    = flag.Bool("version", false, "Show version and exit")
		runJob         = flag.String("run-job", "", "Run a specific job immediately and exit")
		showStats      = flag.Bool("stats", false, "Show sync statistics and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("GitSync v%s (build: %s)\n", common.GetVersion(), common.GetBuild())
		os.Exit(0)
	}

	// Determine config file path
	finalConfigPath := *configPath
	if finalConfigPath == "" {
		// Default to gitsync.toml in the same directory as the executable
		execPath, err := os.Executable()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get executable path: %v\n", err)
			os.Exit(1)
		}
		execDir := filepath.Dir(execPath)
		finalConfigPath = filepath.Join(execDir, "gitsync.toml")
	}

	// Check if config file exists
	if _, err := os.Stat(finalConfigPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Configuration file not found: %s\n", finalConfigPath)
		fmt.Fprintf(os.Stderr, "Create a gitsync.toml file in the same directory as the executable, or specify one with -config\n")
		os.Exit(1)
	}

	cfg, err := common.Load(finalConfigPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if *validateConfig {
		fmt.Println("Configuration is valid")
		os.Exit(0)
	}

	// Calculate enabled jobs
	enabledJobs := cfg.GetEnabledJobs()

	// Initialize logger with config before any logging operations
	if err := common.InitLogger(&cfg.Logging); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	// Now get the configured logger
	logger := common.GetLogger()

	// Show banner after logger is initialized
	common.PrintBanner(cfg.Service.Name, cfg.Service.Environment, len(cfg.Jobs.Names), len(enabledJobs))

	logger.Info().Str("version", common.GetVersion()).Str("build", common.GetBuild()).Msg("Starting GitSync")

	// Test git availability and version at startup
	gitVersion, err := testGitAvailability()
	if err != nil {
		logger.Fatal().Err(err).Msg("Git is not available")
	}
	logger.Info().Str("git_version", gitVersion).Msg("Git availability verified")

	if *showStats {
		fmt.Println("Statistics are now tracked via logging.")
		fmt.Println("Check the log files in ./logs/ for sync history and performance data.")
		os.Exit(0)
	}

	if *runJob != "" {
		logger.Info().Str("job", *runJob).Msg("Running job immediately")
		s := services.NewScheduler(cfg)
		if err := s.RunJobNow(*runJob); err != nil {
			logger.Fatal().Err(err).Msg("Failed to run job")
		}
		logger.Info().Msg("Job completed")
		os.Exit(0)
	}

	sched := services.NewScheduler(cfg)

	// Run all enabled jobs once at startup
	logger.Info().Msg("Running initial sync for all enabled jobs...")
	runInitialJobs(sched, cfg)

	if err := sched.Start(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start scheduler")
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info().Msg("Shutting down GitSync...")
	sched.Stop()
	logger.Info().Msg("Shutdown complete")
}

func testGitAvailability() (string, error) {
	// Test if git command is available and get version
	cmd := exec.Command("git", "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git command not found or not executable: %w", err)
	}
	return string(output), nil
}

func runInitialJobs(sched *services.Scheduler, cfg *common.Config) {
	logger := common.GetLogger()
	enabledJobs := cfg.GetEnabledJobs()
	if len(enabledJobs) == 0 {
		logger.Info().Msg("No enabled jobs found, skipping initial sync")
		return
	}

	var successCount, errorCount int
	logger.Info().Int("job_count", len(enabledJobs)).Msg("Starting initial sync for enabled jobs")

	for _, jobName := range enabledJobs {
		logger.Info().Str("job", jobName).Msg("🔄 Running initial sync for job")

		if err := sched.RunJobNow(jobName); err != nil {
			errorCount++
			logger.Error().Str("job", jobName).Err(err).Msg("❌ INITIAL SYNC FAILED for job")
		} else {
			successCount++
			logger.Info().Str("job", jobName).Msg("✅ Initial sync completed successfully for job")
		}
	}

	logger.Info().Int("successful", successCount).Int("failed", errorCount).Int("total", len(enabledJobs)).Msg("Initial sync summary")

	if errorCount > 0 {
		logger.Error().Int("failed_count", errorCount).Msg("⚠️  WARNING: Jobs failed during initial sync - check configuration and connectivity")
	} else {
		logger.Info().Msg("🎉 All initial sync jobs completed successfully")
	}
}
