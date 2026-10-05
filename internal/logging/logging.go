package logging

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const (
	maxSize    = 10 * 1024 * 1024 // 10 MiB
	maxBackups = 5
)

// Setup configures the default logger to write JSON logs to a rotating file.
func Setup() (func() error, error) {
	path, err := logPath()
	if err != nil {
		return nil, err
	}

	writer, err := newRotatingFile(path, maxSize, maxBackups)
	if err != nil {
		return nil, err
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(writer, nil)))
	return writer.Close, nil
}

func logPath() (string, error) {
	stateHome := os.Getenv("XDG_STATE_HOME")
	if stateHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("getting home directory: %w", err)
		}
		stateHome = filepath.Join(home, ".local", "state")
	}

	dir := filepath.Join(stateHome, "bag")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("creating log directory: %w", err)
	}
	return filepath.Join(dir, "bag.log"), nil
}

type rotatingFile struct {
	path       string
	maxSize    int64
	maxBackups int

	mu   sync.Mutex
	file *os.File
}

func newRotatingFile(path string, maxSize int64, maxBackups int) (*rotatingFile, error) {
	file, err := openLogFile(path)
	if err != nil {
		return nil, err
	}

	return &rotatingFile{
		path:       path,
		maxSize:    maxSize,
		maxBackups: maxBackups,
		file:       file,
	}, nil
}

func openLogFile(path string) (*os.File, error) {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	return file, nil
}

func (r *rotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	info, err := r.file.Stat()
	if err != nil {
		return 0, fmt.Errorf("getting log file size: %w", err)
	}
	if info.Size() > 0 && info.Size()+int64(len(p)) > r.maxSize {
		if err := r.rotate(); err != nil {
			return 0, err
		}
	}

	return r.file.Write(p)
}

func (r *rotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}

func (r *rotatingFile) rotate() error {
	if err := r.file.Close(); err != nil {
		return fmt.Errorf("closing log file for rotation: %w", err)
	}

	backup := r.path + "." + time.Now().UTC().Format("20060102T150405.000000000Z")
	if err := os.Rename(r.path, backup); err != nil {
		return fmt.Errorf("rotating log file: %w", err)
	}

	file, err := openLogFile(r.path)
	if err != nil {
		return err
	}
	r.file = file

	if err := r.pruneBackups(); err != nil {
		return err
	}
	return nil
}

func (r *rotatingFile) pruneBackups() error {
	dir := filepath.Dir(r.path)
	prefix := filepath.Base(r.path) + "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("reading log directory: %w", err)
	}

	backups := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && len(entry.Name()) > len(prefix) && entry.Name()[:len(prefix)] == prefix {
			backups = append(backups, entry.Name())
		}
	}
	sort.Strings(backups)

	for _, backup := range backups[:max(0, len(backups)-r.maxBackups)] {
		if err := os.Remove(filepath.Join(dir, backup)); err != nil {
			return fmt.Errorf("removing old log backup: %w", err)
		}
	}
	return nil
}
