package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"jig/internal/pack"
)

func (s *Server) runPackTool(ctx context.Context, instanceID, name string, args map[string]any) (any, string, error) {
	p := s.Sessions.PackOf(instanceID)
	if p == nil {
		return nil, instanceID, fmt.Errorf("这个会话没有挂分析包")
	}
	tool, ok := p.Lookup(name)
	if !ok {
		return nil, instanceID, fmt.Errorf("分析包 %s 没有工具 %s", p.ID, name)
	}
	switch tool.Kind {
	case pack.KindFieldsFromCalls:
		return s.fieldsFromCalls(ctx, instanceID, tool, args)
	default:
		return nil, instanceID, fmt.Errorf("不支持 kind %s", tool.Kind)
	}
}

func (s *Server) fieldsFromCalls(ctx context.Context, instanceID string, tool pack.Tool, args map[string]any) (any, string, error) {
	address := stringArg(args, "address", "query")
	category := stringArg(args, "category")
	packetID := intArg(args, "packetId", "packet_id")
	if address == "" && category == "" && packetID == 0 {
		return nil, instanceID, fmt.Errorf("需要 address 或 category+packetId")
	}
	handler, err := pack.HandlerFromCodec(tool.Config, category, packetID, address)
	if err != nil && address == "" {
		return nil, instanceID, err
	}
	if handler == "" {
		handler = address
	}

	sites, err := pack.CallSites(tool.Config, category, packetID, handler)
	if err != nil {
		return nil, instanceID, err
	}

	cctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	funcs := s.functionsForSites(cctx, instanceID, sites, handler)
	var calls []pack.Call
	for _, fn := range funcs {
		fctx, done := context.WithTimeout(cctx, 25*time.Second)
		raw, err := s.Sessions.CallTool(fctx, instanceID, "analysis.calls", map[string]any{
			"address": fn,
			"query":   fn,
			"max":     500,
		})
		done()
		if err != nil {
			continue
		}
		calls = append(calls, parseCalls(raw)...)
	}
	fields, err := pack.FieldsFromCalls(tool.Config, category, packetID, handler, calls)
	if err != nil {
		return nil, instanceID, err
	}
	return map[string]any{
		"pack":      pName(s, instanceID),
		"tool":      tool.Name,
		"handler":   handler,
		"functions": funcs,
		"fields":    fields,
		"n":         len(fields),
	}, instanceID, nil
}

const maxFieldFuncs = 24

func (s *Server) functionsForSites(ctx context.Context, instanceID string, sites []string, fallback string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, site := range sites {
		if len(out) >= maxFieldFuncs {
			break
		}
		fctx, done := context.WithTimeout(ctx, 15*time.Second)
		raw, err := s.Sessions.CallTool(fctx, instanceID, "analysis.get_function", map[string]any{
			"address": site,
			"query":   site,
		})
		done()
		if err != nil {
			continue
		}
		root, _ := raw.(map[string]any)
		if root == nil {
			continue
		}
		start := asString(root["addr"])
		if start == "" {
			continue
		}
		canon, err := pack.CanonicalEA(start)
		if err != nil {
			continue
		}
		if _, ok := seen[canon]; ok {
			continue
		}
		seen[canon] = struct{}{}
		out = append(out, canon)
	}
	if len(out) == 0 && fallback != "" {
		return []string{fallback}
	}
	return out
}

func pName(s *Server, instanceID string) string {
	p := s.Sessions.PackOf(instanceID)
	if p == nil {
		return ""
	}
	return p.ID
}

func parseCalls(raw any) []pack.Call {
	root, _ := raw.(map[string]any)
	if root == nil {
		return nil
	}
	data, _ := root["calls"].([]any)
	if data == nil {
		data, _ = root["data"].([]any)
	}
	var out []pack.Call
	for _, item := range data {
		row, _ := item.(map[string]any)
		if row == nil {
			continue
		}
		out = append(out, pack.Call{
			CallSite:     asString(row["callSite"]),
			Callee:       asString(row["callee"]),
			CalleeName:   asString(row["calleeName"]),
			AssignedTo:   asString(row["assignedTo"]),
			AssignedNice: asBool(row["assignedNice"]),
		})
	}
	return out
}

func stringArg(args map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := args[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func intArg(args map[string]any, keys ...string) int {
	for _, k := range keys {
		switch v := args[k].(type) {
		case float64:
			return int(v)
		case int:
			return v
		case int64:
			return int(v)
		case json.Number:
			n, _ := v.Int64()
			return int(n)
		}
	}
	return 0
}

func asString(v any) string {
	s, _ := v.(string)
	return s
}

func asBool(v any) bool {
	b, _ := v.(bool)
	return b
}
