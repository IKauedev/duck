package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Settings map[string]string

func LoadSettings() (Settings, error) {
	path, err := SettingsPath()
	if err != nil {
		return nil, err
	}

	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Settings{}, nil
	}
	if err != nil {
		return nil, err
	}

	var settings Settings
	if err := json.Unmarshal(content, &settings); err != nil {
		return nil, err
	}
	if settings == nil {
		settings = Settings{}
	}
	return settings, nil
}

func SaveSettings(settings Settings) error {
	path, err := SettingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	content, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(content, '\n'), 0644)
}

func SetSetting(key, value string) error {
	settings, err := LoadSettings()
	if err != nil {
		return err
	}
	settings[key] = value
	return SaveSettings(settings)
}

func SettingsPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err == nil && configDir != "" {
		return filepath.Join(configDir, "duck", "config.json"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".duck_config.json"), nil
}

func PrintSettings(settings Settings) {
	if len(settings) == 0 {
		fmt.Println("Nenhuma configuracao salva.")
		return
	}

	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("%s=%s\n", key, settings[key])
	}
}
