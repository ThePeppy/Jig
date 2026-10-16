// Package mcp implements the Streamable HTTP / JSON-RPC 2.0 MCP server on 127.0.0.1:9401.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"time"

	"jig/internal/intake"
	"jig/internal/session"
	"jig/internal/trace"
)

// Server serves /mcp and /health over HTTP.
type Server struct {
	Addr     string
	Sessions *session.Manager
	Traces   *trace.Store
	IdaDir   func() string
	Python   func() string
	OnChange func()

	ln  net.Listener
	srv *http.Server
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
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusNotFound, map[string]any{"error": "not found"})
		return
	}
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
		}))
	case "ping":
		writeJSON(w, http.StatusOK, rpcResult(req.ID, map[string]any{}))
	case "tools/list":
		writeJSON(w, http.StatusOK, rpcResult(req.ID, map[string]any{
			"tools": toolDefinitions(),
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
		py := "python3"
		if s.Python != nil && s.Python() != "" {
			py = s.Python()
		}
		sess, err := s.Sessions.Launch(ctx, decision.Name, decision.Source, decision.OpenPath, idaDir, py)
		if err != nil {
			return nil, decision.Name, err
		}
		return map[string]any{
			"instanceId":   sess.ID,
			"targetName":   sess.TargetName,
			"databasePath": sess.DatabasePath,
			"pid":          sess.PID,
			"state":        sess.State,
		}, decision.Name, nil

	case "ida_close_target":
		if instanceID == "" {
			return nil, "", fmt.Errorf("instanceId is required")
		}
		err := s.Sessions.Close(instanceID)
		return map[string]any{"ok": err == nil, "instanceId": instanceID}, instanceID, err
	}

	workerToolMap := map[string]string{
		"ida_ping_target":         "target.ping",
		"ida_get_metadata":        "target.get_metadata",
		"ida_save_target":         "session.save",
		"ida_start_auto_analysis": "session.start_auto_analysis",
		"ida_stop_auto_analysis":  "session.stop_auto_analysis",
		"ida_list_functions":      "analysis.list_functions",
		"ida_get_function":        "analysis.get_function",
		"ida_decompile":           "analysis.decompile",
		"ida_disassemble":         "analysis.disassemble",
		"ida_xrefs":               "analysis.xrefs",
		"ida_list_strings":        "analysis.list_strings",
		"ida_list_imports":        "analysis.list_imports",
		"ida_get_bytes":           "analysis.get_bytes",
		"ida_search_text":         "analysis.search_text",
		"ida_rename":              "analysis.rename",
		"ida_set_comment":         "analysis.set_comment",
	}

	workerTool, ok := workerToolMap[name]
	if !ok {
		return nil, instanceID, fmt.Errorf("unknown tool: %s", name)
	}

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

func toolDefinitions() []map[string]any {
	names := []struct {
		name, desc string
	}{
		{"ida_list_targets", "List active headless idalib sessions."},
		{"ida_open", "Open a binary or IDA database in a headless idalib worker."},
		{"ida_close_target", "Close an active idalib session."},
		{"ida_save_target", "Save the IDA database for an active session."},
		{"ida_get_metadata", "Return binary and database metadata for a session."},
		{"ida_list_functions", "List functions from an IDA database."},
		{"ida_get_function", "Resolve one function by address or name."},
		{"ida_decompile", "Decompile one function with Hex-Rays."},
		{"ida_disassemble", "Return disassembly lines for one function."},
		{"ida_xrefs", "List code and data cross-references to or from an address."},
		{"ida_list_strings", "List strings from an IDA database."},
		{"ida_list_imports", "List imports from an IDA database."},
		{"ida_get_bytes", "Read raw bytes from an IDA database."},
		{"ida_search_text", "Search IDA's rendered listing with a bounded paged scanner."},
		{"ida_rename", "Rename an address or function in an IDA database."},
		{"ida_set_comment", "Set a regular or repeatable comment at an address."},
	}
	out := make([]map[string]any, 0, len(names))
	for _, t := range names {
		out = append(out, map[string]any{
			"name":        t.name,
			"description": t.desc,
			"inputSchema": map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		})
	}
	return out
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
