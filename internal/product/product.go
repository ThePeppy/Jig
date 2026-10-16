// Package product is the stable identity of the native app.
// DrawMark is the Dock icon and the menu-bar template. It does not read a file.
package product

import (
	"os"
	"path/filepath"
	"strings"
)

const Name = "Jig"

// DataDir is where the library and settings persist.
func DataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".jig"), nil
}

// ShortPath renders a path under the home directory with a leading tilde.
func ShortPath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	home = filepath.Clean(home)
	path = filepath.Clean(path)
	if path == home {
		return "~"
	}
	rel, err := filepath.Rel(home, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return path
	}
	return filepath.Join("~", rel)
}
