package handlers

import (
	"context"

	"github.com/rs/zerolog/log"
	atp "tangled.org/pdewey.com/atp"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/records"
)

// ListRecords fetches every record of nsid the authenticated user owns
// and decodes each one. Records that fail to decode are logged and
// skipped — callers degrade to "missing record" rather than 500ing.
//
// Not tied to a particular app; works for any (nsid, decoder) pair.
func ListRecords[T any](
	ctx context.Context,
	store records.Store,
	nsid string,
	decode func(map[string]any, string) (*T, error),
) []*T {
	raw, err := store.FetchAllRecords(ctx, nsid)
	if err != nil {
		log.Warn().Err(err).Str("nsid", nsid).Msg("FetchAllRecords failed; rendering empty list")
		return nil
	}
	out := make([]*T, 0, len(raw))
	for _, r := range raw {
		t, err := decode(r.Record, r.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", r.URI).Msg("decode failed; skipping record")
			continue
		}
		out = append(out, t)
	}
	return out
}

// ListPublicRecords mirrors ListRecords but reads from an arbitrary
// DID's PDS via an unauthenticated public client. Used by profile
// handlers that surface another user's records.
func ListPublicRecords[T any](
	ctx context.Context,
	client *atp.PublicClient,
	did, nsid string,
	decode func(map[string]any, string) (*T, error),
) []*T {
	records, err := client.ListAllRecords(ctx, did, nsid)
	if err != nil {
		// An empty collection is not an error from the user's POV — the PDS
		// may 404 a never-written collection. Degrade to empty list.
		return nil
	}
	out := make([]*T, 0, len(records))
	for _, rec := range records {
		t, err := decode(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("ListPublicRecords: decode failed; skipping record")
			continue
		}
		out = append(out, t)
	}
	return out
}

// StandardViewTriple builds the fromWitness/fromPDS/fromStore lambdas
// for an entity view config from a record decoder. The three paths
// (witness cache, PDS public client, authenticated own-store) all
// share the same decode-and-set-rkey shape; per-entity ref resolution
// belongs in EntityViewConfig.resolveRefs, not in these lambdas.
func StandardViewTriple[M any](
	nsid string,
	decode func(map[string]any, string) (*M, error),
	setRKey func(*M, string),
) (
	fromWitness func(ctx context.Context, m map[string]any, uri, rkey, ownerDID string) (any, error),
	fromPDS func(ctx context.Context, e *atp.Record, rkey, ownerDID string) (any, error),
	fromStore func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, map[string]any, string, string, error),
) {
	fromWitness = func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
		model, err := decode(m, uri)
		if err != nil {
			return nil, err
		}
		setRKey(model, rkey)
		return model, nil
	}
	fromPDS = func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
		model, err := decode(e.Value, e.URI)
		if err != nil {
			return nil, err
		}
		setRKey(model, rkey)
		return model, nil
	}
	fromStore = func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, map[string]any, string, string, error) {
		raw, uri, cid, err := s.FetchRecord(ctx, nsid, rkey)
		if err != nil {
			return nil, nil, "", "", err
		}
		model, err := decode(raw, uri)
		if err != nil {
			return nil, nil, "", "", err
		}
		setRKey(model, rkey)
		return model, raw, uri, cid, nil
	}
	return
}

// WitnessLookup returns a lookup closure suitable for resolveRefs that
// reads foreign records from the witness cache. Returns false if the
// witness cache is unavailable or the URI is not indexed.
func (h *Handler) WitnessLookup(ctx context.Context) func(refURI string) (map[string]any, bool) {
	return func(refURI string) (map[string]any, bool) {
		if h.witnessCache == nil || refURI == "" {
			return nil, false
		}
		wr, _ := h.witnessCache.GetWitnessRecord(ctx, refURI)
		if wr == nil {
			return nil, false
		}
		m, err := atproto.WitnessRecordToMap(wr)
		if err != nil {
			return nil, false
		}
		return m, true
	}
}

// PublicLookup returns a lookup closure suitable for resolveRefs that
// reads foreign records through the unauthenticated PDS public client.
// Use this on the PDS fallback path where the witness cache may be
// stale or missing.
func PublicLookup(ctx context.Context) func(refURI string) (map[string]any, bool) {
	pub := atproto.NewPublicClient()
	return func(refURI string) (map[string]any, bool) {
		if refURI == "" {
			return nil, false
		}
		parsed, err := atp.ParseATURI(refURI)
		if err != nil {
			return nil, false
		}
		rec, err := pub.GetPublicRecord(ctx, parsed.DID, parsed.Collection, parsed.RKey)
		if err != nil || rec == nil {
			return nil, false
		}
		return rec.Value, true
	}
}
