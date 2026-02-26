package parser

import (
	"testing"
	"time"

	"github.com/mg7d/mg7d/internal/state"
)

func TestParseTimeLine_NotTimeLine(t *testing.T) {
	snap, ok, err := ParseTimeLine("Something else")
	if ok || err != nil {
		t.Fatalf("expected !ok, no err; got ok=%v err=%v", ok, err)
	}
	if snap.FPS != 0 {
		t.Errorf("expected zero snap: %+v", snap)
	}
}

func TestParseTimeLine_TimeLine(t *testing.T) {
	line := "Time: 123.45 FPS: 30.5 Heap: 512.2 RSS: 600 Chunks: 100 Ply: 2 Zom: 50 Ent: 200 CO: 2"
	snap, ok, err := ParseTimeLine(line)
	if !ok || err != nil {
		t.Fatalf("expected ok, no err; got ok=%v err=%v", ok, err)
	}
	if snap.FPS != 30.5 {
		t.Errorf("FPS: got %v", snap.FPS)
	}
	if snap.HeapMB != 512.2 {
		t.Errorf("HeapMB: got %v", snap.HeapMB)
	}
	if snap.RSSMB != 600 {
		t.Errorf("RSSMB: got %v", snap.RSSMB)
	}
	if snap.Chunks != 100 {
		t.Errorf("Chunks: got %v", snap.Chunks)
	}
	if snap.Players != 2 {
		t.Errorf("Players: got %v", snap.Players)
	}
	if snap.Zombies != 50 {
		t.Errorf("Zombies: got %v", snap.Zombies)
	}
	if snap.EntitiesTotal != 200 {
		t.Errorf("EntitiesTotal: got %v", snap.EntitiesTotal)
	}
	if snap.CO != 2 {
		t.Errorf("CO: got %v", snap.CO)
	}
}

func TestParseTimeLine_OrderVariation(t *testing.T) {
	line := "Time: 0 FPS: 60 Chunks: 10 Heap: 100 RSS: 120"
	snap, ok, err := ParseTimeLine(line)
	if !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if snap.FPS != 60 || snap.Chunks != 10 || snap.HeapMB != 100 || snap.RSSMB != 120 {
		t.Errorf("%+v", snap)
	}
}

func TestParseTimeLine_MissingTokens(t *testing.T) {
	line := "Time: 0 FPS: 20 Heap: 50"
	snap, ok, err := ParseTimeLine(line)
	if !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if snap.FPS != 20 && snap.HeapMB != 50 {
		t.Errorf("%+v", snap)
	}
	if snap.EntitiesActive != -1 {
		t.Errorf("EntitiesActive should be -1 when missing, got %d", snap.EntitiesActive)
	}
	if !snap.CGoMissing {
		t.Errorf("CGoMissing should be true when CGo not in line")
	}
}

func TestParseTimeLine_Realistic(t *testing.T) {
	// Example 7DTD-style line with timestamp
	line := "Time: 2024-01-15 14:30:00 FPS: 45.2 Heap: 2048.5 RSS: 2500 Chunks: 500 Ply: 4 Zom: 120 Ent: 1500 CO: 4"
	snap, ok, err := ParseTimeLine(line)
	if !ok || err != nil {
		t.Fatalf("ok=%v err=%v", ok, err)
	}
	if snap.FPS != 45.2 || snap.HeapMB != 2048.5 || snap.RSSMB != 2500 {
		t.Errorf("%+v", snap)
	}
	// Timestamp may be parsed
	if !snap.Timestamp.IsZero() && snap.Timestamp.Year() != 2024 {
		t.Logf("timestamp parsed: %v", snap.Timestamp)
	}
}

func TestSnapshot_ParsedAtSet(t *testing.T) {
	before := time.Now()
	snap, ok, _ := ParseTimeLine("Time: x FPS: 1")
	after := time.Now()
	if !ok {
		t.Fatal("expected ok")
	}
	if snap.ParsedAt.Before(before) || snap.ParsedAt.After(after) {
		t.Errorf("ParsedAt %v not in [%v, %v]", snap.ParsedAt, before, after)
	}
}

func TestParseKeyValuePairs(t *testing.T) {
	m := parseKeyValuePairs("FPS: 60 Heap: 100 MB")
	if m["FPS"] != "60" {
		t.Errorf("FPS: %q", m["FPS"])
	}
	if m["Heap"] != "100 MB" {
		t.Errorf("Heap: %q", m["Heap"])
	}
}

// Ensure we don't return (zero, false, err) for non-Time lines
func TestParseTimeLine_NoErrorForNonTime(t *testing.T) {
	_, ok, err := ParseTimeLine("2024/01/01 12:00:00 Some other log")
	if ok || err != nil {
		t.Errorf("non-Time line: ok=%v err=%v", ok, err)
	}
}

func TestParseMB(t *testing.T) {
	f, err := parseMB("100.5")
	if err != nil || f != 100.5 {
		t.Errorf("parseMB(100.5): %v %v", f, err)
	}
	f, err = parseMB("200 MB")
	if err != nil || f != 200 {
		t.Errorf("parseMB(200 MB): %v %v", f, err)
	}
}

// Ensure state.Snapshot is used (compile check)
var _ = state.Snapshot{}

func TestParseTimeLine_ReplayFixture(t *testing.T) {
	// Replay first and last line from testdata/replay_fps.log
	lines := []string{
		"Time: 0 FPS: 50 Heap: 100 RSS: 200 Chunks: 10 Ply: 0 Zom: 20 Ent: 100 CO: 0",
		"Time: 3 FPS: 20 Heap: 110 RSS: 220 Chunks: 12 Ply: 1 Zom: 25 Ent: 120 CO: 1",
		"Time: 12 FPS: 50 Heap: 100 RSS: 200 Chunks: 10 Ply: 0 Zom: 20 Ent: 100 CO: 0",
	}
	for i, line := range lines {
		snap, ok, err := ParseTimeLine(line)
		if !ok || err != nil {
			t.Fatalf("line %d: ok=%v err=%v", i, ok, err)
		}
		if snap.FPS <= 0 {
			t.Errorf("line %d: FPS should be > 0, got %v", i, snap.FPS)
		}
		if snap.Chunks <= 0 {
			t.Errorf("line %d: Chunks should be > 0, got %v", i, snap.Chunks)
		}
	}
}
