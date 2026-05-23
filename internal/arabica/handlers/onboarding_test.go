package coffeehandlers

import (
	"context"
	"errors"
	"testing"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/database"

	"github.com/stretchr/testify/assert"
)

func TestBrewNewReady_True(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:    func(ctx context.Context) ([]*arabica.Bean, error) { return []*arabica.Bean{{RKey: "a"}}, nil },
		ListBrewersFunc:  func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "b"}}, nil },
		ListRoastersFunc: func(ctx context.Context) ([]*arabica.Roaster, error) { return []*arabica.Roaster{{RKey: "c"}}, nil },
	}

	assert.True(t, brewNewReady(context.Background(), store))
}

func TestBrewNewReady_FalseWhenMissingBean(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:    func(ctx context.Context) ([]*arabica.Bean, error) { return nil, nil },
		ListBrewersFunc:  func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "b"}}, nil },
		ListRoastersFunc: func(ctx context.Context) ([]*arabica.Roaster, error) { return []*arabica.Roaster{{RKey: "c"}}, nil },
	}

	assert.False(t, brewNewReady(context.Background(), store))
}

func TestBrewNewReady_FalseWhenMissingRoaster(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:    func(ctx context.Context) ([]*arabica.Bean, error) { return []*arabica.Bean{{RKey: "a"}}, nil },
		ListBrewersFunc:  func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "b"}}, nil },
		ListRoastersFunc: func(ctx context.Context) ([]*arabica.Roaster, error) { return nil, nil },
	}

	assert.False(t, brewNewReady(context.Background(), store))
}

func TestBrewNewReady_FalseOnError(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:    func(ctx context.Context) ([]*arabica.Bean, error) { return nil, errors.New("pds down") },
		ListBrewersFunc:  func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
		ListRoastersFunc: func(ctx context.Context) ([]*arabica.Roaster, error) { return nil, nil },
	}

	assert.False(t, brewNewReady(context.Background(), store))
}

func TestBuildGetStartedCardProps_Empty(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:    func(ctx context.Context) ([]*arabica.Bean, error) { return nil, nil },
		ListBrewersFunc:  func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
		ListGrindersFunc: func(ctx context.Context) ([]*arabica.Grinder, error) { return nil, nil },
		ListRoastersFunc: func(ctx context.Context) ([]*arabica.Roaster, error) { return nil, nil },
	}

	props, err := buildGetStartedCardProps(context.Background(), store)

	assert.NoError(t, err)
	assert.False(t, props.Readiness.Ready())
	assert.Empty(t, props.Beans)
	assert.Empty(t, props.Brewers)
}

func TestBuildGetStartedCardProps_Populated(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc: func(ctx context.Context) ([]*arabica.Bean, error) {
			return []*arabica.Bean{{RKey: "b1", Name: "Ethiopia"}}, nil
		},
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) {
			return []*arabica.Brewer{{RKey: "br1", Name: "V60"}}, nil
		},
		ListGrindersFunc: func(ctx context.Context) ([]*arabica.Grinder, error) { return nil, nil },
		ListRoastersFunc: func(ctx context.Context) ([]*arabica.Roaster, error) {
			return []*arabica.Roaster{{RKey: "r1", Name: "Onyx"}}, nil
		},
	}

	props, err := buildGetStartedCardProps(context.Background(), store)

	assert.NoError(t, err)
	assert.True(t, props.Readiness.Ready())
	assert.Len(t, props.Beans, 1)
	assert.Len(t, props.Brewers, 1)
	assert.Len(t, props.Roasters, 1)
	assert.Equal(t, "Ethiopia", props.Beans[0].Name)
}
