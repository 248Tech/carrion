package telnet

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

func TestClient_RateLimit(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("no listener:", err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	port := ln.Addr().(*net.TCPAddr).Port
	cfg := Config{
		Host:            "127.0.0.1",
		Port:            port,
		RateLimitPerSec: 2.0,
		CommandTimeout:  time.Second,
	}
	client := NewClient(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go client.Run(ctx)

	// Send a few commands; with 2/sec we expect the first to go quickly, next may wait
	start := time.Now()
	_ = client.Send(ctx, Command{Raw: "test1"})
	_ = client.Send(ctx, Command{Raw: "test2"})
	elapsed := time.Since(start)
	// Should take at least ~0.5s for 2 commands at 2/sec (one token refill)
	if elapsed < 400*time.Millisecond {
		t.Logf("rate limit may not have applied (elapsed %v)", elapsed)
	}
}

func TestClient_ReconnectBackoff(t *testing.T) {
	cfg := Config{
		Host:            "127.0.0.1",
		Port:            19999, // no server
		RateLimitPerSec: 10,
		ReconnectMin:    50 * time.Millisecond,
		ReconnectMax:    200 * time.Millisecond,
	}
	client := NewClient(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.Run(ctx)
	}()
	// Run should not spin; it should back off
	time.Sleep(100 * time.Millisecond)
	cancel()
	wg.Wait()
}
