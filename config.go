package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func readRecent(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var recent []string
	if json.Unmarshal(data, &recent) != nil {
		return nil
	}
	return recent
}

func writeRecent(path string, recent []string) error {
	return writeJSON(path, recent)
}

// settings are preferences kept apart from recent.json, whose format (a
// bare list of paths) stays as it is.
type settings struct {
	Theme string `json:"theme,omitempty"`
}

func settingsPath(recentPath string) string {
	if recentPath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(recentPath), "settings.json")
}

func readSettings(path string) settings {
	var s settings
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &s)
	}
	switch s.Theme {
	case themeLight, themeDark:
	default:
		s.Theme = themeSystem
	}
	return s
}

func writeSettings(path string, s settings) error { return writeJSON(path, s) }

// writeJSON replaces path atomically.
func writeJSON(path string, value any) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err = f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}

func remember(recent []string, path string) []string {
	result := []string{path}
	for _, p := range recent {
		if p != path && len(result) < 10 {
			result = append(result, p)
		}
	}
	return result
}
