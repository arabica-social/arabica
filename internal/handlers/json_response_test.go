package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWriteJSONStatusPreservesStatusAndContentType(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSONStatus(w, http.StatusCreated, map[string]string{"result": "created"}, "test")

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"result":"created"}`, w.Body.String())
}

func TestWriteJSONErrorUsesStableEnvelope(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSONError(w, http.StatusForbidden, "permission_denied", "You cannot edit this record.")

	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"You cannot edit this record.","code":"permission_denied"}`, w.Body.String())
}

func TestWriteJSONValidationErrorIncludesFieldDetails(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSONValidationError(w, map[string]string{"name": "Name is required"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{
		"error":"Please correct the highlighted fields.",
		"code":"validation_failed",
		"fields":{"name":"Name is required"}
	}`, w.Body.String())
}
