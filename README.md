# GitSync

A Git synchronization utility written in Go, ported from [git-mirror](https://github.com/RalfJung/git-mirror). GitSync automatically synchronizes multiple Git repositories across different hosting platforms.

## Features

- **Multi-Repository Sync**: Synchronize multiple repositories in a single job
- **Cron Scheduling**: Schedule sync jobs using cron expressions
- **Multiple Targets**: Push to multiple target repositories from a single source
- **Transaction Store**: Track sync history using BBolt embedded database
- **Flexible Authentication**: Support for tokens and SSH keys
- **Foreground Application**: Run as a long-running foreground application with scheduled jobs
- **Statistics**: View sync statistics and transaction history

## Installation

```bash
go install github.com/ternarybob/gitsync/cmd/gitsync@latest
```

Or build from source:

```bash
git clone https://github.com/ternarybob/gitsync.git
cd gitsync
./scripts/build.sh  # Creates platform-specific binary
```

## Configuration

GitSync uses TOML configuration files. Create a `gitsync.toml` file:

```toml
[service]
name = "gitsync"
environment = "production"

[[jobs]]
name = "main-sync"
description = "Synchronize repositories"
schedule = "0 */5 * * * *"  # Every 5 minutes
enabled = true
timeout = "5m"

  [[jobs.repos]]
  name = "my-project"
  source = "https://github.com/org/project.git"
  targets = ["https://gitlab.com/org/project.git"]
  branch = "main"

  [jobs.git]
  username = "git-bot"
  token_env_var = "GITHUB_TOKEN"
```

Generate an example configuration:

```bash
./bin/gitsync.exe -generate-config > gitsync.toml  # Windows
./bin/gitsync-linux -generate-config > gitsync.toml  # Linux
```

## Usage

### Run as Foreground Application

```bash
# Start the application with default config
./bin/gitsync.exe  # Windows
./bin/gitsync-linux  # Linux

# Use custom config file
./bin/gitsync.exe -config /path/to/config.toml  # Windows
./bin/gitsync-linux -config /path/to/config.toml  # Linux

# For background execution, use process managers:
# PM2: pm2 start ./bin/gitsync-linux --name gitsync -- -config gitsync.toml
# Screen: screen -S gitsync ./bin/gitsync-linux -config gitsync.toml
# Tmux: tmux new-session -d -s gitsync './bin/gitsync-linux -config gitsync.toml'
```

### One-Time Operations

```bash
# Validate configuration
./bin/gitsync.exe -validate  # Windows
./bin/gitsync-linux -validate  # Linux

# Run a specific job immediately
./bin/gitsync.exe -run-job "main-sync"  # Windows
./bin/gitsync-linux -run-job "main-sync"  # Linux

# View sync statistics
./bin/gitsync.exe -stats  # Windows
./bin/gitsync-linux -stats  # Linux

# Show version
./bin/gitsync.exe -version  # Windows
./bin/gitsync-linux -version  # Linux
```

## Environment Variables

- `GITHUB_TOKEN`: GitHub personal access token
- `GITLAB_TOKEN`: GitLab personal access token
- `LOG_LEVEL`: Set logging level (debug, info, warn, error)
- `ENVIRONMENT`: Set environment (development, production)

## Cron Schedule Format

GitSync uses standard cron expressions with seconds:

```
SEC MIN HOUR DAY MONTH WEEKDAY
```

Examples:
- `0 */5 * * * *` - Every 5 minutes
- `0 0 * * * *` - Every hour
- `0 0 0 * * *` - Daily at midnight
- `@hourly` - Every hour
- `@daily` - Every day

## Architecture

- **Configuration**: TOML-based configuration with environment variable support
- **Scheduler**: Uses robfig/cron for job scheduling with foreground execution
- **Storage**: BBolt embedded database for transaction history
- **Logging**: Structured logging with logrus
- **Git Operations**: Native Git commands via exec
- **Process Management**: Foreground application suitable for process managers

## License

MIT License - see [LICENSE](LICENSE) file for details
