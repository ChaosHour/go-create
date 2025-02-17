package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config represents the application configuration
type Config struct {
	MySQL MySQLConfig `json:"mysql"`
}

// MySQLConfig holds MySQL-specific configuration
type MySQLConfig struct {
	Host     string `json:"host"`
	Port     string `json:"port"`
	User     string `json:"user"`     // User for connecting
	Password string `json:"password"` // Password for connecting
}

// LoadConfig reads configuration from a JSON file
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		// Default to config.json in the user's home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(home, ".go-create.json")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &Config{
				MySQL: MySQLConfig{
					Port: "3306",
				},
			}, nil
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig writes the configuration to a file
func SaveConfig(config *Config, path string) error {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path = filepath.Join(home, ".go-create.json")
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0600)
}
