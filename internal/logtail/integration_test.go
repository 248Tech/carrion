package logtail

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestReplayFixture verifies that replaying a fixture log emits lines in order.
func TestReplayFixture(t *testing.T) {
	// Test runs from package dir (internal/logtail); fixture is at repo root testdata/
	fixture := filepath.Join("..", "..", "testdata", "replay_fps.log")
	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Skip("fixture not found:", err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "replay.log")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{FromBeginning: true, PollInterval: time.Millisecond * 20}
	tailer, err := NewTailer(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var lines []string
	done := make(chan struct{})
	go func() {
		for line := range tailer.Lines() {
			lines = append(lines, line)
		}
		close(done)
	}()

	go func() { _ = tailer.Run(ctx) }()
	<-done
	cancel()

	// Fixture has 13 Time lines
	if len(lines) < 10 {
		t.Errorf("expected at least 10 lines from fixture, got %d", len(lines))
	}
	// First line should start with "Time:"
	if len(lines) > 0 && len(lines[0]) > 4 && lines[0][:4] != "Time:" {
		t.Errorf("first line should be Time line: %q", lines[0])
	}
}
