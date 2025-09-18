# GitSync

A Git synchronization utility written in Go, ported from [git-mirror](https://github.com/RalfJung/git-mirror). GitSync automatically synchronizes Git repositories across different hosting platforms with advanced branch filtering, bidirectional sync capabilities, and commit author replacement.

## Features

- **Smart Branch Filtering**: Sync specific branches using wildcard patterns (`main`, `feature-*`, `*-sync`)
- **Bidirectional Sync**: Configure separate jobs for different sync directions
- **Author Replacement**: Rewrite commit authors during sync (e.g., private repo → corporate repo)
- **Cron Scheduling**: Schedule sync jobs using robfig/cron expressions (with seconds support)
- **Multiple Targets**: Push to multiple target repositories from a single source
- **Override Control**: Configure force push behavior per job for safe/unsafe branches
- **Professional Logging**: Structured logging with arbor logger, logs stored in executable directory
- **Foreground Application**: Run as a long-running foreground application with scheduled jobs
- **Git Validation**: Automatic git availability check at startup
- **Self-Contained**: Default config and logs in same directory as executable

## Installation

```bash
go install github.com/ternarybob/gitsync/cmd/gitsync@latest
```

Or build from source:

```bash
git clone https://github.com/ternarybob/gitsync.git
cd gitsync
./scripts/build.ps1   # Windows PowerShell
./scripts/build.sh    # Linux/macOS
```

## Quick Start

1. **Place the executable** in your desired directory
2. **Create `gitsync.toml`** in the same directory as the executable
3. **Configure your repositories** (see examples below)
4. **Run**: `./gitsync.exe` (Windows) or `./gitsync` (Linux/macOS)

## Configuration

GitSync uses TOML configuration files with a clean, hierarchical structure supporting job groups and per-job settings.

### Basic Configuration

```toml
# Service configuration
[service]
name = "gitsync"
environment = "development"  # development, staging, production

# Jobs configuration - shared settings for all jobs
[jobs]
names = ["main-sync", "feature-sync"]  # List of job names
schedule = "0 */5 * * * *"             # Every 5 minutes (SEC MIN HOUR DAY MONTH WEEKDAY)
timeout = "5m"                         # Shared timeout for all jobs

# Individual job: Sync main branch safely
["main-sync"]
description = "Sync main branch to backup"
enabled = true
source = "https://github.com/myorg/project.git"
targets = [
  "https://gitlab.com/myorg/project.git",
  "https://bitbucket.org/myorg/project.git"
]
branches = ["main"]                 # Only sync main branch
override = false                    # Safe push (no force) for main branch
git_username = "sync-bot"
git_token = "${GITHUB_TOKEN}"       # Environment variable

# Individual job: Sync feature branches with wildcards
["feature-sync"]
description = "Sync feature branches with wildcards"
enabled = true
source = "https://github.com/myorg/features.git"
targets = ["https://backup.myorg.com/features.git"]
branches = ["feature-*", "*-sync"]  # Wildcard patterns
override = true                     # Force push allowed for feature branches
git_username = "backup-user"
git_token = "${BACKUP_TOKEN}"

# Logging configuration
[logging]
level = "info"                      # debug, info, warn, error (default: debug)
format = "text"                     # text, json (default: text)
output = "both"                     # stdout, file, both (default: stdout)
max_file_size = 100                 # Log file max size in MB (default: 100)
max_backups = 3                     # Number of backup log files (default: 3)
max_age = 7                         # Days to retain log files (default: 3)
```

## Advanced Features

### Author Replacement (Private → Corporate Repos)

Rewrite commit authors during sync for corporate compliance:

```toml
["private-to-corporate"]
description = "Sync with author replacement"
enabled = true
source = "https://github.com/contractor/private-repo.git"
targets = ["https://github.com/company/corporate-repo.git"]
branches = ["*-sync"]               # Only sync tagged branches
override = true                     # Required for rewritten history
rewrite_history = true              # Enable commit author rewriting

# Author replacement rules
[[private-to-corporate.author_replace]]
from_email = "contractor@external.com"
from_name = "External Contractor"
to_email = "employee@company.com"
to_name = "Company Employee"

[[private-to-corporate.author_replace]]
from_email = "freelancer@gmail.com"
to_email = "employee@company.com"
to_name = "Company Employee"

git_username = "company-sync"
git_token = "${GITHUB_TOKEN}"
```

### Bidirectional Sync

Configure two separate jobs for bidirectional synchronization:

```toml
[jobs]
names = ["sync-down", "sync-up"]
schedule = "0 */10 * * * *"  # Every 10 minutes
timeout = "10m"

# Primary to backup
["sync-down"]
description = "Primary to backup sync"
enabled = true
source = "https://github.com/primary/repo.git"
targets = ["https://backup.example.com/repo.git"]
branches = ["main", "develop"]
override = false

# Backup to primary (with author replacement)
["sync-up"]
description = "Backup to primary with author replacement"
enabled = true
source = "https://backup.example.com/repo.git"
targets = ["https://github.com/primary/repo.git"]
branches = ["*-sync"]               # Only branches ending with -sync
override = true                     # Force push for rewritten history
rewrite_history = true

[[sync-up.author_replace]]
from_email = "backup@external.com"
to_email = "main@company.com"
to_name = "Main Developer"
```

### SSH Authentication

```toml
["ssh-sync"]
description = "SSH key authentication example"
enabled = true
source = "git@github.com:myorg/private-repo.git"
targets = ["git@gitlab.com:myorg/private-repo.git"]
branches = ["main", "develop"]
override = false
ssh_key_path = "/home/user/.ssh/id_rsa"
# OR use environment variable:
# ssh_key_env = "SSH_KEY_PATH"
```

## Key Configuration Options

### Branch Filtering
- `branches = ["main"]` - Sync only the main branch
- `branches = ["feature-*"]` - Sync all branches starting with "feature-"
- `branches = ["*-sync"]` - Sync all branches ending with "-sync"
- `branches = ["main", "develop", "feature-*"]` - Mix exact and wildcard patterns

### Override Behavior
- `override = false` - Safe push, will fail if there are conflicts (recommended for main branches)
- `override = true` - Force push, will overwrite target branch (required for rewritten history)

### Author Replacement
- `rewrite_history = true` - Enable commit history rewriting
- `author_replace` - Array of replacement rules matching by email or name
- **⚠️ Warning**: History rewriting changes commit hashes and requires `override = true`

### Environment Variables
Use `${VAR}` syntax in configuration files:

```toml
git_token = "${GITHUB_TOKEN}"
git_username = "${GIT_USER:-default-user}"  # With fallback value
```

### Logging Configuration Defaults

If no `[logging]` section is specified, GitSync uses these defaults:

```toml
[logging]
level = "debug"                     # Default: debug
format = "text"                     # Default: text
output = "stdout"                   # Default: stdout
max_file_size = 100                 # Default: 100 MB
max_backups = 3                     # Default: 3 files
max_age = 3                         # Default: 3 days
```

**Output Options:**
- `stdout` - Console only (default)
- `file` - File only (logs/{executable_directory}/logs/gitsync-YYYY-MM-DD.log)
- `both` - Console and file output

## Usage

### Command Line Options

```bash
# Show version information
./gitsync.exe -version

# Validate configuration file (uses default gitsync.toml in exe directory)
./gitsync.exe -validate

# Validate specific configuration file
./gitsync.exe -validate -config /path/to/config.toml

# Run a specific job immediately (for testing)
./gitsync.exe -run-job "main-sync"

# View sync statistics (from logs)
./gitsync.exe -stats
```

### Run as Foreground Application

```bash
# Start with default config (gitsync.toml in executable directory)
./gitsync.exe         # Windows
./gitsync            # Linux/macOS

# Use custom config file
./gitsync.exe -config /path/to/config.toml

# For background execution, use process managers:
# PM2: pm2 start ./gitsync --name gitsync -- -config gitsync.toml
# Screen: screen -S gitsync ./gitsync -config gitsync.toml
# Tmux: tmux new-session -d -s gitsync './gitsync -config gitsync.toml'
```

### File Structure

GitSync is self-contained in its directory:

```
project/
├── gitsync.exe         # Executable
├── gitsync.toml        # Default configuration (required)
└── logs/               # Log files (created automatically)
    ├── gitsync-2025-01-15.log
    └── gitsync-2025-01-16.log
```

### Startup Validation

GitSync automatically validates on startup:
- Checks if `git` command is available in PATH
- Verifies git can be executed
- Creates logs directory in executable directory
- Validates configuration file exists and is valid
- Fails fast with clear error messages

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

GitSync performs intelligent repository synchronization with branch filtering, author replacement, and safe/unsafe push modes.

### Sync Process

1. **Startup Validation**
   - Verify git command availability
   - Create logs directory in executable directory
   - Validate configuration file exists

2. **Branch Discovery & Filtering**
   - Fetch all remote branches from source repository
   - Apply wildcard pattern matching (`main`, `feature-*`, `*-sync`)
   - Only sync branches that match configured patterns

3. **Author Replacement (Optional)**
   - If `rewrite_history = true`, apply author replacement rules
   - Rewrite commit history using `git filter-branch`
   - Match authors by email (preferred) or name

4. **Authentication Setup**
   - Configure git credentials per job (token or SSH key)
   - Handle different auth for source vs target if needed

5. **Smart Synchronization**
   - Clone source repository to temporary directory (first run)
   - Fetch and reset to latest changes (subsequent runs)
   - Checkout each matching branch individually
   - Push to targets with override control:
     - `override = false`: Safe push, fails on conflicts
     - `override = true`: Force push, overwrites target

6. **Logging & Monitoring**
   - Structured logging with arbor logger
   - Dual output to console and timestamped log files in executable directory
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
                              │ 6. Apply author replacement (opt)   │
                              │ 7. For each matching branch:        │
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
Configuration file not found: /path/to/gitsync.toml
Create a gitsync.toml file in the same directory as the executable, or specify one with -config
```
- Create `gitsync.toml` in the same directory as the executable
- Or specify custom config with `-config path/to/config.toml`

**Git not available:**
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

**Author replacement not working:**
```
Git filter-branch failed
```
- Ensure `rewrite_history = true` is set
- Verify `override = true` (required for rewritten history)
- Check author replacement rules match actual commit authors
- Use exact email matching for best results

### Log Analysis

GitSync creates detailed logs in `{executable_directory}/logs/`:
```bash
# View latest log
ls -la logs/
tail -f logs/gitsync-$(date +%Y-%m-%d).log

# Search for errors
grep -i error logs/*.log

# Check specific job performance
grep "job=main-sync" logs/*.log

# Check author replacement
grep "rewriting.*authors" logs/*.log
```

## Use Cases

### 1. Multi-Platform Repository Mirroring
- **Primary Development**: GitHub
- **Automatic Mirroring**: GitLab, Bitbucket, private Git servers
- **Branch Strategy**: Sync main/develop safely, sync feature branches with override
- **Benefits**: Platform redundancy, compliance requirements, team access

### 2. Corporate Compliance Workflow
- **Private Development**: Contractors work in private repos with personal emails
- **Corporate Integration**: Author replacement transforms commits to corporate identity
- **Controlled Sync**: Use `*-sync` branches for reviewed code only
- **Audit Trail**: Complete logging of author transformations

### 3. Bidirectional Development Workflow
- **Main Flow**: Primary repo → backup repo (main branches)
- **Feature Flow**: Backup repo → primary repo (feature branches ending in `-sync`)
- **Use Case**: Contractors work in isolated repo, changes flow back through tagged branches
- **Safety**: Override disabled for main branches, enabled for sync branches

### 4. Enterprise Backup Strategy
- **Multi-Site Redundancy**: Sync to multiple geographic locations
- **Compliance**: Maintain copies for regulatory requirements
- **Disaster Recovery**: Automated failover to backup repositories
- **Scheduling**: Different frequencies for different repo priorities

### 5. Branch-Specific Synchronization
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
- **Self-Contained**: All data stored in executable directory

## Security Considerations

- **Token Scope**: Use minimal required permissions (repo read/write)
- **Environment Variables**: Store sensitive tokens in environment, not config files
- **SSH Keys**: Support for key-based authentication
- **Override Control**: Disable force push for protected branches
- **Author Replacement**: Only enabled when explicitly configured
- **Audit Trail**: Comprehensive logging of all sync operations including author changes

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