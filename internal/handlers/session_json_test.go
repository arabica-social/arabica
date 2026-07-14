package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHandleSessionJSONAnonymous(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
	rec := httptest.NewRecorder()

	h.HandleSessionJSON(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	var response SessionResponseJSON
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	require.Equal(t, SessionResponseJSON{TemperatureUnit: "recorded"}, response)
}
