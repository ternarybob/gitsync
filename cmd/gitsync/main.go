package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/ternarybob/gitsync/internal/common"
	"github.com/ternarybob/gitsync/internal/scheduler"
)

func main() {
	var (
		configPath     = flag.String("config", "gitsync.toml", "Path to configuration file")
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

	cfg, err := common.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if *validateConfig {
		fmt.Println("Configuration is valid")
		os.Exit(0)
	}

	if err := common.InitLogger(&cfg.Logging); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	common.Infof("Starting GitSync v%s (build: %s)", common.GetVersion(), common.GetBuild())

	// Test git availability at startup
	if err := testGitAvailability(); err != nil {
		common.Fatalf("Git is not available: %v", err)
	}
	common.Info("Git availability verified")

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
	if err := sched.Start(); err != nil {
		common.Fatalf("Failed to start scheduler: %v", err)
	}

	// Calculate enabled jobs
	enabledJobs := cfg.GetEnabledJobs()

	common.PrintBanner(cfg.Service.Name, cfg.Service.Environment, len(cfg.Jobs.Names), len(enabledJobs))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	common.Info("Shutting down GitSync...")
	sched.Stop()
	common.Info("Shutdown complete")
}

func testGitAvailability() error {
	// Test if git command is available
	cmd := exec.Command("git", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git command not found or not executable: %w", err)
	}
	return nil
}
