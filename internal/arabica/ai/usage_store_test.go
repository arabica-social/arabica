package ai

import (
	"context"
	"database/sql"
	_ "modernc.org/sqlite"
	"path/filepath"
	"testing"
	"time"
)

func newTestDailyUsageStore(t *testing.T) (*DailyUsageStore, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "usage.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	store, err := NewDailyUsageStore(db)
	if err != nil {
		t.Fatal(err)
	}
	return store, db
}

func TestConsumeEnforcesLimitAtomically(t *testing.T) {
	store, _ := newTestDailyUsageStore(t)
	ctx := context.Background()
	var total int64
	for i, chars := range []int64{40, 40, 20} {
		total += chars
		used, allowed, err := store.Consume(ctx, "did:plc:user", "2026-01-01", chars, 100)
		if err != nil || !allowed || used != total {
			t.Fatalf("charge %d: used=%d allowed=%v err=%v", i, used, allowed, err)
		}
	}
	used, allowed, err := store.Consume(ctx, "did:plc:user", "2026-01-01", 1, 100)
	if err != nil || allowed || used != 100 {
		t.Fatalf("over-limit charge: used=%d allowed=%v err=%v", used, allowed, err)
	}
	if used, err := store.Used(ctx, "did:plc:user", "2026-01-01"); err != nil || used != 100 {
		t.Fatalf("refused charge was recorded: %d %v", used, err)
	}
	if used, allowed, err := store.Consume(ctx, "did:plc:other", "2026-01-01", 100, 100); err != nil || !allowed || used != 100 {
		t.Fatalf("per-account budget shared: used=%d allowed=%v err=%v", used, allowed, err)
	}
	if used, allowed, err := store.Consume(ctx, "did:plc:user", "2026-01-02", 100, 100); err != nil || !allowed || used != 100 {
		t.Fatalf("per-day budget shared: used=%d allowed=%v err=%v", used, allowed, err)
	}
	if _, allowed, err := store.Consume(ctx, "", "2026-01-01", 1, 100); err == nil || allowed {
		t.Fatal("empty account accepted")
	}
	if _, allowed, err := store.Consume(ctx, "did:plc:user", "2026-01-01", -1, 100); err == nil || allowed {
		t.Fatal("negative charge accepted")
	}
}

func TestZeroChargeTouchesNothing(t *testing.T) {
	store, _ := newTestDailyUsageStore(t)
	ctx := context.Background()
	used, allowed, err := store.Consume(ctx, "did:plc:user", "2026-01-01", 0, 100)
	if err != nil || !allowed || used != 0 {
		t.Fatalf("zero charge: used=%d allowed=%v err=%v", used, allowed, err)
	}
}

func TestPurgeBeforeDropsOldDaysOnly(t *testing.T) {
	store, _ := newTestDailyUsageStore(t)
	ctx := context.Background()
	for _, day := range []string{"2026-01-01", "2026-01-02", "2026-01-03"} {
		if _, _, err := store.Consume(ctx, "did:plc:user", day, 10, 100); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.PurgeBefore(ctx, "2026-01-02"); err != nil {
		t.Fatal(err)
	}
	for day, want := range map[string]int64{"2026-01-01": 0, "2026-01-02": 10, "2026-01-03": 10} {
		used, err := store.Used(ctx, "did:plc:user", day)
		if err != nil || used != want {
			t.Fatalf("%s: used=%d want=%d err=%v", day, used, want, err)
		}
	}
}

func TestUsageDayIsUTC(t *testing.T) {
	at := time.Date(2026, time.March, 5, 23, 59, 0, 0, time.FixedZone("late", -5*60*60))
	if got := UsageDay(at); got != "2026-03-06" {
		t.Fatalf("UsageDay = %s", got)
	}
}

func TestConsumeReportsDatabaseFailure(t *testing.T) {
	store, db := newTestDailyUsageStore(t)
	db.Close()
	if _, allowed, err := store.Consume(context.Background(), "did:plc:user", "2026-01-01", 1, 100); err == nil || allowed {
		t.Fatal("database failure hidden")
	}
}
