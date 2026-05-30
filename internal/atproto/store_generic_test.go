package atproto

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/bluesky-social/indigo/xrpc"
	"github.com/stretchr/testify/assert"
)

func TestIsRepoNotFoundError(t *testing.T) {
	err := fmt.Errorf("list records social.arabica.alpha.bean: %w", &xrpc.Error{
		StatusCode: http.StatusBadRequest,
		Wrapped: &xrpc.XRPCError{
			ErrStr:  "InvalidRequest",
			Message: "Could not find repo: did:plc:missing",
		},
	})

	assert.True(t, isRepoNotFoundError(err))
}

func TestIsRepoNotFoundErrorRejectsOtherBadRequests(t *testing.T) {
	err := fmt.Errorf("list records social.arabica.alpha.bean: %w", &xrpc.Error{
		StatusCode: http.StatusBadRequest,
		Wrapped: &xrpc.XRPCError{
			ErrStr:  "InvalidRequest",
			Message: "collection is invalid",
		},
	})

	assert.False(t, isRepoNotFoundError(err))
}
