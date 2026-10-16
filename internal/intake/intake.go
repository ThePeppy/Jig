// Package intake decides which paths may enter the library.
//
// It only checks that the path is a regular file. Whether IDA can load it
// is not known here: that answer comes from the worker after init_library,
// via build_loaders_list and open_database.
package intake

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	ErrMissing   = errors.New("intake: not a file")
	ErrDirectory = errors.New("intake: directory")
)

// Decision is the path the library stores. OpenPath is the file the user
// picked. idalib, not this package, chooses an existing database or creates one.
type Decision struct {
	Source   string
	OpenPath string
	Name     string
}

// Accept reports how path should be stored. An extension is neither an
// allowlist nor a denylist.
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
	return Decision{
		Source:   abs,
		OpenPath: abs,
		Name:     filepath.Base(abs),
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
	default:
		return "没能添加这个文件。"
	}
}
