package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ternarybob/gitsync/internal/config"
	"github.com/ternarybob/gitsync/internal/logger"
	"github.com/ternarybob/gitsync/internal/scheduler"
	"github.com/ternarybob/gitsync/internal/store"
	"github.com/ternarybob/gitsync/internal/version"
)

func main() {
	var (
		configPath     = flag.String("config", "gitsync.toml", "Path to configuration file")
		generateConfig = flag.Bool("generate-config", false, "Generate example configuration file")
		validateConfig = flag.Bool("validate", false, "Validate configuration file and exit")
		showVersion    = flag.Bool("version", false, "Show version and exit")
		runJob         = flag.String("run-job", "", "Run a specific job immediately and exit")
		showStats      = flag.Bool("stats", false, "Show sync statistics and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("GitSync v%s (build: %s)\n", version.GetVersion(), version.GetBuild())
		os.Exit(0)
	}

	if *generateConfig {
		fmt.Print(config.GenerateExampleConfig())
		os.Exit(0)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	if *validateConfig {
		fmt.Println("Configuration is valid")
		os.Exit(0)
	}

	if err := logger.Initialize(
		cfg.Logging.Level,
		cfg.Logging.Format,
		cfg.Logging.Output,
		cfg.Logging.MaxSize,
		cfg.Logging.MaxBackups,
		cfg.Logging.MaxAge,
	); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}

	logger.Infof("Starting GitSync v%s (build: %s)", version.GetVersion(), version.GetBuild())

	dataStore, err := store.New(cfg.Store.Path, cfg.Store.BucketName)
	if err != nil {
		logger.Fatalf("Failed to initialize store: %v", err)
	}
	defer dataStore.Close()

	if *showStats {
		stats, err := dataStore.GetStats()
		if err != nil {
			logger.Fatalf("Failed to get stats: %v", err)
		}
		fmt.Printf("Sync Statistics:\n")
		fmt.Printf("  Total Transactions: %v\n", stats["total_transactions"])
		fmt.Printf("  Successful Syncs:   %v\n", stats["successful_syncs"])
		fmt.Printf("  Failed Syncs:       %v\n", stats["failed_syncs"])
		fmt.Printf("  Average Duration:   %.2f seconds\n", stats["avg_duration_seconds"])
		os.Exit(0)
	}

	if *runJob != "" {
		logger.Infof("Running job immediately: %s", *runJob)
		s := scheduler.New(cfg, dataStore)
		if err := s.RunJobNow(*runJob); err != nil {
			logger.Fatalf("Failed to run job: %v", err)
		}
		logger.Info("Job completed")
		os.Exit(0)
	}

	sched := scheduler.New(cfg, dataStore)
	if err := sched.Start(); err != nil {
		logger.Fatalf("Failed to start scheduler: %v", err)
	}

	// Calculate enabled jobs
	enabledCount := 0
	for _, job := range cfg.Jobs {
		if job.Enabled {
			enabledCount++
		}
	}

	version.PrintBanner(cfg.Service.Name, cfg.Service.Environment, len(cfg.Jobs), enabledCount)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down GitSync...")
	sched.Stop()
	logger.Info("Shutdown complete")
}
