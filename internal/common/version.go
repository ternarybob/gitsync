package common

import (
	"fmt"
	"strings"
)

var (
	Version   = "dev"
	Build     = "unknown"
	GitCommit = "unknown"
)

func PrintBanner(serviceName, environment string, jobCount, enabledCount int) {
	width := 60
	border := strings.Repeat("=", width)

	fmt.Println(border)
	fmt.Printf("%s\n", centerText("GitSync", width))
	fmt.Printf("%s\n", centerText(fmt.Sprintf("v%s (build: %s)", Version, Build), width))
	fmt.Println(border)
	fmt.Printf("  Service:      %s\n", serviceName)
	fmt.Printf("  Environment:  %s\n", environment)
	fmt.Printf("  Jobs:         %d configured, %d enabled\n", jobCount, enabledCount)
	fmt.Println(border)
}

func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text
}

func GetVersion() string {
	return Version
}

func GetBuild() string {
	return Build
}

func GetGitCommit() string {
	return GitCommit
}