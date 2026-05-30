package handlers

import (
	"context"
	"net/http"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/metrics"
	"tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"github.com/rs/zerolog/log"
)

// EntityLoadConfig captures per-entity behavior for loading a typed record
// from the authenticated owner store, witness cache, or public PDS fallback.
type EntityLoadConfig struct {
	Descriptor  *entities.Descriptor
	FromWitness func(ctx context.Context, m map[string]any, uri, rkey, ownerDID string) (any, error)
	FromPDS     func(ctx context.Context, e *atp.Record, rkey, ownerDID string) (any, error)
	FromStore   func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, map[string]any, string, string, error)
	// ResolveRefs, if set, runs after a record is decoded on any of the
	// three source paths (own-store, witness, PDS). It receives the
	// typed model, the raw record map (to read ref AT-URIs), and a
	// source-bound lookup that fetches foreign records. Implementations
	// must be idempotent — for the own-store path the codec PostGet
	// may have already populated some ref fields.
	ResolveRefs func(ctx context.Context, model any, raw map[string]any, lookup func(refURI string) (map[string]any, bool))
}

// LoadedEntity is the HTTP-independent result of loading an entity view record.
type LoadedEntity struct {
	Record       any
	SubjectURI   string
	SubjectCID   string
	OwnerDID     string
	IsOwnProfile bool
	Route        domain.EntityRoute
	EntityNoun   string
}

type EntityLoadErrorKind string

const (
	EntityLoadBadRequest EntityLoadErrorKind = "bad_request"
	EntityLoadNotFound   EntityLoadErrorKind = "not_found"
	EntityLoadInternal   EntityLoadErrorKind = "internal"
)

// EntityLoadError lets RenderEntityView map loading failures to HTTP responses
// without the loader writing to the response itself.
type EntityLoadError struct {
	Kind EntityLoadErrorKind
	Msg  string
	Err  error
}

func (e *EntityLoadError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}

func (e *EntityLoadError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func (e *EntityLoadError) HTTPStatus() int {
	switch e.Kind {
	case EntityLoadBadRequest:
		return http.StatusBadRequest
	case EntityLoadNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

type rawAtprotoStore interface {
	RawAtprotoStore() *atproto.AtprotoStore
}

// EntityViewLoader owns the source selection policy for entity detail pages.
// It tries authenticated own-store reads first, then witness cache, then public PDS.
type EntityViewLoader struct {
	h *Handler
}

func (h *Handler) EntityViewLoader() EntityViewLoader {
	return EntityViewLoader{h: h}
}

func (l EntityViewLoader) Load(r *http.Request, rkey string, cfg EntityLoadConfig) (*LoadedEntity, error) {
	h := l.h
	ctx := r.Context()
	route := h.entityRouteFor(cfg.Descriptor)
	entityNoun := route.Noun
	if entityNoun == "" {
		entityNoun = "content"
	}

	owner := r.URL.Query().Get("owner")
	if owner == "" {
		return nil, &EntityLoadError{Kind: EntityLoadBadRequest, Msg: "owner required"}
	}

	entityOwnerDID, err := ResolveOwnerDID(ctx, owner)
	if err != nil {
		log.Warn().Err(err).Str("handle", owner).Msgf("Failed to resolve handle for %s view", entityNoun)
		return nil, &EntityLoadError{Kind: EntityLoadNotFound, Msg: "User not found", Err: err}
	}

	didStr, _ := atpmiddleware.GetDID(ctx)
	isOwnProfile := didStr != "" && didStr == entityOwnerDID

	loaded := &LoadedEntity{
		OwnerDID:     entityOwnerDID,
		IsOwnProfile: isOwnProfile,
		Route:        route,
		EntityNoun:   entityNoun,
	}

	// If the viewer owns the record, read through the authenticated
	// AtprotoStore so locally-written records that the firehose hasn't
	// caught up to are still visible.
	if isOwnProfile {
		l.loadFromOwnStore(ctx, r, loaded, rkey, cfg)
	}

	if loaded.Record == nil {
		l.loadFromWitness(ctx, loaded, rkey, cfg)
	}

	if loaded.Record == nil {
		if err := l.loadFromPDS(ctx, loaded, rkey, cfg); err != nil {
			return nil, err
		}
	}

	return loaded, nil
}

func (l EntityViewLoader) loadFromOwnStore(ctx context.Context, r *http.Request, loaded *LoadedEntity, rkey string, cfg EntityLoadConfig) {
	if cfg.FromStore == nil {
		return
	}
	h := l.h
	store, ok := h.GetRecordStore(r)
	if !ok {
		return
	}
	atprotoStore, _ := store.(*atproto.AtprotoStore)
	if atprotoStore == nil {
		if wrapped, ok := store.(rawAtprotoStore); ok {
			atprotoStore = wrapped.RawAtprotoStore()
		}
	}
	if atprotoStore == nil {
		return
	}
	rec, raw, uri, cid, err := cfg.FromStore(ctx, atprotoStore, rkey)
	if err != nil {
		return
	}
	loaded.Record = rec
	loaded.SubjectURI = uri
	loaded.SubjectCID = cid
	if cfg.ResolveRefs != nil {
		cfg.ResolveRefs(ctx, loaded.Record, raw, h.WitnessLookup(ctx))
	}
}

func (l EntityViewLoader) loadFromWitness(ctx context.Context, loaded *LoadedEntity, rkey string, cfg EntityLoadConfig) {
	h := l.h
	if h.witnessCache == nil || cfg.Descriptor == nil || cfg.FromWitness == nil {
		return
	}
	entityURI := atp.BuildATURI(loaded.OwnerDID, cfg.Descriptor.NSID, rkey)
	wr, _ := h.witnessCache.GetWitnessRecord(ctx, entityURI)
	if wr == nil {
		return
	}
	m, err := atproto.WitnessRecordToMap(wr)
	if err != nil {
		return
	}
	rec, err := cfg.FromWitness(ctx, m, wr.URI, rkey, loaded.OwnerDID)
	if err != nil {
		return
	}
	metrics.WitnessCacheHitsTotal.WithLabelValues(loaded.EntityNoun).Inc()
	loaded.Record = rec
	loaded.SubjectURI = wr.URI
	loaded.SubjectCID = wr.CID
	if cfg.ResolveRefs != nil {
		cfg.ResolveRefs(ctx, loaded.Record, m, h.WitnessLookup(ctx))
	}
}

func (l EntityViewLoader) loadFromPDS(ctx context.Context, loaded *LoadedEntity, rkey string, cfg EntityLoadConfig) error {
	if cfg.Descriptor == nil || cfg.FromPDS == nil {
		return &EntityLoadError{Kind: EntityLoadInternal, Msg: "entity load config is incomplete"}
	}
	metrics.WitnessCacheMissesTotal.WithLabelValues(loaded.EntityNoun).Inc()
	pub := atproto.NewPublicClient()
	entry, err := pub.GetPublicRecord(ctx, loaded.OwnerDID, cfg.Descriptor.NSID, rkey)
	if err != nil {
		log.Error().Err(err).Str("did", loaded.OwnerDID).Str("rkey", rkey).Msgf("Failed to get %s record", loaded.EntityNoun)
		return &EntityLoadError{Kind: EntityLoadNotFound, Msg: cfg.Descriptor.DisplayName + " not found", Err: err}
	}
	rec, err := cfg.FromPDS(ctx, entry, rkey, loaded.OwnerDID)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to convert %s record", loaded.EntityNoun)
		return &EntityLoadError{Kind: EntityLoadInternal, Msg: "Failed to load " + loaded.EntityNoun, Err: err}
	}
	loaded.Record = rec
	loaded.SubjectURI = entry.URI
	loaded.SubjectCID = entry.CID
	if cfg.ResolveRefs != nil {
		cfg.ResolveRefs(ctx, loaded.Record, entry.Value, PublicLookup(ctx))
	}
	return nil
}
