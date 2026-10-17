// Package pack loads a project analysis pack (pack.json).
// It does not talk to IDA, the window, or the worker. Relative paths
// resolve against the directory that contains pack.json.
package pack

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const fileName = "pack.json"

var toolNameRe = regexp.MustCompile(`^ida_[a-z0-9_]+$`)

// KindFieldsFromCalls joins codec rows with Hex-Rays call sites.
const KindFieldsFromCalls = "fields_from_calls"

// Pack is one project's extra MCP tools.
type Pack struct {
	ID    string
	Name  string
	Path  string // absolute pack.json
	Dir   string
	Tools []Tool
}

// Tool is one extra MCP tool advertised while this pack is attached.
type Tool struct {
	Name        string
	Description string
	Kind        string
	Config      Config
}

// Config is kind-specific. Unknown keys are ignored.
type Config struct {
	Codec   []string          // absolute TSV paths
	Readers map[string]string // canonical 0xea → reader name
	Depth   int               // codec depth to keep; default 0
}

type fileShape struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Tools []struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Kind        string          `json:"kind"`
		Config      json.RawMessage `json:"config"`
	} `json:"tools"`
}

type fieldsConfigShape struct {
	Codec   any               `json:"codec"`
	Readers map[string]string `json:"readers"`
	Depth   *int              `json:"depth"`
}

// Load reads path. A directory means path/pack.json.
func Load(path string) (*Pack, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("pack path is required")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("分析包路径无效")
	}
	info, err := os.Stat(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("找不到分析包: %s", abs)
		}
		return nil, fmt.Errorf("打不开分析包: %s", abs)
	}
	file := abs
	if info.IsDir() {
		file = filepath.Join(abs, fileName)
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("找不到 %s", file)
		}
		return nil, fmt.Errorf("读不了 %s", file)
	}
	var shape fileShape
	if err := json.Unmarshal(raw, &shape); err != nil {
		return nil, fmt.Errorf("pack.json 不是合法 JSON")
	}
	id := strings.TrimSpace(shape.ID)
	if id == "" || strings.ContainsAny(id, " \t") {
		return nil, fmt.Errorf("pack.json 需要 id")
	}
	if len(shape.Tools) == 0 {
		return nil, fmt.Errorf("pack.json 没有 tools")
	}
	dir := filepath.Dir(file)
	p := &Pack{
		ID:   id,
		Name: strings.TrimSpace(shape.Name),
		Path: file,
		Dir:  dir,
	}
	if p.Name == "" {
		p.Name = id
	}
	seen := map[string]struct{}{}
	for _, t := range shape.Tools {
		name := strings.TrimSpace(t.Name)
		if !toolNameRe.MatchString(name) {
			return nil, fmt.Errorf("非法工具名 %q", t.Name)
		}
		if Reserved(name) {
			return nil, fmt.Errorf("工具名与核心冲突: %s", name)
		}
		if _, ok := seen[name]; ok {
			return nil, fmt.Errorf("重复工具名 %s", name)
		}
		seen[name] = struct{}{}
		kind := strings.TrimSpace(t.Kind)
		if kind == "" {
			return nil, fmt.Errorf("%s 缺少 kind", name)
		}
		if kind != KindFieldsFromCalls {
			return nil, fmt.Errorf("%s 不支持 kind %s", name, kind)
		}
		cfg, err := parseFieldsConfig(dir, t.Config)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		desc := strings.TrimSpace(t.Description)
		if desc == "" {
			desc = name
		}
		p.Tools = append(p.Tools, Tool{
			Name:        name,
			Description: desc,
			Kind:        kind,
			Config:      cfg,
		})
	}
	return p, nil
}

func parseFieldsConfig(dir string, raw json.RawMessage) (Config, error) {
	cfg := Config{Depth: 0, Readers: map[string]string{}}
	if len(raw) == 0 {
		return cfg, nil
	}
	var shape fieldsConfigShape
	if err := json.Unmarshal(raw, &shape); err != nil {
		return Config{}, fmt.Errorf("config 不是对象")
	}
	for _, rel := range stringList(shape.Codec) {
		abs := rel
		if !filepath.IsAbs(rel) {
			abs = filepath.Join(dir, rel)
		}
		cfg.Codec = append(cfg.Codec, filepath.Clean(abs))
	}
	for ea, reader := range shape.Readers {
		canon, err := CanonicalEA(ea)
		if err != nil {
			return Config{}, fmt.Errorf("readers 地址无效: %s", ea)
		}
		name := strings.TrimSpace(reader)
		if name == "" {
			return Config{}, fmt.Errorf("readers %s 没有类型", ea)
		}
		cfg.Readers[canon] = name
	}
	if shape.Depth != nil {
		if *shape.Depth < 0 {
			return Config{}, fmt.Errorf("depth 不能为负")
		}
		cfg.Depth = *shape.Depth
	}
	return cfg, nil
}

func stringList(v any) []string {
	switch t := v.(type) {
	case nil:
		return nil
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
		return []string{t}
	case []any:
		var out []string
		for _, item := range t {
			s, ok := item.(string)
			if !ok || strings.TrimSpace(s) == "" {
				continue
			}
			out = append(out, s)
		}
		return out
	}
	return nil
}

// Lookup returns the tool with name.
func (p *Pack) Lookup(name string) (Tool, bool) {
	if p == nil {
		return Tool{}, false
	}
	for _, t := range p.Tools {
		if t.Name == name {
			return t, true
		}
	}
	return Tool{}, false
}

// Reserved reports a core MCP tool name. Kept here so pack load fails closed
// without importing mcp (mcp imports pack).
func Reserved(name string) bool {
	_, ok := reservedNames[name]
	return ok
}

var reservedNames = map[string]struct{}{
	"ida_list_targets":        {},
	"ida_open":                {},
	"ida_close_target":        {},
	"ida_save_target":         {},
	"ida_ping_target":         {},
	"ida_get_metadata":        {},
	"ida_start_auto_analysis": {},
	"ida_stop_auto_analysis":  {},
	"ida_list_functions":      {},
	"ida_get_function":        {},
	"ida_decompile":           {},
	"ida_disassemble":         {},
	"ida_xrefs":               {},
	"ida_callees":             {},
	"ida_calls":               {},
	"ida_lvars":               {},
	"ida_list_strings":        {},
	"ida_list_imports":        {},
	"ida_get_bytes":           {},
	"ida_search_text":         {},
	"ida_rename":              {},
	"ida_set_comment":         {},
	"ida_attach_pack":         {},
	"ida_detach_pack":         {},
}
