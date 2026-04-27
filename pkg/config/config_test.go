package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name       string
		setupFile  func() (string, error)
		wantErr    bool
		wantHost   string
		wantPort   string
		wantUser   string
		wantPasswd string
	}{
		{
			name: "valid config file",
			setupFile: func() (string, error) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "test-config.json")
				content := `{
					"mysql": {
						"host": "testhost.example.com",
						"port": "3307",
						"user": "testuser",
						"password": "testpass"
					}
				}`
				err := os.WriteFile(configFile, []byte(content), 0600)
				return configFile, err
			},
			wantErr:    false,
			wantHost:   "testhost.example.com",
			wantPort:   "3307",
			wantUser:   "testuser",
			wantPasswd: "testpass",
		},
		{
			name: "config with only host",
			setupFile: func() (string, error) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "test-config.json")
				content := `{
					"mysql": {
						"host": "localhost"
					}
				}`
				err := os.WriteFile(configFile, []byte(content), 0600)
				return configFile, err
			},
			wantErr:  false,
			wantHost: "localhost",
			wantPort: "",
			wantUser: "",
		},
		{
			name: "empty config file",
			setupFile: func() (string, error) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "test-config.json")
				content := `{}`
				err := os.WriteFile(configFile, []byte(content), 0600)
				return configFile, err
			},
			wantErr:  false,
			wantHost: "",
			wantPort: "",
			wantUser: "",
		},
		{
			name: "malformed JSON",
			setupFile: func() (string, error) {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "test-config.json")
				content := `{"mysql": {invalid json}`
				err := os.WriteFile(configFile, []byte(content), 0600)
				return configFile, err
			},
			wantErr: true,
		},
		{
			name: "non-existent file with empty path",
			setupFile: func() (string, error) {
				return "", nil
			},
			wantErr:  false,  // Should return default config
			wantPort: "3306", // Default port
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, err := tt.setupFile()
			if err != nil {
				t.Fatalf("Failed to setup test file: %v", err)
			}

			got, err := LoadConfig(configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got == nil {
					t.Fatal("LoadConfig() returned nil config")
				}
				if got.MySQL.Host != tt.wantHost {
					t.Errorf("LoadConfig().MySQL.Host = %v, want %v", got.MySQL.Host, tt.wantHost)
				}
				if tt.wantPort != "" && got.MySQL.Port != tt.wantPort {
					t.Errorf("LoadConfig().MySQL.Port = %v, want %v", got.MySQL.Port, tt.wantPort)
				}
				if tt.wantUser != "" && got.MySQL.User != tt.wantUser {
					t.Errorf("LoadConfig().MySQL.User = %v, want %v", got.MySQL.User, tt.wantUser)
				}
				if tt.wantPasswd != "" && got.MySQL.Password != tt.wantPasswd {
					t.Errorf("LoadConfig().MySQL.Password = %v, want %v", got.MySQL.Password, tt.wantPasswd)
				}
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		path    string
		wantErr bool
	}{
		{
			name: "save valid config",
			config: &Config{
				MySQL: MySQLConfig{
					Host:     "localhost",
					Port:     "3306",
					User:     "root",
					Password: "secret",
				},
			},
			path:    "", // Will use temp dir
			wantErr: false,
		},
		{
			name: "save config with custom port",
			config: &Config{
				MySQL: MySQLConfig{
					Host:     "db.example.com",
					Port:     "3307",
					User:     "admin",
					Password: "pass123",
				},
			},
			path:    "", // Will use temp dir
			wantErr: false,
		},
		{
			name: "save empty config",
			config: &Config{
				MySQL: MySQLConfig{},
			},
			path:    "", // Will use temp dir
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configPath := filepath.Join(tmpDir, "saved-config.json")

			err := SaveConfig(tt.config, configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("SaveConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the file was created
				if _, err := os.Stat(configPath); os.IsNotExist(err) {
					t.Errorf("SaveConfig() did not create file at %v", configPath)
				}

				// Verify we can load it back
				loaded, err := LoadConfig(configPath)
				if err != nil {
					t.Errorf("Failed to load saved config: %v", err)
					return
				}

				if loaded.MySQL.Host != tt.config.MySQL.Host {
					t.Errorf("Saved config Host = %v, want %v", loaded.MySQL.Host, tt.config.MySQL.Host)
				}
				if loaded.MySQL.Port != tt.config.MySQL.Port {
					t.Errorf("Saved config Port = %v, want %v", loaded.MySQL.Port, tt.config.MySQL.Port)
				}
				if loaded.MySQL.User != tt.config.MySQL.User {
					t.Errorf("Saved config User = %v, want %v", loaded.MySQL.User, tt.config.MySQL.User)
				}
				if loaded.MySQL.Password != tt.config.MySQL.Password {
					t.Errorf("Saved config Password = %v, want %v", loaded.MySQL.Password, tt.config.MySQL.Password)
				}
			}
		})
	}
}

func TestConfigRoundTrip(t *testing.T) {
	// Test that we can save and load a config without data loss
	original := &Config{
		MySQL: MySQLConfig{
			Host:     "production.db.com",
			Port:     "3308",
			User:     "produser",
			Password: "complexP@ssw0rd!123",
		},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "roundtrip-config.json")

	// Save
	if err := SaveConfig(original, configPath); err != nil {
		t.Fatalf("SaveConfig() failed: %v", err)
	}

	// Load
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() failed: %v", err)
	}

	// Compare
	if loaded.MySQL.Host != original.MySQL.Host {
		t.Errorf("Round trip Host = %v, want %v", loaded.MySQL.Host, original.MySQL.Host)
	}
	if loaded.MySQL.Port != original.MySQL.Port {
		t.Errorf("Round trip Port = %v, want %v", loaded.MySQL.Port, original.MySQL.Port)
	}
	if loaded.MySQL.User != original.MySQL.User {
		t.Errorf("Round trip User = %v, want %v", loaded.MySQL.User, original.MySQL.User)
	}
	if loaded.MySQL.Password != original.MySQL.Password {
		t.Errorf("Round trip Password = %v, want %v", loaded.MySQL.Password, original.MySQL.Password)
	}
}

func TestConfigFilePermissions(t *testing.T) {
	// Test that saved config files have secure permissions
	config := &Config{
		MySQL: MySQLConfig{
			Host:     "localhost",
			Port:     "3306",
			User:     "user",
			Password: "secret",
		},
	}

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "secure-config.json")

	if err := SaveConfig(config, configPath); err != nil {
		t.Fatalf("SaveConfig() failed: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatalf("Failed to stat config file: %v", err)
	}

	mode := info.Mode()
	expectedMode := os.FileMode(0600)

	if mode != expectedMode {
		t.Errorf("Config file permissions = %v, want %v", mode, expectedMode)
	}
}
