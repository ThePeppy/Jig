package pack

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// Call is one Hex-Rays call site from analysis.calls.
type Call struct {
	CallSite     string
	Callee       string
	CalleeName   string
	AssignedTo   string
	AssignedNice bool
}

// Field is one payload slot with a name when the decompiler supplied one.
type Field struct {
	Index      int    `json:"index"`
	Reader     string `json:"reader"`
	Size       int    `json:"size,omitempty"`
	CallSite   string `json:"callSite"`
	ReaderVA   string `json:"readerVa"`
	Lvar       string `json:"lvar,omitempty"`
	Name       string `json:"name"`
	Evidence   string `json:"evidence"` // named | inferred | codec
	Category   string `json:"category,omitempty"`
	PacketID   int    `json:"packetId,omitempty"`
	PacketName string `json:"packetName,omitempty"`
	HandlerVA  string `json:"handlerVa,omitempty"`
}

type codecRow struct {
	category   string
	packetID   int
	packetName string
	handler    string
	index      int
	depth      int
	reader     string
	size       int
	callVA     string
	readerVA   string
}

// HandlerFromCodec returns the handler VA for a packet filter.
func HandlerFromCodec(cfg Config, category string, packetID int, handler string) (string, error) {
	if handler != "" {
		return CanonicalEA(handler)
	}
	rows, err := loadCodec(cfg.Codec)
	if err != nil {
		return "", err
	}
	matched := filterCodec(rows, cfg.Depth, category, packetID, "")
	if len(matched) == 0 {
		return "", fmt.Errorf("codec 里没有这条包")
	}
	return matched[0].handler, nil
}

// CallSites returns unique field call VAs in codec order. Hex-Rays names
// come from the functions that contain these sites, not from handler_va
// (often a nearby error stub).
func CallSites(cfg Config, category string, packetID int, handler string) ([]string, error) {
	rows, err := loadCodec(cfg.Codec)
	if err != nil {
		return nil, err
	}
	handlerCanon := ""
	if handler != "" {
		handlerCanon, err = CanonicalEA(handler)
		if err != nil {
			return nil, fmt.Errorf("handler 地址无效")
		}
	}
	matched := filterCodec(rows, cfg.Depth, category, packetID, handlerCanon)
	seen := map[string]struct{}{}
	var sites []string
	for _, row := range matched {
		if row.callVA == "" {
			continue
		}
		if _, ok := seen[row.callVA]; ok {
			continue
		}
		seen[row.callVA] = struct{}{}
		sites = append(sites, row.callVA)
	}
	return sites, nil
}

// FieldsFromCalls builds the named field table. codec rows at Config.Depth
// (default 0) win order and reader type; call sites supply lvar names.
func FieldsFromCalls(cfg Config, category string, packetID int, handler string, calls []Call) ([]Field, error) {
	rows, err := loadCodec(cfg.Codec)
	if err != nil {
		return nil, err
	}
	handlerCanon := ""
	if handler != "" {
		handlerCanon, err = CanonicalEA(handler)
		if err != nil {
			return nil, fmt.Errorf("handler 地址无效")
		}
	}
	matched := filterCodec(rows, cfg.Depth, category, packetID, handlerCanon)
	if len(matched) == 0 && len(cfg.Codec) > 0 && (category != "" || packetID != 0 || handlerCanon != "") {
		return nil, fmt.Errorf("codec 里没有这条包")
	}

	bySite := map[string]Call{}
	var filtered []Call
	for _, c := range calls {
		site, err := CanonicalEA(c.CallSite)
		if err != nil {
			continue
		}
		callee, err := CanonicalEA(c.Callee)
		if err != nil {
			continue
		}
		c.CallSite = site
		c.Callee = callee
		if len(cfg.Readers) > 0 {
			if _, ok := cfg.Readers[callee]; !ok {
				continue
			}
		}
		bySite[site] = c
		filtered = append(filtered, c)
	}

	if len(matched) > 0 {
		out := make([]Field, 0, len(matched))
		for _, row := range matched {
			f := Field{
				Index:      row.index,
				Reader:     row.reader,
				Size:       row.size,
				CallSite:   row.callVA,
				ReaderVA:   row.readerVA,
				Category:   row.category,
				PacketID:   row.packetID,
				PacketName: row.packetName,
				HandlerVA:  row.handler,
				Evidence:   "codec",
			}
			if c, ok := bySite[row.callVA]; ok {
				applyName(&f, c)
			} else {
				f.Name = inferredName(row.reader, row.index)
			}
			if reader, ok := cfg.Readers[row.readerVA]; ok && f.Reader == "" {
				f.Reader = reader
			}
			out = append(out, f)
		}
		return out, nil
	}

	out := make([]Field, 0, len(filtered))
	for i, c := range filtered {
		reader := cfg.Readers[c.Callee]
		if reader == "" {
			reader = c.CalleeName
		}
		f := Field{
			Index:    i,
			Reader:   reader,
			CallSite: c.CallSite,
			ReaderVA: c.Callee,
			Evidence: "inferred",
		}
		applyName(&f, c)
		if f.Name == "" {
			f.Name = inferredName(reader, i)
		}
		out = append(out, f)
	}
	return out, nil
}

func applyName(f *Field, c Call) {
	f.Lvar = c.AssignedTo
	if c.AssignedTo == "" {
		if f.Name == "" {
			f.Name = inferredName(f.Reader, f.Index)
		}
		return
	}
	if c.AssignedNice && !dummyLvar(c.AssignedTo) {
		f.Name = c.AssignedTo
		f.Evidence = "named"
		return
	}
	if f.Name == "" {
		f.Name = c.AssignedTo
		if dummyLvar(c.AssignedTo) {
			f.Evidence = "inferred"
		} else {
			f.Evidence = "named"
		}
	}
}

func dummyLvar(name string) bool {
	if name == "" {
		return true
	}
	if name[0] != 'v' && name[0] != 'a' && name[0] != 'r' {
		return false
	}
	rest := name[1:]
	if rest == "" {
		return false
	}
	for _, c := range rest {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func inferredName(reader string, index int) string {
	base := reader
	if base == "" {
		base = "field"
	}
	return fmt.Sprintf("%s_%d", base, index)
}

func filterCodec(rows []codecRow, depth int, category string, packetID int, handler string) []codecRow {
	var out []codecRow
	cat := strings.ToUpper(strings.TrimSpace(category))
	for _, row := range rows {
		if row.depth != depth {
			continue
		}
		if cat != "" && strings.ToUpper(row.category) != cat {
			continue
		}
		if packetID != 0 && row.packetID != packetID {
			continue
		}
		if handler != "" && row.handler != handler {
			continue
		}
		if cat == "" && packetID == 0 && handler == "" {
			continue
		}
		out = append(out, row)
	}
	return out
}

func loadCodec(paths []string) ([]codecRow, error) {
	var all []codecRow
	for _, path := range paths {
		rows, err := readCodecTSV(path)
		if err != nil {
			return nil, err
		}
		all = append(all, rows...)
	}
	return all, nil
}

func readCodecTSV(path string) ([]codecRow, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打不开 codec %s", path)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = '\t'
	r.FieldsPerRecord = -1
	r.ReuseRecord = true
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("codec 是空的: %s", path)
	}
	col := map[string]int{}
	for i, name := range header {
		col[strings.TrimSpace(name)] = i
	}
	need := []string{"category", "packet_id", "packet_name", "handler_va", "field_index", "depth", "reader", "size", "call_va", "reader_va"}
	for _, n := range need {
		if _, ok := col[n]; !ok {
			return nil, fmt.Errorf("codec 缺列 %s: %s", n, path)
		}
	}
	var rows []codecRow
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("codec 读失败: %s", path)
		}
		handler, err := CanonicalEA(at(rec, col["handler_va"]))
		if err != nil {
			continue
		}
		callVA, err := CanonicalEA(at(rec, col["call_va"]))
		if err != nil {
			continue
		}
		readerVA, err := CanonicalEA(at(rec, col["reader_va"]))
		if err != nil {
			continue
		}
		id, _ := strconv.Atoi(at(rec, col["packet_id"]))
		index, _ := strconv.Atoi(at(rec, col["field_index"]))
		depth, _ := strconv.Atoi(at(rec, col["depth"]))
		size, _ := strconv.Atoi(at(rec, col["size"]))
		rows = append(rows, codecRow{
			category:   at(rec, col["category"]),
			packetID:   id,
			packetName: at(rec, col["packet_name"]),
			handler:    handler,
			index:      index,
			depth:      depth,
			reader:     at(rec, col["reader"]),
			size:       size,
			callVA:     callVA,
			readerVA:   readerVA,
		})
	}
	return rows, nil
}

func at(rec []string, i int) string {
	if i < 0 || i >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[i])
}
