package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const maxWorkoutNotesBytes int64 = 256 * 1024

func readWorkoutNotesFile(input string) (string, string, error) {
	path := strings.TrimSpace(input)
	if len(path) >= 2 {
		if (path[0] == '\'' && path[len(path)-1] == '\'') || (path[0] == '"' && path[len(path)-1] == '"') {
			path = path[1 : len(path)-1]
		}
	}
	path = strings.ReplaceAll(path, `\ `, " ")
	if path == "" {
		return "", "", fmt.Errorf("choose a workout notes file first")
	}

	if path == "~" || strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", "", fmt.Errorf("could not resolve home directory: %w", err)
		}
		if path == "~" {
			path = home
		} else {
			path = filepath.Join(home, strings.TrimPrefix(path, "~/"))
		}
	}
	path = filepath.Clean(path)

	info, err := os.Stat(path)
	if err != nil {
		return "", "", fmt.Errorf("cannot read workout notes %q: %w", path, err)
	}
	if !info.Mode().IsRegular() {
		return "", "", fmt.Errorf("workout notes path %q is not a regular file", path)
	}
	if info.Size() > maxWorkoutNotesBytes {
		return "", "", fmt.Errorf("workout notes file is too large (maximum 256 KiB)")
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("cannot read workout notes %q: %w", path, err)
	}
	if int64(len(content)) > maxWorkoutNotesBytes {
		return "", "", fmt.Errorf("workout notes file is too large (maximum 256 KiB)")
	}
	if !utf8.Valid(content) {
		return "", "", fmt.Errorf("workout notes must be UTF-8 text")
	}
	notes := strings.TrimSpace(string(content))
	if notes == "" {
		return "", "", fmt.Errorf("workout notes file is empty")
	}
	return notes, path, nil
}
