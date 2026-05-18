package onboarding

import (
	"context"
	"errors"
	"testing"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/database"

	"github.com/stretchr/testify/assert"
)

func TestCheckBrewReadiness_None(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return nil, nil },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
	}

	got, err := CheckBrewReadiness(context.Background(), store)

	assert.NoError(t, err)
	assert.False(t, got.HasBean)
	assert.False(t, got.HasBrewer)
	assert.False(t, got.Ready())
}

func TestCheckBrewReadiness_BrewerOnly(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return nil, nil },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "a"}}, nil },
	}

	got, err := CheckBrewReadiness(context.Background(), store)

	assert.NoError(t, err)
	assert.True(t, got.HasBrewer)
	assert.False(t, got.HasBean)
	assert.False(t, got.Ready())
}

func TestCheckBrewReadiness_BeanOnly(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return []*arabica.Bean{{RKey: "a"}}, nil },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
	}

	got, err := CheckBrewReadiness(context.Background(), store)

	assert.NoError(t, err)
	assert.True(t, got.HasBean)
	assert.False(t, got.HasBrewer)
	assert.False(t, got.Ready())
}

func TestCheckBrewReadiness_Both(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return []*arabica.Bean{{RKey: "a"}}, nil },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "b"}}, nil },
	}

	got, err := CheckBrewReadiness(context.Background(), store)

	assert.NoError(t, err)
	assert.True(t, got.HasBean)
	assert.True(t, got.HasBrewer)
	assert.True(t, got.Ready())
}

func TestCheckBrewReadiness_BeanError(t *testing.T) {
	want := errors.New("boom")
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return nil, want },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
	}

	_, err := CheckBrewReadiness(context.Background(), store)

	assert.ErrorIs(t, err, want)
}
