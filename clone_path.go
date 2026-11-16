package main

import (
	"net/url"
	"path"
	"strings"
)

func cloneFolderName(source string) string {
	source = strings.TrimSpace(source)
	if source == "" {
		return ""
	}
	if strings.Contains(source, "://") {
		u, err := url.Parse(source)
		if err != nil {
			return ""
		}
		source = u.Path
	} else if colon := strings.Index(source, ":"); colon >= 0 && !strings.ContainsAny(source[:colon], "/\\") {
		source = source[colon+1:]
	}
	source = strings.TrimRight(strings.ReplaceAll(source, "\\", "/"), "/")
	name := strings.TrimSuffix(path.Base(source), ".git")
	if name == "" || name == "." || name == ".." || name == "/" || strings.ContainsAny(name, "\x00\r\n") {
		return ""
	}
	return name
}
