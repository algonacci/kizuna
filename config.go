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
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(recent, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".recent-*")
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
