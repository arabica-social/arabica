// Package onboarding holds Arabica helpers for the new-user onboarding flow.
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

// BrewPrerequisiteStore is the narrow slice of Arabica store behavior that
// CheckBrewReadiness needs.
type BrewPrerequisiteStore interface {
	ListBeans(ctx context.Context) ([]*arabica.Bean, error)
	ListBrewers(ctx context.Context) ([]*arabica.Brewer, error)
	ListRoasters(ctx context.Context) ([]*arabica.Roaster, error)
}

// ReadinessStatus reports which brew-prerequisite collections the user owns.
//
// Roaster is required for *initial* onboarding so the user is nudged into
// recording at least one — but bean records themselves continue to allow a
// nil roaster (the "random blend" case). Once a user owns any roaster, this
// gate stays satisfied even if later beans are roaster-less.
type ReadinessStatus struct {
	HasBean    bool
	HasBrewer  bool
	HasRoaster bool
}

// Ready returns true when the user owns at least one brewer, one roaster,
// and one bean — the minimum required to log a brew.
func (s ReadinessStatus) Ready() bool {
	return s.HasBean && s.HasBrewer && s.HasRoaster
}

// CheckBrewReadiness derives the user's readiness from the store. It calls
// ListBeans / ListBrewers / ListRoasters; the AtprotoStore implementation
// uses its caches, so this is cheap on repeat calls within a request.
func CheckBrewReadiness(ctx context.Context, store BrewPrerequisiteStore) (ReadinessStatus, error) {
	beans, err := store.ListBeans(ctx)
	if err != nil {
		return ReadinessStatus{}, fmt.Errorf("list beans: %w", err)
	}
	brewers, err := store.ListBrewers(ctx)
	if err != nil {
		return ReadinessStatus{}, fmt.Errorf("list brewers: %w", err)
	}
	roasters, err := store.ListRoasters(ctx)
	if err != nil {
		return ReadinessStatus{}, fmt.Errorf("list roasters: %w", err)
	}
	return ReadinessStatus{
		HasBean:    len(beans) > 0,
		HasBrewer:  len(brewers) > 0,
		HasRoaster: len(roasters) > 0,
	}, nil
}
