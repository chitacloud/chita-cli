package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type authConfig struct {
	Token   string    `json:"token"`
	SavedAt time.Time `json:"savedAt"`
}

func SaveToken(token string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(authConfig{Token: token, SavedAt: time.Now()}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "auth.json"), data, 0600)
}

func LoadToken() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(dir, "auth.json"))
	if err != nil {
		return "", err
	}
	var cfg authConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return "", err
	}
	return cfg.Token, nil
}

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".chita"), nil
}
