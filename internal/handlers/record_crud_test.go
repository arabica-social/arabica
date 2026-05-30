package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/arabica.social/arabica/internal/records"
)

type crudTestStore struct {
	putErr    error
	putNSID   string
	putRKey   string
	putRecord any
}

func (s *crudTestStore) DID() string { return "did:plc:test" }

func (s *crudTestStore) FetchRecord(context.Context, string, string) (map[string]any, string, string, error) {
	return nil, "", "", nil
}

func (s *crudTestStore) FetchAllRecords(context.Context, string) ([]records.RawRecord, error) {
	return nil, nil
}

func (s *crudTestStore) PutRecord(_ context.Context, nsid, rkey string, record any) (string, string, error) {
	s.putNSID = nsid
	s.putRKey = rkey
	s.putRecord = record
	if s.putErr != nil {
		return "", "", s.putErr
	}
	if rkey == "" {
		return "new-rkey", "cid", nil
	}
	return "", "", nil
}

func (s *crudTestStore) RemoveRecord(context.Context, string, string) error { return nil }

type crudTestRequest struct {
	Name string `json:"name"`
}

func (r *crudTestRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

type crudTestModel struct {
	RKey string `json:"rkey"`
	Name string `json:"name"`
}

func TestRecordCRUDWriteCreateSuccess(t *testing.T) {
	store := &crudTestStore{}
	var invalidated bool
	req := httptest.NewRequest(http.MethodPost, "/api/things", strings.NewReader(`{"name":"thing"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	RecordCRUDWrite[crudTestRequest, *crudTestRequest, crudTestModel](
		w, req, store, "social.test.thing", "thing", "", nil,
		func(req *crudTestRequest) *crudTestModel { return &crudTestModel{Name: req.Name} },
		func(m *crudTestModel, rkey string) { m.RKey = rkey },
		func(_ records.Store, _ *crudTestRequest, m *crudTestModel) (map[string]any, error) {
			return map[string]any{"name": m.Name}, nil
		},
		func() { invalidated = true }, false,
	)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"rkey":"new-rkey"`)
	assert.True(t, invalidated)
	assert.Equal(t, "social.test.thing", store.putNSID)
	assert.Equal(t, "", store.putRKey)
	assert.Equal(t, map[string]any{"name": "thing"}, store.putRecord)
}

func TestRecordCRUDWriteUpdateSuccessFallsBackToExistingRKey(t *testing.T) {
	store := &crudTestStore{}
	req := httptest.NewRequest(http.MethodPut, "/api/things/existing", strings.NewReader(`{"name":"updated"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	RecordCRUDWrite[crudTestRequest, *crudTestRequest, crudTestModel](
		w, req, store, "social.test.thing", "thing", "existing", nil,
		func(req *crudTestRequest) *crudTestModel { return &crudTestModel{Name: req.Name} },
		func(m *crudTestModel, rkey string) { m.RKey = rkey },
		func(_ records.Store, _ *crudTestRequest, m *crudTestModel) (map[string]any, error) {
			return map[string]any{"name": m.Name}, nil
		},
		nil, false,
	)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"rkey":"existing"`)
	assert.Equal(t, "existing", store.putRKey)
}

func TestRecordCRUDWriteDecodeError(t *testing.T) {
	store := &crudTestStore{}
	req := httptest.NewRequest(http.MethodPost, "/api/things", strings.NewReader(`{"name"`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	RecordCRUDWrite[crudTestRequest, *crudTestRequest, crudTestModel](
		w, req, store, "social.test.thing", "thing", "", nil,
		func(req *crudTestRequest) *crudTestModel { return &crudTestModel{Name: req.Name} },
		func(m *crudTestModel, rkey string) { m.RKey = rkey },
		func(_ records.Store, _ *crudTestRequest, _ *crudTestModel) (map[string]any, error) { return nil, nil },
		nil, false,
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid request body")
}

func TestRecordCRUDWriteValidationError(t *testing.T) {
	store := &crudTestStore{}
	req := httptest.NewRequest(http.MethodPost, "/api/things", strings.NewReader(`{"name":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	RecordCRUDWrite[crudTestRequest, *crudTestRequest, crudTestModel](
		w, req, store, "social.test.thing", "thing", "", nil,
		func(req *crudTestRequest) *crudTestModel { return &crudTestModel{Name: req.Name} },
		func(m *crudTestModel, rkey string) { m.RKey = rkey },
		func(_ records.Store, _ *crudTestRequest, _ *crudTestModel) (map[string]any, error) { return nil, nil },
		nil, false,
	)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "name is required")
}

func TestRecordCRUDWriteStoreError(t *testing.T) {
	store := &crudTestStore{putErr: errors.New("boom")}
	req := httptest.NewRequest(http.MethodPost, "/api/things", strings.NewReader(`{"name":"thing"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	RecordCRUDWrite[crudTestRequest, *crudTestRequest, crudTestModel](
		w, req, store, "social.test.thing", "thing", "", nil,
		func(req *crudTestRequest) *crudTestModel { return &crudTestModel{Name: req.Name} },
		func(m *crudTestModel, rkey string) { m.RKey = rkey },
		func(_ records.Store, _ *crudTestRequest, m *crudTestModel) (map[string]any, error) {
			return map[string]any{"name": m.Name}, nil
		},
		nil, false,
	)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to save thing")
}
