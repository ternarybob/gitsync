package common

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Service ServiceConfig `toml:"service"`
	Jobs    JobsConfig    `toml:"jobs"`
	JobDefs map[string]*JobConfig
	Logging LoggingConfig `toml:"logging"`
}

type ServiceConfig struct {
	Name        string `toml:"name"`
	Environment string `toml:"environment"`
}

type JobsConfig struct {
	Names    []string      `toml:"names"`
	Schedule string        `toml:"schedule"`
	Timeout  time.Duration `toml:"timeout"`
}

type AuthorReplacement struct {
	FromEmail string `toml:"from_email"`
	FromName  string `toml:"from_name"`
	ToEmail   string `toml:"to_email"`
	ToName    string `toml:"to_name"`
}

type JobConfig struct {
	Description    string              `toml:"description"`
	Enabled        bool                `toml:"enabled"`
	Source         string              `toml:"source"`
	Targets        []string            `toml:"targets"`
	Branches       []string            `toml:"branches"`
	Override       bool                `toml:"override"`
	GitUsername    string              `toml:"git_username"`
	GitToken       string              `toml:"git_token"`
	GitTokenEnv    string              `toml:"git_token_env"`
	SSHKeyPath     string              `toml:"ssh_key_path"`
	SSHKeyEnv      string              `toml:"ssh_key_env"`
	AuthorReplace  []AuthorReplacement `toml:"author_replace"`  // Replace existing commit authors
	RewriteHistory bool                `toml:"rewrite_history"` // Enable commit rewriting
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
		Jobs: JobsConfig{
			Names:    []string{},
			Schedule: "",
			Timeout:  5 * time.Minute,
		},
		JobDefs: make(map[string]*JobConfig),
		Logging: *DefaultLoggingConfig(),
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

			var rawConfig map[string]interface{}
			if err := toml.Unmarshal([]byte(content), &rawConfig); err != nil {
				return nil, fmt.Errorf("failed to parse config file %s: %w", filename, err)
			}

			if err := parseConfig(rawConfig, config); err != nil {
				return nil, fmt.Errorf("failed to process config: %w", err)
			}
		}
	}

	applyEnvOverrides(config)

	// Apply environment variables to credentials
	for _, jobConfig := range config.JobDefs {
		applyJobEnvOverrides(jobConfig)
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

func applyJobEnvOverrides(jobConfig *JobConfig) {
	if jobConfig.GitTokenEnv != "" {
		jobConfig.GitToken = os.Getenv(jobConfig.GitTokenEnv)
	}
	if jobConfig.SSHKeyEnv != "" {
		jobConfig.SSHKeyPath = os.Getenv(jobConfig.SSHKeyEnv)
	}
}

func parseConfig(rawConfig map[string]interface{}, config *Config) error {
	for key, value := range rawConfig {
		switch key {
		case "service":
			if serviceMap, ok := value.(map[string]interface{}); ok {
				config.Service.Name = getString(serviceMap, "name", "gitsync")
				config.Service.Environment = getString(serviceMap, "environment", "development")
			}
		case "jobs":
			if jobsMap, ok := value.(map[string]interface{}); ok {
				if namesArray, exists := jobsMap["names"].([]interface{}); exists {
					for _, name := range namesArray {
						if nameStr, ok := name.(string); ok {
							config.Jobs.Names = append(config.Jobs.Names, nameStr)
						}
					}
				}
				config.Jobs.Schedule = getString(jobsMap, "schedule", "")
				config.Jobs.Timeout = getDuration(jobsMap, "timeout", 5*time.Minute)
			}
		case "logging":
			if loggingMap, ok := value.(map[string]interface{}); ok {
				config.Logging.Level = getString(loggingMap, "level", "info")
				config.Logging.Format = getString(loggingMap, "format", "json")
				config.Logging.Output = getString(loggingMap, "output", "both")
				config.Logging.MaxSize = getInt(loggingMap, "max_size", 100)
				config.Logging.MaxBackups = getInt(loggingMap, "max_backups", 3)
				config.Logging.MaxAge = getInt(loggingMap, "max_age", 7)
			}
		default:
			// Job definition
			if jobMap, ok := value.(map[string]interface{}); ok {
				jobConfig := &JobConfig{
					Description:    getString(jobMap, "description", ""),
					Enabled:        getBool(jobMap, "enabled", true),
					Source:         getString(jobMap, "source", ""),
					Override:       getBool(jobMap, "override", false),
					GitUsername:    getString(jobMap, "git_username", ""),
					GitToken:       getString(jobMap, "git_token", ""),
					GitTokenEnv:    getString(jobMap, "git_token_env", ""),
					SSHKeyPath:     getString(jobMap, "ssh_key_path", ""),
					SSHKeyEnv:      getString(jobMap, "ssh_key_env", ""),
					RewriteHistory: getBool(jobMap, "rewrite_history", false),
				}

				// Parse author replacement rules
				if authorReplaceArray, exists := jobMap["author_replace"].([]interface{}); exists {
					for _, replacement := range authorReplaceArray {
						if replaceMap, ok := replacement.(map[string]interface{}); ok {
							authorReplace := AuthorReplacement{
								FromEmail: getString(replaceMap, "from_email", ""),
								FromName:  getString(replaceMap, "from_name", ""),
								ToEmail:   getString(replaceMap, "to_email", ""),
								ToName:    getString(replaceMap, "to_name", ""),
							}
							jobConfig.AuthorReplace = append(jobConfig.AuthorReplace, authorReplace)
						}
					}
				}

				// Parse targets array
				if targetsArray, exists := jobMap["targets"].([]interface{}); exists {
					for _, target := range targetsArray {
						if targetStr, ok := target.(string); ok {
							jobConfig.Targets = append(jobConfig.Targets, targetStr)
						}
					}
				}

				// Parse branches array
				if branchesArray, exists := jobMap["branches"].([]interface{}); exists {
					for _, branch := range branchesArray {
						if branchStr, ok := branch.(string); ok {
							jobConfig.Branches = append(jobConfig.Branches, branchStr)
						}
					}
				}

				config.JobDefs[key] = jobConfig
			}
		}
	}
	return nil
}

func getString(m map[string]interface{}, key, defaultValue string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return defaultValue
}

func getInt(m map[string]interface{}, key string, defaultValue int) int {
	if v, ok := m[key].(int64); ok {
		return int(v)
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	return defaultValue
}

func getBool(m map[string]interface{}, key string, defaultValue bool) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return defaultValue
}

func getDuration(m map[string]interface{}, key string, defaultValue time.Duration) time.Duration {
	if v, ok := m[key].(string); ok {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	if v, ok := m[key].(int64); ok {
		return time.Duration(v)
	}
	return defaultValue
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
}

func (c *Config) Validate() error {
	if c.Service.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if len(c.Jobs.Names) == 0 {
		return fmt.Errorf("at least one job must be configured")
	}

	if c.Jobs.Schedule == "" {
		return fmt.Errorf("jobs schedule cannot be empty")
	}

	for i, jobName := range c.Jobs.Names {
		jobConfig, exists := c.JobDefs[jobName]
		if !exists {
			return fmt.Errorf("job[%d]: job definition '%s' not found", i, jobName)
		}

		if jobConfig.Source == "" {
			return fmt.Errorf("job[%d]: source cannot be empty for job '%s'", i, jobName)
		}

		if len(jobConfig.Targets) == 0 {
			return fmt.Errorf("job[%d]: at least one target must be configured for job '%s'", i, jobName)
		}

		// Validate branch configuration - if no branches specified, default to main
		if len(jobConfig.Branches) == 0 {
			jobConfig.Branches = []string{"main"}
		}
	}

	return nil
}

func (c *Config) IsProduction() bool {
	return c.Service.Environment == "production"
}

func (c *Config) GetJobConfig(name string) (*JobConfig, bool) {
	jobConfig, exists := c.JobDefs[name]
	return jobConfig, exists
}

func (c *Config) GetEnabledJobs() []string {
	var enabled []string
	for _, jobName := range c.Jobs.Names {
		if jobConfig, exists := c.JobDefs[jobName]; exists && jobConfig.Enabled {
			enabled = append(enabled, jobName)
		}
	}
	return enabled
}

func (jc *JobConfig) ShouldSyncBranch(branchName string) bool {
	// Check against all patterns in branches list
	for _, pattern := range jc.Branches {
		if matchesBranchPattern(branchName, pattern) {
			return true
		}
	}
	return false
}

func (jc *JobConfig) GetSyncBranches() []string {
	return jc.Branches
}

func matchesBranchPattern(branchName, pattern string) bool {
	// Simple wildcard matching
	if pattern == "*" {
		return true
	}

	// Prefix matching with *
	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(branchName, suffix)
	}

	// Suffix matching with *
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(branchName, prefix)
	}

	// Contains matching with * in middle
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			return strings.HasPrefix(branchName, parts[0]) && strings.HasSuffix(branchName, parts[1])
		}
	}

	// Exact match
	return branchName == pattern
}
