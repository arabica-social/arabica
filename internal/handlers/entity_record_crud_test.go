package handlers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

func TestHandleRoasterCreateUsesGenericRecordStore(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)

	var gotNSID, gotRKey string
	var gotRecord any
	tc.MockStore.PutRecordFunc = func(_ context.Context, nsid, rkey string, record any) (string, string, error) {
		gotNSID = nsid
		gotRKey = rkey
		gotRecord = record
		return "3jzfcijpj2z2a", "cid", nil
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/api/roasters")
	req.Body = ioNopCloser(`{"name":"Test Roaster","location":"Test City","website":"https://example.com"}`)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleRoasterCreate(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), `"rkey":"3jzfcijpj2z2a"`)
	assert.Equal(t, arabica.NSIDRoaster, gotNSID)
	assert.Equal(t, "", gotRKey)
	recordMap, ok := gotRecord.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "Test Roaster", recordMap["name"])
	assert.Equal(t, "Test City", recordMap["location"])
}

func TestHandleRoasterCreateReturnsJSONValidationError(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)

	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/api/roasters")
	req.Body = ioNopCloser(`{"name":""}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleRoasterCreate(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"name is required","code":"validation_failed"}`, rec.Body.String())
}

func TestHandleRoasterCreateReturnsJSONAuthenticationError(t *testing.T) {
	tc := NewTestContext()
	req := httptest.NewRequest(http.MethodPost, "/api/roasters", strings.NewReader(`{"name":"Test Roaster"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleRoasterCreate(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Authentication required","code":"authentication_required"}`, rec.Body.String())
}

func TestHandleRoasterCreateReturnsSafeJSONStoreError(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.PutRecordFunc = func(context.Context, string, string, any) (string, string, error) {
		return "", "", errors.New("private PDS failure detail")
	}
	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/api/roasters")
	req.Body = ioNopCloser(`{"name":"Test Roaster"}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleRoasterCreate(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Failed to save roaster","code":"internal_error"}`, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "private PDS failure detail")
}

func TestHandleRoasterUpdatePreservesCreatedAt(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)

	createdAt := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	tc.MockStore.FetchRecordFunc = func(_ context.Context, nsid, rkey string) (map[string]any, string, string, error) {
		assert.Equal(t, arabica.NSIDRoaster, nsid)
		assert.Equal(t, "3jzfcijpj2z2a", rkey)
		return map[string]any{"createdAt": createdAt.Format(time.RFC3339)}, "", "", nil
	}
	var gotRecord any
	tc.MockStore.PutRecordFunc = func(_ context.Context, nsid, rkey string, record any) (string, string, error) {
		assert.Equal(t, arabica.NSIDRoaster, nsid)
		assert.Equal(t, "3jzfcijpj2z2a", rkey)
		gotRecord = record
		return "", "", nil
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPut, "/api/roasters/3jzfcijpj2z2a")
	req.SetPathValue("id", "3jzfcijpj2z2a")
	req.Body = ioNopCloser(`{"name":"Updated Roaster","location":"Seattle","website":"https://example.com"}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleRoasterUpdate(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), `"rkey":"3jzfcijpj2z2a"`)
	recordMap, ok := gotRecord.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "Updated Roaster", recordMap["name"])
	assert.Equal(t, createdAt.Format(time.RFC3339), recordMap["createdAt"])
}

func TestHandleRoasterDeleteReturnsJSON(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)

	var deletedRKey string
	tc.MockStore.DeleteRoasterByRKeyFunc = func(_ context.Context, rkey string) error {
		deletedRKey = rkey
		return nil
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodDelete, "/api/roasters/3jzfcijpj2z2a")
	req.SetPathValue("id", "3jzfcijpj2z2a")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleRoasterDelete(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"deleted":true}`, rec.Body.String())
	assert.Equal(t, "3jzfcijpj2z2a", deletedRKey)
}

func TestHandleGrinderUpdatePreservesCreatedAt(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)

	createdAt := time.Date(2024, 5, 1, 12, 0, 0, 0, time.UTC)
	tc.MockStore.FetchRecordFunc = func(_ context.Context, nsid, rkey string) (map[string]any, string, string, error) {
		assert.Equal(t, arabica.NSIDGrinder, nsid)
		assert.Equal(t, "3jzfcijpj2z2a", rkey)
		return map[string]any{"createdAt": createdAt.Format(time.RFC3339)}, "", "", nil
	}
	var gotRecord any
	tc.MockStore.PutRecordFunc = func(_ context.Context, nsid, rkey string, record any) (string, string, error) {
		assert.Equal(t, arabica.NSIDGrinder, nsid)
		assert.Equal(t, "3jzfcijpj2z2a", rkey)
		gotRecord = record
		return "", "", nil
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPut, "/api/grinders/3jzfcijpj2z2a")
	req.SetPathValue("id", "3jzfcijpj2z2a")
	req.Body = ioNopCloser(`{"name":"Updated Grinder","grinder_type":"Hand","burr_type":"Conical"}`)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleGrinderUpdate(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"rkey":"3jzfcijpj2z2a"`)
	recordMap, ok := gotRecord.(map[string]any)
	assert.True(t, ok)
	assert.Equal(t, "Updated Grinder", recordMap["name"])
	assert.Equal(t, createdAt.Format(time.RFC3339), recordMap["createdAt"])
}

func ioNopCloser(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}

func newMiddlewareAuthenticatedRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	ctx := atpmiddleware.ContextWithAuth(req.Context(), "did:plc:test123456789", "test-session-id")
	return req.WithContext(ctx)
}
