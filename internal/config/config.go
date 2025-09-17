package config

import (
	"fmt"
	"os"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Service ServiceConfig `toml:"service"`
	Jobs    []JobConfig   `toml:"jobs"`
	Store   StoreConfig   `toml:"store"`
	Logging LoggingConfig `toml:"logging"`
}

type ServiceConfig struct {
	Name        string `toml:"name"`
	Environment string `toml:"environment"`
}

type JobConfig struct {
	Name        string        `toml:"name"`
	Description string        `toml:"description"`
	Schedule    string        `toml:"schedule"`
	Enabled     bool          `toml:"enabled"`
	Timeout     time.Duration `toml:"timeout"`
	Repos       []RepoConfig  `toml:"repos"`
	Git         GitConfig     `toml:"git"`
}

type RepoConfig struct {
	Name    string   `toml:"name"`
	Source  string   `toml:"source"`
	Targets []string `toml:"targets"`
	Branch  string   `toml:"branch"`
}

type GitConfig struct {
	Username     string `toml:"username"`
	Token        string `toml:"token"`
	TokenEnvVar  string `toml:"token_env_var"`
	SSHKeyPath   string `toml:"ssh_key_path"`
	SSHKeyEnvVar string `toml:"ssh_key_env_var"`
	CommitAuthor string `toml:"commit_author"`
	CommitEmail  string `toml:"commit_email"`
}

type StoreConfig struct {
	Path            string `toml:"path"`
	BucketName      string `toml:"bucket_name"`
	MaxTransactions int    `toml:"max_transactions"`
	RetentionDays   int    `toml:"retention_days"`
}

type LoggingConfig struct {
	Level      string `toml:"level"`
	Format     string `toml:"format"`
	Output     string `toml:"output"`
	MaxSize    int    `toml:"max_size"`
	MaxBackups int    `toml:"max_backups"`
	MaxAge     int    `toml:"max_age"`
}

func DefaultConfig() *Config {
	return &Config{
		Service: ServiceConfig{
			Name:        "gitsync",
			Environment: "development",
		},
		Jobs: []JobConfig{},
		Store: StoreConfig{
			Path:            "./data/gitsync.db",
			BucketName:      "sync_transactions",
			MaxTransactions: 1000,
			RetentionDays:   30,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
		},
	}
}

func Load(filename string) (*Config, error) {
	config := DefaultConfig()

	if filename != "" {
		if _, err := os.Stat(filename); err == nil {
			data, err := os.ReadFile(filename)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file %s: %w", filename, err)
			}

			content := os.ExpandEnv(string(data))

			if err := toml.Unmarshal([]byte(content), config); err != nil {
				return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
			}
		}
	}

	applyEnvOverrides(config)

	for i := range config.Jobs {
		if config.Jobs[i].Git.TokenEnvVar != "" {
			config.Jobs[i].Git.Token = os.Getenv(config.Jobs[i].Git.TokenEnvVar)
		}
		if config.Jobs[i].Git.SSHKeyEnvVar != "" {
			config.Jobs[i].Git.SSHKeyPath = os.Getenv(config.Jobs[i].Git.SSHKeyEnvVar)
		}
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func applyEnvOverrides(config *Config) {
	if serviceName := os.Getenv("SERVICE_NAME"); serviceName != "" {
		config.Service.Name = serviceName
	}
	if environment := os.Getenv("ENVIRONMENT"); environment != "" {
		config.Service.Environment = environment
	}
	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		config.Logging.Level = logLevel
	}
	if logFormat := os.Getenv("LOG_FORMAT"); logFormat != "" {
		config.Logging.Format = logFormat
	}
	if storePath := os.Getenv("STORE_PATH"); storePath != "" {
		config.Store.Path = storePath
	}
}

func (c *Config) Validate() error {
	if c.Service.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if len(c.Jobs) == 0 {
		return fmt.Errorf("at least one job must be configured")
	}

	for i, job := range c.Jobs {
		if job.Name == "" {
			return fmt.Errorf("job[%d]: name cannot be empty", i)
		}
		if job.Schedule == "" && job.Enabled {
			return fmt.Errorf("job[%d]: schedule cannot be empty for enabled job", i)
		}
		if len(job.Repos) == 0 {
			return fmt.Errorf("job[%d]: at least one repository must be configured", i)
		}
		for j, repo := range job.Repos {
			if repo.Source == "" {
				return fmt.Errorf("job[%d].repos[%d]: source cannot be empty", i, j)
			}
			if len(repo.Targets) == 0 {
				return fmt.Errorf("job[%d].repos[%d]: at least one target must be configured", i, j)
			}
		}
	}

	if c.Store.Path == "" {
		return fmt.Errorf("store path cannot be empty")
	}
	if c.Store.BucketName == "" {
		return fmt.Errorf("store bucket name cannot be empty")
	}

	return nil
}

func (c *Config) IsProduction() bool {
	return c.Service.Environment == "production"
}

func GenerateExampleConfig() string {
	config := &Config{
		Service: ServiceConfig{
			Name:        "gitsync",
			Environment: "development",
		},
		Jobs: []JobConfig{
			{
				Name:        "main-sync",
				Description: "Synchronize main repositories",
				Schedule:    "*/5 * * * *",
				Enabled:     true,
				Timeout:     5 * time.Minute,
				Repos: []RepoConfig{
					{
						Name:   "my-project",
						Source: "https://github.com/myorg/my-project.git",
						Targets: []string{
							"https://gitlab.com/myorg/my-project.git",
							"https://bitbucket.org/myorg/my-project.git",
						},
						Branch: "main",
					},
				},
				Git: GitConfig{
					Username:     "git-sync-bot",
					TokenEnvVar:  "GITHUB_TOKEN",
					CommitAuthor: "Git Sync Bot",
					CommitEmail:  "gitsync@example.com",
				},
			},
		},
		Store: StoreConfig{
			Path:            "./data/gitsync.db",
			BucketName:      "sync_transactions",
			MaxTransactions: 1000,
			RetentionDays:   30,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
		},
	}

	data, _ := toml.Marshal(config)

	header := `# GitSync Configuration
#
# This file configures the GitSync service for synchronizing git repositories.
#
# Environment variables can be used with ${VAR} syntax.
# Cron expressions: https://crontab.guru/
#
# Schedule format: "MIN HOUR DAY MONTH WEEKDAY"
# Examples:
#   "*/5 * * * *"    - Every 5 minutes
#   "0 */2 * * *"    - Every 2 hours
#   "0 0 * * *"      - Daily at midnight
#   "@hourly"        - Every hour
#   "@daily"         - Every day at midnight
#

`

	return header + string(data)
}
