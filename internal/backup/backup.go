package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// SourceStatus is a snapshot of the most recent backup activity for a single
// source, suitable for rendering in an admin dashboard.
type SourceStatus struct {
	Source        string
	LastRun       time.Time
	LastSuccess   time.Time
	LastFailure   time.Time
	LastError     string
	LastDuration  time.Duration
	LastSize      int64
	RetainedCount int
	NextRun       time.Time
}

// Healthy reports whether the most recent run for this source succeeded.
// Returns false if the source has never run.
func (s SourceStatus) Healthy() bool {
	if s.LastRun.IsZero() {
		return false
	}
	return !s.LastSuccess.IsZero() && !s.LastSuccess.Before(s.LastFailure)
}

// Source represents a database that can be backed up.
type Source interface {
	Name() string
	Backup(ctx context.Context, destPath string) error
}

type Destination interface {
	Write(ctx context.Context, key string, srcPath string) error
	List(ctx context.Context, prefix string) ([]string, error)
	Delete(ctx context.Context, key string) error
}

type Config struct {
	ScheduleHour int // Hour (0-23) in UTC -- Default: 11 (11:00 UTC = 3:00 AM PST)
	Retain       int // Number of backups to keep  -- Default: 2
	Dest         Destination
}

type Service struct {
	config  Config
	sources []Source

	mu     sync.RWMutex
	status map[string]SourceStatus
}

func NewService(cfg Config) *Service {
	if cfg.Retain == 0 {
		cfg.Retain = 2
	}
	return &Service{
		config: cfg,
		status: make(map[string]SourceStatus),
	}
}

// Status returns a snapshot of the most recent run state for each registered
// source. Sources that have never run still appear with zero-valued time fields
// and a populated NextRun.
func (s *Service) Status() []SourceStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	next := nextOccurrence(time.Now().UTC(), s.config.ScheduleHour)
	out := make([]SourceStatus, 0, len(s.sources))
	for _, src := range s.sources {
		st, ok := s.status[src.Name()]
		if !ok {
			st = SourceStatus{Source: src.Name()}
		}
		st.NextRun = next
		out = append(out, st)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Source < out[j].Source })
	return out
}

func (s *Service) updateStatus(name string, mutate func(*SourceStatus)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.status[name]
	st.Source = name
	mutate(&st)
	s.status[name] = st
}

func (s *Service) AddSource(src Source) {
	s.sources = append(s.sources, src)
}

// Start runs an initial backup after a short delay, then schedules daily
// backups at the configured hour (UTC).
func (s *Service) Start(ctx context.Context) {
	go func() {
		// Short delay on startup to let the app stabilize, then run immediately.
		select {
		case <-time.After(1 * time.Minute):
		case <-ctx.Done():
			return
		}

		s.runAll(ctx)

		// Sleep until the next scheduled hour, then repeat daily.
		for {
			next := nextOccurrence(time.Now().UTC(), s.config.ScheduleHour)
			delay := time.Until(next)
			log.Debug().Time("next_backup", next).Str("delay", delay.String()).Msg("Scheduled next backup")

			select {
			case <-time.After(delay):
				s.runAll(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func nextOccurrence(now time.Time, hour int) time.Time {
	today := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.UTC)
	if today.After(now) {
		return today
	}
	return today.Add(24 * time.Hour)
}

func (s *Service) runAll(ctx context.Context) {
	for _, src := range s.sources {
		if err := s.backupSource(ctx, src); err != nil {
			log.Error().Err(err).Str("source", src.Name()).Msg("Backup failed")
		} else {
			log.Info().Str("source", src.Name()).Msg("Backup completed")
		}
	}
}

func (s *Service) backupSource(ctx context.Context, src Source) error {
	name := src.Name()
	start := time.Now().UTC()
	timestamp := start.Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.bak", name, timestamp)

	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, filename)
	defer func() { _ = os.Remove(tmpPath) }()

	if err := src.Backup(ctx, tmpPath); err != nil {
		s.recordFailure(name, start, fmt.Errorf("creating backup: %w", err))
		return fmt.Errorf("creating backup: %w", err)
	}

	if err := s.config.Dest.Write(ctx, filename, tmpPath); err != nil {
		s.recordFailure(name, start, fmt.Errorf("writing backup: %w", err))
		return fmt.Errorf("writing backup: %w", err)
	}

	// Capture size before pruning so the gauge reflects the run we just did.
	var size int64
	if info, err := os.Stat(tmpPath); err == nil {
		size = info.Size()
	}

	// Prune old backups.
	prefix := name + "-"
	keys, err := s.config.Dest.List(ctx, prefix)
	if err != nil {
		log.Warn().Err(err).Str("source", name).Msg("Failed to list backups for pruning")
		s.recordSuccess(name, start, size, 0)
		return nil // backup itself succeeded
	}
	if len(keys) > s.config.Retain {
		for _, key := range keys[s.config.Retain:] {
			if err := s.config.Dest.Delete(ctx, key); err != nil {
				log.Warn().Err(err).Str("key", key).Msg("Failed to delete old backup")
			} else {
				log.Info().Str("key", key).Msg("Pruned old backup")
			}
		}
		keys = keys[:s.config.Retain]
	}

	s.recordSuccess(name, start, size, len(keys))
	return nil
}

func (s *Service) recordSuccess(name string, start time.Time, size int64, retained int) {
	end := time.Now().UTC()
	duration := end.Sub(start)
	s.updateStatus(name, func(st *SourceStatus) {
		st.LastRun = end
		st.LastSuccess = end
		st.LastError = ""
		st.LastDuration = duration
		st.LastSize = size
		st.RetainedCount = retained
	})

	lastSuccessTimestamp.WithLabelValues(name).Set(float64(end.Unix()))
	lastDurationSeconds.WithLabelValues(name).Set(duration.Seconds())
	lastSizeBytes.WithLabelValues(name).Set(float64(size))
	retainedCount.WithLabelValues(name).Set(float64(retained))
	runsTotal.WithLabelValues(name, "success").Inc()
}

func (s *Service) recordFailure(name string, start time.Time, runErr error) {
	end := time.Now().UTC()
	duration := end.Sub(start)
	s.updateStatus(name, func(st *SourceStatus) {
		st.LastRun = end
		st.LastFailure = end
		st.LastError = runErr.Error()
		st.LastDuration = duration
	})

	lastFailureTimestamp.WithLabelValues(name).Set(float64(end.Unix()))
	lastDurationSeconds.WithLabelValues(name).Set(duration.Seconds())
	runsTotal.WithLabelValues(name, "failure").Inc()
}

type SQLiteSource struct {
	name string
	db   *sql.DB
}

func NewSQLiteSource(name string, db *sql.DB) *SQLiteSource {
	return &SQLiteSource{name: name, db: db}
}

func (s *SQLiteSource) Name() string { return s.name }

func (s *SQLiteSource) Backup(ctx context.Context, destPath string) error {
	// VACUUM INTO refuses to write to an existing file; clear any stale
	// target left over from a previous interrupted run.
	if err := os.Remove(destPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing stale backup target: %w", err)
	}

	_, err := s.db.ExecContext(ctx, "VACUUM INTO ?", destPath)
	if err != nil {
		return fmt.Errorf("VACUUM INTO: %w", err)
	}

	return nil
}

type LocalDestination struct {
	dir string
}

func NewLocalDestination(dir string) (*LocalDestination, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating backup directory: %w", err)
	}
	return &LocalDestination{dir: dir}, nil
}

func (d *LocalDestination) Write(_ context.Context, key string, srcPath string) error {
	destPath := filepath.Join(d.dir, key)

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer func() { _ = dst.Close() }()

	if _, err := dst.ReadFrom(src); err != nil {
		_ = os.Remove(destPath)
		return err
	}
	return dst.Close()
}

func (d *LocalDestination) List(_ context.Context, prefix string) ([]string, error) {
	entries, err := os.ReadDir(d.dir)
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasPrefix(e.Name(), prefix) {
			keys = append(keys, e.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	return keys, nil
}

func (d *LocalDestination) Delete(_ context.Context, key string) error {
	return os.Remove(filepath.Join(d.dir, key))
}
