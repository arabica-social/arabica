package atproto

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"
	"tangled.org/arabica.social/arabica/internal/atproto/storecodec"
	"tangled.org/arabica.social/arabica/internal/records"
)

type EntityCodec[M any] = storecodec.EntityCodec[M]

type entityStore interface {
	records.Store
	FetchRecordSource(ctx context.Context, nsid, rkey string) (record map[string]any, uri, cid string, hit, fromWitness bool, err error)
	Cache() *SessionCache
	SessionID() string
}

// CreateEntity creates a new record and returns the freshly-keyed model.
func CreateEntity[M any](ctx context.Context, s entityStore, c *EntityCodec[M], model *M) (*M, error) {
	rec, err := c.ToRecord(s, model)
	if err != nil {
		return nil, fmt.Errorf("convert %s: %w", c.NSID, err)
	}
	rkey, _, err := s.PutRecord(ctx, c.NSID, "", rec)
	if err != nil {
		return nil, err
	}
	c.SetRKey(model, rkey)
	return model, nil
}

// GetEntity reads one record from witness or PDS and converts it to *M.
// PostGet, if set, runs after decode to resolve foreign references.
func GetEntity[M any](ctx context.Context, s entityStore, c *EntityCodec[M], rkey string) (*M, error) {
	rec, uri, _, hit, _, err := s.FetchRecordSource(ctx, c.NSID, rkey)
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
func ListEntity[M any](ctx context.Context, s entityStore, c *EntityCodec[M], cached func() []*M) ([]*M, error) {
	if cached != nil {
		if uc := s.Cache().Get(s.SessionID()); uc != nil && uc.IsValid() {
			if cs := cached(); cs != nil {
				return cs, nil
			}
		}
	}
	raws, err := s.FetchAllRecords(ctx, c.NSID)
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
	s.Cache().SetRecords(s.SessionID(), c.NSID, out)
	s.Cache().ClearDirty(s.SessionID(), c.NSID)
	return out, nil
}

// UpdateEntity overwrites an existing record. The supplied model must
// already carry whatever fields should be preserved across the update
// (e.g. CreatedAt copied from the existing record by the caller).
func UpdateEntity[M any](ctx context.Context, s entityStore, c *EntityCodec[M], rkey string, model *M) error {
	rec, err := c.ToRecord(s, model)
	if err != nil {
		return fmt.Errorf("convert %s: %w", c.NSID, err)
	}
	_, _, err = s.PutRecord(ctx, c.NSID, rkey, rec)
	return err
}

// DeleteEntity removes a record by rkey.
func DeleteEntity(ctx context.Context, s entityStore, nsid, rkey string) error {
	return s.RemoveRecord(ctx, nsid, rkey)
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
func GetEntityRecord[M any](ctx context.Context, s entityStore, c *EntityCodec[M], rkey string) (*EntityRecord[M], error) {
	rec, uri, cid, hit, _, err := s.FetchRecordSource(ctx, c.NSID, rkey)
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
