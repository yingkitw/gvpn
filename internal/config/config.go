package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Config struct {
	ListenAddr string `json:"listen_addr"`
	VPNNetwork string `json:"vpn_network"`
	MTU        int    `json:"mtu"`
	Debug      bool   `json:"debug"`
}

func Load() (*Config, error) {
	cfg := &Config{
		ListenAddr: ":1194",
		VPNNetwork: "10.8.0.0/24",
		MTU:        1500,
		Debug:      false,
	}

	configPath := getConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := Save(cfg); err != nil {
			return nil, fmt.Errorf("failed to create default config: %v", err)
		}
		return cfg, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %v", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	return cfg, nil
}

func Save(cfg *Config) error {
	configPath := getConfigPath()
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %v", err)
	}

	return nil
}

func getConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".gvpn", "config.json")
}