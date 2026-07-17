package coffeehandlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atproto"
)

func TestHandleBrewCreateReturnsJSONSessionExpired(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.CreateBrewFunc = func(context.Context, *arabica.CreateBrewRequest, int) (*arabica.Brew, error) {
		return nil, atproto.ErrSessionExpired
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/brews")
	req.Body = ioNopCloser("bean_rkey=3jzfcijpj2z2a")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewCreate(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Your session has expired. Please log in again.","code":"session_expired"}`, rec.Body.String())
}

func TestHandleBrewCreateUnauthenticatedReturnsJSONNotRedirect(t *testing.T) {
	tc := NewTestContext()
	req := httptest.NewRequest(http.MethodPost, "/brews", strings.NewReader("bean_rkey=3jzfcijpj2z2a"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewCreate(rec, req)

	// A JSON SPA request must get a JSON 401 it can react to, not a same-tab
	// redirect that would discard an in-progress brew form.
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Authentication required","code":"authentication_required"}`, rec.Body.String())
	assert.Empty(t, rec.Header().Get("Location"))
}

func TestHandleBrewUpdateReturnsJSONSessionExpired(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.UpdateBrewByRKeyFunc = func(context.Context, string, *arabica.CreateBrewRequest) error {
		return atproto.ErrSessionExpired
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPut, "/brews/3jzfcijpj2z2a")
	req.SetPathValue("id", "3jzfcijpj2z2a")
	req.Body = ioNopCloser("bean_rkey=3jzfcijpj2z2a")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewUpdate(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Your session has expired. Please log in again.","code":"session_expired"}`, rec.Body.String())
}

func TestHandleRecipeCreateReturnsJSONSessionExpired(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.CreateRecipeFunc = func(context.Context, *arabica.CreateRecipeRequest) (*arabica.Recipe, error) {
		return nil, atproto.ErrSessionExpired
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/api/recipes")
	req.Body = ioNopCloser(`{"name":"V60","brewer_rkey":"3jzfcijpj2z2a","coffee_amount":15,"water_amount":250}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleRecipeCreate(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Your session has expired. Please log in again.","code":"session_expired"}`, rec.Body.String())
}

func TestHandleRecipeUpdateReturnsJSONSessionExpired(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.UpdateRecipeByRKeyFunc = func(context.Context, string, *arabica.UpdateRecipeRequest) error {
		return atproto.ErrSessionExpired
	}

	req := newMiddlewareAuthenticatedRequest(http.MethodPut, "/api/recipes/3jzfcijpj2z2a")
	req.SetPathValue("id", "3jzfcijpj2z2a")
	req.Body = ioNopCloser(`{"name":"V60","brewer_rkey":"3jzfcijpj2z2a","coffee_amount":15,"water_amount":250}`)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleRecipeUpdate(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/json; charset=utf-8", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"Your session has expired. Please log in again.","code":"session_expired"}`, rec.Body.String())
}

func TestBrewSessionErrorsFallbackToPlainTextForLegacyRequests(t *testing.T) {
	tc := NewTestContext()
	tc.Handler.SetStoreOverrideForTest(tc.MockStore)
	tc.MockStore.CreateBrewFunc = func(context.Context, *arabica.CreateBrewRequest, int) (*arabica.Brew, error) {
		return nil, atproto.ErrSessionExpired
	}

	// Legacy HTMX callers (no Accept: application/json) keep the plain-text
	// 401 body so existing flows are unchanged.
	req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/brews")
	req.Body = ioNopCloser("bean_rkey=3jzfcijpj2z2a")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewCreate(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.NotContains(t, rec.Header().Get("Content-Type"), "application/json")
	assert.Contains(t, rec.Body.String(), "Your session has expired")
}
