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
}

func NewService(cfg Config) *Service {
	if cfg.Retain == 0 {
		cfg.Retain = 2
	}
	return &Service{config: cfg}
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
	timestamp := time.Now().UTC().Format("20060102-150405")
	filename := fmt.Sprintf("%s-%s.bak", src.Name(), timestamp)

	tmpDir := os.TempDir()
	tmpPath := filepath.Join(tmpDir, filename)
	defer func() { _ = os.Remove(tmpPath) }()

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
