package main

import (
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"

	// Blank import runs init() in arabica's web/components package,
	// wiring per-app templ render hooks onto descriptors.
	_ "tangled.org/arabica.social/arabica/internal/arabica/web/components"
)

// newArabicaApp builds the App value for the coffee-tracking app.
func newArabicaApp() *domain.App {
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
