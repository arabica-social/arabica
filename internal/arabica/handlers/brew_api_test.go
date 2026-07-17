package coffeehandlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atproto"
)

// TestHandleBrewCreateJSONSessionExpired asserts the typed JSON create
// endpoint surfaces a 401 session_expired envelope when the store reports an
// expired OAuth session, mirroring the legacy HandleBrewCreate JSON path.
func TestHandleBrewCreateJSONSessionExpired(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.CreateBrewFunc = func(context.Context, *arabica.CreateBrewRequest, int) (*arabica.Brew, error) {
		return nil, atproto.ErrSessionExpired
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/api/brews")
	req.Body = ioNopCloser(`{"bean_rkey":"3jzfcijpj2z2a","coffee_amount":18,"water_amount":250}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewCreateJSON(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Your session has expired. Please log in again.","code":"session_expired"}`, rec.Body.String())
}

// TestHandleBrewCreateJSONUnauthenticated asserts the typed JSON create
// endpoint returns a JSON 401 for a missing session instead of redirecting.
func TestHandleBrewCreateJSONUnauthenticated(t *testing.T) {
	tc := NewTestContext()
	req := httptest.NewRequest(http.MethodPost, "/api/brews", ioNopCloser(`{"bean_rkey":"3jzfcijpj2z2a"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewCreateJSON(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Authentication required","code":"authentication_required"}`, rec.Body.String())
	assert.Empty(t, rec.Header().Get("Location"))
}

// TestHandleBrewCreateJSONValidationMissingBean asserts a JSON create without
// a bean_rkey returns a 400 validation_failed envelope.
func TestHandleBrewCreateJSONValidationMissingBean(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)

	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/api/brews")
	req.Body = ioNopCloser(`{"coffee_amount":18,"water_amount":250}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewCreateJSON(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"Bean selection is required","code":"validation_failed"}`, rec.Body.String())
}

// TestHandleBrewCreateJSONValidationOutOfRange asserts the numeric range
// validation (replicated from the legacy validateBrewRequest) rejects an
// out-of-range temperature via the JSON envelope.
func TestHandleBrewCreateJSONValidationOutOfRange(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)

	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/api/brews")
	req.Body = ioNopCloser(`{"bean_rkey":"3jzfcijpj2z2a","temperature":300}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewCreateJSON(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "temperature")
	assert.Contains(t, rec.Body.String(), "validation_failed")
}

// TestHandleBrewCreateJSONHappyPath asserts a successful JSON create returns
// the brew model envelope with author_did, and skips the incomplete-bean nudge
// when the referenced bean fetch fails.
func TestHandleBrewCreateJSONHappyPath(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.CreateBrewFunc = func(ctx context.Context, req *arabica.CreateBrewRequest, userID int) (*arabica.Brew, error) {
		return &arabica.Brew{RKey: "3jzfcijpj2z2a", BeanRKey: req.BeanRKey}, nil
	}
	// Avoid the nudge path by failing the bean lookup.
	tc.MockStore.GetBeanByRKeyFunc = func(context.Context, string) (*arabica.Bean, error) {
		return nil, errors.New("none")
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/api/brews")
	req.Body = ioNopCloser(`{"bean_rkey":"3jzfcijpj2z2a","coffee_amount":18,"water_amount":250,"rating":8}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewCreateJSON(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	var resp BrewMutationJSONResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Brew)
	assert.Equal(t, "3jzfcijpj2z2a", resp.Brew.RKey)
	assert.Equal(t, "did:plc:test123456789", resp.AuthorDID)
	assert.Nil(t, resp.IncompleteNudge)
}

// TestHandleBrewCreateJSONIncompleteBeanNudge asserts the nudge is populated
// when the referenced bean is incomplete.
func TestHandleBrewCreateJSONIncompleteBeanNudge(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.CreateBrewFunc = func(ctx context.Context, req *arabica.CreateBrewRequest, userID int) (*arabica.Brew, error) {
		return &arabica.Brew{RKey: "3jzfcijpj2z2a", BeanRKey: req.BeanRKey}, nil
	}
	tc.MockStore.GetBeanByRKeyFunc = func(context.Context, string) (*arabica.Bean, error) {
		// Incomplete: missing roaster_rkey and roast_level.
		return &arabica.Bean{RKey: "3jzfcijpj2z2a", Name: "Test Bean"}, nil
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/api/brews")
	req.Body = ioNopCloser(`{"bean_rkey":"3jzfcijpj2z2a"}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewCreateJSON(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp BrewMutationJSONResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotNil(t, resp.IncompleteNudge)
	assert.Equal(t, "bean", resp.IncompleteNudge.EntityType)
	assert.Equal(t, "Test Bean", resp.IncompleteNudge.Name)
}

// TestHandleBrewUpdateJSONSessionExpired asserts the typed JSON update endpoint
// surfaces a 401 session_expired envelope when the store reports an expired
// OAuth session.
func TestHandleBrewUpdateJSONSessionExpired(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.UpdateBrewByRKeyFunc = func(context.Context, string, *arabica.CreateBrewRequest) error {
		return atproto.ErrSessionExpired
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPut, "/api/brews/3jzfcijpj2z2a")
	req.SetPathValue("id", "3jzfcijpj2z2a")
	req.Body = ioNopCloser(`{"bean_rkey":"3jzfcijpj2z2a","coffee_amount":18,"water_amount":250}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewUpdateJSON(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Your session has expired. Please log in again.","code":"session_expired"}`, rec.Body.String())
}

// TestHandleBrewUpdateJSONHappyPath asserts a successful JSON update returns
// the re-fetched brew model envelope with author_did.
func TestHandleBrewUpdateJSONHappyPath(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.UpdateBrewByRKeyFunc = func(context.Context, string, *arabica.CreateBrewRequest) error {
		return nil
	}
	tc.MockStore.GetBrewByRKeyFunc = func(context.Context, string) (*arabica.Brew, error) {
		return &arabica.Brew{RKey: "3jzfcijpj2z2a", BeanRKey: "3jzfcijpj2z2a", Rating: 9}, nil
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPut, "/api/brews/3jzfcijpj2z2a")
	req.SetPathValue("id", "3jzfcijpj2z2a")
	req.Body = ioNopCloser(`{"bean_rkey":"3jzfcijpj2z2a","rating":9}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewUpdateJSON(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var resp BrewMutationJSONResponse
	assert.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.NotNil(t, resp.Brew)
	assert.Equal(t, "3jzfcijpj2z2a", resp.Brew.RKey)
	assert.Equal(t, 9, resp.Brew.Rating)
	assert.Equal(t, "did:plc:test123456789", resp.AuthorDID)
	assert.Nil(t, resp.IncompleteNudge)
}

// TestHandleBrewUpdateJSONValidationMissingBean asserts a JSON update without
// a bean_rkey returns a 400 validation_failed envelope.
func TestHandleBrewUpdateJSONValidationMissingBean(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)

	req := newMiddlewareAuthenticatedRequest(http.MethodPut, "/api/brews/3jzfcijpj2z2a")
	req.SetPathValue("id", "3jzfcijpj2z2a")
	req.Body = ioNopCloser(`{"coffee_amount":18}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewUpdateJSON(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"Bean selection is required","code":"validation_failed"}`, rec.Body.String())
}
