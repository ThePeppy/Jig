package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"jig/internal/session"
)

func TestInitializeSetsSessionID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc((&Server{}).handleMCP))
	t.Cleanup(ts.Close)
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}`))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("Mcp-Session-Id") == "" {
		t.Fatal("missing Mcp-Session-Id")
	}
}

func TestGETWithoutSSEIs405(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc((&Server{}).handleMCP))
	t.Cleanup(ts.Close)
	req, err := http.NewRequest(http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestAttachPackNotifiesListChanged(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "codec.tsv"), []byte("category\tpacket_id\tpacket_name\thandler_va\tfield_index\tdepth\treader\tsize\tcall_va\treader_va\nCMD\t1\tLOGIN\t0x1\t0\t0\tu8\t1\t0x2\t0x3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pack.json"), []byte(`{"id":"dnf90","tools":[{"name":"ida_dnf90_packet_fields","kind":"fields_from_calls","config":{"codec":"codec.tsv"}}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mgr := session.NewManager()
	mgr.Register(&session.Session{ID: "sess", State: session.StateReady, TargetName: "t"})
	s := &Server{Sessions: mgr}
	ts := httptest.NewServer(http.HandlerFunc(s.handleMCP))
	t.Cleanup(ts.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Accept", "text/event-stream")
	got := make(chan string, 1)
	go func() {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return
		}
		defer resp.Body.Close()
		sc := bufio.NewScanner(resp.Body)
		for sc.Scan() {
			line := sc.Text()
			if strings.HasPrefix(line, "data: ") {
				got <- strings.TrimPrefix(line, "data: ")
				return
			}
		}
	}()

	deadline := time.Now().Add(time.Second)
	for {
		s.mu.Lock()
		n := len(s.subs)
		s.mu.Unlock()
		if n > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("SSE listener never subscribed")
		}
		time.Sleep(5 * time.Millisecond)
	}

	call := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "ida_attach_pack",
			"arguments": map[string]any{
				"instanceId": "sess",
				"path":       dir,
			},
		},
	}
	raw, err := json.Marshal(call)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(ts.URL, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("attach status %d", resp.StatusCode)
	}

	select {
	case msg := <-got:
		if !strings.Contains(msg, `"notifications/tools/list_changed"`) {
			t.Fatalf("notify %s", msg)
		}
	case <-ctx.Done():
		t.Fatal("no list_changed")
	}
}
