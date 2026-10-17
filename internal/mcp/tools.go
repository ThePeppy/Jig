package mcp

import "jig/internal/pack"

type toolDef struct {
	name       string
	desc       string
	worker     string // empty: handled in this process
	required   []string
	properties map[string]any
}

func coreTools() []toolDef {
	addr := map[string]any{
		"type":        "string",
		"description": "Address or name. 0xhex, sub_X, or a symbol. Alias: query.",
	}
	instance := map[string]any{
		"type":        "string",
		"description": "idalib session id from ida_open / ida_list_targets. Required when more than one session is ready.",
	}
	offset := map[string]any{"type": "integer", "description": "Page start.", "default": 0}
	count := map[string]any{"type": "integer", "description": "Page size, 1–1000.", "default": 100}
	filter := map[string]any{"type": "string", "description": "Case-insensitive substring."}

	return []toolDef{
		{name: "ida_list_targets", desc: "List active headless idalib sessions."},
		{name: "ida_open", desc: "Open a binary or IDA database in a headless idalib worker.",
			required: []string{"path"},
			properties: map[string]any{
				"path": map[string]any{"type": "string", "description": "File to open. Not rewritten to a sibling .i64."},
				"pack": map[string]any{"type": "string", "description": "Optional pack.json or its directory."},
			}},
		{name: "ida_close_target", desc: "Close an active idalib session.",
			required:   []string{"instanceId"},
			properties: map[string]any{"instanceId": instance}},
		{name: "ida_save_target", desc: "Save the IDA database for an active session.", worker: "session.save",
			properties: map[string]any{"instanceId": instance}},
		{name: "ida_ping_target", desc: "Ping one idalib worker.", worker: "target.ping",
			properties: map[string]any{"instanceId": instance}},
		{name: "ida_get_metadata", desc: "Return binary and database metadata for a session.", worker: "target.get_metadata",
			properties: map[string]any{"instanceId": instance}},
		{name: "ida_start_auto_analysis", desc: "Enable background auto-analysis pumping.", worker: "session.start_auto_analysis",
			properties: map[string]any{"instanceId": instance}},
		{name: "ida_stop_auto_analysis", desc: "Stop background auto-analysis pumping.", worker: "session.stop_auto_analysis",
			properties: map[string]any{"instanceId": instance}},
		{name: "ida_list_functions", desc: "List functions from an IDA database.", worker: "analysis.list_functions",
			properties: map[string]any{"instanceId": instance, "offset": offset, "count": count, "filter": filter}},
		{name: "ida_get_function", desc: "Resolve one function by address or name.", worker: "analysis.get_function",
			required:   []string{"address"},
			properties: map[string]any{"instanceId": instance, "address": addr, "query": addr, "create": map[string]any{"type": "boolean", "description": "Create a function if none exists.", "default": false}}},
		{name: "ida_decompile", desc: "Decompile one function with Hex-Rays. Includes lvars.", worker: "analysis.decompile",
			required:   []string{"address"},
			properties: map[string]any{"instanceId": instance, "address": addr, "query": addr, "create": map[string]any{"type": "boolean", "default": false}}},
		{name: "ida_disassemble", desc: "Return disassembly lines for one function.", worker: "analysis.disassemble",
			required:   []string{"address"},
			properties: map[string]any{"instanceId": instance, "address": addr, "query": addr, "maxLines": map[string]any{"type": "integer", "default": 200}}},
		{name: "ida_xrefs", desc: "List code and data cross-references to or from an address.", worker: "analysis.xrefs",
			required: []string{"address"},
			properties: map[string]any{
				"instanceId": instance, "address": addr, "query": addr,
				"kind":      map[string]any{"type": "string", "enum": []string{"code", "data", "both"}, "default": "both"},
				"direction": map[string]any{"type": "string", "enum": []string{"to", "from", "both"}, "default": "both"},
				"max":       map[string]any{"type": "integer", "default": 200},
			}},
		{name: "ida_callees", desc: "Distinct call targets inside one function body.", worker: "analysis.callees",
			required:   []string{"address"},
			properties: map[string]any{"instanceId": instance, "address": addr, "query": addr, "max": map[string]any{"type": "integer", "default": 200}}},
		{name: "ida_calls", desc: "Hex-Rays call sites in one function, with assigned lvar when present.", worker: "analysis.calls",
			required:   []string{"address"},
			properties: map[string]any{"instanceId": instance, "address": addr, "query": addr, "max": map[string]any{"type": "integer", "default": 200}}},
		{name: "ida_lvars", desc: "Hex-Rays local variables for one function.", worker: "analysis.lvars",
			required:   []string{"address"},
			properties: map[string]any{"instanceId": instance, "address": addr, "query": addr}},
		{name: "ida_list_strings", desc: "List strings from an IDA database.", worker: "analysis.list_strings",
			properties: map[string]any{"instanceId": instance, "offset": offset, "count": count, "filter": filter}},
		{name: "ida_list_imports", desc: "List imports from an IDA database.", worker: "analysis.list_imports",
			properties: map[string]any{"instanceId": instance, "offset": offset, "count": count, "filter": filter}},
		{name: "ida_get_bytes", desc: "Read raw bytes from an IDA database.", worker: "analysis.get_bytes",
			required:   []string{"address"},
			properties: map[string]any{"instanceId": instance, "address": addr, "size": map[string]any{"type": "integer", "default": 16}}},
		{name: "ida_search_text", desc: "Search IDA's rendered listing with a bounded paged scanner.", worker: "analysis.search_text",
			required:   []string{"pattern"},
			properties: map[string]any{"instanceId": instance, "pattern": map[string]any{"type": "string"}, "limit": map[string]any{"type": "integer", "default": 30}}},
		{name: "ida_rename", desc: "Rename a function, data address, or Hex-Rays lvar.", worker: "analysis.rename",
			required: []string{"address", "newName"},
			properties: map[string]any{
				"instanceId": instance, "address": addr,
				"newName": map[string]any{"type": "string"},
				"oldName": map[string]any{"type": "string", "description": "Required when kind is lvar."},
				"kind":    map[string]any{"type": "string", "enum": []string{"function", "data", "lvar"}, "default": "function"},
			}},
		{name: "ida_set_comment", desc: "Set a regular or repeatable comment at an address.", worker: "analysis.set_comment",
			required:   []string{"address", "text"},
			properties: map[string]any{"instanceId": instance, "address": addr, "text": map[string]any{"type": "string"}, "repeatable": map[string]any{"type": "boolean", "default": false}}},
		{name: "ida_attach_pack", desc: "Attach a project analysis pack to a session.",
			required:   []string{"path"},
			properties: map[string]any{"instanceId": instance, "path": map[string]any{"type": "string", "description": "pack.json or the directory that contains it."}}},
		{name: "ida_detach_pack", desc: "Detach the analysis pack from a session.",
			properties: map[string]any{"instanceId": instance}},
	}
}

func workerToolMap() map[string]string {
	m := map[string]string{}
	for _, t := range coreTools() {
		if t.worker != "" {
			m[t.name] = t.worker
		}
	}
	return m
}

func coreToolNames() map[string]struct{} {
	m := map[string]struct{}{}
	for _, t := range coreTools() {
		m[t.name] = struct{}{}
	}
	return m
}

func toolDefinitionsFor(packs []*pack.Pack) []map[string]any {
	out := make([]map[string]any, 0, 32)
	seen := map[string]struct{}{}
	for _, t := range coreTools() {
		seen[t.name] = struct{}{}
		props := t.properties
		if props == nil {
			props = map[string]any{}
		}
		schema := map[string]any{"type": "object", "properties": props}
		if len(t.required) > 0 {
			schema["required"] = t.required
		}
		out = append(out, map[string]any{
			"name":        t.name,
			"description": t.desc,
			"inputSchema": schema,
		})
	}
	for _, p := range packs {
		if p == nil {
			continue
		}
		for _, t := range p.Tools {
			if _, ok := seen[t.Name]; ok {
				continue
			}
			seen[t.Name] = struct{}{}
			out = append(out, map[string]any{
				"name":        t.Name,
				"description": t.Description + " (pack " + p.ID + ")",
				"inputSchema": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"instanceId": map[string]any{"type": "string"},
						"address":    map[string]any{"type": "string", "description": "Handler address."},
						"category":   map[string]any{"type": "string", "description": "CMD or NOTI when using packetId."},
						"packetId":   map[string]any{"type": "integer"},
					},
				},
			})
		}
	}
	return out
}
