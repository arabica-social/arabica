package handlers

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tangled.org/arabica.social/arabica/internal/firehose"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

func TestHandleNotificationsMarkReadJSONRequiresAuthentication(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, Config{})
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/read", nil)
	w := httptest.NewRecorder()

	h.HandleNotificationsMarkReadJSON(w, req)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Authentication required","code":"authentication_required"}`, w.Body.String())
}

func TestHandleNotificationsMarkReadJSONReturnsJSONOnStoreFailure(t *testing.T) {
	idx, err := firehose.NewFeedIndex(filepath.Join(t.TempDir(), "feed.db"), 0)
	require.NoError(t, err)
	require.NoError(t, idx.Close())

	h := NewHandler(nil, nil, nil, nil, nil, Config{})
	h.SetFeedIndex(idx)
	req := httptest.NewRequest(http.MethodPost, "/api/notifications/read", nil)
	req = req.WithContext(atpmiddleware.ContextWithAuth(req.Context(), "did:plc:test", "session"))
	w := httptest.NewRecorder()

	h.HandleNotificationsMarkReadJSON(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Failed to mark notifications as read","code":"internal_error"}`, w.Body.String())
}
