package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderEntityViewJSONReturnsJSONForInvalidRecordKey(t *testing.T) {
	h := NewHandler(nil, nil, nil, nil, nil, Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/roasters/alice.test/bad_key", nil)
	req.SetPathValue("id", "bad/key")
	w := httptest.NewRecorder()

	h.RenderEntityViewJSON(w, req, EntityViewConfig{})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Invalid record key format","code":"invalid_request"}`, w.Body.String())
}

func TestWriteEntityLoadJSONErrorMapsPublicFailureSemantics(t *testing.T) {
	tests := []struct {
		name    string
		err     error
		status  int
		code    string
		message string
	}{
		{
			name:    "bad request",
			err:     &EntityLoadError{Kind: EntityLoadBadRequest, Msg: "owner required"},
			status:  http.StatusBadRequest,
			code:    "invalid_request",
			message: "owner required",
		},
		{
			name:    "not found",
			err:     &EntityLoadError{Kind: EntityLoadNotFound, Msg: "Record not found"},
			status:  http.StatusNotFound,
			code:    "not_found",
			message: "Record not found",
		},
		{
			name:    "internal details stay private",
			err:     &EntityLoadError{Kind: EntityLoadInternal, Msg: "decode failed", Err: assert.AnError},
			status:  http.StatusInternalServerError,
			code:    "internal_error",
			message: "Failed to load record",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()

			writeEntityLoadJSONError(w, tt.err)

			assert.Equal(t, tt.status, w.Code)
			assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
			assert.JSONEq(t, `{"error":"`+tt.message+`","code":"`+tt.code+`"}`, w.Body.String())
			assert.NotContains(t, w.Body.String(), assert.AnError.Error())
		})
	}
}
