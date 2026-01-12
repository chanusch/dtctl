package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dynatrace-oss/dtctl/pkg/config"
)

func TestConfigSetCmd(t *testing.T) {
	// Create a temporary directory for the config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config")

	// Set the config path
	os.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer os.Unsetenv("XDG_CONFIG_HOME")

	tests := []struct {
		name      string
		key       string
		value     string
		wantError bool
		validate  func(t *testing.T, cfg *config.Config)
	}{
		{
			name:      "set editor preference",
			key:       "preferences.editor",
			value:     "vim",
			wantError: false,
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Preferences.Editor != "vim" {
					t.Errorf("expected editor to be 'vim', got %q", cfg.Preferences.Editor)
				}
			},
		},
		{
			name:      "set editor to micro",
			key:       "preferences.editor",
			value:     "micro",
			wantError: false,
			validate: func(t *testing.T, cfg *config.Config) {
				if cfg.Preferences.Editor != "micro" {
					t.Errorf("expected editor to be 'micro', got %q", cfg.Preferences.Editor)
				}
			},
		},
		{
			name:      "unknown key",
			key:       "unknown.key",
			value:     "value",
			wantError: true,
			validate:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up for fresh test
			os.Remove(configPath)

			// Execute the RunE function directly with args
			err := configSetCmd.RunE(configSetCmd, []string{tt.key, tt.value})

			if tt.wantError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Validate the result
			if tt.validate != nil {
				cfg, err := config.Load()
				if err != nil {
					t.Fatalf("failed to load config: %v", err)
				}
				tt.validate(t, cfg)
			}
		})
	}
}
