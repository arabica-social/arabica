// Package oolongapp constructs the Oolong product configuration.
package oolongapp

import (
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"
	oolong "tangled.org/arabica.social/arabica/internal/oolong/entities"
)

func New() *domain.App {
	return &domain.App{
		Name:        "oolong",
		NSIDBase:    oolong.NSIDBase,
		Descriptors: entities.AllForApp(oolong.NSIDBase),
		Brand: domain.BrandConfig{
			DisplayName: "Oolong",
			Tagline:     "Your tea, your data",
		},
	}
}
