package atproto

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"tangled.org/arabica.social/arabica/internal/social"
	"tangled.org/pdewey.com/atp"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
)

// AtprotoStore implements generic AT Protocol record CRUD using atproto records.
// Context is passed as a parameter to each method rather than stored in the struct,
// following Go best practices for context propagation.
type AtprotoStore struct {
	client       *Client
	did          syntax.DID
	sessionID    string
	cache        *SessionCache
	witnessCache WitnessCache // optional; enables cache-first reads without PDS calls

	// likeNSID and commentNSID are the collection NSIDs this store reads
	// and writes for likes/comments. They must be set by app-aware
	// production callers before shared social handlers can write records.
	likeNSID    string
	commentNSID string
}

// NewAtprotoStore creates a new atproto store for a specific user session.
// The cache parameter allows for dependency injection and testability.
func NewAtprotoStore(client *Client, did syntax.DID, sessionID string, cache *SessionCache) *AtprotoStore {
	return &AtprotoStore{
		client:    client,
		did:       did,
		sessionID: sessionID,
		cache:     cache,
	}
}

// NewAtprotoStoreWithWitness creates a store that uses the witness cache for
// cache-first reads, falling back to the PDS on cache misses.
func NewAtprotoStoreWithWitness(client *Client, did syntax.DID, sessionID string, cache *SessionCache, witness WitnessCache) *AtprotoStore {
	return &AtprotoStore{
		client:       client,
		did:          did,
		sessionID:    sessionID,
		cache:        cache,
		witnessCache: witness,
	}
}

// NewAtprotoStoreForApp builds a store wired with per-app like/comment NSIDs.
func NewAtprotoStoreForApp(client *Client, did syntax.DID, sessionID string, cache *SessionCache, witness WitnessCache, likeNSID, commentNSID string) *AtprotoStore {
	return &AtprotoStore{
		client:       client,
		did:          did,
		sessionID:    sessionID,
		cache:        cache,
		witnessCache: witness,
		likeNSID:     likeNSID,
		commentNSID:  commentNSID,
	}
}

func (s *AtprotoStore) likeCollection() string {
	return s.likeNSID
}

func (s *AtprotoStore) commentCollection() string {
	return s.commentNSID
}

// atpClient returns an *atp.Client scoped to this store's DID and session.
func (s *AtprotoStore) atpClient(ctx context.Context) (*atp.Client, error) {
	return s.client.AtpClient(ctx, s.did, s.sessionID)
}

// witnessRecordToMap is a package-internal alias for WitnessRecordToMap.
func witnessRecordToMap(wr *WitnessRecord) (map[string]any, error) {
	return WitnessRecordToMap(wr)
}

// getFromWitness fetches a single record by collection+rkey from the witness cache.
// Returns nil when the cache is not configured, the record is not found,
// or the collection was recently written to (dirty).
func (s *AtprotoStore) getFromWitness(ctx context.Context, collection, rkey string) *WitnessRecord {
	if s.witnessCache == nil {
		return nil
	}
	// Skip witness cache for collections with pending writes
	if userCache := s.cache.Get(s.sessionID); userCache.IsDirty(collection) {
		log.Debug().Str("collection", collection).Msg("witness: skipping dirty collection for single record, falling back to PDS")
		return nil
	}
	uri := atp.BuildATURI(s.did.String(), collection, rkey)
	wr, err := s.witnessCache.GetWitnessRecord(ctx, uri)
	if err != nil {
		log.Debug().Err(err).Str("uri", uri).Msg("witness: GetWitnessRecord error")
		return nil
	}
	return wr
}

// listFromWitness returns all cached records for a collection.
// Returns nil when the cache is not configured or returns nothing.
// Skips the witness cache if the collection was recently written to
// (dirty), since the firehose may not have indexed the new record yet.
func (s *AtprotoStore) listFromWitness(ctx context.Context, collection string) []*WitnessRecord {
	if s.witnessCache == nil {
		return nil
	}
	// Skip witness cache for collections with pending writes
	if userCache := s.cache.Get(s.sessionID); userCache.IsDirty(collection) {
		log.Debug().Str("collection", collection).Msg("witness: skipping dirty collection, falling back to PDS")
		return nil
	}
	records, err := s.witnessCache.ListWitnessRecords(ctx, s.did.String(), collection)
	if err != nil {
		log.Debug().Err(err).Str("collection", collection).Msg("witness: ListWitnessRecords error")
		return nil
	}
	if len(records) == 0 {
		return nil
	}
	return records
}

// writeThroughWitness upserts a record into the witness cache after a
// successful PDS write so subsequent reads see the latest data without
// waiting for the firehose to re-index.
func (s *AtprotoStore) writeThroughWitness(collection, rkey, cid string, record any) {
	if s.witnessCache == nil {
		return
	}
	data, err := json.Marshal(record)
	if err != nil {
		log.Warn().Err(err).Str("collection", collection).Str("rkey", rkey).
			Msg("witness write-through: failed to marshal record")
		return
	}
	if err := s.witnessCache.UpsertWitnessRecord(context.Background(), s.did.String(), collection, rkey, cid, data); err != nil {
		log.Warn().Err(err).Str("collection", collection).Str("rkey", rkey).
			Msg("witness write-through: failed to upsert record")
	}
}

// updateThroughWitness updates a witness record's body without touching cid,
// for use after a successful PDS PutRecord (which doesn't return a CID).
func (s *AtprotoStore) updateThroughWitness(collection, rkey string, record any) {
	if s.witnessCache == nil {
		return
	}
	data, err := json.Marshal(record)
	if err != nil {
		log.Warn().Err(err).Str("collection", collection).Str("rkey", rkey).
			Msg("witness write-through: failed to marshal record")
		return
	}
	if err := s.witnessCache.UpdateWitnessRecord(context.Background(), s.did.String(), collection, rkey, data); err != nil {
		log.Warn().Err(err).Str("collection", collection).Str("rkey", rkey).
			Msg("witness write-through: failed to update record")
	}
}

// deleteFromWitness removes a record from the witness cache after a
// successful PDS deletion.
func (s *AtprotoStore) deleteFromWitness(collection, rkey string) {
	if s.witnessCache == nil {
		return
	}
	if err := s.witnessCache.DeleteWitnessRecord(context.Background(), s.did.String(), collection, rkey); err != nil {
		log.Warn().Err(err).Str("collection", collection).Str("rkey", rkey).
			Msg("witness write-through: failed to delete record")
	}
}

// getWitnessRecordByURI fetches a single record by full AT-URI from the witness cache.
// Returns nil when the cache is not configured or the record is not found.
func (s *AtprotoStore) getWitnessRecordByURI(ctx context.Context, uri string) *WitnessRecord {
	if s.witnessCache == nil {
		return nil
	}
	wr, err := s.witnessCache.GetWitnessRecord(ctx, uri)
	if err != nil {
		log.Debug().Err(err).Str("uri", uri).Msg("witness: GetWitnessRecord error")
		return nil
	}
	return wr
}

// ========== Like Operations ==========

func (s *AtprotoStore) CreateLike(ctx context.Context, req *social.CreateLikeRequest) (*social.Like, error) {
	if req.SubjectURI == "" {
		return nil, fmt.Errorf("subject_uri is required")
	}
	if req.SubjectCID == "" {
		return nil, fmt.Errorf("subject_cid is required")
	}

	collection := s.likeCollection()
	if collection == "" {
		return nil, fmt.Errorf("like collection is not configured")
	}

	likeModel := &social.Like{
		SubjectURI: req.SubjectURI,
		SubjectCID: req.SubjectCID,
		CreatedAt:  time.Now().UTC(),
	}

	record, err := social.LikeToRecord(collection, likeModel)
	if err != nil {
		return nil, fmt.Errorf("failed to convert like to record: %w", err)
	}
	rkey, _, err := s.PutRecord(ctx, collection, "", record)
	if err != nil {
		return nil, fmt.Errorf("failed to create like record: %w", err)
	}
	likeModel.RKey = rkey

	return likeModel, nil
}

func (s *AtprotoStore) DeleteLikeByRKey(ctx context.Context, rkey string) error {
	collection := s.likeCollection()
	if collection == "" {
		return fmt.Errorf("like collection is not configured")
	}
	if err := s.RemoveRecord(ctx, collection, rkey); err != nil {
		return fmt.Errorf("failed to delete like record: %w", err)
	}
	return nil
}

func (s *AtprotoStore) GetUserLikeForSubject(ctx context.Context, subjectURI string) (*social.Like, error) {
	// List all likes and find the one matching the subject URI
	likes, err := s.ListUserLikes(ctx)
	if err != nil {
		return nil, err
	}

	for _, like := range likes {
		if like.SubjectURI == subjectURI {
			return like, nil
		}
	}

	return nil, nil // Not found (not an error)
}

func (s *AtprotoStore) ListUserLikes(ctx context.Context) ([]*social.Like, error) {
	collection := s.likeCollection()
	if collection == "" {
		return nil, fmt.Errorf("like collection is not configured")
	}
	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get atp client: %w", err)
	}
	records, err := atpClient.ListAllRecords(ctx, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to list like records: %w", err)
	}

	likes := make([]*social.Like, 0, len(records))

	for _, rec := range records {
		like, err := social.RecordToLike(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert like record")
			continue
		}

		// Extract rkey from URI
		if rkey := atp.RKeyFromURI(rec.URI); rkey != "" {
			like.RKey = rkey
		}

		likes = append(likes, like)
	}

	return likes, nil
}

// ========== Comment Operations ==========

func (s *AtprotoStore) CreateComment(ctx context.Context, req *social.CreateCommentRequest) (*social.Comment, error) {
	if req.SubjectURI == "" {
		return nil, fmt.Errorf("subject_uri is required")
	}
	if req.SubjectCID == "" {
		return nil, fmt.Errorf("subject_cid is required")
	}
	if req.Text == "" {
		return nil, fmt.Errorf("text is required")
	}

	collection := s.commentCollection()
	if collection == "" {
		return nil, fmt.Errorf("comment collection is not configured")
	}

	commentModel := &social.Comment{
		SubjectURI: req.SubjectURI,
		SubjectCID: req.SubjectCID,
		Text:       req.Text,
		CreatedAt:  time.Now().UTC(),
		ParentURI:  req.ParentURI,
		ParentCID:  req.ParentCID,
	}

	record, err := social.CommentToRecord(collection, commentModel)
	if err != nil {
		return nil, fmt.Errorf("failed to convert comment to record: %w", err)
	}
	rkey, cid, err := s.PutRecord(ctx, collection, "", record)
	if err != nil {
		return nil, fmt.Errorf("failed to create comment record: %w", err)
	}
	commentModel.RKey = rkey
	// Store the CID of this comment record (useful for threading)
	commentModel.CID = cid

	return commentModel, nil
}

func (s *AtprotoStore) DeleteCommentByRKey(ctx context.Context, rkey string) error {
	collection := s.commentCollection()
	if collection == "" {
		return fmt.Errorf("comment collection is not configured")
	}
	if err := s.RemoveRecord(ctx, collection, rkey); err != nil {
		return fmt.Errorf("failed to delete comment record: %w", err)
	}
	return nil
}

func (s *AtprotoStore) GetCommentsForSubject(ctx context.Context, subjectURI string) ([]*social.Comment, error) {
	// List all comments and filter by subject URI
	// Note: This is inefficient for large numbers of comments.
	// The firehose index provides a more efficient lookup.
	comments, err := s.ListUserComments(ctx)
	if err != nil {
		return nil, err
	}

	var filtered []*social.Comment
	for _, comment := range comments {
		if comment.SubjectURI == subjectURI {
			filtered = append(filtered, comment)
		}
	}

	return filtered, nil
}

func (s *AtprotoStore) ListUserComments(ctx context.Context) ([]*social.Comment, error) {
	collection := s.commentCollection()
	if collection == "" {
		return nil, fmt.Errorf("comment collection is not configured")
	}
	atpClient, err := s.atpClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("get atp client: %w", err)
	}
	records, err := atpClient.ListAllRecords(ctx, collection)
	if err != nil {
		return nil, fmt.Errorf("failed to list comment records: %w", err)
	}

	comments := make([]*social.Comment, 0, len(records))

	for _, rec := range records {
		comment, err := social.RecordToComment(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("Failed to convert comment record")
			continue
		}

		// Extract rkey from URI
		if rkey := atp.RKeyFromURI(rec.URI); rkey != "" {
			comment.RKey = rkey
		}

		comments = append(comments, comment)
	}

	return comments, nil
}

func (s *AtprotoStore) Close() error {
	// No persistent connection to close for atproto
	return nil
}
