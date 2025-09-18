# GitSync

A Git synchronization utility written in Go, ported from [git-mirror](https://github.com/RalfJung/git-mirror). GitSync automatically synchronizes Git repositories across different hosting platforms with advanced branch filtering and bidirectional sync capabilities.

## Features

- **Smart Branch Filtering**: Sync specific branches using wildcard patterns (`main`, `feature-*`, `*-sync`)
- **Bidirectional Sync**: Configure separate jobs for different sync directions
- **Cron Scheduling**: Schedule sync jobs using robfig/cron expressions (with seconds support)
- **Multiple Targets**: Push to multiple target repositories from a single source
- **Override Control**: Configure force push behavior per job for safe/unsafe branches
- **Professional Logging**: Structured logging with arbor logger, dual console/file output
- **Foreground Application**: Run as a long-running foreground application with scheduled jobs
- **Git Validation**: Automatic git availability check at startup

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

GitSync uses TOML configuration files with a clean, hierarchical structure supporting job groups and per-job settings.

### Configuration Structure

```toml
# Service configuration
[service]
name = "gitsync"                    # Service identifier
environment = "development"         # Environment: development, staging, production

# Jobs configuration - shared settings for all jobs
[jobs]
names = ["main-sync", "feature-sync", "bidirectional-up"]  # List of job names
schedule = "*/5 * * * *"            # Cron expression (with seconds support)
timeout = "5m"                      # Shared timeout for all jobs

# Individual job definitions
["main-sync"]
description = "Sync main branch safely"
enabled = true
source = "https://github.com/myorg/project.git"
targets = [
  "https://gitlab.com/myorg/project.git",
  "https://bitbucket.org/myorg/project.git"
]
branches = ["main"]                 # Only sync main branch
override = false                    # No force push for main branch
git_username = "sync-bot"
git_token = "${GITHUB_TOKEN}"       # Environment variable
commit_author = "Git Sync Bot"
commit_email = "sync@example.com"

["feature-sync"]
description = "Sync feature branches with wildcards"
enabled = true
source = "https://github.com/myorg/features.git"
targets = ["https://backup.myorg.com/features.git"]
branches = ["feature-*", "*-sync"]  # Wildcard patterns
override = true                     # Allow force push for feature branches
git_username = "backup-user"
git_token = "${BACKUP_TOKEN}"

["bidirectional-up"]
description = "Sync back to main repo (bidirectional)"
enabled = false
source = "https://backup.myorg.com/features.git"
targets = ["https://github.com/myorg/features.git"]
branches = ["*-sync"]               # Only sync-tagged branches back
override = true                     # Force push allowed
git_username = "sync-bot"
git_token = "${GITHUB_TOKEN}"

# Logging configuration
[logging]
level = "debug"                     # Log level: debug, info, warn, error
format = "text"                     # Output format: text, json
output = "both"                     # Output: stdout, both (console + file)
max_file_size = 100                 # Log file max size in MB
max_backups = 3                     # Number of backup log files
max_age = 3                         # Days to retain log files
```

## Key Configuration Options

### Branch Filtering
- `branches = ["main"]` - Sync only the main branch
- `branches = ["feature-*"]` - Sync all branches starting with "feature-"
- `branches = ["*-sync"]` - Sync all branches ending with "-sync"
- `branches = ["main", "develop", "feature-*"]` - Mix exact and wildcard patterns

### Override Behavior
- `override = false` - Safe push, will fail if there are conflicts (recommended for main branches)
- `override = true` - Force push, will overwrite target branch (use for feature/sync branches)

### Environment Variables
Use `${VAR}` syntax in configuration files:

```toml
git_token = "${GITHUB_TOKEN}"
git_username = "${GIT_USER:-default-user}"  # With fallback value
```

## Bidirectional Sync Example

Configure two separate jobs for bidirectional synchronization:

```toml
[jobs]
names = ["sync-down", "sync-up"]
schedule = "*/5 * * * *"
timeout = "10m"

# Sync main branch from primary to backup
["sync-down"]
description = "Primary to backup sync"
enabled = true
source = "https://github.com/primary/repo.git"
targets = ["https://backup.example.com/repo.git"]
branches = ["main", "develop"]
override = false

# Sync feature branches back from backup to primary
["sync-up"]
description = "Backup to primary sync"
enabled = true
source = "https://backup.example.com/repo.git"
targets = ["https://github.com/primary/repo.git"]
branches = ["*-sync"]  # Only branches ending with -sync
override = true
```

## Usage

### Command Line Options

```bash
# Show version information
./bin/gitsync.exe -version

# Validate configuration file
./bin/gitsync.exe -validate -config gitsync.toml

# Run a specific job immediately (for testing)
./bin/gitsync.exe -run-job "main-sync" -config gitsync.toml

# View sync statistics (from logs)
./bin/gitsync.exe -stats
```

### Run as Foreground Application

```bash
# Start with default config (gitsync.toml)
./bin/gitsync.exe  # Windows
./bin/gitsync-linux  # Linux

# Use custom config file
./bin/gitsync.exe -config /path/to/config.toml

# For background execution, use process managers:
# PM2: pm2 start ./bin/gitsync-linux --name gitsync -- -config gitsync.toml
# Screen: screen -S gitsync ./bin/gitsync-linux -config gitsync.toml
# Tmux: tmux new-session -d -s gitsync './bin/gitsync-linux -config gitsync.toml'
```

### Startup Validation

GitSync automatically validates git availability on startup:
- Checks if `git` command is available in PATH
- Verifies git can be executed
- Fails fast with clear error message if git is not available

## Environment Variables

Set these environment variables for authentication:
- `GITHUB_TOKEN`: GitHub personal access token
- `GITLAB_TOKEN`: GitLab personal access token
- `BACKUP_TOKEN`: Token for backup repositories
- `LOG_LEVEL`: Override logging level (debug, info, warn, error)
- `ENVIRONMENT`: Override environment setting

## Cron Schedule Format

GitSync uses robfig/cron expressions with **seconds support**:

```
SEC MIN HOUR DAY MONTH WEEKDAY
```

Examples:
- `*/30 * * * * *` - Every 30 seconds
- `0 */5 * * * *` - Every 5 minutes
- `0 0 */2 * * *` - Every 2 hours
- `0 0 0 * * *` - Daily at midnight
- `@hourly` - Every hour
- `@daily` - Every day at midnight

## How Git Sync Works

GitSync performs intelligent repository synchronization with branch filtering and safe/unsafe push modes.

### Sync Process

1. **Git Availability Check**
   - Verify git command is available at startup
   - Fail fast if git is not installed or accessible

2. **Branch Discovery & Filtering**
   - Fetch all remote branches from source repository
   - Apply wildcard pattern matching (`main`, `feature-*`, `*-sync`)
   - Only sync branches that match configured patterns

3. **Authentication Setup**
   - Configure git credentials per job (token or SSH key)
   - Set commit author information
   - Handle different auth for source vs target if needed

4. **Smart Synchronization**
   - Clone source repository to temporary directory (first run)
   - Fetch and reset to latest changes (subsequent runs)
   - Checkout each matching branch individually
   - Push to targets with override control:
     - `override = false`: Safe push, fails on conflicts
     - `override = true`: Force push, overwrites target

5. **Logging & Monitoring**
   - Structured logging with arbor logger
   - Dual output to console and timestamped log files
   - Performance metrics (duration, commit hashes)
   - Error tracking and retry information

### Sync Workflow Diagram

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Cron Schedule │───▶│  Job Execution   │───▶│  Branch Filter  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                                         │
                                                         ▼
                              ┌─────────────────────────────────────┐
                              │         Repository Sync             │
                              │                                     │
                              │ 1. Git availability check           │
                              │ 2. Setup git authentication         │
                              │ 3. Clone/update source repo         │
                              │ 4. Fetch all remote branches        │
                              │ 5. Filter by wildcard patterns      │
                              │ 6. For each matching branch:        │
                              │    - Checkout branch                │
                              │    - Push to targets (safe/force)   │
                              │    - Log results & performance      │
                              └─────────────────────────────────────┘
```

### Authentication Methods

**Token-based Authentication:**
```toml
git_username = "sync-bot"
git_token = "${GITHUB_TOKEN}"  # Uses token from environment
```

**SSH Key Authentication:**
```toml
ssh_key_path = "/path/to/private/key"
# OR
ssh_key_env = "SSH_KEY_PATH"  # Path from environment
```

## Branch Pattern Examples

GitSync supports powerful wildcard patterns for branch filtering:

```toml
# Exact branch matching
branches = ["main", "develop"]

# Wildcard patterns
branches = ["feature-*"]     # feature-auth, feature-payment
branches = ["*-sync"]        # auth-sync, payment-sync
branches = ["hotfix/*"]      # hotfix/critical, hotfix/security

# Combined patterns
branches = ["main", "develop", "feature-*", "*-sync"]
```

## Troubleshooting

### Common Issues

**Git not found:**
```
Git is not available: git command not found or not executable
```
- Install git and ensure it's in your system PATH
- Verify with: `git --version`

**Authentication failures:**
```
Failed to push: authentication required
```
- Check token has correct permissions (repo read/write)
- Verify environment variables are set correctly
- Test git access manually: `git clone <repo-url>`

**Branch filtering not working:**
```
No branches to sync
```
- Check branch patterns match actual branch names
- Use `git branch -r` to list remote branches
- Test patterns with `-run-job` for immediate feedback

### Log Analysis

GitSync creates detailed logs in `./logs/` directory:
```bash
# View latest log
ls -la logs/
tail -f logs/log-$(date +%Y-%m-%d)*.log

# Search for errors
grep -i error logs/*.log

# Check specific job performance
grep "job=main-sync" logs/*.log
```

## Use Cases

### 1. Multi-Platform Repository Mirroring
- **Primary Development**: GitHub
- **Automatic Mirroring**: GitLab, Bitbucket, private Git servers
- **Branch Strategy**: Sync main/develop safely, sync feature branches with override
- **Benefits**: Platform redundancy, compliance requirements, team access

### 2. Bidirectional Development Workflow
- **Main Flow**: Primary repo → backup repo (main branches)
- **Feature Flow**: Backup repo → primary repo (feature branches ending in `-sync`)
- **Use Case**: Contractors work in isolated repo, changes flow back through tagged branches
- **Safety**: Override disabled for main branches, enabled for sync branches

### 3. Enterprise Backup Strategy
- **Multi-Site Redundancy**: Sync to multiple geographic locations
- **Compliance**: Maintain copies for regulatory requirements
- **Disaster Recovery**: Automated failover to backup repositories
- **Scheduling**: Different frequencies for different repo priorities

### 4. Branch-Specific Synchronization
- **Development Branches**: `feature-*` patterns sync to development servers
- **Release Branches**: `release-*` patterns sync to staging environments
- **Hotfix Branches**: `hotfix-*` patterns sync immediately with high priority
- **Custom Patterns**: Any wildcard pattern to match team workflows

## Performance & Scalability

- **Concurrent Jobs**: Multiple jobs run independently
- **Efficient Cloning**: Reuses local clones, only fetches changes
- **Branch Filtering**: Only processes matching branches, saves bandwidth
- **Timeout Control**: Prevents hung operations from blocking other jobs
- **Resource Management**: Temporary directories cleaned up automatically

## Security Considerations

- **Token Scope**: Use minimal required permissions (repo read/write)
- **Environment Variables**: Store sensitive tokens in environment, not config files
- **SSH Keys**: Support for key-based authentication
- **Override Control**: Disable force push for protected branches
- **Audit Trail**: Comprehensive logging of all sync operations

## Contributing

1. Fork the repository
2. Create feature branch: `git checkout -b feature/amazing-feature`
3. Commit changes: `git commit -m 'Add amazing feature'`
4. Push to branch: `git push origin feature/amazing-feature`
5. Open Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Original [git-mirror](https://github.com/RalfJung/git-mirror) Python implementation
- [robfig/cron](https://github.com/robfig/cron) for advanced cron scheduling
- [ternarybob/arbor](https://github.com/ternarybob/arbor) for structured logging
- Sync from multiple internal repositories
- Centralized backup location
- Compliance and audit trails

**4. Open Source Distribution**
- Mirror projects across platforms
- Ensure availability and redundancy
- Automated release distribution

## Architecture

- **Configuration**: TOML-based configuration with environment variable support
- **Scheduler**: Uses robfig/cron for job scheduling with foreground execution
- **Storage**: BBolt embedded database for transaction history
- **Logging**: Structured logging with logrus
- **Git Operations**: Native Git commands via exec
- **Process Management**: Foreground application suitable for process managers
- **Authentication**: Support for tokens, SSH keys, and environment variables
- **Monitoring**: Built-in statistics and transaction tracking

## License

MIT License - see [LICENSE](LICENSE) file for details
