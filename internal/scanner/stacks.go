// Package scanner provides functionality to scan directories for projects
// and collect project metadata including Git information and language stacks.
package scanner

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// StackMarker holds detection markers for a language or technology stack.
// Filenames contains specific filenames that indicate the stack (e.g., "package.json" for Node.js).
// Extensions contains file extensions that indicate the stack (e.g., ".py" for Python).
type StackMarker struct {
	Filenames  map[string]struct{} // Filenames that indicate this stack
	Extensions map[string]struct{} // File extensions that indicate this stack
}

// Base project markers (to decide "this is a project").
var projectMarkers = map[string]struct{}{
	// Python
	"pyproject.toml":   {},
	"requirements.txt": {},
	"setup.cfg":        {},
	"setup.py":         {},
	// Node / JS / TS
	"package.json": {},
	// Docker / generic
	"Dockerfile":         {},
	"docker-compose.yml": {},
	"Makefile":           {},
	".env":               {},
	// Gradle (Kotlin/Java)
	"build.gradle":        {},
	"build.gradle.kts":    {},
	"settings.gradle":     {},
	"settings.gradle.kts": {},
	// Swift Package Manager
	"Package.swift": {},
}

// Stack-specific markers for classification.
var stackMarkers = map[string]StackMarker{
	"python": {
		Filenames: map[string]struct{}{
			"pyproject.toml":   {},
			"requirements.txt": {},
			"setup.cfg":        {},
			"setup.py":         {},
		},
		Extensions: map[string]struct{}{
			".py": {},
		},
	},
	"node": {
		Filenames: map[string]struct{}{
			"package.json": {},
		},
		Extensions: map[string]struct{}{},
	},
	"dotnet": {
		Filenames: map[string]struct{}{},
		Extensions: map[string]struct{}{
			".sln":    {},
			".csproj": {},
		},
	},
	"kotlin": {
		Filenames: map[string]struct{}{
			"build.gradle":        {},
			"build.gradle.kts":    {},
			"settings.gradle":     {},
			"settings.gradle.kts": {},
		},
		Extensions: map[string]struct{}{
			".kt": {},
		},
	},
	"swift": {
		Filenames: map[string]struct{}{
			"Package.swift": {},
		},
		Extensions: map[string]struct{}{
			".swift":       {},
			".xcodeproj":   {},
			".xcworkspace": {},
		},
	},
}

// hasProjectMarker returns true if directory contains any "project-ish" marker.
// It checks both projectMarkers (generic markers) and stackMarkers (language-specific).
func hasProjectMarker(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}

	filenames := make(map[string]struct{})
	exts := make(map[string]struct{})

	for _, entry := range entries {
		name := entry.Name()
		filenames[name] = struct{}{}

		// Check generic project markers
		if _, ok := projectMarkers[name]; ok {
			return true
		}

		// Collect extensions for stack marker checking
		if entry.Type().IsRegular() || entry.Type().Type() == fs.ModeDir {
			ext := filepath.Ext(name)
			if ext != "" {
				exts[ext] = struct{}{}
			}
		}
	}

	// Check stack markers (reuse same logic as detectStacks)
	for _, markers := range stackMarkers {
		// Check filenames
		for fn := range markers.Filenames {
			if _, ok := filenames[fn]; ok {
				return true
			}
		}
		// Check extensions
		for ext := range markers.Extensions {
			if _, ok := exts[ext]; ok {
				return true
			}
		}
	}

	return false
}

// detectStacks examines files in the project root directory and returns
// a sorted list of detected language/technology stacks based on marker files
// and extensions found.
func detectStacks(path string) []string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	filenames := make(map[string]struct{})
	exts := make(map[string]struct{})

	for _, entry := range entries {
		name := entry.Name()
		filenames[name] = struct{}{}

		// regular files and dirs (.xcodeproj is a dir) still have suffix
		if entry.Type().IsRegular() || entry.Type().Type() == fs.ModeDir {
			ext := filepath.Ext(name)
			if ext != "" {
				exts[ext] = struct{}{}
			}
		}
	}

	var stacks []string
	for stack, markers := range stackMarkers {
		found := false

		if len(markers.Filenames) > 0 {
			for fn := range markers.Filenames {
				if _, ok := filenames[fn]; ok {
					found = true
					break
				}
			}
		}

		if !found && len(markers.Extensions) > 0 {
			for ext := range markers.Extensions {
				if _, ok := exts[ext]; ok {
					found = true
					break
				}
			}
		}

		if found {
			stacks = append(stacks, stack)
		}
	}

	sort.Strings(stacks)
	return stacks
}
