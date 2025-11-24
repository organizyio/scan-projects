// Package scanner provides functionality to scan directories for projects
// and collect project metadata including Git information and language stacks.
package scanner

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"

	"scan-projects/internal/model"
)

// runGit executes a git command in the specified directory and returns
// the trimmed stdout output and any error that occurred.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %v: %w", args, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// parseRemoteURL tries to parse a remote URL into host, owner, repoSlug.
// Supports SSH (git@host:owner/repo.git), HTTPS (https://host/owner/repo.git),
// and handles ports, trailing slashes, and enterprise URLs.
func parseRemoteURL(url string) (host, owner, repoSlug *string) {
	if url == "" {
		return nil, nil, nil
	}

	// Remove trailing slashes
	url = strings.TrimSuffix(url, "/")

	// SSH: git@github.com:owner/repo.git or git@github.com:2222:owner/repo.git
	if strings.HasPrefix(url, "git@") {
		parts := strings.SplitN(url, "@", 2)
		if len(parts) != 2 {
			return nil, nil, nil
		}
		right := parts[1]

		// Handle port in SSH URL (git@host:port:path)
		// First try to find the last colon which separates host:port from path
		lastColon := strings.LastIndex(right, ":")
		if lastColon == -1 {
			return nil, nil, nil
		}

		hostPart := right[:lastColon]
		path := right[lastColon+1:]

		// Remove port if present (host:port -> host)
		hostParts := strings.SplitN(hostPart, ":", 2)
		h := hostParts[0]

		// Parse path (owner/repo.git)
		path = strings.TrimPrefix(path, "/")
		p := strings.SplitN(path, "/", 2)
		if len(p) != 2 {
			return &h, nil, nil
		}
		o := p[0]
		r := strings.TrimSuffix(p[1], ".git")
		return &h, &o, &r
	}

	// HTTPS: https://github.com/owner/repo.git or http://gitlab.example.com/owner/repo.git
	if strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://") {
		noProto := strings.SplitN(url, "://", 2)
		if len(noProto) != 2 {
			return nil, nil, nil
		}

		// Split host from path
		parts := strings.SplitN(noProto[1], "/", 2)
		if len(parts) < 1 {
			return nil, nil, nil
		}

		// Host may contain port (host:port)
		hostParts := strings.SplitN(parts[0], ":", 2)
		h := hostParts[0]

		if len(parts) < 2 {
			return &h, nil, nil
		}

		path := strings.TrimPrefix(parts[1], "/")
		segments := strings.Split(path, "/")
		if len(segments) < 2 {
			return &h, nil, nil
		}
		o := segments[0]
		r := strings.TrimSuffix(segments[1], ".git")
		return &h, &o, &r
	}

	return nil, nil, nil
}

// collectGitInfo enriches a ProjectInfo struct with Git metadata by
// querying the Git repository at the given path. It populates remotes,
// branch information, and working directory status.
// If logger is provided, debug information will be logged when verbose mode is enabled.
func collectGitInfo(path string, info *model.ProjectInfo, logger *Logger) {
	info.IsGit = true
	info.HasOrigin = false
	info.Remotes = []model.RemoteInfo{}
	info.Host = nil
	info.Owner = nil
	info.RepoSlug = nil
	info.CurrentBranch = ""
	info.LocalBranches = []string{}
	info.TrackingBranch = ""
	info.Dirty = nil

	// remotes
	remotesOut, err := runGit(path, "remote", "-v")
	if err != nil && logger != nil {
		logger.Debugf("Failed to get remotes for %s: %v", path, err)
	}
	if err == nil && remotesOut != "" {
		lines := strings.Split(remotesOut, "\n")
		seen := make(map[string]struct{})

		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			name := fields[0]
			url := fields[1]
			key := name + "|" + url
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			info.Remotes = append(info.Remotes, model.RemoteInfo{Name: name, URL: url})

			if name == "origin" {
				info.HasOrigin = true
				h, o, r := parseRemoteURL(url)
				if h != nil {
					info.Host = h
				}
				if o != nil {
					info.Owner = o
				}
				if r != nil {
					info.RepoSlug = r
				}
			}
		}
	}

	// current branch
	cur, err := runGit(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil && logger != nil {
		logger.Debugf("Failed to get current branch for %s: %v", path, err)
	}
	if err == nil && cur != "" {
		info.CurrentBranch = cur
	}

	// local branches
	branchesOut, err := runGit(path, "branch", "--format", "%(refname:short)")
	if err != nil && logger != nil {
		logger.Debugf("Failed to get branches for %s: %v", path, err)
	}
	if err == nil && branchesOut != "" {
		scanner := bufio.NewScanner(strings.NewReader(branchesOut))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				info.LocalBranches = append(info.LocalBranches, line)
			}
		}
	}

	// tracking branch
	if info.CurrentBranch != "" {
		tracking, err := runGit(path, "rev-parse", "--abbrev-ref", info.CurrentBranch+"@{upstream}")
		if err != nil && logger != nil {
			logger.Debugf("Failed to get tracking branch for %s: %v", path, err)
		}
		if err == nil && tracking != "" {
			info.TrackingBranch = tracking
		}
	}

	// dirty?
	statusOut, err := runGit(path, "status", "--porcelain")
	if err != nil && logger != nil {
		logger.Debugf("Failed to get status for %s: %v", path, err)
	}
	d := strings.TrimSpace(statusOut) != ""
	info.Dirty = &d
}
