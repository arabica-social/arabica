# Future Architecture Notes

## Witness Cache Pattern (from Paul Frazee)

### Concept
Many Atmosphere backends should start with a local "witness cache" of repositories.

**What is a Witness Cache?**
- A copy of repository records
- Plus a timestamp of when the record was indexed (the "witness time")
- Must be kept/preserved

### Key Benefits

#### 1. Local Replay Capability
With local replay, you can:
- Add new tables or indexes to your backend
- Quickly backfill the data from local cache
- **Without** having to backfill from the network (which is slow)

#### 2. Fast Iteration
- Change your data model
- Add new indexes
- Reprocess data quickly
- No network bottleneck

### Technology Recommendations

#### Good Candidates:

**RocksDB or other LSMs (Log-Structured Merge-Trees)**
- Excellent write throughput
- Good for high-volume ingestion
- Used by many distributed systems

**ClickHouse**
- Good compression ratio
- Analytics-focused
- Fast columnar queries

**DuckDB**
- Good compression ratio
- Embedded database
- Great for analytics
- Easy integration

### Implementation Timeline

**Phase 1-4 (Current):** Direct PDS queries (no cache)
- Simple implementation
- Works for single-user or small datasets
- Good for understanding the data model

**Phase 6-7 (AppView + Indexing):** Add witness cache
- When building firehose consumer
- When indexing multiple users
- When cross-user queries become important

**Phase 10 (Performance & Scale):** Optimize cache
- Choose between RocksDB/ClickHouse/DuckDB based on usage patterns
- Implement replay/backfill mechanisms
- Add monitoring and metrics

### Architecture Sketch

```
┌─────────────────────────────────────────────┐
│           Firehose Consumer                 │
│  (Subscribes to repo commit events)         │
└────────────────┬────────────────────────────┘
                 │
                 ↓ Write with witness time
┌─────────────────────────────────────────────┐
│          Witness Cache                      │
│  (RocksDB / ClickHouse / DuckDB)            │
│                                             │
│  Record: {                                  │
│    did: "did:plc:abc123"                    │
│    collection: "com.arabica.brew"           │
│    rkey: "3jxy..."                          │
│    record: {...}                            │
│    witness_time: "2024-01-04T20:00:00Z"     │
│    cid: "baf..."                            │
│  }                                          │
└────────────────┬────────────────────────────┘
                 │
                 ↓ Query/Transform
┌─────────────────────────────────────────────┐
│         Application Database                │
│  (PostgreSQL / SQLite)                      │
│                                             │
│  - Denormalized views                       │
│  - Application-specific indexes             │
│  - Can be rebuilt from witness cache        │
└─────────────────────────────────────────────┘
```

### Key Principles

1. **Witness cache is immutable append-only log**
   - Never delete records
   - Keep deletion markers
   - Keep all historical data

2. **Application DB is derived state**
   - Can be dropped and rebuilt
   - Optimized for queries
   - Contains denormalized data

3. **Replay = rebuild application DB from cache**
   - Fast because it's local
   - No network calls
   - Consistent state

### Migration Strategy

**Current (Phase 1-4):**
```
User's PDS → XRPC → Your App → SQLite (for UI)
```

**Future (Phase 6+):**
```
Firehose → Witness Cache → Application DB → Your App
     ↓
User's PDS (for writes)
```

### Open Questions for Later

1. **Retention policy?**
   - Keep all historical data forever?
   - Compress old data?
   - Archive after N days?

2. **Consistency guarantees?**
   - Eventually consistent OK?
   - Need stronger guarantees?

3. **Witness time vs repo commit time?**
   - What if we receive events out of order?
   - How to handle backfills?

4. **Compression strategy?**
   - Compress old records?
   - Trade-off: space vs replay speed

---

**Reference:** Paul Frazee's advice on Atmosphere backend architecture
**Status:** Future enhancement (Phase 6+)
**Priority:** Not needed for personal tracker v1
