package config

import (
	"encoding/json"
	"myxb/internal/auth"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the user configuration
type Config struct {
	Username        string `json:"username"`
	PasswordHash    string `json:"password_hash"` // MD5 hash of password
	ScheduleProfile string `json:"schedule_profile,omitempty"`
}

const (
	// ScheduleProfileStandard keeps the raw times returned by Xiaobao.
	ScheduleProfileStandard = "standard"
	// ScheduleProfileHighSchool adjusts periods 1-8 to the high-school bell schedule.
	ScheduleProfileHighSchool = "highschool"
)

// NormalizeScheduleProfile converts aliases into a saved schedule profile value.
func NormalizeScheduleProfile(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "standard", "default", "normal", "other", "others", "regular":
		return ScheduleProfileStandard
	case "highschool", "high-school", "hs", "senior", "gaozhong", "高中":
		return ScheduleProfileHighSchool
	default:
		return ""
	}
}

// HasCredentials reports whether the config contains a usable saved login.
func (c *Config) HasCredentials() bool {
	if c == nil {
		return false
	}

	return strings.TrimSpace(c.Username) != "" && strings.TrimSpace(c.PasswordHash) != ""
}

// ConfiguredScheduleProfile returns the explicitly saved schedule profile, if any.
func (c *Config) ConfiguredScheduleProfile() (string, bool) {
	if c == nil {
		return "", false
	}

	raw := strings.TrimSpace(c.ScheduleProfile)
	if raw == "" {
		return "", false
	}

	normalized := NormalizeScheduleProfile(raw)
	if normalized == "" {
		return "", false
	}

	return normalized, true
}

// EffectiveScheduleProfile returns the saved profile or the default fallback.
func (c *Config) EffectiveScheduleProfile() string {
	if c == nil {
		return ScheduleProfileStandard
	}

	if normalized, ok := c.ConfiguredScheduleProfile(); ok {
		return normalized
	}

	return ScheduleProfileStandard
}

// GetConfigDir returns the configuration directory path.
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	configDir := filepath.Join(homeDir, ".myxb")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", err
	}

	return configDir, nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.json"), nil
}

// GetScheduleCachePath returns the path to the schedule cache file.
func GetScheduleCachePath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(configDir, "schedule_cache.json"), nil
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No config file exists
		}
		return nil, err
	}

	var config Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Save saves the configuration to disk
func Save(config *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// HashPassword returns MD5 hash of the password (first hash only, for storage)
func HashPassword(password string) string {
	return auth.FirstHash(password)
}

// Delete removes the configuration file
func Delete() error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	cfg, err := Load()
	if err != nil {
		return err
	}

	if cfg != nil && strings.TrimSpace(cfg.ScheduleProfile) != "" {
		cfg.Username = ""
		cfg.PasswordHash = ""
		return Save(cfg)
	}

	if err := os.Remove(configPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return nil
}
