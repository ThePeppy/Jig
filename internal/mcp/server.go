// Package mcp implements the Streamable HTTP / JSON-RPC 2.0 MCP server on 127.0.0.1:9401.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"jig/internal/intake"
	"jig/internal/pack"
	"jig/internal/session"
	"jig/internal/trace"
)

// Server serves /mcp and /health over HTTP.
type Server struct {
	Addr     string
	Sessions *session.Manager
	Traces   *trace.Store
	IdaDir   func() string

	OnChange func()

	ln  net.Listener
	srv *http.Server

	mu   sync.Mutex
	subs map[chan []byte]struct{}
	seq  atomic.Uint64
}

type rpcRequest struct {
	JSONRPC string         `json:"jsonrpc"`
	ID      any            `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params"`
}

// Start begins listening on s.Addr (defaults to "127.0.0.1:9401").
func (s *Server) Start() error {
	addr := s.Addr
	if addr == "" {
		addr = "127.0.0.1:9401"
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.ln = ln

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/mcp", s.handleMCP)

	s.srv = &http.Server{Handler: mux}
	go func() {
		_ = s.srv.Serve(ln)
	}()
	return nil
}

// URL returns the active MCP endpoint URL.
func (s *Server) URL() string {
	if s.ln != nil {
		return "http://" + s.ln.Addr().String() + "/mcp"
	}
	return "http://127.0.0.1:9401/mcp"
}

// Close shuts down the HTTP server.
func (s *Server) Close() error {
	s.closeSubs()
	if s.srv != nil {
		return s.srv.Close()
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"status": "ok",
		"mcp":    s.URL(),
	})
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleSSE(w, r)
		return
	case http.MethodDelete:
		w.WriteHeader(http.StatusOK)
		return
	case http.MethodPost:
		s.handleRPC(w, r)
		return
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{"error": "method not allowed"})
	}
}

func (s *Server) handleRPC(w http.ResponseWriter, r *http.Request) {
	var req rpcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, rpcError(nil, -32700, "Parse error."))
		return
	}
	// JSON-RPC notifications (no id) get 202 Accepted
	if req.ID == nil && req.Method != "" {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	switch req.Method {
	case "initialize":
		proto := "2025-06-18"
		if v, ok := req.Params["protocolVersion"].(string); ok && v != "" {
			proto = v
		}
		w.Header().Set("Mcp-Session-Id", newMCPSessionID())
		writeJSON(w, http.StatusOK, rpcResult(req.ID, map[string]any{
			"protocolVersion": proto,
			"capabilities": map[string]any{
				"tools": map[string]any{"listChanged": true},
			},
			"serverInfo": map[string]any{
				"name":    "jig",
				"title":   "Jig",
				"version": "0.2.0",
			},
			"instructions": "Address tools accept address or query. Pass instanceId when more than one session is ready. ida_open may take pack; otherwise ida_attach_pack. Pack tools appear on tools/list after attach and disappear on detach; clients must honor notifications/tools/list_changed. Default decompile waits for Hex-Rays; create=false does not invent functions.",
		}))
	case "ping":
		writeJSON(w, http.StatusOK, rpcResult(req.ID, map[string]any{}))
	case "tools/list":
		var packs []*pack.Pack
		if s.Sessions != nil {
			packs = s.Sessions.AttachedPacks()
		}
		writeJSON(w, http.StatusOK, rpcResult(req.ID, map[string]any{
			"tools": toolDefinitionsFor(packs),
		}))
	case "tools/call":
		res := s.callTool(r.Context(), req.Params)
		writeJSON(w, http.StatusOK, rpcResult(req.ID, res))
	case "resources/list":
		writeJSON(w, http.StatusOK, rpcResult(req.ID, map[string]any{"resources": []any{}}))
	case "prompts/list":
		writeJSON(w, http.StatusOK, rpcResult(req.ID, map[string]any{"prompts": []any{}}))
	default:
		writeJSON(w, http.StatusOK, rpcError(req.ID, -32601, "Method not found: "+req.Method))
	}
}

func (s *Server) callTool(ctx context.Context, params map[string]any) map[string]any {
	name, _ := params["name"].(string)
	args, _ := params["arguments"].(map[string]any)
	if args == nil {
		args = map[string]any{}
	}

	start := time.Now()
	out, target, err := s.dispatchTool(ctx, name, args)
	dur := time.Since(start)

	if s.Traces != nil {
		status := "ok"
		summary := "执行完成"
		if err != nil {
			status = "error"
			summary = err.Error()
		} else if b, jerr := json.Marshal(out); jerr == nil {
			summary = string(b)
			if len(summary) > 120 {
				summary = summary[:120] + "..."
			}
		}
		if target == "" {
			target = "local"
		}
		s.Traces.Append(trace.Entry{
			Timestamp: time.Now(),
			Function:  name,
			Target:    target,
			Duration:  dur,
			Status:    status,
			Summary:   summary,
		})
		if s.OnChange != nil {
			s.OnChange()
		}
	}

	if err != nil {
		return map[string]any{
			"isError": true,
			"content": []map[string]any{
				{"type": "text", "text": err.Error()},
			},
		}
	}

	b, _ := json.MarshalIndent(out, "", "  ")
	return map[string]any{
		"isError": false,
		"content": []map[string]any{
			{"type": "text", "text": string(b)},
		},
	}
}

func (s *Server) dispatchTool(ctx context.Context, name string, args map[string]any) (any, string, error) {
	instanceID, _ := args["instanceId"].(string)

	switch name {
	case "ida_list_targets":
		var list []map[string]any
		for _, sess := range s.Sessions.Sessions() {
			list = append(list, map[string]any{
				"instanceId":   sess.ID,
				"targetName":   sess.TargetName,
				"sourcePath":   sess.SourcePath,
				"databasePath": sess.DatabasePath,
				"sessionState": sess.State,
				"pid":          sess.PID,
				"packId":       sess.PackID,
				"packPath":     sess.PackPath,
			})
		}
		return map[string]any{"targets": list}, "center", nil

	case "ida_open":
		rawPath, _ := args["path"].(string)
		decision, err := intake.Accept(rawPath)
		if err != nil {
			return nil, filepath.Base(rawPath), err
		}
		idaDir := ""
		if s.IdaDir != nil {
			idaDir = s.IdaDir()
		}
		sess, err := s.Sessions.Launch(ctx, decision.Name, decision.Source, decision.OpenPath, idaDir)
		if err != nil {
			return nil, decision.Name, err
		}
		out := map[string]any{
			"instanceId":   sess.ID,
			"targetName":   sess.TargetName,
			"databasePath": sess.DatabasePath,
			"pid":          sess.PID,
			"state":        sess.State,
		}
		if packPath, _ := args["pack"].(string); packPath != "" {
			p, perr := s.Sessions.AttachPack(sess.ID, packPath)
			if perr != nil {
				out["packError"] = perr.Error()
			} else {
				out["packId"] = p.ID
				out["packPath"] = p.Path
				s.notifyToolsChanged()
			}
		}
		return out, decision.Name, nil

	case "ida_close_target":
		if instanceID == "" {
			return nil, "", fmt.Errorf("instanceId is required")
		}
		err := s.Sessions.Close(instanceID)
		if err == nil {
			s.notifyToolsChanged()
		}
		return map[string]any{"ok": err == nil, "instanceId": instanceID}, instanceID, err

	case "ida_attach_pack":
		path, _ := args["path"].(string)
		p, err := s.Sessions.AttachPack(instanceID, path)
		if err != nil {
			return nil, instanceID, err
		}
		s.notifyToolsChanged()
		return map[string]any{"ok": true, "packId": p.ID, "packPath": p.Path, "instanceId": instanceID}, instanceID, nil

	case "ida_detach_pack":
		err := s.Sessions.DetachPack(instanceID)
		if err == nil {
			s.notifyToolsChanged()
		}
		return map[string]any{"ok": err == nil, "instanceId": instanceID}, instanceID, err
	}

	if _, ok := workerToolMap()[name]; !ok {
		return s.runPackTool(ctx, instanceID, name, args)
	}

	workerTool := workerToolMap()[name]
	targetLabel := instanceID
	for _, sess := range s.Sessions.Sessions() {
		if instanceID == "" || sess.ID == instanceID {
			targetLabel = sess.TargetName
			break
		}
	}

	cctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	res, err := s.Sessions.CallTool(cctx, instanceID, workerTool, args)
	return res, targetLabel, err
}

func rpcResult(id any, result any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}
}

func rpcError(id any, code int, msg string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]any{
			"code":    code,
			"message": msg,
		},
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
