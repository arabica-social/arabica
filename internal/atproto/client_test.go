package atproto

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapPDSError(t *testing.T) {
	t.Run("nil error passes through", func(t *testing.T) {
		assert.Nil(t, wrapPDSError(nil))
	})

	t.Run("unrelated error passes through unchanged", func(t *testing.T) {
		err := errors.New("network timeout")
		result := wrapPDSError(err)
		assert.False(t, errors.Is(result, ErrSessionExpired))
		assert.Equal(t, err, result)
	})

	t.Run("invalid_grant is wrapped as ErrSessionExpired", func(t *testing.T) {
		err := fmt.Errorf("auth server request failed (HTTP 400): invalid_grant")
		result := wrapPDSError(err)
		assert.True(t, errors.Is(result, ErrSessionExpired))
	})

	t.Run("token refresh failure is wrapped as ErrSessionExpired", func(t *testing.T) {
		err := fmt.Errorf("failed to refresh OAuth tokens: token refresh failed")
		result := wrapPDSError(err)
		assert.True(t, errors.Is(result, ErrSessionExpired))
	})

	t.Run("token expired is wrapped as ErrSessionExpired", func(t *testing.T) {
		err := fmt.Errorf("token is expired")
		result := wrapPDSError(err)
		assert.True(t, errors.Is(result, ErrSessionExpired))
	})

	t.Run("nested invalid_grant is detected", func(t *testing.T) {
		inner := fmt.Errorf("auth server request failed (HTTP 400): invalid_grant")
		err := fmt.Errorf("failed to refresh OAuth tokens: token refresh failed: %w", inner)
		result := wrapPDSError(err)
		assert.True(t, errors.Is(result, ErrSessionExpired))
	})
}
