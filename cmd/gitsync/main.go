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
	"github.com/ternarybob/gitsync/internal/scheduler"
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

	// Show banner first - before any logging
	common.PrintBanner(cfg.Service.Name, cfg.Service.Environment, len(cfg.Jobs.Names), len(enabledJobs))

	if err := common.InitLogger(&cfg.Logging); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	common.Infof("Starting GitSync v%s (build: %s)", common.GetVersion(), common.GetBuild())

	// Test git availability and version at startup
	gitVersion, err := testGitAvailability()
	if err != nil {
		common.Fatalf("Git is not available: %v", err)
	}
	common.Infof("Git availability verified: %s", gitVersion)

	if *showStats {
		fmt.Println("Statistics are now tracked via logging.")
		fmt.Println("Check the log files in ./logs/ for sync history and performance data.")
		os.Exit(0)
	}

	if *runJob != "" {
		common.Infof("Running job immediately: %s", *runJob)
		s := scheduler.New(cfg)
		if err := s.RunJobNow(*runJob); err != nil {
			common.Fatalf("Failed to run job: %v", err)
		}
		common.Info("Job completed")
		os.Exit(0)
	}

	sched := scheduler.New(cfg)

	// Run all enabled jobs once at startup
	common.Info("Running initial sync for all enabled jobs...")
	runInitialJobs(sched, cfg)

	if err := sched.Start(); err != nil {
		common.Fatalf("Failed to start scheduler: %v", err)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	common.Info("Shutting down GitSync...")
	sched.Stop()
	common.Info("Shutdown complete")
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

func runInitialJobs(sched *scheduler.Scheduler, cfg *common.Config) {
	enabledJobs := cfg.GetEnabledJobs()
	if len(enabledJobs) == 0 {
		common.Info("No enabled jobs found, skipping initial sync")
		return
	}

	var successCount, errorCount int
	common.Infof("Starting initial sync for %d enabled jobs", len(enabledJobs))

	for _, jobName := range enabledJobs {
		common.Infof("🔄 Running initial sync for job: %s", jobName)

		if err := sched.RunJobNow(jobName); err != nil {
			errorCount++
			common.Errorf("❌ INITIAL SYNC FAILED for job '%s': %v", jobName, err)
		} else {
			successCount++
			common.Infof("✅ Initial sync completed successfully for job: %s", jobName)
		}
	}

	common.Infof("Initial sync summary: %d successful, %d failed out of %d jobs",
		successCount, errorCount, len(enabledJobs))

	if errorCount > 0 {
		common.Errorf("⚠️  WARNING: %d jobs failed during initial sync - check configuration and connectivity", errorCount)
	} else {
		common.Info("🎉 All initial sync jobs completed successfully")
	}
}
