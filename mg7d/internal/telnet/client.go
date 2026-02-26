package telnet

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// Config holds telnet client settings.
type Config struct {
	Host            string
	Port            int
	Password        string
	RateLimitPerSec float64
	CommandTimeout  time.Duration
	ReconnectMin    time.Duration
	ReconnectMax    time.Duration
	CircuitBreakAfter int  // consecutive send failures before opening breaker
	CircuitBreakWindow time.Duration
}

const (
	DefaultCommandTimeout   = 10 * time.Second
	DefaultReconnectMin    = 2 * time.Second
	DefaultReconnectMax    = 60 * time.Second
	DefaultCircuitBreakAfter = 3
	DefaultCircuitBreakWindow = 30 * time.Second
)

// Client maintains one persistent telnet connection with rate limiting and safe reconnect.
type Client struct {
	cfg    Config
	addr   string
	conn   net.Conn
	mu     sync.Mutex
	closed bool

	// token bucket for rate limiting
	tokens   float64
	lastTick time.Time
	tickMu   sync.Mutex

	// circuit breaker
	failCount   int
	breakerOpen bool
	breakerAt   time.Time

	// command queue: bounded
	commands chan commandReq
	done     chan struct{}
}

type commandReq struct {
	cmd    Command
	result chan error
}

// NewClient creates a telnet client. Call Run to start the connection and send loop.
func NewClient(cfg Config) *Client {
	if cfg.RateLimitPerSec <= 0 {
		cfg.RateLimitPerSec = 2.0
	}
	if cfg.CommandTimeout == 0 {
		cfg.CommandTimeout = DefaultCommandTimeout
	}
	if cfg.ReconnectMin == 0 {
		cfg.ReconnectMin = DefaultReconnectMin
	}
	if cfg.ReconnectMax == 0 {
		cfg.ReconnectMax = DefaultReconnectMax
	}
	if cfg.CircuitBreakAfter <= 0 {
		cfg.CircuitBreakAfter = DefaultCircuitBreakAfter
	}
	if cfg.CircuitBreakWindow == 0 {
		cfg.CircuitBreakWindow = DefaultCircuitBreakWindow
	}
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	return &Client{
		cfg:      cfg,
		addr:     addr,
		tokens:   cfg.RateLimitPerSec,
		lastTick: time.Now(),
		commands: make(chan commandReq, 64),
		done:     make(chan struct{}),
	}
}

// Send enqueues a command. Returns when sent or context/timeout. Returns error if queue full or client closed.
func (c *Client) Send(ctx context.Context, cmd Command) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("telnet: client closed")
	}
	c.mu.Unlock()
	req := commandReq{cmd: cmd, result: make(chan error, 1)}
	select {
	case c.commands <- req:
		select {
		case err := <-req.result:
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	default:
		return fmt.Errorf("telnet: command queue full")
	}
}

// Run maintains the connection, drain loop, and send loop. Exits when ctx is cancelled.
func (c *Client) Run(ctx context.Context) {
	backoff := c.cfg.ReconnectMin
	for {
		select {
		case <-ctx.Done():
			c.closeConn()
			close(c.done)
			return
		default:
		}

		conn, err := c.connect(ctx)
		if err != nil {
			if ctx.Err() != nil {
				close(c.done)
				return
			}
			time.Sleep(backoff)
			if backoff < c.cfg.ReconnectMax {
				backoff *= 2
				if backoff > c.cfg.ReconnectMax {
					backoff = c.cfg.ReconnectMax
				}
			}
			continue
		}

		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()
		backoff = c.cfg.ReconnectMin
		c.failCount = 0
		c.breakerOpen = false

		// Authenticate
		if c.cfg.Password != "" {
			_ = c.writeLine(conn, c.cfg.Password)
		}

		// Drain output in background so server doesn't block us
		go c.drain(conn)

		c.sendLoop(ctx, conn)
		c.closeConn()
	}
}

func (c *Client) connect(ctx context.Context) (net.Conn, error) {
	dialer := net.Dialer{}
	return dialer.DialContext(ctx, "tcp", c.addr)
}

func (c *Client) closeConn() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *Client) drain(conn net.Conn) {
	r := bufio.NewReader(conn)
	buf := make([]byte, 4096)
	for {
		_, err := r.Read(buf)
		if err != nil {
			return
		}
	}
}

// takeToken blocks until one token is available or ctx done. Returns true if token acquired.
func (c *Client) takeToken(ctx context.Context) bool {
	c.tickMu.Lock()
	defer c.tickMu.Unlock()
	for c.tokens < 1 {
		now := time.Now()
		elapsed := now.Sub(c.lastTick).Seconds()
		c.tokens += elapsed * c.cfg.RateLimitPerSec
		c.lastTick = now
		if c.tokens > c.cfg.RateLimitPerSec {
			c.tokens = c.cfg.RateLimitPerSec
		}
		if c.tokens >= 1 {
			break
		}
		// wait a bit
		c.tickMu.Unlock()
		select {
		case <-ctx.Done():
			c.tickMu.Lock()
			return false
		case <-time.After(100 * time.Millisecond):
		}
		c.tickMu.Lock()
	}
	c.tokens--
	return true
}

func (c *Client) sendLoop(ctx context.Context, conn net.Conn) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-c.commands:
			if !ok {
				return
			}
			// Circuit breaker
			c.mu.Lock()
			if c.breakerOpen {
				if time.Since(c.breakerAt) < c.cfg.CircuitBreakWindow {
					c.mu.Unlock()
					req.result <- fmt.Errorf("circuit breaker open")
					continue
				}
				c.breakerOpen = false
				c.failCount = 0
			}
			c.mu.Unlock()

			if !c.takeToken(ctx) {
				req.result <- ctx.Err()
				continue
			}

			done := make(chan error, 1)
			go func() {
				done <- c.sendOne(conn, req.cmd)
			}()
			var err error
			select {
			case err = <-done:
			case <-time.After(c.cfg.CommandTimeout):
				err = fmt.Errorf("command timeout")
			case <-ctx.Done():
				err = ctx.Err()
			}
			if err != nil {
				c.mu.Lock()
				c.failCount++
				if c.failCount >= c.cfg.CircuitBreakAfter {
					c.breakerOpen = true
					c.breakerAt = time.Now()
				}
				c.mu.Unlock()
				c.closeConn()
				req.result <- err
				return // exit send loop to reconnect
			}
			req.result <- err
		}
	}
}

func (c *Client) sendOne(conn net.Conn, cmd Command) error {
	c.mu.Lock()
	if c.conn != conn {
		c.mu.Unlock()
		return fmt.Errorf("connection closed")
	}
	c.mu.Unlock()
	return c.writeLine(conn, cmd.Raw)
}

func (c *Client) writeLine(conn net.Conn, line string) error {
	_, err := conn.Write([]byte(line + "\r\n"))
	return err
}

// Close marks the client closed. Does not close the command channel so callers do not panic.
// Cancel the context passed to Run() to stop the connection; then Close() returns after Run exits.
func (c *Client) Close() {
	c.mu.Lock()
	c.closed = true
	c.mu.Unlock()
	<-c.done
}
