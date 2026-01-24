#!/bin/bash
# Diagnostic script to check feed database status

set -e

DB_PATH="${ARABICA_FEED_INDEX_PATH:-$HOME/.local/share/arabica/feed-index.db}"

echo "=== Feed Database Diagnostics ==="
echo "Database path: $DB_PATH"
echo ""

if [ ! -f "$DB_PATH" ]; then
    echo "ERROR: Database file does not exist at $DB_PATH"
    exit 1
fi

echo "Database file size: $(du -h "$DB_PATH" | cut -f1)"
echo "Last modified: $(stat -c %y "$DB_PATH" 2>/dev/null || stat -f "%Sm" "$DB_PATH")"
echo ""

# Create a simple Go program to inspect the database
cat > /tmp/inspect-feed-db.go << 'EOF'
package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"time"

	bolt "go.etcd.io/bbolt"
)

type IndexedRecord struct {
	URI        string          `json:"uri"`
	DID        string          `json:"did"`
	Collection string          `json:"collection"`
	RKey       string          `json:"rkey"`
	Record     json.RawMessage `json:"record"`
	CID        string          `json:"cid"`
	IndexedAt  time.Time       `json:"indexed_at"`
	CreatedAt  time.Time       `json:"created_at"`
}

func main() {
	dbPath := os.Args[1]
	
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{ReadOnly: true, Timeout: 5 * time.Second})
	if err != nil {
		fmt.Printf("ERROR: Failed to open database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	err = db.View(func(tx *bolt.Tx) error {
		// Check buckets
		records := tx.Bucket([]byte("records"))
		byTime := tx.Bucket([]byte("by_time"))
		meta := tx.Bucket([]byte("meta"))
		knownDIDs := tx.Bucket([]byte("known_dids"))
		backfilled := tx.Bucket([]byte("backfilled"))

		if records == nil {
			fmt.Println("ERROR: 'records' bucket does not exist")
			return nil
		}

		recordCount := records.Stats().KeyN
		fmt.Printf("Total records: %d\n", recordCount)

		if byTime != nil {
			timeIndexCount := byTime.Stats().KeyN
			fmt.Printf("Time index entries: %d\n", timeIndexCount)
		}

		if knownDIDs != nil {
			didCount := knownDIDs.Stats().KeyN
			fmt.Printf("Known DIDs: %d\n", didCount)
			knownDIDs.ForEach(func(k, v []byte) error {
				fmt.Printf("  - %s\n", string(k))
				return nil
			})
		}

		if backfilled != nil {
			backfilledCount := backfilled.Stats().KeyN
			fmt.Printf("Backfilled DIDs: %d\n", backfilledCount)
		}

		// Check cursor
		if meta != nil {
			cursorBytes := meta.Get([]byte("cursor"))
			if cursorBytes != nil && len(cursorBytes) == 8 {
				cursor := int64(binary.BigEndian.Uint64(cursorBytes))
				cursorTime := time.UnixMicro(cursor)
				fmt.Printf("\nCursor position: %d (%s)\n", cursor, cursorTime.Format(time.RFC3339))
			} else {
				fmt.Println("\nNo cursor found in database")
			}
		}

		// Get first 5 and last 5 records by time
		if byTime != nil && records != nil {
			fmt.Println("\n=== First 5 records (oldest) ===")
			c := byTime.Cursor()
			count := 0
			for k, _ := c.First(); k != nil && count < 5; k, _ = c.Next() {
				uri := extractURI(k)
				if record := getRecord(records, uri); record != nil {
					fmt.Printf("%s - %s - %s\n", record.CreatedAt.Format("2006-01-02 15:04:05"), record.Collection, uri)
				}
				count++
			}

			fmt.Println("\n=== Last 5 records (newest with inverted timestamps) ===")
			c = byTime.Cursor()
			count = 0
			for k, _ := c.Last(); k != nil && count < 5; k, _ = c.Prev() {
				uri := extractURI(k)
				if record := getRecord(records, uri); record != nil {
					fmt.Printf("%s - %s - %s\n", record.CreatedAt.Format("2006-01-02 15:04:05"), record.Collection, uri)
				}
				count++
			}
		}

		return nil
	})

	if err != nil {
		fmt.Printf("ERROR: %v\n", err)
		os.Exit(1)
	}
}

func extractURI(key []byte) string {
	if len(key) < 10 {
		return ""
	}
	return string(key[9:])
}

func getRecord(bucket *bolt.Bucket, uri string) *IndexedRecord {
	data := bucket.Get([]byte(uri))
	if data == nil {
		return nil
	}
	var record IndexedRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return nil
	}
	return &record
}
EOF

cd "$(dirname "$0")/.."
go run /tmp/inspect-feed-db.go "$DB_PATH"

rm -f /tmp/inspect-feed-db.go
