package main

import (
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
)

// newArabicaApp builds the App value for the arabica binary. It pulls
// descriptors from the global entities registry — populated by
// internal/entities/arabica's init() (imported here for both the
// NSIDBase value and that side effect) — and pairs them with the
// arabica NSID base.
func newArabicaApp() *domain.App {
	return &domain.App{
		Name:        "arabica",
		NSIDBase:    arabica.NSIDBase,
		Descriptors: entities.All(),
		Brand: domain.BrandConfig{
			DisplayName: "Arabica",
			Tagline:     "Your brew, your data",
		},
	}
}
