package ai

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "modernc.org/sqlite"
	"path/filepath"
	"testing"
)

func TestAccessStorePersistenceAndBounds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "access.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, err := NewAccessStore(db)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := store.Grant(ctx, "bad", "did:plc:admin"); !errors.Is(err, ErrInvalidDID) {
		t.Fatal(err)
	}
	for i := 0; i < MaxAccessEntries; i++ {
		if err := store.Grant(ctx, fmt.Sprintf("did:plc:user%d", i), "did:plc:admin"); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Grant(ctx, "did:plc:overflow", "did:plc:admin"); !errors.Is(err, ErrAccessFull) {
		t.Fatal(err)
	}
	entries, err := store.List(ctx)
	if err != nil || len(entries) != MaxAccessEntries {
		t.Fatalf("list %d %v", len(entries), err)
	}
	if err := store.Grant(ctx, "did:plc:user0", "did:plc:other"); err != nil {
		t.Fatal(err)
	}
	entries, _ = store.List(ctx)
	for _, entry := range entries {
		if entry.GrantedBy != "did:plc:admin" || entry.CreatedAt == "" {
			t.Fatalf("entry %+v", entry)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store, err = NewAccessStore(db)
	if err != nil {
		t.Fatal(err)
	}
	if allowed, err := store.Allows(ctx, "did:plc:user0"); err != nil || !allowed {
		t.Fatalf("persisted %v %v", allowed, err)
	}
	for i := 0; i < 2; i++ {
		if err := store.Revoke(ctx, "did:plc:user0"); err != nil {
			t.Fatal(err)
		}
	}
	if allowed, err := store.Allows(ctx, "did:plc:user0"); err != nil || allowed {
		t.Fatalf("revoked %v %v", allowed, err)
	}
	if err := store.Grant(ctx, "did:plc:replacement", "did:plc:admin"); err != nil {
		t.Fatal(err)
	}
	db.Close()
	if _, err := store.Allows(ctx, "did:plc:replacement"); err == nil {
		t.Fatal("database failure hidden")
	}
}
