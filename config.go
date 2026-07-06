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

	fmt.Println()
	fmt.Printf("%s⚙️  OGC Configuration Initialized!%s\n", BoldGreen, Reset)
	fmt.Printf("A fresh configuration file has been created at:\n  %s%s%s\n\n", BoldCyan, configPath, Reset)
	fmt.Printf("%s👉 NEXT STEPS FOR FIRST-TIME SETUP:%s\n", BoldYellow, Reset)
	fmt.Printf("  1. Get a Gemini API Key from Google AI Studio:\n     %shttps://aistudio.google.com/app/apikey%s\n", BoldBlue, Reset)
	fmt.Printf("  2. Open the config file in your editor:\n     %snano %s%s\n", BoldCyan, configPath, Reset)
	fmt.Printf("  3. Insert your API key in the %sapi_key = \"...\"%s field.\n", BoldCyan, Reset)
	fmt.Printf("  4. (Optional) Choose a model, e.g., %smodel = \"gemini-2.5-pro\"%s (default: %sgemini-2.5-flash%s).\n\n", BoldCyan, Reset, BoldCyan, Reset)
	fmt.Printf("%sOnce configured, run OGC again to start generating clean commit messages!%s\n", BoldGreen, Reset)
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
		return nil, fmt.Errorf("api_key is empty or missing in %s.\n\nPlease open that file and paste your Gemini API Key. You can get one for free at:\n%shttps://aistudio.google.com/app/apikey%s", configPath, BoldBlue, Reset)
	}

	if cfg.Model == "" {
		return nil, fmt.Errorf("model is empty or missing in %s.\n\nPlease set a model like \"gemini-2.5-flash\" or \"gemini-2.5-pro\".", configPath)
	}

	if cfg.LineLength == 0 {
		cfg.LineLength = 80
	}

	return &cfg, nil
}
