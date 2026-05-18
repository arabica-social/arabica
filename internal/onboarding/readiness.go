// Package onboarding holds shared helpers for the new-user onboarding flow.
//
// Currently it derives "is this user ready to log a brew?" from the user's
// PDS state. There is no persistence: deleting all beans puts the user back
// into onboarding, which is the correct behavior.
package onboarding

import (
	"context"
	"fmt"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

// BrewPrerequisiteStore is the narrow slice of database.Store that
// CheckBrewReadiness needs. Callers may pass any database.Store value;
// tests may use database.MockStore directly.
type BrewPrerequisiteStore interface {
	ListBeans(ctx context.Context) ([]*arabica.Bean, error)
	ListBrewers(ctx context.Context) ([]*arabica.Brewer, error)
}

// ReadinessStatus reports which brew-prerequisite collections the user owns.
type ReadinessStatus struct {
	HasBean   bool
	HasBrewer bool
}

// Ready returns true when the user owns at least one bean and one brewer —
// the minimum required to log a brew.
func (s ReadinessStatus) Ready() bool {
	return s.HasBean && s.HasBrewer
}

// CheckBrewReadiness derives the user's readiness from the store. It calls
// ListBeans / ListBrewers; the AtprotoStore implementation uses its caches,
// so this is cheap on repeat calls within a request.
func CheckBrewReadiness(ctx context.Context, store BrewPrerequisiteStore) (ReadinessStatus, error) {
	beans, err := store.ListBeans(ctx)
	if err != nil {
		return ReadinessStatus{}, fmt.Errorf("list beans: %w", err)
	}
	brewers, err := store.ListBrewers(ctx)
	if err != nil {
		return ReadinessStatus{}, fmt.Errorf("list brewers: %w", err)
	}
	return ReadinessStatus{
		HasBean:   len(beans) > 0,
		HasBrewer: len(brewers) > 0,
	}, nil
}
