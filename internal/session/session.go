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
	"strconv"
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
	exited chan struct{}
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
	hubAddr  string
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
		host, _, err := net.SplitHostPort(addr)
		if err != nil || host == "" {
			host = "127.0.0.1"
		}
		m.hubAddr = net.JoinHostPort(host, strconv.Itoa(tcpAddr.Port))
	} else {
		m.hubAddr = addr
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

// Launch starts one native idalib worker for openPath. The worker links the
// installed libidalib and loads Hex-Rays itself. This process does not start Python.
func (m *Manager) Launch(ctx context.Context, targetName, sourcePath, openPath, idaDir string) (*Session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if openPath == "" {
		return nil, fmt.Errorf("target path is required")
	}
	if existing, ok := m.FindByPath(openPath); ok {
		return existing, nil
	}
	label := targetName
	if label == "" {
		label = filepath.Base(sourcePath)
	}
	if label == "" || label == "." {
		label = filepath.Base(openPath)
	}
	if idalib.LibraryFile(idaDir) == "" {
		return nil, fmt.Errorf("无法打开 %s：没有找到 libidalib", label)
	}
	bin, err := workerBinary()
	if err != nil {
		return nil, fmt.Errorf("无法打开 %s：%w", label, err)
	}
	m.mu.Lock()
	hub := m.hubAddr
	m.mu.Unlock()
	if hub == "" {
		return nil, fmt.Errorf("无法打开 %s：worker hub 没有在监听", label)
	}
	host, port, err := net.SplitHostPort(hub)
	if err != nil {
		return nil, fmt.Errorf("无法打开 %s：%w", label, err)
	}
	install := idalib.ResolveInstallDir(idaDir)
	id := fmt.Sprintf("jig-%d", time.Now().UnixNano())
	cmd := exec.Command(bin, "--center-host", host, "--center-port", port, "--path", openPath, "--instance-id", id, "--ida-dir", install)
	cmd.Env = append(os.Environ(), "IDADIR="+install)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	s := &Session{
		ID:           id,
		TargetName:   label,
		SourcePath:   sourcePath,
		DatabasePath: openPath,
		State:        StateLaunching,
		StartedAt:    time.Now(),
		cmd:          cmd,
		exited:       make(chan struct{}),
	}
	m.Register(s)
	if err := cmd.Start(); err != nil {
		m.failLaunch(s, err.Error())
		return nil, fmt.Errorf("无法打开 %s：%w", label, err)
	}
	s.mu.Lock()
	s.PID = cmd.Process.Pid
	s.mu.Unlock()

	exited := s.exited
	go func() {
		waitErr := cmd.Wait()
		s.mu.Lock()
		switch s.State {
		case StateLaunching, StateReady:
			s.State = StateError
			if waitErr != nil {
				s.Error = waitErr.Error()
			}
		case StateStopping:
			if waitErr != nil && s.Error == "" {
				s.Error = waitErr.Error()
			}
			s.State = StateClosed
		}
		s.mu.Unlock()
		close(exited)
		m.notify()
	}()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for {
		s.mu.Lock()
		ready := s.State == StateReady && s.conn != nil
		failed := s.State == StateError
		s.mu.Unlock()
		if ready {
			return s, nil
		}
		if failed {
			msg := strings.TrimSpace(stderr.String())
			if msg == "" {
				msg = "worker 退出"
			}
			m.failLaunch(s, msg)
			return nil, fmt.Errorf("无法打开 %s：%s", label, msg)
		}
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
			m.failLaunch(s, ctx.Err().Error())
			return nil, ctx.Err()
		case <-exited:
		case <-ticker.C:
		}
	}
}

func (m *Manager) failLaunch(s *Session, msg string) {
	s.mu.Lock()
	s.State = StateClosed
	s.Error = msg
	s.mu.Unlock()
	m.notify()
}

func workerBinary() (string, error) {
	if p := os.Getenv("JIG_IDALIB_WORKER"); p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p, nil
		}
		return "", fmt.Errorf("JIG_IDALIB_WORKER 不存在: %s", p)
	}
	var roots []string
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, wd)
	}
	for _, root := range roots {
		dir := root
		for range 5 {
			candidate := filepath.Join(dir, "native", "idalib-worker", "jig-idalib")
			if st, err := os.Stat(candidate); err == nil && !st.IsDir() {
				return candidate, nil
			}
			next := filepath.Dir(dir)
			if next == dir {
				break
			}
			dir = next
		}
		beside := filepath.Join(root, "jig-idalib")
		if st, err := os.Stat(beside); err == nil && !st.IsDir() {
			return beside, nil
		}
	}
	return "", fmt.Errorf("没有找到原生 worker jig-idalib")
}

// Close asks the worker to pack the database and exit. It does not kill the
// process: SIGKILL leaves the .i64 unpacked. The session stays Stopping until
// the process actually exits.
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
	exited := s.exited
	s.mu.Unlock()

	if conn != nil {
		s.writeM.Lock()
		_ = writeWireMessage(conn, map[string]any{"type": "shutdown", "instanceId": id})
		s.writeM.Unlock()
		_ = conn.Close()
	}
	if exited == nil {
		s.mu.Lock()
		s.State = StateClosed
		s.mu.Unlock()
	} else {
		select {
		case <-exited:
			s.mu.Lock()
			if s.State == StateStopping || s.State == StateError {
				s.State = StateClosed
			}
			s.mu.Unlock()
		default:
		}
	}
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
