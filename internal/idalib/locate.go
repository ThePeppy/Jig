// Package idalib finds a local IDA install and its libidalib.
// It does not load the library, start IDA, or launch a worker.
package idalib

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Status is a probe result.
type Status struct {
	Install  string
	Found    bool
	Missing  bool
	Library  string
	Searched string
}

// ApplicationsDir is the platform directory Probe searches when the user
// has not chosen an install.
func ApplicationsDir() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Applications"
	case "windows":
		return `C:\Program Files`
	default:
		return "/opt"
	}
}

// Probe prefers configured. Otherwise it looks in applications for a
// directory whose name contains "IDA". On macOS that directory must be an
// .app bundle. It does not walk the disk.
func Probe(configured, applications string) Status {
	st := Status{Searched: applications}
	if configured != "" {
		st.Install = configured
		info, err := os.Stat(configured)
		st.Found = err == nil && info.IsDir()
		st.Missing = !st.Found
	} else {
		st.Install = findInstall(applications)
		st.Found = st.Install != ""
	}
	if st.Found {
		st.Library = LibraryFile(st.Install)
	}
	return st
}

// ResolveInstallDir returns the directory containing libidalib.dylib / idalib.dll.
// On macOS, if dir is an .app bundle, it resolves to Contents/MacOS when present.
func ResolveInstallDir(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(dir), ".app") {
		macOSDir := filepath.Join(dir, "Contents", "MacOS")
		if info, err := os.Stat(macOSDir); err == nil && info.IsDir() {
			return macOSDir
		}
	}
	return dir
}

// LibraryFile returns libidalib inside installDir, or "" if it is not there.
// The file is only stated. Loading it starts the IDA kernel, so Probe does not.
func LibraryFile(installDir string) string {
	resolved := ResolveInstallDir(installDir)
	if resolved == "" {
		return ""
	}
	name := "libidalib.so"
	switch runtime.GOOS {
	case "windows":
		name = "idalib.dll"
	case "darwin":
		name = "libidalib.dylib"
	}
	candidate := filepath.Join(resolved, name)
	info, err := os.Stat(candidate)
	if err != nil || info.IsDir() {
		return ""
	}
	return candidate
}

func findInstall(root string) string {
	if root == "" {
		return ""
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return ""
	}
	var hits []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.Contains(strings.ToUpper(name), "IDA") {
			continue
		}
		if runtime.GOOS == "darwin" && !strings.HasSuffix(name, ".app") {
			continue
		}
		hits = append(hits, filepath.Join(root, name))
	}
	if len(hits) == 0 {
		return ""
	}
	sort.Strings(hits)
	chosen := ""
	for _, hit := range hits {
		if strings.Contains(strings.ToLower(filepath.Base(hit)), "professional") {
			chosen = hit
		}
	}
	if chosen != "" {
		return chosen
	}
	return hits[len(hits)-1]
}
