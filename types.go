package main


type commitConfig struct {
	useClipboard bool
	useEditor    bool
	moduleName   string
	path         string
	targetTag    string
	taskID       string
}

type Config struct {
	APIKey     string `toml:"api_key"`
	Model      string `toml:"model"`
	LineLength int    `toml:"line_length"`
}
