# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GitSync is a Git synchronization utility written in Go, ported from [git-mirror](https://github.com/RalfJung/git-mirror). It automatically synchronizes multiple Git repositories across different hosting platforms using scheduled jobs.

## Project Structure

```
├── cmd/gitsync/          # Main application entry point
├── internal/             # Internal packages
│   ├── config/          # Configuration management (TOML)
│   ├── logger/          # Structured logging with logrus
│   ├── store/           # BBolt transaction store
│   ├── sync/            # Git synchronization logic
│   └── scheduler/       # Cron job scheduling
├── scripts/             # Build and test scripts
│   ├── build.ps1/.sh   # Cross-platform build scripts
│   └── test.ps1/.sh    # Test runner scripts
├── deployments/         # Deployment configurations
│   ├── configs/        # Environment-specific configs
│   └── local/          # Local development files
└── gitsync.example.toml # Example configuration
```

## Development Commands

### Build Commands
```bash
# Build for development
./scripts/build.sh

# Build for production with optimizations
./scripts/build.sh --release

# Cross-compile for different platforms
./scripts/build.sh --os linux --arch amd64 --release
./scripts/build.ps1 -OS windows -Arch amd64 -Release

# Build with tests
./scripts/build.sh --test --clean
```

### Testing
```bash
# Run all tests
./scripts/test.sh

# Run tests with coverage
./scripts/test.sh --coverage --verbose

# Run specific tests
./scripts/test.sh --run "TestSync" --package "./internal/sync"
```

### Development Workflow
```bash
# Generate example configuration
go run ./cmd/gitsync -generate-config > gitsync.toml

# Validate configuration
go run ./cmd/gitsync -validate -config gitsync.toml

# Run a job immediately
go run ./cmd/gitsync -run-job "main-sync"

# Start service
go run ./cmd/gitsync -config gitsync.toml
```

## Architecture

### Core Components

1. **Configuration (`internal/config`)**
   - TOML-based configuration with environment variable support
   - Job definitions with cron schedules
   - Git authentication settings per job

2. **Scheduler (`internal/scheduler`)**
   - Uses robfig/cron for job scheduling
   - Supports cron expressions with seconds
   - Job timeout and retry handling

3. **Sync Engine (`internal/sync`)**
   - Git repository cloning and updating
   - Multi-target pushing with force updates
   - SSH key and token-based authentication

4. **Transaction Store (`internal/store`)**
   - BBolt embedded database for sync history
   - Transaction tracking with status and timing
   - Cleanup and statistics functionality

5. **Logging (`internal/logger`)**
   - Structured logging with logrus
   - JSON and text output formats
   - File rotation with lumberjack

### Configuration Structure

Jobs are defined in TOML with the following structure:
- Each job has a cron schedule and multiple repositories
- Each repository can sync to multiple targets
- Git authentication is configured per job
- Environment variables can be used for sensitive data

### Key Features

- **Multi-Repository Sync**: One job can handle multiple repositories
- **Multiple Targets**: Each source can sync to multiple destinations
- **Flexible Authentication**: Supports tokens and SSH keys
- **Transaction Tracking**: All sync operations are logged to BBolt
- **Foreground Application**: Runs as long-running foreground application with scheduled jobs
- **Statistics**: View sync history and performance metrics

## Dependencies

- **github.com/pelletier/go-toml/v2**: TOML configuration parsing
- **github.com/robfig/cron/v3**: Cron job scheduling
- **github.com/sirupsen/logrus**: Structured logging
- **go.etcd.io/bbolt**: Embedded key-value database
- **gopkg.in/natefinch/lumberjack.v2**: Log file rotation

## Usage

### Development
```bash
# Set up environment
cp deployments/local/env.example .env
# Edit .env with your tokens

# Build and run
./scripts/build.sh
./bin/gitsync.exe -generate-config > gitsync.toml  # On Windows
./bin/gitsync-linux -generate-config > gitsync.toml  # On Linux
# Edit gitsync.toml
./bin/gitsync.exe  # On Windows
./bin/gitsync-linux  # On Linux
```

### Production
```bash
# Build for production
./scripts/build.sh --release --os linux --arch amd64

# Run directly as foreground application
./bin/gitsync.exe -config deployments/configs/production.toml  # On Windows
./bin/gitsync-linux -config deployments/configs/production.toml  # On Linux

# Use process manager like PM2, supervisor, or screen/tmux for background execution
```

## Environment Variables

Common environment variables used in configurations:
- `GITHUB_TOKEN`, `GITLAB_TOKEN`, `BITBUCKET_TOKEN`: Git platform tokens
- `GIT_USERNAME`, `GIT_EMAIL`: Git user configuration
- `LOG_LEVEL`: Logging level (debug, info, warn, error)
- `ENVIRONMENT`: Runtime environment (development, production)

## Testing

The project includes comprehensive test scripts for:
- Unit tests with coverage reporting
- Integration tests for Git operations
- Performance benchmarks
- Race condition detection