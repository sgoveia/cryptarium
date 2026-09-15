package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// FindingIDInput is the stable inputs that determine a finding's ID.
// See DESIGN.md §9: sha256(detector | ruleID | relPath | line | primitive | canonicalParams),
// truncated to 16 hex characters. No counters, timestamps, or map-iteration order.
type FindingIDInput struct {
	Detector   string
	RuleID     string
	RelPath    string
	Line       int
	Primitive  string
	Parameters map[string]any
}

// FindingID returns a deterministic 16-hex-character ID for a finding.
// RelPath is normalized to forward slashes. Parameters are canonicalized by
// sorting keys and encoding values as stable JSON where needed.
func FindingID(in FindingIDInput) string {
	path := strings.ReplaceAll(in.RelPath, "\\", "/")
	payload := strings.Join([]string{
		in.Detector,
		in.RuleID,
		path,
		strconv.Itoa(in.Line),
		in.Primitive,
		canonicalParams(in.Parameters),
	}, "|")
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])[:16]
}

// canonicalParams produces a stable string for a parameter map.
// Keys are sorted; values use a deterministic encoding.
func canonicalParams(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(canonicalValue(params[k]))
	}
	return b.String()
}

func canonicalValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return x
	case bool:
		return strconv.FormatBool(x)
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		// JSON numbers decode as float64; format without scientific notation when possible.
		return strconv.FormatFloat(x, 'f', -1, 64)
	case json.Number:
		return x.String()
	default:
		raw, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprintf("%v", x)
		}
		return string(raw)
	}
}
