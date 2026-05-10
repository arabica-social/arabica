package atproto

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"tangled.org/pdewey.com/atp"
)

func TestWrapPDSError(t *testing.T) {
	t.Run("nil error passes through", func(t *testing.T) {
		assert.Nil(t, atp.WrapPDSError(nil))
	})

	t.Run("unrelated error passes through unchanged", func(t *testing.T) {
		err := errors.New("network timeout")
		result := atp.WrapPDSError(err)
		assert.False(t, errors.Is(result, ErrSessionExpired))
		assert.Equal(t, err, result)
	})

	t.Run("invalid_grant is wrapped as ErrSessionExpired", func(t *testing.T) {
		err := fmt.Errorf("auth server request failed (HTTP 400): invalid_grant")
		result := atp.WrapPDSError(err)
		assert.True(t, errors.Is(result, ErrSessionExpired))
	})

	t.Run("token refresh failure is wrapped as ErrSessionExpired", func(t *testing.T) {
		err := fmt.Errorf("failed to refresh OAuth tokens: token refresh failed")
		result := atp.WrapPDSError(err)
		assert.True(t, errors.Is(result, ErrSessionExpired))
	})

	t.Run("token expired is wrapped as ErrSessionExpired", func(t *testing.T) {
		err := fmt.Errorf("token is expired")
		result := atp.WrapPDSError(err)
		assert.True(t, errors.Is(result, ErrSessionExpired))
	})

	t.Run("nested invalid_grant is detected", func(t *testing.T) {
		inner := fmt.Errorf("auth server request failed (HTTP 400): invalid_grant")
		err := fmt.Errorf("failed to refresh OAuth tokens: token refresh failed: %w", inner)
		result := atp.WrapPDSError(err)
		assert.True(t, errors.Is(result, ErrSessionExpired))
	})
}
