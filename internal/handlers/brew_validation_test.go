package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

func TestHandleBrewCreateRequiresBrewer(t *testing.T) {
	for _, transport := range []string{"form", "api_json", "api_form"} {
		for _, selection := range []struct {
			name    string
			value   string
			omit    bool
			status  int
			message string
		}{
			{name: "omitted", omit: true, status: http.StatusBadRequest, message: "Brewer selection is required"},
			{name: "empty", status: http.StatusBadRequest, message: "Brewer selection is required"},
			{name: "invalid", value: "invalid/rkey", status: http.StatusBadRequest, message: "Brewer selection has invalid format"},
			{name: "selected_without_rating", value: "brewer123", status: http.StatusOK},
		} {
			t.Run(transport+"/"+selection.name, func(t *testing.T) {
				tc := NewTestContext()
				tc.Handler.SetStoreOverrideForTest(tc.MockStore)
				writes := 0
				tc.MockStore.CreateBrewFunc = func(_ context.Context, req *arabica.CreateBrewRequest, _ int) (*arabica.Brew, error) {
					writes++
					assert.Equal(t, "brewer123", req.BrewerRKey)
					assert.Zero(t, req.Rating, "rating remains optional")
					return &arabica.Brew{RKey: "brew123", BeanRKey: req.BeanRKey, BrewerRKey: req.BrewerRKey}, nil
				}
				tc.MockStore.GetBeanByRKeyFunc = func(context.Context, string) (*arabica.Bean, error) {
					return nil, errors.New("not found")
				}

				fields := map[string]string{"bean_rkey": "bean123"}
				if !selection.omit {
					fields["brewer_rkey"] = selection.value
				}
				path := "/api/brews"
				if transport == "form" {
					path = "/brews"
				}
				req := newMiddlewareAuthenticatedRequest(http.MethodPost, path)
				req.Header.Set("Accept", "application/json")
				if transport == "api_json" {
					body, err := json.Marshal(fields)
					require.NoError(t, err)
					req.Body = ioNopCloser(string(body))
					req.Header.Set("Content-Type", "application/json")
				} else {
					values := url.Values{}
					for key, value := range fields {
						values.Set(key, value)
					}
					req.Body = ioNopCloser(values.Encode())
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				}
				rec := httptest.NewRecorder()
				if transport == "form" {
					tc.Handler.HandleBrewCreate(rec, req)
				} else {
					tc.Handler.HandleBrewCreateJSON(rec, req)
				}

				require.Equal(t, selection.status, rec.Code, rec.Body.String())
				if selection.status == http.StatusOK {
					assert.Equal(t, 1, writes)
				} else {
					assert.Zero(t, writes, "invalid creates must not reach the store")
					assert.Contains(t, rec.Body.String(), selection.message)
					if transport != "form" {
						assert.Contains(t, rec.Body.String(), `"code":"validation_failed"`)
					}
				}
			})
		}
	}
}

func TestHandleBrewCreateTimeSeconds(t *testing.T) {
	for _, transport := range []string{"form", "api_form", "api_json"} {
		for _, seconds := range []string{"0", "3601", "45000", "-1", "1.5"} {
			t.Run(transport+"/"+seconds, func(t *testing.T) {
				tc := NewTestContext()
				tc.Handler.SetStoreOverrideForTest(tc.MockStore)
				writes := 0
				tc.MockStore.CreateBrewFunc = func(_ context.Context, req *arabica.CreateBrewRequest, _ int) (*arabica.Brew, error) {
					writes++
					assert.Equal(t, seconds, strconv.Itoa(req.TimeSeconds))
					return &arabica.Brew{RKey: "brew123", TimeSeconds: req.TimeSeconds}, nil
				}
				tc.MockStore.GetBeanByRKeyFunc = func(context.Context, string) (*arabica.Bean, error) { return nil, errors.New("not found") }
				req := newMiddlewareAuthenticatedRequest(http.MethodPost, "/api/brews")
				req.Header.Set("Accept", "application/json")
				if transport == "api_json" {
					req.Header.Set("Content-Type", "application/json")
					req.Body = ioNopCloser(`{"bean_rkey":"bean123","brewer_rkey":"brewer123","time_seconds":` + seconds + `}`)
				} else {
					req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
					req.Body = ioNopCloser(url.Values{"bean_rkey": {"bean123"}, "brewer_rkey": {"brewer123"}, "time_seconds": {seconds}}.Encode())
				}
				rec := httptest.NewRecorder()
				if transport == "form" {
					tc.Handler.HandleBrewCreate(rec, req)
				} else {
					tc.Handler.HandleBrewCreateJSON(rec, req)
				}
				if seconds == "-1" || seconds == "1.5" {
					require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
					assert.Zero(t, writes)
				} else {
					require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
					assert.Equal(t, 1, writes)
				}
			})
		}
	}
}
