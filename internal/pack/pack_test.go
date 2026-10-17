package pack

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDirectoryAndRejectsCoreName(t *testing.T) {
	dir := t.TempDir()
	codec := filepath.Join(dir, "codec.tsv")
	if err := os.WriteFile(codec, []byte(sampleCodec), 0o644); err != nil {
		t.Fatal(err)
	}
	body := `{
  "id": "dnf90",
  "name": "90DNF",
  "tools": [{
    "name": "ida_dnf90_packet_fields",
    "description": "fields",
    "kind": "fields_from_calls",
    "config": {
      "codec": "codec.tsv",
      "readers": {"0x02e00c70": "u8", "0x2e00d50": "u32"},
      "depth": 0
    }
  }]
}`
	if err := os.WriteFile(filepath.Join(dir, "pack.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "dnf90" || len(p.Tools) != 1 {
		t.Fatalf("%+v", p)
	}
	if p.Tools[0].Config.Readers["0x2e00c70"] != "u8" {
		t.Fatalf("readers %+v", p.Tools[0].Config.Readers)
	}
	if p.Tools[0].Config.Codec[0] != codec {
		t.Fatalf("codec %s", p.Tools[0].Config.Codec[0])
	}

	bad := `{
  "id": "x",
  "tools": [{"name":"ida_decompile","kind":"fields_from_calls"}]
}`
	if err := os.WriteFile(filepath.Join(dir, "pack.json"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil {
		t.Fatal("expected reserved name")
	}
}

func TestLoadSiblingDNF90Pack(t *testing.T) {
	path := filepath.Join("..", "..", "..", "90DNF", ".jig", "pack.json")
	if _, err := os.Stat(path); err != nil {
		t.Skip("90DNF pack not beside Jig")
	}
	p, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "dnf90" || p.Tools[0].Name != "ida_dnf90_packet_fields" {
		t.Fatalf("%+v", p)
	}
	if len(p.Tools[0].Config.Codec) != 2 {
		t.Fatalf("codec %+v", p.Tools[0].Config.Codec)
	}
	for _, c := range p.Tools[0].Config.Codec {
		if _, err := os.Stat(c); err != nil {
			t.Fatalf("codec missing %s", c)
		}
	}
}

func TestFieldsFromCallsNamesFromLvar(t *testing.T) {
	dir := t.TempDir()
	codec := filepath.Join(dir, "codec.tsv")
	if err := os.WriteFile(codec, []byte(sampleCodec), 0o644); err != nil {
		t.Fatal(err)
	}
	fields, err := FieldsFromCalls(Config{
		Codec:   []string{codec},
		Depth:   0,
		Readers: map[string]string{"0x2e00c70": "u8", "0x2e00d50": "u32"},
	}, "CMD", 1, "0x16e69c0", []Call{
		{CallSite: "0x016e6ad8", Callee: "0x02e00c70", AssignedTo: "channel", AssignedNice: true},
		{CallSite: "0x016e6b2f", Callee: "0x02e00c70", AssignedTo: "v5", AssignedNice: false},
		{CallSite: "0xdead", Callee: "0x02e00d50", AssignedTo: "other", AssignedNice: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 {
		t.Fatalf("len %d", len(fields))
	}
	if fields[0].Name != "channel" || fields[0].Evidence != "named" {
		t.Fatalf("field0 %+v", fields[0])
	}
	if fields[1].Name != "v5" || fields[1].Evidence != "inferred" {
		t.Fatalf("field1 %+v", fields[1])
	}
	if fields[0].PacketName != "ENUM_CMDPACKET_LOGIN" {
		t.Fatalf("name %s", fields[0].PacketName)
	}
}

func TestHandlerFromCodec(t *testing.T) {
	dir := t.TempDir()
	codec := filepath.Join(dir, "codec.tsv")
	if err := os.WriteFile(codec, []byte(sampleCodec), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := HandlerFromCodec(Config{Codec: []string{codec}}, "CMD", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "0x16e69c0" {
		t.Fatalf("%s", got)
	}
}

func TestCallSitesSkipsDuplicates(t *testing.T) {
	dir := t.TempDir()
	codec := filepath.Join(dir, "codec.tsv")
	if err := os.WriteFile(codec, []byte(sampleCodec), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := CallSites(Config{Codec: []string{codec}}, "CMD", 1, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "0x16e6ad8" || got[1] != "0x16e6b2f" {
		t.Fatalf("%v", got)
	}
}

func TestCanonicalEA(t *testing.T) {
	a, err := CanonicalEA("0x016e69c0")
	if err != nil || a != "0x16e69c0" {
		t.Fatalf("%s %v", a, err)
	}
}

const sampleCodec = `category	packet_id	packet_id_hex	packet_name	handler_va	field_index	depth	reader	size	call_va	reader_va
CMD	1	0x1	ENUM_CMDPACKET_LOGIN	0x016e69c0	0	0	u8	1	0x016e6ad8	0x02e00c70
CMD	1	0x1	ENUM_CMDPACKET_LOGIN	0x016e69c0	1	0	u8	1	0x016e6b2f	0x02e00c70
CMD	1	0x1	ENUM_CMDPACKET_LOGIN	0x016e69c0	2	1	u32	4	0x0208f6c0	0x02e00d50
`
