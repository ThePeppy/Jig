// Package desk is the state the window shows. File dialogs stay in main.
package desk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"app/internal/catalog"
	"app/internal/idalib"
	"app/internal/intake"
	"app/internal/mcp"
	"app/internal/session"
	"app/internal/trace"
)

// Tab is the window's current page.
type Tab int

const (
	TabLibrary Tab = iota
	TabSessions
	TabTrace
	TabSettings
)

// SettingsSection is the group the settings page shows.
type SettingsSection int

const (
	SettingsIDA SettingsSection = iota
	SettingsPorts
	SettingsAppearance
	SettingsAgents
)

// Desk is main-thread state. Dialog results come back through Take.
type Desk struct {
	Tab             Tab
	Section         SettingsSection
	SelectedSession string
	Listening       bool
	Notice          string
	IdaDir          string
	Catalog         *catalog.Catalog
	Sessions        *session.Manager
	Traces          *trace.Store
	Now             func() time.Time
	RequestAdd      func()
	RequestIdaDir   func()
	NotifyUI        func()

	dir     string
	runtime idalib.Status
	probed  bool
	mcpSrv  *mcp.Server
}

type stored struct {
	Entries []catalog.Entry `json:"entries"`
	IdaDir  string          `json:"idaDir"`
}

// Open loads dir/state.json, creating dir when needed.
func Open(dir string) (*Desk, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	d := &Desk{
		Catalog:  catalog.New(),
		Sessions: session.NewManager(),
		Traces:   trace.NewStore(200),
		Now:      time.Now,
		dir:      dir,
	}
	b, err := os.ReadFile(d.file())
	if err != nil {
		if os.IsNotExist(err) {
			return d, nil
		}
		return nil, err
	}
	var raw stored
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	d.IdaDir = raw.IdaDir
	d.Catalog.Restore(raw.Entries)

	// Persist repaired IDs if state.json contained duplicate or empty IDs.
	restored := d.Catalog.Entries()
	for i := range restored {
		if i < len(raw.Entries) && restored[i].ID != raw.Entries[i].ID {
			_ = d.Save()
			break
		}
	}
	return d, nil
}

// StartServers starts the local Worker TCP Hub (127.0.0.1:9399) and MCP HTTP Server (127.0.0.1:9401).
func (d *Desk) StartServers() {
	d.Sessions.OnChange = func() {
		if d.NotifyUI != nil {
			d.NotifyUI()
		}
	}
	hubErr := d.Sessions.StartHub("127.0.0.1:9399")
	d.mcpSrv = &mcp.Server{
		Addr:     "127.0.0.1:9401",
		Sessions: d.Sessions,
		Traces:   d.Traces,
		IdaDir: func() string {
			if d.IdaDir != "" {
				return d.IdaDir
			}
			return d.Runtime().Install
		},
		Python: func() string {
			return d.Runtime().Python
		},
		OnChange: func() {
			if d.NotifyUI != nil {
				d.NotifyUI()
			}
		},
	}
	mcpErr := d.mcpSrv.Start()
	d.Listening = hubErr == nil && mcpErr == nil
}

// AddPath keeps path and saves. A duplicate selects the existing entry.
func (d *Desk) AddPath(path string) error {
	_, _, err := d.Catalog.Add(path)
	if err != nil {
		d.Notice = intake.Message(err)
		return err
	}
	d.Tab = TabLibrary
	d.Notice = ""
	return d.Save()
}

// Take applies paths from a file dialog or a drop. A cancel is ignored.
func (d *Desk) Take(paths []string, err error) {
	if err != nil || len(paths) == 0 {
		return
	}
	d.Notice = ""
	for _, p := range paths {
		if aerr := d.AddPath(p); aerr != nil {
			d.Notice = intake.Message(aerr)
		}
	}
}

// RemoveSelected drops the entry the library is showing.
func (d *Desk) RemoveSelected() {
	if e, ok := d.Catalog.Selected(); ok {
		d.Catalog.Remove(e.ID)
		_ = d.Save()
	}
}

// StartSession launches an idalib worker for entry, registers it, and logs a trace.
func (d *Desk) StartSession(e catalog.Entry) error {
	rt := d.Runtime()
	idaDir := d.IdaDir
	if idaDir == "" {
		idaDir = rt.Install
	}
	start := time.Now()
	s, err := d.Sessions.Launch(context.Background(), e.Name, e.Source, e.Path, idaDir, rt.Python)
	dur := time.Since(start)
	if err != nil {
		d.Traces.Append(trace.Entry{
			Function: "session.launch",
			Target:   e.Name,
			Duration: dur,
			Status:   "error",
			Summary:  "启动失败: " + err.Error(),
		})
		d.Notice = "启动失败: " + err.Error()
		return err
	}
	d.Traces.Append(trace.Entry{
		Function: "session.launch",
		Target:   e.Name,
		Duration: dur,
		Status:   "ok",
		Summary:  fmt.Sprintf("Worker 已就绪 (PID: %d)", s.PID),
	})
	d.Notice = ""
	d.SelectedSession = s.ID
	d.Tab = TabSessions
	return nil
}

// CloseSession terminates an active worker session.
func (d *Desk) CloseSession(id string) error {
	start := time.Now()
	err := d.Sessions.Close(id)
	dur := time.Since(start)
	status := "ok"
	summary := "会话已正常关闭"
	if err != nil {
		status = "error"
		summary = "关闭异常: " + err.Error()
	}
	d.Traces.Append(trace.Entry{
		Function: "session.close",
		Target:   id,
		Duration: dur,
		Status:   status,
		Summary:  summary,
	})
	return err
}

// SetIdaDir stores the directory the user picked.
func (d *Desk) SetIdaDir(path string) {
	d.IdaDir = path
	d.RefreshRuntime()
	_ = d.Save()
}

// Runtime is the last IDA probe. It scans only on first use and after a
// directory change or RefreshRuntime.
func (d *Desk) Runtime() idalib.Status {
	if !d.probed {
		d.runtime = idalib.Probe(d.IdaDir, idalib.ApplicationsDir())
		d.probed = true
	}
	return d.runtime
}

// RefreshRuntime scans again.
func (d *Desk) RefreshRuntime() {
	d.runtime = idalib.Probe(d.IdaDir, idalib.ApplicationsDir())
	d.probed = true
}

// Save writes state.json.
func (d *Desk) Save() error {
	s := stored{
		Entries: d.Catalog.Entries(),
		IdaDir:  d.IdaDir,
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(d.file(), b, 0o600)
}

func (d *Desk) file() string {
	return filepath.Join(d.dir, "state.json")
}
