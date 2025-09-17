package version

import (
	"fmt"
	"strings"
)

// These variables are set via ldflags during build
var (
	// Version is the semantic version from .version file
	Version = "dev"
	// Build is the build timestamp from .version file
	Build = "unknown"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
)

// GetVersion returns the semantic version
func GetVersion() string {
	return Version
}

// GetBuild returns the build timestamp
func GetBuild() string {
	return Build
}

// GetGitCommit returns the git commit hash
func GetGitCommit() string {
	return GitCommit
}

// GetFullVersion returns the complete version information
func GetFullVersion() string {
	if Build != "unknown" && Build != "dev" {
		return fmt.Sprintf("%s-%s", Version, Build)
	}
	return Version
}

// GetVersionInfo returns detailed version information
func GetVersionInfo() map[string]string {
	return map[string]string{
		"version": Version,
		"build":   Build,
		"commit":  GitCommit,
		"full":    GetFullVersion(),
	}
}

// PrintBanner displays the GitSync banner with version information
func PrintBanner(serviceName, environment string, jobCount, enabledCount int) {
	// ANSI color codes
	const (
		Cyan   = "\033[36m"
		Green  = "\033[32m"
		Yellow = "\033[33m"
		White  = "\033[37m"
		Reset  = "\033[0m"
	)

	banner := `
   _____ _ _   _____
  / ____(_) | / ____|
 | |  __ _| |_| (___  _   _ _ __   ___
 | | |_ | | __|\___ \| | | | '_ \ / __|
 | |__| | | |_ ____) | |_| | | | | (__
  \_____|_|\__|_____/ \__, |_| |_|\___|
                       __/ |
                      |___/
`

	fmt.Printf("%s%s%s\n", Cyan, banner, Reset)

	// Service information box
	fmt.Printf("%s┌────────────────────────────────────────────────────────────┐%s\n", Cyan, Reset)
	fmt.Printf("%s│%s %-58s %s│%s\n", Cyan, White, centerText("GitSync - Git Repository Synchronization", 58), Cyan, Reset)
	fmt.Printf("%s│%s %-58s %s│%s\n", Cyan, White, centerText("Automated Multi-Platform Git Mirroring Service", 58), Cyan, Reset)
	fmt.Printf("%s├────────────────────────────────────────────────────────────┤%s\n", Cyan, Reset)
	fmt.Printf("%s│%s %-58s %s│%s\n", Cyan, White, fmt.Sprintf("Service:     %s", serviceName), Cyan, Reset)
	fmt.Printf("%s│%s %-58s %s│%s\n", Cyan, White, fmt.Sprintf("Environment: %s", environment), Cyan, Reset)
	fmt.Printf("%s│%s %-58s %s│%s\n", Cyan, White, fmt.Sprintf("Version:     %s", GetFullVersion()), Cyan, Reset)
	if GitCommit != "unknown" {
		fmt.Printf("%s│%s %-58s %s│%s\n", Cyan, White, fmt.Sprintf("Commit:      %s", GitCommit[:8]), Cyan, Reset)
	}
	fmt.Printf("%s├────────────────────────────────────────────────────────────┤%s\n", Cyan, Reset)
	fmt.Printf("%s│%s %-58s %s│%s\n", Cyan, White, fmt.Sprintf("Jobs:        %d configured", jobCount), Cyan, Reset)
	fmt.Printf("%s│%s %-58s %s│%s\n", Cyan, White, fmt.Sprintf("Active:      %d enabled", enabledCount), Cyan, Reset)
	fmt.Printf("%s└────────────────────────────────────────────────────────────┘%s\n", Cyan, Reset)
	fmt.Printf("\n")

	// Status indicators
	if enabledCount > 0 {
		fmt.Printf("%s🔄 GitSync service is running. Press Ctrl+C to stop.%s\n", Green, Reset)
	} else {
		fmt.Printf("%s⚠️  No jobs are enabled. Check your configuration.%s\n", Yellow, Reset)
	}
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
