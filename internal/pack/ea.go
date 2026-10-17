package pack

import (
	"fmt"
	"strconv"
	"strings"
)

// CanonicalEA turns 0x16e69c0 / 0x016e69c0 / 24009152 into "0x16e69c0".
func CanonicalEA(v string) (string, error) {
	n, err := ParseEA(v)
	if err != nil {
		return "", err
	}
	return FormatEA(n), nil
}

// ParseEA accepts hex with 0x, bare hex via sub_, or decimal.
func ParseEA(v string) (uint64, error) {
	text := strings.TrimSpace(v)
	if text == "" {
		return 0, fmt.Errorf("empty address")
	}
	if strings.HasPrefix(strings.ToLower(text), "sub_") {
		text = "0x" + text[4:]
	}
	n, err := strconv.ParseUint(text, 0, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid address %q", v)
	}
	return n, nil
}

// FormatEA is lowercase hex with 0x and no leading zeros beyond a single 0.
func FormatEA(n uint64) string {
	return "0x" + strconv.FormatUint(n, 16)
}
