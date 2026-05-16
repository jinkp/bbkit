package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	appDirName         = "bbkit"
	configFileName     = "config.yaml"
	defaultOutputTable = "table"
	defaultOutputJSON  = "json"
	envWorkspace       = "BITBUCKET_WORKSPACE"
	envUsername        = "BITBUCKET_USERNAME"
)

type Config struct {
	Workspace     string `yaml:"workspace,omitempty"`
	Repo          string `yaml:"repo,omitempty"`
	Username      string `yaml:"username,omitempty"`
	DefaultOutput string `yaml:"defaultOutput,omitempty"`
}

func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if len(data) == 0 {
		return &Config{}, nil
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	if err := Validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	if err := Validate(cfg); err != nil {
		return err
	}

	path, err := configPath()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config yaml: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}

func Validate(cfg *Config) error {
	if cfg == nil {
		return errors.New("config is required")
	}

	if cfg.DefaultOutput == "" {
		return nil
	}

	if cfg.DefaultOutput != defaultOutputTable && cfg.DefaultOutput != defaultOutputJSON {
		return fmt.Errorf("invalid defaultOutput %q: must be %q or %q", cfg.DefaultOutput, defaultOutputTable, defaultOutputJSON)
	}

	return nil
}

func GetWorkspace() (string, error) {
	if workspace := os.Getenv(envWorkspace); workspace != "" {
		return workspace, nil
	}

	cfg, err := Load()
	if err != nil {
		return "", err
	}

	return cfg.Workspace, nil
}

func GetUsername() (string, error) {
	if username := os.Getenv(envUsername); username != "" {
		return username, nil
	}

	cfg, err := Load()
	if err != nil {
		return "", err
	}

	return cfg.Username, nil
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}

	return filepath.Join(dir, appDirName, configFileName), nil
}
