package logtail

import (
	"bufio"
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Options configures the tailer.
type Options struct {
	PollInterval   time.Duration // when no fs events; default 1s
	MaxLineBytes   int           // max line size before truncating; default 64k
	FromBeginning  bool          // if true, read from start (for tests)
}

const (
	defaultPollInterval = time.Second
	defaultMaxLineBytes = 64 * 1024
)

// Tailer follows a log file and emits complete lines on a channel.
// Rotation-safe: handles copytruncate and rename+recreate. Partial-line safe.
type Tailer struct {
	path    string
	opts    Options
	linesCh chan string
	mu      sync.Mutex
	closed  bool
}

// NewTailer creates a tailer for path. Lines() must be consumed to avoid blocking.
func NewTailer(path string, opts Options) (*Tailer, error) {
	if opts.PollInterval == 0 {
		opts.PollInterval = defaultPollInterval
	}
	if opts.MaxLineBytes <= 0 {
		opts.MaxLineBytes = defaultMaxLineBytes
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return &Tailer{
		path:    abs,
		opts:    opts,
		linesCh: make(chan string, 256),
	}, nil
}

// Lines returns the channel of complete log lines. Closed when tailer stops.
func (t *Tailer) Lines() <-chan string {
	return t.linesCh
}

// Run runs the tailer until ctx is cancelled. Survives rotation and temporary missing file.
func (t *Tailer) Run(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	dir := filepath.Dir(t.path)
	if err := watcher.Add(dir); err != nil {
		return err
	}

	var (
		backoff   = time.Millisecond * 100
		maxBackoff = time.Second * 5
		pollTicker *time.Ticker
	)
	if t.opts.PollInterval > 0 {
		pollTicker = time.NewTicker(t.opts.PollInterval)
		defer pollTicker.Stop()
	}

	for {
		// Open and read until EOF; then wait for events or poll.
		readErr := t.followFile(ctx, watcher, pollTicker, &backoff, maxBackoff)
		if ctx.Err() != nil {
			break
		}
		// File missing or rotated; backoff and retry open.
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
		select {
		case <-ctx.Done():
			break
		case <-time.After(backoff):
			// retry
		}
	}

	t.mu.Lock()
	if !t.closed {
		t.closed = true
		close(t.linesCh)
	}
	t.mu.Unlock()
	return ctx.Err()
}

// followFile opens path, reads new content, handles rotation. Returns when file is gone or ctx done.
func (t *Tailer) followFile(ctx context.Context, watcher *fsnotify.Watcher, pollTicker *time.Ticker, backoff *time.Duration, maxBackoff time.Duration) error {
	f, err := os.Open(t.path)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	startOffset := int64(0)
	if !t.opts.FromBeginning {
		startOffset = info.Size()
	}
	if _, err := f.Seek(startOffset, io.SeekStart); err != nil {
		return err
	}

	*backoff = time.Millisecond * 100
	origInode := inodeOf(info)
	reader := bufio.NewReaderSize(f, 32*1024)
	var partial []byte
	var tickCh <-chan time.Time
	if pollTicker != nil {
		tickCh = pollTicker.C
	}

	readMore := func() (bool, error) {
		buf := make([]byte, 4096)
		n, err := reader.Read(buf)
		if n > 0 {
			partial = append(partial, buf[:n]...)
			for {
				idx := 0
				for idx < len(partial) && partial[idx] != '\n' {
					idx++
				}
				if idx < len(partial) {
					line := partial[:idx]
					partial = partial[idx+1:]
					if len(line) > t.opts.MaxLineBytes {
						line = line[:t.opts.MaxLineBytes]
					}
					select {
					case t.linesCh <- string(line):
					case <-ctx.Done():
						return false, ctx.Err()
					}
					continue
				}
				if len(partial) > t.opts.MaxLineBytes {
					select {
					case t.linesCh <- string(partial[:t.opts.MaxLineBytes]):
					case <-ctx.Done():
						return false, ctx.Err()
					}
					partial = partial[t.opts.MaxLineBytes:]
				}
				break
			}
		}
		if err == io.EOF {
			return true, nil
		}
		return true, err
	}

	for {
		advanced, err := readMore()
		if err != nil {
			return err
		}
		if !advanced {
			return nil
		}

		// Check rotation: same path but different inode or truncated
		curInfo, statErr := os.Stat(t.path)
		if statErr != nil {
			// file missing
			return statErr
		}
		if inodeOf(curInfo) != origInode || curInfo.Size() < info.Size() {
			// rotated (new file or copytruncate)
			return nil
		}
		if curInfo.Size() > info.Size() {
			info = curInfo
		}

		// Wait for more data: fsnotify or poll
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-watcher.Events:
			if filepath.Clean(e.Name) != filepath.Clean(t.path) {
				continue
			}
			if e.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				// more data or new file at path
				continue
			}
			if e.Op&fsnotify.Remove != 0 {
				return nil
			}
			// Chmod etc.: fall through to poll
		case <-watcher.Errors:
			// ignore
		case <-tickCh:
			// periodic poll when ticker is configured
		}
	}
}

func inodeOf(info os.FileInfo) uint64 {
	// Use ModTime+Size as a simple "identity" on Windows where inode may not exist.
	// On Unix we could use syscall.Stat_t.Ino. For cross-platform we use size + modtime.
	return uint64(info.Size())<<32 | uint64(info.ModTime().UnixNano()&0xffffffff)
}
