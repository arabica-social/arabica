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
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return []*arabica.Bean{{RKey: "a"}}, nil },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "b"}}, nil },
	}

	assert.True(t, brewNewReady(context.Background(), store))
}

func TestBrewNewReady_FalseWhenMissing(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return nil, nil },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return []*arabica.Brewer{{RKey: "b"}}, nil },
	}

	assert.False(t, brewNewReady(context.Background(), store))
}

func TestBrewNewReady_FalseOnError(t *testing.T) {
	store := &database.MockStore{
		ListBeansFunc:   func(ctx context.Context) ([]*arabica.Bean, error) { return nil, errors.New("pds down") },
		ListBrewersFunc: func(ctx context.Context) ([]*arabica.Brewer, error) { return nil, nil },
	}

	assert.False(t, brewNewReady(context.Background(), store))
}
