package atproto

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/metrics"
)

// rawRecord is what fetchAllRecords returns: a parsed record map plus the
// URI, rkey, and CID it was fetched from. Per-entity wrappers convert the
// map to their typed model (arabica.RecordToBean, arabica.RecordToRoaster, etc.).
type rawRecord struct {
	URI    string
	RKey   string
	CID    string
	Record map[string]any
}

// FetchRecord is the exported entry point for callers outside this package
// that need a witness-then-PDS record read for an arbitrary NSID (e.g.
// per-app view handlers that don't have typed Get* wrappers). It just
// forwards to fetchRecord with a simplified return shape.
func (s *AtprotoStore) FetchRecord(ctx context.Context, nsid, rkey string) (record map[string]any, uri, cid string, err error) {
	rec, u, c, hit, _, err := s.fetchRecord(ctx, nsid, rkey)
	if err != nil {
		return nil, "", "", err
	}
	if !hit {
		return nil, "", "", fmt.Errorf("record not found: %s/%s", nsid, rkey)
	}
	return rec, u, c, nil
}

// fetchRecord returns the record at nsid/rkey, hitting the witness cache
// first and falling back to PDS. The "hit" return is true when the record
// was found anywhere; uri/cid are set in either case. fromWitness is true
// when the witness cache served the read (callers like brew use this to
// decide whether ref resolution should also stay in the witness cache).
// An error is returned only on transport or parse failure.
func (s *AtprotoStore) fetchRecord(ctx context.Context, nsid, rkey string) (record map[string]any, uri, cid string, hit, fromWitness bool, err error) {
	if wr := s.getFromWitness(ctx, nsid, rkey); wr != nil {
		m, perr := witnessRecordToMap(wr)
		if perr == nil {
			metrics.WitnessCacheHitsTotal.WithLabelValues(metricLabelFor(nsid)).Inc()
			return m, wr.URI, wr.CID, true, true, nil
		}
		log.Warn().Err(perr).Str("nsid", nsid).Str("rkey", rkey).Msg("witness: parse failed, falling back to PDS")
	} else {
		metrics.WitnessCacheMissesTotal.WithLabelValues(metricLabelFor(nsid)).Inc()
	}

	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return nil, "", "", false, false, fmt.Errorf("get atp client: %w", err)
	}
	rec, err := atpClient.GetRecord(ctx, nsid, rkey)
	if err != nil {
		return nil, "", "", false, false, fmt.Errorf("get record %s/%s: %w", nsid, rkey, err)
	}
	return rec.Value, rec.URI, rec.CID, true, false, nil
}

// fetchAllRecords returns every record of the given NSID, hitting the
// witness cache first and falling back to PDS pagination. The session
// cache is not consulted here — typed wrappers are responsible for the
// session-cache short-circuit because they hold the typed slice.
func (s *AtprotoStore) fetchAllRecords(ctx context.Context, nsid string) ([]rawRecord, error) {
	if wRecords := s.listFromWitness(ctx, nsid); wRecords != nil {
		metrics.WitnessCacheHitsTotal.WithLabelValues(metricLabelFor(nsid)).Inc()
		out := make([]rawRecord, 0, len(wRecords))
		for _, wr := range wRecords {
			m, err := witnessRecordToMap(wr)
			if err != nil {
				log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: parse failed in list, skipping")
				continue
			}
			out = append(out, rawRecord{URI: wr.URI, RKey: wr.RKey, CID: wr.CID, Record: m})
		}
		return out, nil
	}
	metrics.WitnessCacheMissesTotal.WithLabelValues(metricLabelFor(nsid)).Inc()

	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get atp client: %w", err)
	}
	records, err := atpClient.ListAllRecords(ctx, nsid)
	if err != nil {
		return nil, fmt.Errorf("list records %s: %w", nsid, err)
	}
	out := make([]rawRecord, 0, len(records))
	for _, rec := range records {
		atURI, err := syntax.ParseATURI(rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("list: invalid AT-URI, skipping")
			continue
		}
		out = append(out, rawRecord{URI: rec.URI, RKey: atURI.RecordKey().String(), CID: rec.CID, Record: rec.Value})
	}
	return out, nil
}

// fetchPaginatedRecords returns a page of records of the given NSID, ordered by
// created_at descending. Uses the paginated witness cache when limit > 0 and the
// cache is available and not dirty. Falls back to PDS ListAllRecords when the
// witness cache misses (returning all records — callers should slice in Go for
// the PDS fallback case, which only happens before firehose indexing).
func (s *AtprotoStore) fetchPaginatedRecords(ctx context.Context, nsid string, offset, limit int) ([]rawRecord, error) {
	if limit > 0 && s.witnessCache != nil {
		if userCache := s.cache.Get(s.sessionID); !userCache.IsDirty(nsid) {
			wRecords, err := s.witnessCache.ListWitnessRecordsPaginated(ctx, s.did.String(), nsid, offset, limit)
			if err != nil {
				log.Debug().Err(err).Str("collection", nsid).Msg("witness: ListWitnessRecordsPaginated error")
			} else {
				// Paginated query returned results (or empty at end of page) — use them.
				// An empty slice means end of results; nil means cache miss (shouldn't happen
				// when the non-paginated ListWitnessRecords succeeds, but we handle it gracefully).
				if wRecords != nil {
					metrics.WitnessCacheHitsTotal.WithLabelValues(metricLabelFor(nsid)).Inc()
					out := make([]rawRecord, 0, len(wRecords))
					for _, wr := range wRecords {
						m, err := witnessRecordToMap(wr)
						if err != nil {
							log.Warn().Err(err).Str("uri", wr.URI).Msg("witness: parse failed in paginated list, skipping")
							continue
						}
						out = append(out, rawRecord{URI: wr.URI, RKey: wr.RKey, CID: wr.CID, Record: m})
					}
					return out, nil
				}
			}
		}
	}
	metrics.WitnessCacheMissesTotal.WithLabelValues(metricLabelFor(nsid)).Inc()
	// Fall back to fetching all records from PDS
	return s.fetchAllRecords(ctx, nsid)
}

// putRecord creates or updates a record at nsid/rkey. If rkey is empty,
// a new record is created and the PDS-assigned rkey is returned along
// with the CID. For updates (rkey non-empty), the underlying PDS API
// does not return a CID, so cid will be "". The witness cache is updated
// write-through and the session cache is invalidated for the NSID.
func (s *AtprotoStore) putRecord(ctx context.Context, nsid, rkey string, record any) (resultRKey, cid string, err error) {
	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return "", "", fmt.Errorf("get atp client: %w", err)
	}

	if rkey == "" {
		newURI, newCID, err := atpClient.CreateRecord(ctx, nsid, record)
		if err != nil {
			return "", "", fmt.Errorf("create record %s: %w", nsid, err)
		}
		atURI, err := syntax.ParseATURI(newURI)
		if err != nil {
			return "", "", fmt.Errorf("parse created URI %q: %w", newURI, err)
		}
		newRKey := atURI.RecordKey().String()
		s.writeThroughWitness(nsid, newRKey, newCID, record)
		s.cache.InvalidateRecords(s.sessionID, nsid)
		return newRKey, newCID, nil
	}

	if _, _, err := atpClient.PutRecord(ctx, nsid, rkey, record); err != nil {
		return "", "", fmt.Errorf("put record %s/%s: %w", nsid, rkey, err)
	}
	// PutRecord does not return a CID; pass empty string to writeThroughWitness,
	// matching the behavior of pre-refactor per-entity Update methods.
	s.writeThroughWitness(nsid, rkey, "", record)
	s.cache.InvalidateRecords(s.sessionID, nsid)
	return rkey, "", nil
}

// removeRecord deletes from PDS, witness, and invalidates session cache.
func (s *AtprotoStore) removeRecord(ctx context.Context, nsid, rkey string) error {
	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return fmt.Errorf("get atp client: %w", err)
	}
	if err := atpClient.DeleteRecord(ctx, nsid, rkey); err != nil {
		return fmt.Errorf("delete record %s/%s: %w", nsid, rkey, err)
	}
	s.deleteFromWitness(nsid, rkey)
	s.cache.InvalidateRecords(s.sessionID, nsid)
	return nil
}

// metricLabelFor returns the short metric label for an NSID. Centralizes
// the mapping that was previously hardcoded in each typed CRUD method.
// Unknown NSIDs return "unknown" so metric collection never panics on a
// new entity that's not yet been added here.
func metricLabelFor(nsid string) string {
	switch nsid {
	case arabica.NSIDBean:
		return "bean"
	case arabica.NSIDBrew:
		return "brew"
	case arabica.NSIDBrewer:
		return "brewer"
	case arabica.NSIDGrinder:
		return "grinder"
	case arabica.NSIDRecipe:
		return "recipe"
	case arabica.NSIDRoaster:
		return "roaster"
	default:
		return "unknown"
	}
}
