package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func EnsureConfigFile() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".config", "ogc")
	configPath := filepath.Join(configDir, "config.toml")

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	defaultConfig := `# OGC Configuration File
api_key = ""
model = "gemini-2.5-flash"

# Character limit per line for word wrapping
line_length = 80
`

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}

	fmt.Printf("Config file created: %s\n", configPath)
	fmt.Println("Please add your API key and run the command again.")
	os.Exit(0)

	return nil
}

func LoadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".config", "ogc", "config.toml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config TOML: %w", err)
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("api_key is missing in %s", configPath)
	}

	if cfg.Model == "" {
		return nil, fmt.Errorf("model is missing in %s", configPath)
	}

	if cfg.LineLength == 0 {
		cfg.LineLength = 80
	}

	return &cfg, nil
}
