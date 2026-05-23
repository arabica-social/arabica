package teahandlers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/records"
)

type fakeRecordStore struct {
	did        string
	putNSID    string
	putRKey    string
	putRecord  any
	putResult  string
	removeNSID string
	removeRKey string
}

func (s *fakeRecordStore) DID() string { return s.did }

func (s *fakeRecordStore) FetchRecord(context.Context, string, string) (map[string]any, string, string, error) {
	return nil, "", "", nil
}

func (s *fakeRecordStore) FetchAllRecords(context.Context, string) ([]records.RawRecord, error) {
	return nil, nil
}

func (s *fakeRecordStore) PutRecord(_ context.Context, nsid, rkey string, record any) (string, string, error) {
	s.putNSID = nsid
	s.putRKey = rkey
	s.putRecord = record
	return s.putResult, "cid", nil
}

func (s *fakeRecordStore) RemoveRecord(_ context.Context, nsid, rkey string) error {
	s.removeNSID = nsid
	s.removeRKey = rkey
	return nil
}

func TestPutOolongRecordCreateUsesReturnedRKey(t *testing.T) {
	store := &fakeRecordStore{did: "did:plc:test", putResult: "new-rkey"}
	record := map[string]any{"name": "tea"}

	rkey, err := handlers.PutRecord(context.Background(), store, "social.oolong.alpha.tea", "", func(records.Store) (map[string]any, error) {
		return record, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "new-rkey", rkey)
	assert.Equal(t, "social.oolong.alpha.tea", store.putNSID)
	assert.Equal(t, "", store.putRKey)
	assert.Equal(t, record, store.putRecord)
}

func TestPutOolongRecordUpdateFallsBackToExistingRKey(t *testing.T) {
	store := &fakeRecordStore{did: "did:plc:test"}
	record := map[string]any{"name": "updated"}

	rkey, err := handlers.PutRecord(context.Background(), store, "social.oolong.alpha.tea", "existing-rkey", func(records.Store) (map[string]any, error) {
		return record, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "existing-rkey", rkey)
	assert.Equal(t, "existing-rkey", store.putRKey)
	assert.Equal(t, record, store.putRecord)
}

func TestBuildOolongRefUsesGenericStoreDID(t *testing.T) {
	store := &fakeRecordStore{did: "did:plc:test"}

	got := buildOolongRef(store, "rkey123", "social.oolong.alpha.vendor")

	assert.Equal(t, "at://did:plc:test/social.oolong.alpha.vendor/rkey123", got)
}
