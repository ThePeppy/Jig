// Package session manages headless idalib worker instances and the TCP worker hub (127.0.0.1:9399).
package session

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"jig/internal/idalib"
)

// State describes the lifecycle stage of a worker session.
type State string

const (
	StateLaunching State = "launching"
	StateReady     State = "ready"
	StateStopping  State = "stopping"
	StateClosed    State = "closed"
	StateError     State = "error"
)

// Session represents one active headless idalib analysis process.
type Session struct {
	ID           string         `json:"id"`
	TargetName   string         `json:"targetName"`
	SourcePath   string         `json:"sourcePath"`
	DatabasePath string         `json:"databasePath"`
	State        State          `json:"state"`
	PID          int            `json:"pid"`
	StartedAt    time.Time      `json:"startedAt"`
	Error        string         `json:"error,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`

	cmd    *exec.Cmd
	cancel context.CancelFunc
	conn   net.Conn
	writeM sync.Mutex
	mu     sync.Mutex
}

// Elapsed returns how long the session has been active.
func (s *Session) Elapsed() string {
	d := time.Since(s.StartedAt).Round(time.Second)
	m := int(d.Minutes())
	sec := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", m, sec)
}

type pendingCall struct {
	ch chan toolResultPayload
}

type wireMessage struct {
	Type       string         `json:"type"`
	ID         string         `json:"id,omitempty"`
	InstanceID string         `json:"instanceId,omitempty"`
	Payload    map[string]any `json:"payload,omitempty"`
}

type toolResultPayload struct {
	OK     bool   `json:"ok"`
	Result any    `json:"result"`
	Error  string `json:"error"`
}

// Manager manages worker session lifecycles and the TCP hub that workers connect to.
type Manager struct {
	mu       sync.Mutex
	sessions map[string]*Session
	order    []string
	pending  map[string]*pendingCall
	reqSeq   atomic.Uint64
	ln       net.Listener
	hubPort  int
	OnChange func()
}

// NewManager creates an empty session manager.
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
		pending:  make(map[string]*pendingCall),
		hubPort:  9399,
	}
}

// StartHub listens on addr (e.g. "127.0.0.1:9399") for incoming worker TCP connections.
func (m *Manager) StartHub(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.ln = ln
	if tcpAddr, ok := ln.Addr().(*net.TCPAddr); ok {
		m.hubPort = tcpAddr.Port
	}
	m.mu.Unlock()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go m.handleConn(conn)
		}
	}()
	return nil
}

// CloseHub stops the TCP listener.
func (m *Manager) CloseHub() {
	m.mu.Lock()
	ln := m.ln
	m.ln = nil
	m.mu.Unlock()
	if ln != nil {
		_ = ln.Close()
	}
}

func (m *Manager) handleConn(conn net.Conn) {
	defer conn.Close()
	var boundID string
	defer func() {
		if boundID != "" {
			m.mu.Lock()
			if s, ok := m.sessions[boundID]; ok {
				s.mu.Lock()
				if s.conn == conn {
					s.conn = nil
				}
				s.mu.Unlock()
			}
			m.mu.Unlock()
			m.notify()
		}
	}()

	for {
		msg, err := readWireMessage(conn)
		if err != nil {
			return
		}
		switch msg.Type {
		case "hello", "heartbeat":
			if msg.InstanceID != "" {
				boundID = msg.InstanceID
				m.mu.Lock()
				s, ok := m.sessions[boundID]
				if !ok {
					targetName := boundID
					if p, ok := msg.Payload["inputPath"].(string); ok && p != "" {
						targetName = filepath.Base(p)
					}
					s = &Session{
						ID:           boundID,
						TargetName:   targetName,
						SourcePath:   stringField(msg.Payload, "inputPath"),
						DatabasePath: stringField(msg.Payload, "databasePath"),
						State:        StateReady,
						StartedAt:    time.Now(),
					}
					m.sessions[boundID] = s
					m.order = append(m.order, boundID)
				}
				s.mu.Lock()
				s.conn = conn
				s.State = StateReady
				s.Metadata = msg.Payload
				if dbPath := stringField(msg.Payload, "databasePath"); dbPath != "" {
					s.DatabasePath = dbPath
				}
				s.mu.Unlock()
				m.mu.Unlock()
				m.notify()
			}
		case "tool_result":
			if msg.ID != "" {
				m.mu.Lock()
				p := m.pending[msg.ID]
				delete(m.pending, msg.ID)
				m.mu.Unlock()
				if p != nil {
					res := toolResultPayload{
						OK:     boolField(msg.Payload, "ok"),
						Result: msg.Payload["result"],
						Error:  stringField(msg.Payload, "error"),
					}
					p.ch <- res
				}
			}
		}
	}
}

// CallTool invokes a tool on a connected worker session and waits for the result.
func (m *Manager) CallTool(ctx context.Context, instanceID, toolName string, arguments map[string]any) (any, error) {
	m.mu.Lock()
	var target *Session
	if instanceID != "" {
		target = m.sessions[instanceID]
	} else {
		for _, id := range m.order {
			if s := m.sessions[id]; s != nil && s.State == StateReady && s.conn != nil {
				target = s
				break
			}
		}
	}
	if target == nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("no active idalib session found")
	}
	target.mu.Lock()
	conn := target.conn
	targetID := target.ID
	target.mu.Unlock()
	if conn == nil {
		m.mu.Unlock()
		return nil, fmt.Errorf("session %s is not connected yet", targetID)
	}

	reqID := fmt.Sprintf("req-%d", m.reqSeq.Add(1))
	pc := &pendingCall{ch: make(chan toolResultPayload, 1)}
	m.pending[reqID] = pc
	m.mu.Unlock()

	if arguments == nil {
		arguments = map[string]any{}
	}
	msg := map[string]any{
		"type":       "tool_call",
		"id":         reqID,
		"instanceId": targetID,
		"payload": map[string]any{
			"tool":      toolName,
			"arguments": arguments,
		},
	}

	target.writeM.Lock()
	err := writeWireMessage(conn, msg)
	target.writeM.Unlock()
	if err != nil {
		m.mu.Lock()
		delete(m.pending, reqID)
		m.mu.Unlock()
		return nil, err
	}

	select {
	case <-ctx.Done():
		m.mu.Lock()
		delete(m.pending, reqID)
		m.mu.Unlock()
		return nil, ctx.Err()
	case res := <-pc.ch:
		if !res.OK {
			if res.Error != "" {
				return nil, fmt.Errorf("%s", res.Error)
			}
			return nil, fmt.Errorf("tool %s failed", toolName)
		}
		return res.Result, nil
	}
}

// Sessions returns all active registered sessions in creation order.
func (m *Manager) Sessions() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	res := make([]*Session, 0, len(m.order))
	for _, id := range m.order {
		if s, ok := m.sessions[id]; ok && s.State != StateClosed {
			res = append(res, s)
		}
	}
	return res
}

// FindByPath returns an active session for the given target path if one exists.
func (m *Manager) FindByPath(path string) (*Session, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	clean := filepath.Clean(path)
	for _, id := range m.order {
		s := m.sessions[id]
		if filepath.Clean(s.SourcePath) == clean || filepath.Clean(s.DatabasePath) == clean {
			if s.State != StateClosed && s.State != StateError {
				return s, true
			}
		}
	}
	return nil, false
}

// Register tracks a pre-constructed session.
func (m *Manager) Register(s *Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[s.ID] = s
	m.order = append(m.order, s.ID)
}

// Launch starts a new headless worker session for openPath.
func (m *Manager) Launch(ctx context.Context, targetName, sourcePath, openPath, idaDir, pythonPath string) (*Session, error) {
	if openPath == "" {
		return nil, fmt.Errorf("target path is required")
	}
	if existing, ok := m.FindByPath(openPath); ok {
		return existing, nil
	}

	m.mu.Lock()
	id := fmt.Sprintf("s-%x", time.Now().UnixNano()%0xffffff)
	port := m.hubPort
	if port == 0 {
		port = 9399
	}
	m.mu.Unlock()

	if idaDir != "" {
		_ = idalib.EnsureUserConfig(idaDir)
	}
	resolvedIdaDir := idalib.ResolveInstallDir(idaDir)
	idalibPyDir := idalib.IdalibPythonDir(idaDir)

	s := &Session{
		ID:           id,
		TargetName:   targetName,
		SourcePath:   sourcePath,
		DatabasePath: openPath,
		State:        StateLaunching,
		StartedAt:    time.Now(),
	}

	if pythonPath == "" {
		pythonPath = "python3"
	}

	repoRoot := findRepoRoot()
	workerDir := filepath.Join(repoRoot, "idalib-worker", "src")
	pluginDir := filepath.Join(repoRoot, "ida-plugin", "src")

	cctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	cmd := exec.CommandContext(cctx, pythonPath, "-m", "supper_idalib_worker",
		"--center-host", "127.0.0.1",
		"--center-port", fmt.Sprintf("%d", port),
		"--path", openPath,
		"--instance-id", id,
	)

	var pyPaths []string
	if idalibPyDir != "" {
		pyPaths = append(pyPaths, idalibPyDir)
	}
	pyPaths = append(pyPaths, workerDir, pluginDir)
	if existing := os.Getenv("PYTHONPATH"); existing != "" {
		pyPaths = append(pyPaths, existing)
	}

	env := os.Environ()
	env = append(env, "PYTHONPATH="+strings.Join(pyPaths, string(filepath.ListSeparator)), "PYTHONUNBUFFERED=1")
	if resolvedIdaDir != "" {
		env = append(env, "IDADIR="+resolvedIdaDir, "IDA_PATH="+resolvedIdaDir)
	}
	cmd.Env = env

	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	s.cmd = cmd
	if err := cmd.Start(); err != nil {
		cancel()
		s.State = StateError
		s.Error = err.Error()
		m.mu.Lock()
		m.sessions[id] = s
		m.order = append(m.order, id)
		m.mu.Unlock()
		return s, err
	}

	s.PID = cmd.Process.Pid
	s.State = StateReady
	m.mu.Lock()
	m.sessions[id] = s
	m.order = append(m.order, id)
	m.mu.Unlock()
	m.notify()

	go func() {
		err := cmd.Wait()
		s.mu.Lock()
		if s.State != StateClosed && s.State != StateStopping {
			if err != nil {
				s.State = StateClosed
				errText := strings.TrimSpace(stderrBuf.String())
				if errText != "" {
					lines := strings.Split(errText, "\n")
					s.Error = lines[len(lines)-1]
				} else {
					s.Error = err.Error()
				}
			} else {
				s.State = StateClosed
			}
		}
		s.mu.Unlock()
		m.notify()
	}()

	return s, nil
}

// Close gracefully terminates a worker session.
func (m *Manager) Close(id string) error {
	m.mu.Lock()
	s, ok := m.sessions[id]
	m.mu.Unlock()
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}

	s.mu.Lock()
	s.State = StateStopping
	conn := s.conn
	s.conn = nil
	cancel := s.cancel
	cmd := s.cmd
	s.mu.Unlock()

	if conn != nil {
		s.writeM.Lock()
		_ = writeWireMessage(conn, map[string]any{"type": "shutdown", "instanceId": id})
		s.writeM.Unlock()
		_ = conn.Close()
	}
	if cancel != nil {
		cancel()
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}

	s.mu.Lock()
	s.State = StateClosed
	s.mu.Unlock()
	m.notify()
	return nil
}

func (m *Manager) notify() {
	if m.OnChange != nil {
		m.OnChange()
	}
}

func readWireMessage(r io.Reader) (wireMessage, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return wireMessage{}, err
	}
	length := binary.BigEndian.Uint32(hdr[:])
	if length == 0 || length > 64*1024*1024 {
		return wireMessage{}, fmt.Errorf("invalid frame length %d", length)
	}
	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return wireMessage{}, err
	}
	var msg wireMessage
	if err := json.Unmarshal(buf, &msg); err != nil {
		return wireMessage{}, err
	}
	return msg, nil
}

func writeWireMessage(w io.Writer, msg any) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	var hdr [4]byte
	binary.BigEndian.PutUint32(hdr[:], uint32(len(b)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, _ := m[key].(string)
	return v
}

func boolField(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, _ := m[key].(bool)
	return v
}

func findRepoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for dir := wd; dir != "" && dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "idalib-worker")); err == nil {
			return dir
		}
	}
	return "."
}
