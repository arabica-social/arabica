package storecodec

import "context"

// EntityCodec describes how to move one typed record kind between the
// store's wire format (map[string]any) and a Go model. It lets generic
// CRUD helpers replace the bodies of per-entity Create/Get/List/Update/Delete
// methods without losing type safety at the call site.
type EntityCodec[M any] struct {
	NSID string
	// ToRecord receives the backing store as any so app-owned codec packages can
	// depend only on this neutral type instead of importing the atproto store.
	ToRecord   func(store any, model *M) (any, error)
	FromRecord func(rec map[string]any, uri string) (*M, error)
	SetRKey    func(model *M, rkey string)
	// PostGet runs once after a Get decodes the model. Typical use is resolving
	// foreign references. Optional.
	PostGet func(ctx context.Context, store any, model *M, rec map[string]any)
	// PostList runs after each List element is decoded. Must be pure (no I/O).
	// Typical use is extracting foreign rkeys from raw refs so callers can
	// batch-resolve later. Optional.
	PostList func(model *M, rec map[string]any)
}
