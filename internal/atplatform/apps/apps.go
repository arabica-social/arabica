// Package apps constructs the product configurations served by the shared
// AT-platform bootstrap.
package apps

import (
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"
	oolong "tangled.org/arabica.social/arabica/internal/oolong/entities"

	// Blank imports run init() in each app's web/components package, wiring
	// per-app templ render hooks onto descriptors.
	_ "tangled.org/arabica.social/arabica/internal/arabica/web/components"
	_ "tangled.org/arabica.social/arabica/internal/oolong/web/components"
)

// NewArabica builds the App value for the coffee-tracking app.
func NewArabica() *domain.App {
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

// NewOolong builds the App value for the tea-tracking app.
func NewOolong() *domain.App {
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
