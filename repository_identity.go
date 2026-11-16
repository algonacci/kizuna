package main

import (
	"net/url"
	"path"
	"strings"
)

// Handles HTTPS, ssh:// and scp-style SSH, preserving organization/subgroup names.
func remoteIdentity(raw string) string {
	var repository string
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" || u.Scheme == "file" {
			return ""
		}
		repository = strings.Trim(u.Path, "/")
	} else if colon := strings.Index(raw, ":"); colon > 0 && !strings.Contains(raw[:colon], "/") {
		repository = strings.Trim(raw[colon+1:], "/")
	} else {
		return ""
	}
	repository = strings.TrimSuffix(repository, ".git")
	if !strings.Contains(repository, "/") {
		return ""
	}
	return path.Clean(repository)
}
func (r *repository) identity(remote string) string {
	if r == nil {
		return ""
	}
	if id := remoteIdentity(r.RemoteURLs[remote]); id != "" {
		return id
	}
	if id := remoteIdentity(r.RemoteURLs["origin"]); id != "" {
		return id
	}
	for _, name := range r.Remotes {
		if id := remoteIdentity(r.RemoteURLs[name]); id != "" {
			return id
		}
	}
	return repoName(r)
}
