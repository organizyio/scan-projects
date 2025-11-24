// Package scanner provides functionality to scan directories for projects
// and collect project metadata including Git information and language stacks.
package scanner

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"scan-projects/internal/model"
)

// FindProjectsWithProgress scans a root directory recursively to find all projects,
// displaying a progress bar during the scan. A project is identified by the presence
// of a .git directory or project marker files. Returns a slice of project paths.
// excludeMap contains directory names to skip during scanning.
// If logger is provided, debug information will be logged when verbose mode is enabled.
func FindProjectsWithProgress(root string, excludeMap map[string]struct{}, logger *Logger) []string {
	var projects []string
	var totalDirs int

	pb := NewProgressBar(0) // Will update total dynamically

	// Single pass: scan and track progress
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Log error but continue scanning
			fmt.Fprintf(os.Stderr, "[WARN] Error accessing %s: %v\n", path, err)
			if logger != nil {
				logger.Debugf("Skipping directory due to error: %s", path)
			}
			return nil
		}
		if !d.IsDir() {
			return nil
		}

		// Skip excluded directories
		// Check the immediate directory name first
		dirName := d.Name()
		if _, excluded := excludeMap[dirName]; excluded {
			if logger != nil {
				logger.Debugf("Skipping excluded directory: %s", path)
			}
			return filepath.SkipDir
		}

		// Also check if any parent directory in the path is excluded
		// This handles cases where we're deep inside an excluded directory
		// that wasn't caught at the top level (e.g., venv/Lib/site-packages)
		// Walk up the path and check each component name
		checkPath := path
		rootClean := filepath.Clean(root)
		for {
			// Get the base name of current path component
			component := filepath.Base(checkPath)
			checkPathClean := filepath.Clean(checkPath)

			// Stop if we've reached the root or hit an invalid component
			if component == "." || component == "" {
				break
			}
			if checkPathClean == rootClean {
				break
			}

			// Check if this component is excluded
			if _, excluded := excludeMap[component]; excluded {
				if logger != nil {
					logger.Debugf("Skipping directory inside excluded parent: %s (excluded: %s)", path, component)
				}
				return filepath.SkipDir
			}

			// Move up one level
			parent := filepath.Dir(checkPath)
			parentClean := filepath.Clean(parent)
			if parentClean == checkPathClean || parentClean == rootClean {
				break
			}
			checkPath = parent
		}

		totalDirs++
		pb.Total = totalDirs

		isProject := false
		gitDir := filepath.Join(path, ".git")
		hasGit := false
		if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
			projects = append(projects, path)
			isProject = true
			hasGit = true
			if logger != nil {
				logger.Debugf("Found Git project: %s", path)
			}
		} else if hasProjectMarker(path) {
			// Check if we're inside another project (unless we have our own .git)
			// Walk up the path to see if any parent is already a detected project
			isInsideProject := false
			checkPath := filepath.Dir(path)
			rootClean := filepath.Clean(root)
			for {
				checkPathClean := filepath.Clean(checkPath)
				if checkPathClean == rootClean {
					break
				}
				// Check if this parent directory is a project
				parentGitDir := filepath.Join(checkPath, ".git")
				if info, err := os.Stat(parentGitDir); err == nil && info.IsDir() {
					// Parent has .git, so this is inside a Git repo
					// Only treat as separate project if current dir has .git (submodule)
					isInsideProject = true
					break
				}
				if hasProjectMarker(checkPath) {
					// Parent has project markers, so this is inside a project
					isInsideProject = true
					break
				}
				// Move up one level
				parent := filepath.Dir(checkPath)
				parentClean := filepath.Clean(parent)
				if parentClean == checkPathClean || parentClean == rootClean {
					break
				}
				checkPath = parent
			}

			if !isInsideProject {
				projects = append(projects, path)
				isProject = true
				if logger != nil {
					logger.Debugf("Found project with markers: %s", path)
				}
			} else {
				// This directory is inside another project, skip it and all subdirectories
				if logger != nil {
					logger.Debugf("Skipping subdirectory inside project: %s", path)
				}
				return filepath.SkipDir
			}
		}

		pb.Increment(isProject)

		// If this is a project detected by markers (not .git), skip its subdirectories
		// to avoid detecting nested subdirectories as separate projects
		if isProject && !hasGit {
			return filepath.SkipDir
		}

		return nil
	})

	return projects
}

// CollectProjectInfo builds a ProjectInfo struct for a given directory path.
// It detects language stacks from files in the directory and, if a .git directory
// exists, enriches the information with Git metadata.
// If logger is provided, debug information will be logged when verbose mode is enabled.
func CollectProjectInfo(path string, logger *Logger) model.ProjectInfo {
	name := filepath.Base(path)
	stacks := detectStacks(path)
	var primary string
	if len(stacks) > 0 {
		primary = stacks[0]
	}

	p := model.ProjectInfo{
		Name:           name,
		Path:           path,
		Stacks:         stacks,
		PrimaryStack:   primary,
		IsGit:          false,
		HasOrigin:      false,
		Remotes:        []model.RemoteInfo{},
		Host:           nil,
		Owner:          nil,
		RepoSlug:       nil,
		CurrentBranch:  "",
		LocalBranches:  []string{},
		TrackingBranch: "",
		Dirty:          nil,
	}

	// If .git exists, enrich with git data
	gitDir := filepath.Join(path, ".git")
	if info, err := os.Stat(gitDir); err == nil && info.IsDir() {
		if logger != nil {
			logger.Debugf("Collecting Git info for: %s", path)
		}
		collectGitInfo(path, &p, logger)
	}

	return p
}
