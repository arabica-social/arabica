package backup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Source represents a database that can be backed up.
type Source interface {
	// Name returns a short identifier used in backup filenames (e.g. "feed-index").
	Name() string
	// Backup creates a consistent snapshot at the given file path.
	Backup(ctx context.Context, destPath string) error
}

// Destination controls where backups are stored and how old ones are pruned.
type Destination interface {
	// Write stores the backup from srcPath. The key is the relative filename.
	Write(ctx context.Context, key string, srcPath string) error
	// List returns keys matching the given prefix, newest first.
	List(ctx context.Context, prefix string) ([]string, error)
	// Delete removes a backup by key.
	Delete(ctx context.Context, key string) error
}

// Config holds backup service configuration.
type Config struct {
	// ScheduleHour is the hour (0-23) in UTC to run the daily backup.
	// Default: 11 (11:00 UTC = 3:00 AM PST).
	ScheduleHour int
	// Retain is the number of backups to keep per source (default: 2).
	Retain int
	// Destination for storing backups.
	Dest Destination
}

// Service manages periodic backups of registered sources.
type Service struct {
	config  Config
	sources []Source
}

// NewService creates a backup service with the given config.
func NewService(cfg Config) *Service {
	if cfg.Retain == 0 {
		cfg.Retain = 2
	}
	return &Service{
		config: cfg,
	}
}

// AddSource registers a database source for backup.
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

// nextOccurrence returns the next time.Time at the given hour (UTC).
// If the hour hasn't passed today, it returns today at that hour;
// otherwise it returns tomorrow at that hour.
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
	timestamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.bak", src.Name(), timestamp)

	// Write to a temp file first, then hand off to the destination.
	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, filename)
	defer os.Remove(tmpPath)

	if err := src.Backup(ctx, tmpPath); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	if err := s.config.Dest.Write(ctx, filename, tmpPath); err != nil {
		return fmt.Errorf("writing backup: %w", err)
	}

	// Prune old backups.
	prefix := src.Name() + "-"
	keys, err := s.config.Dest.List(ctx, prefix)
	if err != nil {
		log.Warn().Err(err).Str("source", src.Name()).Msg("Failed to list backups for pruning")
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
	}

	return nil
}

// SQLiteSource backs up a SQLite database using VACUUM INTO.
type SQLiteSource struct {
	name string
	db   *sql.DB
}

// NewSQLiteSource creates a backup source for a SQLite database.
func NewSQLiteSource(name string, db *sql.DB) *SQLiteSource {
	return &SQLiteSource{name: name, db: db}
}

func (s *SQLiteSource) Name() string { return s.name }

func (s *SQLiteSource) Backup(ctx context.Context, destPath string) error {
	_, err := s.db.ExecContext(ctx, "VACUUM INTO ?", destPath)
	if err != nil {
		return fmt.Errorf("VACUUM INTO: %w", err)
	}
	return nil
}

// LocalDestination stores backups in a local directory.
type LocalDestination struct {
	dir string
}

// NewLocalDestination creates a destination that writes to a local directory.
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
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Use io.Copy via ReadFrom for efficiency.
	if _, err := dst.ReadFrom(src); err != nil {
		os.Remove(destPath)
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

	// Sort descending (newest first) — timestamp in filename makes this work.
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))
	return keys, nil
}

func (d *LocalDestination) Delete(_ context.Context, key string) error {
	return os.Remove(filepath.Join(d.dir, key))
}
