// Package idalib knows how this app finds and configures a local IDA runtime.
// It does not start IDA GUI and does not install a GUI plugin.
package idalib

import (
	"encoding/json"
	"os"
	"os/exec"
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
	Python   string
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
	if p, err := exec.LookPath("python3"); err == nil {
		st.Python = p
	}
	if configured != "" {
		st.Install = configured
		info, err := os.Stat(configured)
		st.Found = err == nil && info.IsDir()
		st.Missing = !st.Found
		return st
	}
	st.Install = findInstall(applications)
	st.Found = st.Install != ""
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

// IdalibPythonDir returns the path to idalib/python (containing idapro) inside installDir.
func IdalibPythonDir(installDir string) string {
	resolved := ResolveInstallDir(installDir)
	if resolved == "" {
		return ""
	}
	candidate := filepath.Join(resolved, "idalib", "python")
	if info, err := os.Stat(filepath.Join(candidate, "idapro")); err == nil && info.IsDir() {
		return candidate
	}
	return ""
}

// EnsureUserConfig writes ~/.idapro/ida-config.json so `import idapro` locates libidalib.
func EnsureUserConfig(installDir string) error {
	resolved := ResolveInstallDir(installDir)
	if resolved == "" {
		return nil
	}
	configDir := os.Getenv("IDAUSR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		if runtime.GOOS == "windows" {
			if appData := os.Getenv("APPDATA"); appData != "" {
				configDir = filepath.Join(appData, "Hex-Rays", "IDA Pro")
			} else {
				configDir = filepath.Join(home, "AppData", "Roaming", "Hex-Rays", "IDA Pro")
			}
		} else {
			configDir = filepath.Join(home, ".idapro")
		}
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return err
	}
	payload := map[string]any{
		"Paths": map[string]string{
			"ida-install-dir": resolved,
		},
	}
	b, err := json.MarshalIndent(payload, "", "    ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(configDir, "ida-config.json"), b, 0o644)
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
