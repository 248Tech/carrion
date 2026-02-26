package logtail

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTailerPartialLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	if err := os.WriteFile(path, []byte("line1\nline2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{FromBeginning: true, PollInterval: time.Millisecond * 50}
	tailer, err := NewTailer(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var got []string
	done := make(chan struct{})
	go func() {
		for line := range tailer.Lines() {
			got = append(got, line)
			if len(got) >= 2 {
				break
			}
		}
		close(done)
	}()

	go func() { _ = tailer.Run(ctx) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		cancel()
		t.Fatal("timeout waiting for lines")
	}
	cancel()

	if len(got) < 2 {
		t.Fatalf("expected at least 2 lines, got %d: %v", len(got), got)
	}
	if got[0] != "line1" || got[1] != "line2" {
		t.Errorf("got %v", got)
	}
}

func TestTailerPartialLineAcrossReads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	// Start with incomplete line (no newline)
	if err := os.WriteFile(path, []byte("incomplete"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{FromBeginning: true, PollInterval: time.Millisecond * 20}
	tailer, err := NewTailer(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var got []string
	go func() {
		for line := range tailer.Lines() {
			got = append(got, line)
		}
	}()

	go func() { _ = tailer.Run(ctx) }()

	// Append the rest of the line and a newline
	time.Sleep(50 * time.Millisecond)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("line\n")
	_ = f.Close()

	time.Sleep(200 * time.Millisecond)
	cancel()

	if len(got) != 1 {
		t.Fatalf("expected 1 complete line, got %d: %v", len(got), got)
	}
	if got[0] != "incompleteline" {
		t.Errorf("expected 'incompleteline', got %q", got[0])
	}
}

func TestTailerRotationSimulation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "game.log")
	if err := os.WriteFile(path, []byte("before_rotation\n"), 0644); err != nil {
		t.Fatal(err)
	}

	opts := Options{FromBeginning: true, PollInterval: time.Millisecond * 30}
	tailer, err := NewTailer(path, opts)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var got []string
	go func() {
		for line := range tailer.Lines() {
			got = append(got, line)
		}
	}()

	go func() { _ = tailer.Run(ctx) }()

	time.Sleep(80 * time.Millisecond)
	// Simulate copytruncate: truncate file then append new content
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString("after_rotation\n")
	_ = f.Close()

	time.Sleep(300 * time.Millisecond)
	cancel()

	// We should see "before_rotation" and then after reopen "after_rotation"
	before := false
	after := false
	for _, s := range got {
		if strings.TrimSpace(s) == "before_rotation" {
			before = true
		}
		if strings.TrimSpace(s) == "after_rotation" {
			after = true
		}
	}
	if !before {
		t.Errorf("expected to see before_rotation, got: %v", got)
	}
	if !after {
		t.Errorf("expected to see after_rotation after rotation, got: %v", got)
	}
}
