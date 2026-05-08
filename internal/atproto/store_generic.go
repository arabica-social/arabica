package atproto

import (
	"context"
	"fmt"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
	"tangled.org/arabica.social/arabica/internal/metrics"
)

// rawRecord is what fetchAllRecords returns: a parsed record map plus the
// URI, rkey, and CID it was fetched from. Per-entity wrappers convert the
// map to their typed model (RecordToBean, RecordToRoaster, etc.).
type rawRecord struct {
	URI    string
	RKey   string
	CID    string
	Record map[string]any
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

	output, err := s.client.GetRecord(ctx, s.did, s.sessionID, &GetRecordInput{
		Collection: nsid,
		RKey:       rkey,
	})
	if err != nil {
		return nil, "", "", false, false, fmt.Errorf("get record %s/%s: %w", nsid, rkey, err)
	}
	return output.Value, output.URI, output.CID, true, false, nil
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

	output, err := s.client.ListAllRecords(ctx, s.did, s.sessionID, nsid)
	if err != nil {
		return nil, fmt.Errorf("list records %s: %w", nsid, err)
	}
	out := make([]rawRecord, 0, len(output.Records))
	for _, rec := range output.Records {
		atURI, err := syntax.ParseATURI(rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("list: invalid AT-URI, skipping")
			continue
		}
		out = append(out, rawRecord{URI: rec.URI, RKey: atURI.RecordKey().String(), CID: rec.CID, Record: rec.Value})
	}
	return out, nil
}

// putRecord creates or updates a record at nsid/rkey. If rkey is empty,
// a new record is created and the PDS-assigned rkey is returned along
// with the CID. For updates (rkey non-empty), the underlying PDS API
// does not return a CID, so cid will be "". The witness cache is updated
// write-through and the session cache is invalidated for the NSID.
func (s *AtprotoStore) putRecord(ctx context.Context, nsid, rkey string, record any) (resultRKey, cid string, err error) {
	if rkey == "" {
		out, err := s.client.CreateRecord(ctx, s.did, s.sessionID, &CreateRecordInput{
			Collection: nsid,
			Record:     record,
		})
		if err != nil {
			return "", "", fmt.Errorf("create record %s: %w", nsid, err)
		}
		atURI, err := syntax.ParseATURI(out.URI)
		if err != nil {
			return "", "", fmt.Errorf("parse created URI %q: %w", out.URI, err)
		}
		newRKey := atURI.RecordKey().String()
		s.writeThroughWitness(nsid, newRKey, out.CID, record)
		s.cache.InvalidateRecords(s.sessionID, nsid)
		return newRKey, out.CID, nil
	}

	if err := s.client.PutRecord(ctx, s.did, s.sessionID, &PutRecordInput{
		Collection: nsid,
		RKey:       rkey,
		Record:     record,
	}); err != nil {
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
	if err := s.client.DeleteRecord(ctx, s.did, s.sessionID, &DeleteRecordInput{
		Collection: nsid,
		RKey:       rkey,
	}); err != nil {
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
	case NSIDBean:
		return "bean"
	case NSIDBrew:
		return "brew"
	case NSIDBrewer:
		return "brewer"
	case NSIDGrinder:
		return "grinder"
	case NSIDRecipe:
		return "recipe"
	case NSIDRoaster:
		return "roaster"
	default:
		return "unknown"
	}
}
