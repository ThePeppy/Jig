package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"jig/internal/pack"
)

func TestCoreToolSchemasRequireAddress(t *testing.T) {
	need := map[string]bool{
		"ida_get_function": true,
		"ida_decompile":    true,
		"ida_disassemble":  true,
		"ida_xrefs":        true,
		"ida_calls":        true,
		"ida_callees":      true,
		"ida_lvars":        true,
		"ida_get_bytes":    true,
		"ida_rename":       true,
	}
	seen := map[string]bool{}
	for _, tool := range coreTools() {
		seen[tool.name] = true
		if !need[tool.name] {
			continue
		}
		ok := false
		for _, r := range tool.required {
			if r == "address" {
				ok = true
			}
		}
		if !ok {
			t.Fatalf("%s missing required address", tool.name)
		}
		if _, ok := tool.properties["address"]; !ok {
			t.Fatalf("%s missing address property", tool.name)
		}
	}
	for name := range need {
		if !seen[name] {
			t.Fatalf("missing tool %s", name)
		}
	}
	if _, ok := workerToolMap()["ida_calls"]; !ok {
		t.Fatal("ida_calls not mapped")
	}
	if workerToolMap()["ida_calls"] != "analysis.calls" {
		t.Fatalf("ida_calls maps to %s", workerToolMap()["ida_calls"])
	}
}

func TestToolDefinitionsIncludeAttachedPack(t *testing.T) {
	dir := t.TempDir()
	codec := filepath.Join(dir, "codec.tsv")
	if err := os.WriteFile(codec, []byte("category\tpacket_id\tpacket_name\thandler_va\tfield_index\tdepth\treader\tsize\tcall_va\treader_va\nCMD\t1\tLOGIN\t0x1\t0\t0\tu8\t1\t0x2\t0x3\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	body := `{
  "id": "dnf90",
  "tools": [{"name":"ida_dnf90_packet_fields","kind":"fields_from_calls","config":{"codec":"codec.tsv"}}]
}`
	if err := os.WriteFile(filepath.Join(dir, "pack.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := pack.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	defs := toolDefinitionsFor([]*pack.Pack{p})
	found := false
	for _, d := range defs {
		if d["name"] == "ida_dnf90_packet_fields" {
			found = true
		}
		if d["name"] == "ida_decompile" {
			schema, _ := d["inputSchema"].(map[string]any)
			req, _ := schema["required"].([]string)
			ok := false
			for _, r := range req {
				if r == "address" {
					ok = true
				}
			}
			if !ok {
				t.Fatalf("ida_decompile schema %+v", d["inputSchema"])
			}
		}
	}
	if !found {
		t.Fatal("pack tool missing from tools/list")
	}
	if pack.Reserved("ida_lvars") != true {
		t.Fatal("ida_lvars should be reserved")
	}
}
