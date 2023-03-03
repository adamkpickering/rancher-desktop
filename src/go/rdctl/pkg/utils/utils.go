package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var devMode *bool

// Get the steps-th parent directory of fullPath.
func getParentDir(fullPath string, steps int) string {
	fullPath = filepath.Clean(fullPath)
	for ; steps > 0; steps-- {
		fullPath = filepath.Dir(fullPath)
	}
	return fullPath
}

// Determine whether rdctl is running in development mode. "Development mode"
// means that Rancher Desktop has been run from `npm run dev`.
func DevMode() (bool, error) {
	if devMode != nil {
		return *devMode, nil
	}

	value, ok := os.LookupEnv("NODE_ENV")
	if ok && value == "development" {
		devMode = new(bool)
		*devMode = true
		return *devMode, nil
	}

	rdctlSymlinkPath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("failed to get path to rdctl symlink: %w", err)
	}
	rdctlPath, err := filepath.EvalSymlinks(rdctlSymlinkPath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve %q: %w", rdctlSymlinkPath, err)
	}

	parentPath := rdctlPath
	for {
		parentPath = filepath.Dir(parentPath)
		// Means the iteration has reached the root directory
		if strings.HasSuffix(parentPath, string(filepath.Separator)) {
			break
		}
		// If true, means that RD is running from a build produced by `npm run package`
		if filepath.Base(parentPath) == "dist" {
			devMode = new(bool)
			*devMode = false
			return *devMode, nil
		}
		// Check current directory for a directory named ".git"; if it exists,
		// and the above conditions are satisfied, rdctl has been put in place
		// by an instance of Rancher Desktop that has been run via `npm run dev`
		dirEntries, err := os.ReadDir(parentPath)
		if err != nil {
			return false, fmt.Errorf("failed to read directory %q: %w", parentPath, err)
		}
		for _, dirEntry := range dirEntries {
			if dirEntry.Name() == ".git" && dirEntry.Type().IsDir() {
				devMode = new(bool)
				*devMode = true
				return *devMode, nil
			}
		}
	}

	return false, nil
}
