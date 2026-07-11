package config

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

//go:embed sample.yaml
var sampleConfig string

// Config holds the application configuration.
type Config struct {
	Source    SourceConfig    `mapstructure:"source"`
	Analyzer  AnalyzerConfig  `mapstructure:"analyzer"`
	Storage   StorageConfig   `mapstructure:"storage"`
	Workspace WorkspaceConfig `mapstructure:"workspace"`
	Queue     QueueConfig     `mapstructure:"queue"`
	Ranking   RankingConfig   `mapstructure:"ranking"`
}

// SourceConfig configures the repository provider.
type SourceConfig struct {
	Type  string `mapstructure:"type"`
	URL   string `mapstructure:"url"`
	Token string `mapstructure:"token"`
}

// AnalyzerConfig configures the language analyzer.
type AnalyzerConfig struct {
	Language string `mapstructure:"language"`
}

// StorageConfig configures the storage backend.
type StorageConfig struct {
	Driver string `mapstructure:"driver"`
	Path   string `mapstructure:"path"`
}

// WorkspaceConfig configures workspace directories.
type WorkspaceConfig struct {
	DataDir     string `mapstructure:"data_dir"`
	DatasetsDir string `mapstructure:"datasets_dir"`
	ReposDir    string `mapstructure:"repos_dir"`
}

// QueueConfig configures the job queue.
type QueueConfig struct {
	Driver string `mapstructure:"driver"`
}

// RankingConfig configures repository ranking weights.
type RankingConfig struct {
	Weights RankingWeights `mapstructure:"weights"`
}

// RankingWeights holds configurable score weights.
type RankingWeights struct {
	AgentsMD    int `mapstructure:"agents_md"`
	ClaudeMD    int `mapstructure:"claude_md"`
	CursorDir   int `mapstructure:"cursor_dir"`
	AICommits   int `mapstructure:"ai_commits"`
	Tests       int `mapstructure:"tests"`
	CI          int `mapstructure:"ci"`
	README      int `mapstructure:"readme"`
	Docs        int `mapstructure:"docs"`
	Activity    int `mapstructure:"activity"`
	Maintainers int `mapstructure:"maintainers"`
}

// DefaultRankingWeights returns sensible default weights.
func DefaultRankingWeights() RankingWeights {
	return RankingWeights{
		AgentsMD: 30, ClaudeMD: 25, CursorDir: 15, AICommits: 10,
		Tests: 20, CI: 15, README: 10, Docs: 10, Activity: 15, Maintainers: 10,
	}
}

// ProviderConfig returns provider settings as a map for the registry.
func (c *Config) ProviderConfig() map[string]string {
	return map[string]string{
		"url":   c.Source.URL,
		"token": c.Source.Token,
	}
}

// AnalyzerConfigMap returns analyzer settings as a map for the registry.
func (c *Config) AnalyzerConfigMap() map[string]string {
	return map[string]string{
		"language": c.Analyzer.Language,
	}
}

// Load reads configuration from the given file path.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

// WriteSample writes the embedded sample config to the given path if it does not exist.
func WriteSample(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
	}
	return os.WriteFile(path, []byte(sampleConfig), 0o644)
}

// EnsureDirs creates workspace directories defined in the config.
func EnsureDirs(cfg *Config) error {
	dirs := []string{
		cfg.Workspace.DataDir,
		cfg.Workspace.DatasetsDir,
		cfg.Workspace.ReposDir,
	}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create dir %s: %w", dir, err)
		}
	}
	// Ensure parent dir for DB file exists
	if cfg.Storage.Path != "" {
		dbDir := filepath.Dir(cfg.Storage.Path)
		if dbDir != "." && dbDir != "" {
			if err := os.MkdirAll(dbDir, 0o755); err != nil {
				return fmt.Errorf("create db dir %s: %w", dbDir, err)
			}
		}
	}
	return nil
}
