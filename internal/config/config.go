package config

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	DirName    = ".fdup"
	ConfigFile = "config.yaml"
	DBFile     = "fdup.db"
)

// Config represents the fdup configuration.
type Config struct {
	Patterns []Pattern   `yaml:"patterns"`
	Ignore   []string    `yaml:"ignore"`
	Test     []TestCase  `yaml:"test,omitempty"`
}

// Pattern represents a regex pattern for code extraction.
type Pattern struct {
	Name  string `yaml:"name"`
	Regex string `yaml:"regex"`
}

// TestCase represents a test case for pattern validation.
type TestCase struct {
	Input    string  `yaml:"input"`
	Expected *string `yaml:"expected"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Patterns: []Pattern{
			{Name: "standard", Regex: `([A-Z]{2,5}-\d{3,5})`},
			{Name: "no_hyphen", Regex: `([A-Z]{2,5})(\d{3,5})`},
		},
		Ignore: []string{
			"node_modules/",
			".git/",
			"*.tmp",
			"*.log",
			".DS_Store",
			".fdup/",
		},
		Test: []TestCase{
			{Input: "PRJ-001_final.zip", Expected: strPtr("PRJ001")},
			{Input: "doc123.pdf", Expected: strPtr("DOC123")},
			{Input: "random_file.txt", Expected: nil},
		},
	}
}

func strPtr(s string) *string {
	return &s
}

// FindConfigDir finds the nearest .fdup directory by walking up from cwd.
func FindConfigDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		configPath := filepath.Join(dir, DirName)
		if info, err := os.Stat(configPath); err == nil && info.IsDir() {
			return configPath, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("not initialized. Run 'fdup init' first")
		}
		dir = parent
	}
}

// Load loads the configuration from the given directory.
func Load(configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, ConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Save saves the configuration to the given directory.
func Save(configDir string, cfg *Config) error {
	configPath := filepath.Join(configDir, ConfigFile)
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, data, 0644)
}

// GetPatternRegexes returns the regex strings from the patterns.
func (c *Config) GetPatternRegexes() []string {
	regexes := make([]string, len(c.Patterns))
	for i, p := range c.Patterns {
		regexes[i] = p.Regex
	}
	return regexes
}
