package mcp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func newMCPSessionID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func acceptsSSE(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept"), ",") {
		media := strings.TrimSpace(strings.Split(part, ";")[0])
		if media == "text/event-stream" {
			return true
		}
	}
	return false
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	if !acceptsSSE(r) {
		w.Header().Set("Allow", "GET")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "SSE required"})
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]any{"error": "streaming unsupported"})
		return
	}
	ch := make(chan []byte, 8)
	s.addSub(ch)
	defer s.removeSub(ch)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			id := s.seq.Add(1)
			fmt.Fprintf(w, "id: %d\nevent: message\ndata: %s\n\n", id, msg)
			flusher.Flush()
		}
	}
}

func (s *Server) addSub(ch chan []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.subs == nil {
		s.subs = map[chan []byte]struct{}{}
	}
	s.subs[ch] = struct{}{}
}

func (s *Server) removeSub(ch chan []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.subs, ch)
}

func (s *Server) closeSubs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		close(ch)
		delete(s.subs, ch)
	}
}

// notifyToolsChanged tells Streamable HTTP clients to re-run tools/list.
func (s *Server) notifyToolsChanged() {
	payload, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/tools/list_changed",
	})
	if err != nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for ch := range s.subs {
		select {
		case ch <- payload:
		default:
		}
	}
}
