// Package arabicaapp constructs the Arabica product configuration.
package arabicaapp

import (
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"
)

func New() *domain.App {
	return &domain.App{
		Name:        "arabica",
		NSIDBase:    arabica.NSIDBase,
		Descriptors: entities.AllForApp(arabica.NSIDBase),
		Brand: domain.BrandConfig{
			DisplayName: "Arabica",
			Tagline:     "Your brew, your data",
		},
	}
}
