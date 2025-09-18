package common

import (
	"fmt"
	"strings"
)

// ANSI color codes
const (
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Reset  = "\033[0m"
)

// PrintBanner displays the GitSync banner with clean purple box style
func PrintBanner(serviceName, environment string, jobCount, enabledCount int) {
	fmt.Printf("\n")
	// Clean banner in purple box style
	fmt.Printf("%s┌────────────────────────────────────────────────────────────────────────────────┐%s\n", Purple, Reset)
	fmt.Printf("%s│%s %-78s %s│%s\n", Purple, White, centerText("GITSYNC SERVICE", 78), Purple, Reset)
	fmt.Printf("%s│%s %-78s %s│%s\n", Purple, White, centerText("Intelligent Git Repository Synchronization Engine", 78), Purple, Reset)
	fmt.Printf("%s├────────────────────────────────────────────────────────────────────────────────┤%s\n", Purple, Reset)
	fmt.Printf("%s│%s %-78s %s│%s\n", Purple, White, "Configuration loaded from gitsync.toml", Purple, Reset)
	fmt.Printf("%s│%s %-78s %s│%s\n", Purple, White, "Git availability verified, scheduler initialized", Purple, Reset)
	fmt.Printf("%s├────────────────────────────────────────────────────────────────────────────────┤%s\n", Purple, Reset)
	fmt.Printf("%s│%s %-78s %s│%s\n", Purple, White, fmt.Sprintf("Service: %s", serviceName), Purple, Reset)
	fmt.Printf("%s│%s %-78s %s│%s\n", Purple, White, fmt.Sprintf("Version: %s", GetVersion()), Purple, Reset)
	fmt.Printf("%s│%s %-78s %s│%s\n", Purple, White, fmt.Sprintf("Build: %s", GetBuild()), Purple, Reset)
	fmt.Printf("%s│%s %-78s %s│%s\n", Purple, White, fmt.Sprintf("Environment: %s", environment), Purple, Reset)
	fmt.Printf("%s│%s %-78s %s│%s\n", Purple, White, fmt.Sprintf("Jobs: %d configured, %d enabled", jobCount, enabledCount), Purple, Reset)
	fmt.Printf("%s│%s %-78s %s│%s\n", Purple, White, "Mode: Repository Synchronization", Purple, Reset)
	fmt.Printf("%s└────────────────────────────────────────────────────────────────────────────────┘%s\n", Purple, Reset)
	fmt.Printf("\n")

	// Print sync capabilities with emojis
	printSyncCapabilities()
	fmt.Printf("\n")
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	if len(text) >= width {
		return text[:width]
	}

	totalPadding := width - len(text)
	leftPad := totalPadding / 2
	rightPad := totalPadding - leftPad

	return strings.Repeat(" ", leftPad) + text + strings.Repeat(" ", rightPad)
}

// printSyncCapabilities displays the sync features with emojis
func printSyncCapabilities() {
	// Core Features
	fmt.Printf("🚀 Core Synchronization Features:\n")
	fmt.Printf("   • Multi-platform repository mirroring (GitHub, GitLab, Bitbucket)\n")
	fmt.Printf("   • Branch pattern filtering with wildcards (main, feature-*, *-sync)\n")
	fmt.Printf("   • Bidirectional sync support with separate job configurations\n")
	fmt.Printf("   • Safe vs force push modes per job (override control)\n")
	fmt.Printf("\n")

	// Scheduling & Automation
	fmt.Printf("⏰ Scheduling & Automation:\n")
	fmt.Printf("   • Cron-based scheduling with seconds precision\n")
	fmt.Printf("   • Concurrent job execution with timeout control\n")
	fmt.Printf("   • Automatic retry on transient failures\n")
	fmt.Printf("   • Real-time job status tracking\n")
	fmt.Printf("\n")

	// Security & Authentication
	fmt.Printf("🔐 Security & Authentication:\n")
	fmt.Printf("   • Token-based authentication (GitHub, GitLab, Bitbucket)\n")
	fmt.Printf("   • SSH key authentication support\n")
	fmt.Printf("   • Environment variable substitution for secrets\n")
	fmt.Printf("   • Per-job authentication configuration\n")
	fmt.Printf("\n")

	// Monitoring & Logging
	fmt.Printf("📊 Monitoring & Logging:\n")
	fmt.Printf("   • Structured logging with arbor logger\n")
	fmt.Printf("   • Dual console and file output\n")
	fmt.Printf("   • Performance metrics and timing data\n")
	fmt.Printf("   • Comprehensive error tracking\n")
	fmt.Printf("\n")

	// Command Line Options
	fmt.Printf("⚡ Command Line Options:\n")
	fmt.Printf("   • -config <file>    : Specify configuration file\n")
	fmt.Printf("   • -validate        : Validate configuration and exit\n")
	fmt.Printf("   • -run-job <name>  : Run specific job immediately\n")
	fmt.Printf("   • -version         : Show version information\n")
	fmt.Printf("   • -stats           : Display sync statistics\n")
}
