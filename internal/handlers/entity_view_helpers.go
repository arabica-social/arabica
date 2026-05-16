package handlers

import (
	"context"

	atp "tangled.org/pdewey.com/atp"

	"tangled.org/arabica.social/arabica/internal/atproto"
)

// standardViewTriple builds the fromWitness/fromPDS/fromStore lambdas
// for an entity view config from a record decoder. The three paths
// (witness cache, PDS public client, authenticated own-store) all
// share the same decode-and-set-rkey shape; per-entity ref resolution
// belongs in entityViewConfig.resolveRefs, not in these lambdas.
func standardViewTriple[M any](
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

// witnessLookup returns a lookup closure suitable for resolveRefs that
// reads foreign records from the witness cache. Returns false if the
// witness cache is unavailable or the URI is not indexed.
func (h *Handler) witnessLookup(ctx context.Context) func(refURI string) (map[string]any, bool) {
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

// publicLookup returns a lookup closure suitable for resolveRefs that
// reads foreign records through the unauthenticated PDS public client.
// Use this on the PDS fallback path where the witness cache may be
// stale or missing.
func publicLookup(ctx context.Context) func(refURI string) (map[string]any, bool) {
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

