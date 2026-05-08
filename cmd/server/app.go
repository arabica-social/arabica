package main

import (
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities"
)

// newArabicaApp builds the App value for the arabica binary. It pulls
// descriptors from the global entities registry (populated at init time)
// and pairs them with the arabica NSID base. Subsequent phases of the
// tea-multitenant refactor will move construction into internal/arabica/.
func newArabicaApp() *domain.App {
	return &domain.App{
		Name:        "arabica",
		NSIDBase:    atproto.NSIDBase,
		Descriptors: entities.All(),
		Brand: domain.BrandConfig{
			DisplayName: "Arabica",
			Tagline:     "Your brew, your data",
		},
	}
}
