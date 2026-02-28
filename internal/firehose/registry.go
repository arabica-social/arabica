package firehose

import (
	"time"

	"arabica/internal/feed"
)

// Ensure FeedIndex implements feed.PersistentStore at compile time.
var _ feed.PersistentStore = (*FeedIndex)(nil)

// Register adds a DID to the registered_dids table.
// This records users who have explicitly logged into Arabica.
func (idx *FeedIndex) Register(did string) error {
	_, err := idx.db.Exec(
		`INSERT OR IGNORE INTO registered_dids (did, registered_at) VALUES (?, ?)`,
		did, time.Now().Format(time.RFC3339),
	)
	return err
}

// Unregister removes a DID from the registered_dids table.
func (idx *FeedIndex) Unregister(did string) error {
	_, err := idx.db.Exec(`DELETE FROM registered_dids WHERE did = ?`, did)
	return err
}

// IsRegistered reports whether a DID is in the registered_dids table.
func (idx *FeedIndex) IsRegistered(did string) bool {
	var exists int
	_ = idx.db.QueryRow(`SELECT 1 FROM registered_dids WHERE did = ?`, did).Scan(&exists)
	return exists == 1
}

// List returns all registered DIDs.
func (idx *FeedIndex) List() []string {
	rows, err := idx.db.Query(`SELECT did FROM registered_dids`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var dids []string
	for rows.Next() {
		var did string
		if err := rows.Scan(&did); err != nil {
			continue
		}
		dids = append(dids, did)
	}
	return dids
}

// Count returns the number of registered users.
func (idx *FeedIndex) Count() int {
	var count int
	_ = idx.db.QueryRow(`SELECT COUNT(*) FROM registered_dids`).Scan(&count)
	return count
}

// SeedRegisteredDIDs inserts DIDs into registered_dids, ignoring duplicates.
// Used for one-time migration from the legacy BoltDB feed_registry.
func (idx *FeedIndex) SeedRegisteredDIDs(dids []string) (int, error) {
	if len(dids) == 0 {
		return 0, nil
	}
	var added int
	for _, did := range dids {
		res, err := idx.db.Exec(
			`INSERT OR IGNORE INTO registered_dids (did, registered_at) VALUES (?, ?)`,
			did, time.Now().Format(time.RFC3339),
		)
		if err != nil {
			return added, err
		}
		n, _ := res.RowsAffected()
		added += int(n)
	}
	return added, nil
}
