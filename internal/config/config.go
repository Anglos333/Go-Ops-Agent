package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Provider ProviderConfig `yaml:"provider"`
}

type ProviderConfig struct {
	BaseURL string `yaml:"base_url"`
	APIKey  string `yaml:"api_key"`
	Model   string `yaml:"model"`
}

func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	configPath, err := resolveConfigPath(path)
	if err != nil {
		return nil, err
	}

	if configPath != "" {
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	overrideFromEnv(cfg)
	return cfg, nil
}

func defaultConfig() *Config {
	return &Config{
		Provider: ProviderConfig{
			BaseURL: "https://api.deepseek.com",
			Model:   "deepseek-chat",
		},
	}
}

func resolveConfigPath(path string) (string, error) {
	if path != "" {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("resolve home directory failed")
	}

	defaultPath := filepath.Join(homeDir, ".ops-agent.yaml")
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	return "", nil
}

func overrideFromEnv(cfg *Config) {
	if v := os.Getenv("OPS_AGENT_BASE_URL"); v != "" {
		cfg.Provider.BaseURL = v
	}
	if v := os.Getenv("OPS_AGENT_API_KEY"); v != "" {
		cfg.Provider.APIKey = v
	}
	if v := os.Getenv("OPS_AGENT_MODEL"); v != "" {
		cfg.Provider.Model = v
	}
}
