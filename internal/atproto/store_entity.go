package atproto

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
)

// EntityCodec describes how to move one typed record kind between the
// store's wire format (map[string]any) and a Go model. It lets the
// generic CRUD helpers below replace the bodies of every per-entity
// Create/Get/List/Update/Delete method without losing type safety at
// the call site.
//
// ToRecord converts a populated model into the value PutRecord expects;
// FromRecord is the inverse for fetched records (URI gives reference
// resolution a stable identity); SetRKey assigns the PDS-returned rkey
// to the model field.
type EntityCodec[M any] struct {
	NSID string
	// ToRecord receives the store so encoders for entities with foreign
	// references (e.g. Bean → Roaster) can call s.DID() to build AT-URIs.
	// Simple entities ignore the first argument.
	ToRecord   func(s *AtprotoStore, m *M) (any, error)
	FromRecord func(rec map[string]any, uri string) (*M, error)
	SetRKey    func(model *M, rkey string)
	// PostGet runs once after a Get decodes the model. Typical use is
	// resolving foreign references (witness lookup, PDS fallback) — it
	// may make network calls. Optional.
	PostGet func(ctx context.Context, s *AtprotoStore, m *M, rec map[string]any)
	// PostList runs after each List element is decoded. Must be pure
	// (no I/O) — runs in a tight loop. Typical use is extracting
	// foreign rkeys from raw refs so callers can batch-resolve later
	// (see LinkBeansToRoasters). Optional.
	PostList func(m *M, rec map[string]any)
}

// CreateEntity creates a new record and returns the freshly-keyed model.
func CreateEntity[M any](ctx context.Context, s *AtprotoStore, c *EntityCodec[M], model *M) (*M, error) {
	rec, err := c.ToRecord(s, model)
	if err != nil {
		return nil, fmt.Errorf("convert %s: %w", c.NSID, err)
	}
	rkey, _, err := s.putRecord(ctx, c.NSID, "", rec)
	if err != nil {
		return nil, err
	}
	c.SetRKey(model, rkey)
	return model, nil
}

// GetEntity reads one record from witness or PDS and converts it to *M.
// PostGet, if set, runs after decode to resolve foreign references.
func GetEntity[M any](ctx context.Context, s *AtprotoStore, c *EntityCodec[M], rkey string) (*M, error) {
	rec, uri, _, hit, _, err := s.fetchRecord(ctx, c.NSID, rkey)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, fmt.Errorf("%s %s not found", c.NSID, rkey)
	}
	model, err := c.FromRecord(rec, uri)
	if err != nil {
		return nil, fmt.Errorf("convert %s: %w", c.NSID, err)
	}
	c.SetRKey(model, rkey)
	if c.PostGet != nil {
		c.PostGet(ctx, s, model, rec)
	}
	return model, nil
}

// ListEntity returns every record of the codec's NSID, populating the
// session cache. cached is a getter for the typed cache slice (returning
// nil signals "not cached"); pass nil to skip the cache short-circuit
// entirely. The caller's getter exists because UserCache exposes typed
// accessors (Beans(), Roasters()) that callers reach via concrete types.
func ListEntity[M any](ctx context.Context, s *AtprotoStore, c *EntityCodec[M], cached func() []*M) ([]*M, error) {
	if cached != nil {
		if uc := s.cache.Get(s.sessionID); uc != nil && uc.IsValid() {
			if cs := cached(); cs != nil {
				return cs, nil
			}
		}
	}
	raws, err := s.fetchAllRecords(ctx, c.NSID)
	if err != nil {
		return nil, err
	}
	out := make([]*M, 0, len(raws))
	for _, r := range raws {
		model, err := c.FromRecord(r.Record, r.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", r.URI).Str("nsid", c.NSID).Msg("failed to convert record")
			continue
		}
		c.SetRKey(model, r.RKey)
		if c.PostList != nil {
			c.PostList(model, r.Record)
		}
		out = append(out, model)
	}
	s.cache.SetRecords(s.sessionID, c.NSID, out)
	s.cache.ClearDirty(s.sessionID, c.NSID)
	return out, nil
}

// UpdateEntity overwrites an existing record. The supplied model must
// already carry whatever fields should be preserved across the update
// (e.g. CreatedAt copied from the existing record by the caller).
func UpdateEntity[M any](ctx context.Context, s *AtprotoStore, c *EntityCodec[M], rkey string, model *M) error {
	rec, err := c.ToRecord(s, model)
	if err != nil {
		return fmt.Errorf("convert %s: %w", c.NSID, err)
	}
	_, _, err = s.putRecord(ctx, c.NSID, rkey, rec)
	return err
}

// DeleteEntity removes a record by rkey.
func DeleteEntity(ctx context.Context, s *AtprotoStore, nsid, rkey string) error {
	return s.removeRecord(ctx, nsid, rkey)
}

// EntityRecord pairs a typed model with the AT Protocol metadata
// (URI + CID) needed for like/comment subject references. Replaces
// the per-entity wrapper structs (BeanRecord, RoasterRecord, …) that
// were structurally identical.
type EntityRecord[M any] struct {
	Model *M
	URI   string
	CID   string
}

// GetEntityRecord is like GetEntity but additionally returns the URI
// and CID of the fetched record, wrapped in EntityRecord[M]. View
// handlers use these to wire like/comment widgets to the subject.
// PostGet runs the same as in GetEntity.
func GetEntityRecord[M any](ctx context.Context, s *AtprotoStore, c *EntityCodec[M], rkey string) (*EntityRecord[M], error) {
	rec, uri, cid, hit, _, err := s.fetchRecord(ctx, c.NSID, rkey)
	if err != nil {
		return nil, err
	}
	if !hit {
		return nil, fmt.Errorf("%s record %s not found", c.NSID, rkey)
	}
	model, err := c.FromRecord(rec, uri)
	if err != nil {
		return nil, fmt.Errorf("convert %s: %w", c.NSID, err)
	}
	c.SetRKey(model, rkey)
	if c.PostGet != nil {
		c.PostGet(ctx, s, model, rec)
	}
	return &EntityRecord[M]{Model: model, URI: uri, CID: cid}, nil
}
