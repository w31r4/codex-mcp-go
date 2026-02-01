package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Load loads configuration with the following precedence:
// defaults < config file (optional) < environment variables.
func Load(path string) (*Config, error) {
	cfg := Default()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read config file: %w", err)
		}
		if err := toml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config file: %w", err)
		}
	}

	cfg.LoadFromEnv()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}
