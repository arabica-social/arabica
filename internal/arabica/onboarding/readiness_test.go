package onboarding

import (
	"context"
	"errors"
	"testing"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"

	"github.com/stretchr/testify/assert"
)

type fakeBrewPrerequisiteStore struct {
	listBeans    func(context.Context) ([]*arabica.Bean, error)
	listBrewers  func(context.Context) ([]*arabica.Brewer, error)
	listRoasters func(context.Context) ([]*arabica.Roaster, error)
}

func emptyStore() *fakeBrewPrerequisiteStore {
	return &fakeBrewPrerequisiteStore{
		listBeans:    func(ctx context.Context) ([]*arabica.Bean, error) { return nil, nil },
		listBrewers:  func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
		listRoasters: func(ctx context.Context) ([]*arabica.Roaster, error) { return nil, nil },
	}
}

func (s *fakeBrewPrerequisiteStore) ListBeans(ctx context.Context) ([]*arabica.Bean, error) {
	return s.listBeans(ctx)
}

func (s *fakeBrewPrerequisiteStore) ListBrewers(ctx context.Context) ([]*arabica.Brewer, error) {
	return s.listBrewers(ctx)
}

func (s *fakeBrewPrerequisiteStore) ListRoasters(ctx context.Context) ([]*arabica.Roaster, error) {
	return s.listRoasters(ctx)
}

func TestCheckBrewReadiness_None(t *testing.T) {
	got, err := CheckBrewReadiness(context.Background(), emptyStore())

	assert.NoError(t, err)
	assert.False(t, got.HasBean)
	assert.False(t, got.HasBrewer)
	assert.False(t, got.HasRoaster)
	assert.False(t, got.Ready())
}

func TestCheckBrewReadiness_BrewerOnly(t *testing.T) {
	store := emptyStore()
	store.listBrewers = func(ctx context.Context) ([]*arabica.Brewer, error) {
		return []*arabica.Brewer{{RKey: "a"}}, nil
	}

	got, err := CheckBrewReadiness(context.Background(), store)

	assert.NoError(t, err)
	assert.True(t, got.HasBrewer)
	assert.False(t, got.HasBean)
	assert.False(t, got.HasRoaster)
	assert.False(t, got.Ready())
}

func TestCheckBrewReadiness_MissingRoaster(t *testing.T) {
	store := emptyStore()
	store.listBeans = func(ctx context.Context) ([]*arabica.Bean, error) {
		return []*arabica.Bean{{RKey: "a"}}, nil
	}
	store.listBrewers = func(ctx context.Context) ([]*arabica.Brewer, error) {
		return []*arabica.Brewer{{RKey: "b"}}, nil
	}

	got, err := CheckBrewReadiness(context.Background(), store)

	assert.NoError(t, err)
	assert.True(t, got.HasBean)
	assert.True(t, got.HasBrewer)
	assert.False(t, got.HasRoaster)
	assert.False(t, got.Ready(), "roaster is required for initial readiness")
}

func TestCheckBrewReadiness_All(t *testing.T) {
	store := emptyStore()
	store.listBeans = func(ctx context.Context) ([]*arabica.Bean, error) {
		return []*arabica.Bean{{RKey: "a"}}, nil
	}
	store.listBrewers = func(ctx context.Context) ([]*arabica.Brewer, error) {
		return []*arabica.Brewer{{RKey: "b"}}, nil
	}
	store.listRoasters = func(ctx context.Context) ([]*arabica.Roaster, error) {
		return []*arabica.Roaster{{RKey: "c"}}, nil
	}

	got, err := CheckBrewReadiness(context.Background(), store)

	assert.NoError(t, err)
	assert.True(t, got.HasBean)
	assert.True(t, got.HasBrewer)
	assert.True(t, got.HasRoaster)
	assert.True(t, got.Ready())
}

func TestCheckBrewReadiness_BeanError(t *testing.T) {
	want := errors.New("boom")
	store := emptyStore()
	store.listBeans = func(ctx context.Context) ([]*arabica.Bean, error) { return nil, want }

	_, err := CheckBrewReadiness(context.Background(), store)

	assert.ErrorIs(t, err, want)
}

func TestCheckBrewReadiness_RoasterError(t *testing.T) {
	want := errors.New("roaster boom")
	store := emptyStore()
	store.listRoasters = func(ctx context.Context) ([]*arabica.Roaster, error) { return nil, want }

	_, err := CheckBrewReadiness(context.Background(), store)

	assert.ErrorIs(t, err, want)
}
