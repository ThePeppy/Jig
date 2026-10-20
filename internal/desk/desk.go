// Package desk is the state the window shows. File dialogs stay in main.
package desk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"jig/internal/catalog"
	"jig/internal/idalib"
	"jig/internal/intake"
	"jig/internal/mcp"
	"jig/internal/session"
	"jig/internal/trace"
)

// Tab is the window's current page. An open worker is shown on TabLibrary.
type Tab int

const (
	TabLibrary Tab = iota
	TabTrace
	TabSettings
)

// TraceFilter narrows the record list.
type TraceFilter int

const (
	TraceAll TraceFilter = iota
	TraceOK
	TraceFailed
)

// copyHold is how long a copy key says 已复制.
const copyHold = 1500 * time.Millisecond

// Desk is main-thread state. Dialog results come back through Take.
type Desk struct {
	Tab Tab
	// SelectedSession focuses a worker that is not the selected catalog entry.
	SelectedSession string
	// Query filters the library list by name and path.
	Query       string
	TraceFilter TraceFilter
	// SelectedTrace is the record the detail column shows.
	SelectedTrace string
	Listening     bool
	Notice        string
	IdaDir        string
	Catalog       *catalog.Catalog
	Sessions      *session.Manager
	Traces        *trace.Store
	Now           func() time.Time
	RequestAdd    func()
	RequestIdaDir func()
	// RequestPack asks for a pack.json to attach to the session.
	RequestPack func(sessionID string)
	NotifyUI    func()
	// Post runs fn on the window thread. Nil runs fn on the caller.
	Post func(func())

	dir       string
	runtime   idalib.Status
	probed    bool
	mcpSrv    *mcp.Server
	launching map[string]time.Time
	failures  map[string]string
	saving    map[string]bool
	copiedKey string
	copiedAt  time.Time
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
		d.settleCloseTraces()
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

		OnChange: func() {
			if d.NotifyUI != nil {
				d.NotifyUI()
			}
		},
	}
	mcpErr := d.mcpSrv.Start()
	d.Listening = hubErr == nil && mcpErr == nil
}

// touch asks the window and the menu bar to redraw. Nil is a no-op.
func (d *Desk) touch() {
	if d.NotifyUI != nil {
		d.NotifyUI()
	}
}

// AddPath keeps path and saves. A duplicate selects the existing entry.
func (d *Desk) AddPath(path string) error {
	_, _, err := d.Catalog.Add(path)
	if err != nil {
		d.Notice = intake.Message(err)
		return err
	}
	d.SelectedSession = ""
	d.Tab = TabLibrary
	d.Notice = ""
	if err := d.Save(); err != nil {
		return err
	}
	d.touch()
	return nil
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
			d.Tab = TabLibrary
		}
	}
}

// RemoveSelected drops the entry the library is showing.
func (d *Desk) RemoveSelected() {
	if e, ok := d.Catalog.Selected(); ok {
		d.Catalog.Remove(e.ID)
		delete(d.failures, e.ID)
		_ = d.Save()
		d.touch()
	}
}

// Failure is why the last open of entry id failed, if it did.
func (d *Desk) Failure(id string) (string, bool) {
	msg, ok := d.failures[id]
	return msg, ok
}

// Dir is the data directory state.json lives in.
func (d *Desk) Dir() string { return d.dir }

// MarkCopied starts the 已复制 moment for key.
func (d *Desk) MarkCopied(key string) {
	d.copiedKey = key
	d.copiedAt = d.Now()
}

// CopiedFor is how much longer key says 已复制. Zero when it does not.
func (d *Desk) CopiedFor(key string) time.Duration {
	if key == "" || d.copiedKey != key {
		return 0
	}
	if left := copyHold - d.Now().Sub(d.copiedAt); left > 0 {
		return left
	}
	return 0
}

// Saving reports that a save of session id is still running.
func (d *Desk) Saving(id string) bool { return d.saving[id] }

// SaveSession saves the session's database off the window thread, then
// reads its metadata again so the window shows the state after the save.
func (d *Desk) SaveSession(id string) {
	if d.saving[id] {
		return
	}
	s, ok := d.Sessions.Get(id)
	if !ok {
		return
	}
	if d.saving == nil {
		d.saving = map[string]bool{}
	}
	d.saving[id] = true
	name := s.TargetName
	d.touch()
	run := func() {
		start := time.Now()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		_, err := d.Sessions.CallTool(ctx, id, "session.save", map[string]any{})
		var meta map[string]any
		if err == nil {
			if res, merr := d.Sessions.CallTool(ctx, id, "target.get_metadata", map[string]any{}); merr == nil {
				meta, _ = res.(map[string]any)
			}
		}
		dur := time.Since(start)
		d.post(func() { d.finishSave(id, name, meta, err, dur) })
	}
	if d.Post == nil {
		run()
		return
	}
	go run()
}

func (d *Desk) finishSave(id, name string, meta map[string]any, err error, dur time.Duration) {
	delete(d.saving, id)
	e := trace.Entry{Function: "session.save", Target: name, Duration: dur, Status: "ok", Summary: "数据库已保存"}
	if err != nil {
		e.Status, e.Summary, e.Result = "error", "保存失败: "+err.Error(), err.Error()
	}
	d.Traces.Append(e)
	if s, ok := d.Sessions.Get(id); ok && meta != nil {
		s.Metadata = meta
	}
}

// AttachPack puts pack.json at path on the session.
func (d *Desk) AttachPack(id, path string) {
	if _, err := d.Sessions.AttachPack(id, path); err != nil {
		d.Notice = "挂载失败: " + err.Error()
		return
	}
	d.Notice = ""
	d.toolsChanged()
}

// DetachPack takes the pack off the session. Core tools stay.
func (d *Desk) DetachPack(id string) {
	if err := d.Sessions.DetachPack(id); err != nil {
		d.Notice = "卸下失败: " + err.Error()
		return
	}
	d.toolsChanged()
}

// CloseAll asks every live worker to pack and leave.
func (d *Desk) CloseAll() {
	for _, s := range d.Sessions.Sessions() {
		if s.State == session.StateStopping || s.State == session.StateClosed {
			continue
		}
		if _, ok := d.Sessions.Get(s.ID); !ok {
			continue
		}
		_ = d.CloseSession(s.ID)
	}
}

func (d *Desk) toolsChanged() {
	if d.mcpSrv != nil {
		d.mcpSrv.ToolsChanged()
	}
	d.touch()
}

// StartSession launches an idalib worker for entry and logs a trace.
// The library keeps showing that file. The wait runs off the caller when
// Post is set, so the window can show the launch instead of freezing.
func (d *Desk) StartSession(e catalog.Entry) error {
	if d.Launching(e.ID) {
		return nil
	}
	if _, ok := d.Sessions.FindByPath(e.Path); ok {
		return nil
	}
	d.markLaunching(e.ID)
	delete(d.failures, e.ID)
	d.Catalog.Select(e.ID)
	d.Notice = ""
	d.touch()
	idaDir := d.IdaDir
	if idaDir == "" {
		idaDir = d.Runtime().Install
	}
	d.launchWorker(e, idaDir)
	return nil
}

// Launching reports that entry's worker is still starting.
func (d *Desk) Launching(id string) bool {
	_, ok := d.LaunchStarted(id)
	return ok
}

// LaunchStarted is when the launch was asked for.
func (d *Desk) LaunchStarted(id string) (time.Time, bool) {
	if d.launching == nil {
		return time.Time{}, false
	}
	t, ok := d.launching[id]
	return t, ok
}

func (d *Desk) markLaunching(id string) {
	if d.launching == nil {
		d.launching = map[string]time.Time{}
	}
	d.launching[id] = time.Now()
}

func (d *Desk) launchWorker(e catalog.Entry, idaDir string) {
	run := func() {
		start := time.Now()
		s, err := d.Sessions.Launch(context.Background(), e.Name, e.Source, e.Path, idaDir)
		dur := time.Since(start)
		d.post(func() { d.finishLaunch(e, s, err, dur) })
	}
	if d.Post == nil {
		run()
		return
	}
	go run()
}

func (d *Desk) finishLaunch(e catalog.Entry, s *session.Session, err error, dur time.Duration) {
	delete(d.launching, e.ID)
	if err != nil {
		d.Traces.Append(trace.Entry{
			Function: "session.launch",
			Target:   e.Name,
			Duration: dur,
			Status:   "error",
			Summary:  "启动失败: " + err.Error(),
			Result:   err.Error(),
		})
		if d.failures == nil {
			d.failures = map[string]string{}
		}
		d.failures[e.ID] = err.Error()
		return
	}
	d.Traces.Append(trace.Entry{
		Function: "session.launch",
		Target:   e.Name,
		Duration: dur,
		Status:   "ok",
		Summary:  fmt.Sprintf("Worker 已就绪 (PID: %d)", s.PID),
	})
	d.Notice = ""
	d.Catalog.Select(e.ID)
	d.SelectedSession = s.ID
}

func (d *Desk) post(fn func()) {
	if d.Post != nil {
		d.Post(fn)
		return
	}
	fn()
}

// CloseSession terminates the worker and every other live worker on the same
// file. The call log is retargeted to the file name, and launch rows that
// still say the worker is ready are rewritten so the bottom strip cannot
// keep showing 就绪 after the session is gone.
func (d *Desk) CloseSession(id string) error {
	ids, name, pids := d.closeTargets(id)
	start := time.Now()
	var err error
	for _, sid := range ids {
		if cerr := d.Sessions.Close(sid); cerr != nil && err == nil {
			err = cerr
		}
	}
	dur := time.Since(start)
	status := "ok"
	summary := "正在收拢数据库"
	if err != nil {
		status = "error"
		summary = "关闭异常: " + err.Error()
	} else if d.sessionsClosed(ids) {
		summary = "数据库已收拢"
	}
	if name == "" {
		name = id
	}
	d.Traces.Append(trace.Entry{
		Function: "session.close",
		Target:   name,
		Duration: dur,
		Status:   status,
		Summary:  summary,
	})
	for _, pid := range pids {
		if pid == 0 {
			continue
		}
		d.Traces.ReplaceSummary(
			fmt.Sprintf("Worker 已就绪 (PID: %d)", pid),
			fmt.Sprintf("Worker 已关闭 (PID: %d)", pid),
		)
	}
	d.settleCloseTraces()
	d.toolsChanged()
	return err
}

func (d *Desk) closeTargets(id string) (ids []string, name string, pids []int) {
	var seed *session.Session
	for _, s := range d.Sessions.Sessions() {
		if s.ID == id {
			seed = s
			break
		}
	}
	if seed == nil {
		return []string{id}, "", nil
	}
	name = seed.TargetName
	if name == "" {
		name = filepath.Base(firstNonEmpty(seed.SourcePath, seed.DatabasePath))
	}
	for _, s := range d.Sessions.Sessions() {
		if s.ID == id || sameSessionFile(seed, s) {
			ids = append(ids, s.ID)
			pids = append(pids, s.PID)
		}
	}
	if len(ids) == 0 {
		ids = []string{id}
	}
	return ids, name, pids
}

func (d *Desk) sessionsClosed(ids []string) bool {
	live := map[string]session.State{}
	for _, s := range d.Sessions.Sessions() {
		live[s.ID] = s.State
	}
	for _, id := range ids {
		if st, ok := live[id]; ok && st != session.StateClosed {
			return false
		}
	}
	return true
}

// settleCloseTraces finishes close rows once no live worker remains for that
// file. CloseSession records the request immediately; the worker may still be
// packing when that row is written.
func (d *Desk) settleCloseTraces() {
	live := map[string]bool{}
	for _, s := range d.Sessions.Sessions() {
		if s.TargetName != "" {
			live[s.TargetName] = true
		}
		if base := filepath.Base(firstNonEmpty(s.SourcePath, s.DatabasePath)); base != "" && base != "." {
			live[base] = true
		}
	}
	now := time.Now()
	d.Traces.Update(func(e trace.Entry) bool {
		return e.Function == "session.close" && e.Status == "ok" && e.Summary == "正在收拢数据库" && !live[e.Target]
	}, func(e *trace.Entry) {
		e.Summary = "数据库已收拢"
		if !e.Timestamp.IsZero() {
			e.Duration = now.Sub(e.Timestamp)
		}
	})
}

func sameSessionFile(a, b *session.Session) bool {
	for _, left := range []string{a.SourcePath, a.DatabasePath} {
		if left == "" {
			continue
		}
		for _, right := range []string{b.SourcePath, b.DatabasePath} {
			if right != "" && filepath.Clean(left) == filepath.Clean(right) {
				return true
			}
		}
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
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
