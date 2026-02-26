package parser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mg7d/mg7d/internal/state"
)

// ParseTimeLine parses a 7DTD "Time:" status line into a Snapshot.
// Returns (snap, true, nil) when the line is a Time line; (zero, false, nil) when not;
// (zero, false, err) on parse error for a line that looked like a Time line.
// Timestamp: if the line contains a parseable time, use it; otherwise use time.Now() (monotonic at parse time).
func ParseTimeLine(line string) (state.Snapshot, bool, error) {
	line = strings.TrimSpace(line)
	if !strings.HasPrefix(line, "Time:") {
		return state.Snapshot{}, false, nil
	}

	var snap state.Snapshot
	snap.ParsedAt = time.Now()
	snap.Timestamp = snap.ParsedAt
	snap.EntitiesActive = -1
	snap.CGo = 0
	snap.CGoMissing = true

	// Tokenize: "Key: value" pairs; value runs until next " Key:" or EOL.
	rest := strings.TrimSpace(strings.TrimPrefix(line, "Time:"))
	pairs := parseKeyValuePairs(rest)
	for key, val := range pairs {
		switch strings.ToLower(key) {
		case "time":
			if t, err := parseTimeVal(val); err == nil {
				snap.Timestamp = t
			}
		case "fps":
			if f, err := strconv.ParseFloat(strings.TrimSpace(val), 64); err == nil {
				snap.FPS = f
			}
		case "heap":
			if f, err := parseMB(val); err == nil {
				snap.HeapMB = f
			}
		case "rss":
			if f, err := parseMB(val); err == nil {
				snap.RSSMB = f
			}
		case "chunks":
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				snap.Chunks = n
			}
		case "cgo":
			snap.CGoMissing = false
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				snap.CGo = n
			}
		case "ply", "players":
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				snap.Players = n
			}
		case "zom", "zombies":
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				snap.Zombies = n
			}
		case "ent", "entities":
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				snap.EntitiesTotal = n
			}
		case "ent_active", "entities_active":
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				snap.EntitiesActive = n
			}
		case "co", "connections":
			if n, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
				snap.CO = n
			}
		}
	}

	return snap, true, nil
}

// parseKeyValuePairs splits "val0 Key1: val1 Key2: val2" where values can contain spaces.
// Leading value (before first " Word:") is stored as "Time". Keys are words ending with ':'.
func parseKeyValuePairs(s string) map[string]string {
	out := make(map[string]string)
	s = strings.TrimSpace(s)
	// Optional leading value before first " Word:" (e.g. "123.45 FPS: 30.5" -> Time=123.45)
	if idx := firstKeyStart(s); idx > 0 {
		out["Time"] = strings.TrimSpace(s[:idx])
		s = strings.TrimSpace(s[idx:])
	}
	for s != "" {
		// Find "Word:" (key)
		i := 0
		for i < len(s) && s[i] != ' ' && s[i] != '\t' && s[i] != ':' {
			i++
		}
		if i >= len(s) || s[i] != ':' {
			break
		}
		key := s[:i+1] // "Word:"
		s = strings.TrimSpace(s[i+1:])
		// Value is everything until next " Word:"
		valEnd := len(s)
		for j := 0; j < len(s); j++ {
			if s[j] != ' ' && s[j] != '\t' {
				continue
			}
			rest := strings.TrimSpace(s[j:])
			if len(rest) > 0 {
				k := 0
				for k < len(rest) && rest[k] != ' ' && rest[k] != '\t' && rest[k] != ':' {
					k++
				}
				if k < len(rest) && rest[k] == ':' {
					valEnd = j
					break
				}
			}
		}
		val := strings.TrimSpace(s[:valEnd])
		out[strings.TrimSuffix(key, ":")] = val
		s = strings.TrimSpace(s[valEnd:])
	}
	return out
}

// firstKeyStart returns the index of the first " Word:" (space + word + colon).
func firstKeyStart(s string) int {
	for j := 0; j < len(s); j++ {
		if s[j] != ' ' && s[j] != '\t' {
			continue
		}
		rest := strings.TrimSpace(s[j:])
		if len(rest) > 0 {
			k := 0
			for k < len(rest) && rest[k] != ' ' && rest[k] != '\t' && rest[k] != ':' {
				k++
			}
			if k < len(rest) && rest[k] == ':' {
				return j
			}
		}
	}
	return -1
}

func parseMB(s string) (float64, error) {
	s = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(s), "mb"))
	return strconv.ParseFloat(s, 64)
}

func parseTimeVal(s string) (time.Time, error) {
	// Try common formats
	for _, layout := range []string{
		time.RFC3339,
		"2006-01-02 15:04:05",
		"01/02/2006 15:04:05",
	} {
		if t, err := time.Parse(layout, strings.TrimSpace(s)); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unknown time format: %s", s)
}
