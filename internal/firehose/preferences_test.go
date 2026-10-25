package firehose

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"tangled.org/arabica.social/arabica/internal/profileprefs"
)

func TestPreferencesPartialUpdatesAndIsolation(t *testing.T) {
	idx, err := NewFeedIndex(t.TempDir()+"/index.db", time.Hour)
	require.NoError(t, err)
	defer idx.Close()
	ctx := context.Background()
	const did = "did:plc:alice"
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			if err := idx.UpdateUserPreferences(ctx, did, func(p profileprefs.UserPreferences) profileprefs.UserPreferences {
				p.BrewFormMode = profileprefs.BrewFormDetailed
				return p
			}); err != nil {
				t.Error(err)
			}
		}()
		go func() {
			defer wg.Done()
			if err := idx.UpdateUserPreferences(ctx, did, func(p profileprefs.UserPreferences) profileprefs.UserPreferences {
				p.SmartAutofill = profileprefs.SmartAutofillOff
				return p
			}); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	got := idx.GetUserPreferences(ctx, did)
	require.Equal(t, profileprefs.BrewFormDetailed, got.BrewFormMode)
	require.Equal(t, profileprefs.SmartAutofillOff, got.SmartAutofill)
	require.Equal(t, profileprefs.DefaultUserPreferences(), idx.GetUserPreferences(ctx, "did:plc:bob"))
	require.NoError(t, idx.profileStorage.setUserPreferences(ctx, did, `{"brew_form_mode":"broken"}`))
	require.Equal(t, profileprefs.BrewFormQuick, idx.GetUserPreferences(ctx, did).BrewFormMode)
	require.NoError(t, idx.profileStorage.setUserPreferences(ctx, did, `not json`))
	require.Equal(t, profileprefs.DefaultUserPreferences(), idx.GetUserPreferences(ctx, did))
}
