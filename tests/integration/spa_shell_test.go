//go:build integration

package integration

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTP_SPAShellIncludesTestAuthSession(t *testing.T) {
	h := StartHarness(t, nil)

	resp := h.Get("/brews/new")
	body := ReadBody(t, resp)
	require.Equal(t, http.StatusOK, resp.StatusCode, statusErr(resp, body))
	assert.Contains(t, body, `data-frontend="sveltekit"`)
	assert.Contains(t, body, `data-user-did="`+h.PrimaryAccount.DID+`"`)
}
