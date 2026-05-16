package main

import (
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/entities/oolong"

	// Blank import runs init() in oolong's web/components package,
	// wiring per-app templ render hooks onto descriptors.
	_ "tangled.org/arabica.social/arabica/internal/oolong/web/components"
)

// newOolongApp builds the App value for the tea-tracking app.
func newOolongApp() *domain.App {
	return &domain.App{
		Name:           "oolong",
		NSIDBase:       oolong.NSIDBase,
		SocialNSIDBase: oolong.SocialNSIDBase,
		Descriptors:    entities.AllForApp(oolong.NSIDBase),
		Brand: domain.BrandConfig{
			DisplayName: "Oolong",
			Tagline:     "Your tea, your data",
		},
	}
}
