// Package intake decides which paths may enter the library.
//
// An extension is never an allowlist. idalib opens whatever IDA's loader
// can read; the old desktop treated that as ".i64 or .exe".
package intake

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const (
	KindDatabase = "database"
	KindFile     = "file"
)

var (
	ErrMissing   = errors.New("intake: not a file")
	ErrDirectory = errors.New("intake: directory")
	ErrSidecar   = errors.New("intake: ida sidecar")
)

// Decision is what the library stores after a path is accepted.
type Decision struct {
	Source    string
	OpenPath  string
	Name      string
	Kind      string
	Companion bool
}

// Accept reports how path should be stored. OpenPath is a companion
// database when one already exists, so a later open does not ingest the
// binary again. Name stays the file the user picked.
func Accept(path string) (Decision, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return Decision{}, ErrMissing
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return Decision{}, err
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return Decision{}, ErrMissing
		}
		return Decision{}, err
	}
	if info.IsDir() {
		return Decision{}, ErrDirectory
	}
	if !info.Mode().IsRegular() {
		return Decision{}, ErrMissing
	}
	if isSidecar(abs) {
		return Decision{}, ErrSidecar
	}

	openPath := abs
	companion := false
	if !isDatabase(abs) {
		if found := findCompanion(abs); found != "" {
			openPath = found
			companion = true
		}
	}
	kind := KindFile
	if isDatabase(openPath) {
		kind = KindDatabase
	}
	return Decision{
		Source:    abs,
		OpenPath:  openPath,
		Name:      filepath.Base(abs),
		Kind:      kind,
		Companion: companion,
	}, nil
}

// Message is the Chinese sentence the window shows for err.
func Message(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, ErrMissing):
		return "找不到这个文件。"
	case errors.Is(err, ErrDirectory):
		return "这是目录，请添加文件。"
	case errors.Is(err, ErrSidecar):
		return "这是 IDA 的附属文件。请添加数据库或原始二进制。"
	default:
		return "没能添加这个文件。"
	}
}

func isDatabase(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".i64", ".idb":
		return true
	default:
		return false
	}
}

func isSidecar(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".id0", ".id1", ".id2", ".nam", ".til":
		return true
	default:
		return false
	}
}

func findCompanion(path string) string {
	dir := filepath.Dir(path)
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	var candidates []string
	if dir != "" && stem != "" {
		candidates = append(candidates,
			filepath.Join(dir, stem+".i64"),
			filepath.Join(dir, stem+".idb"),
		)
	}
	candidates = append(candidates, path+".i64", path+".idb")
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}
		return candidate
	}
	return ""
}
