package main

import (
	"strings"

	"tangled.org/arabica.social/arabica/internal/atplatform/domain"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/entities/oolong"

	// Blank imports run init() in each app's web/components package, which
	// wires per-app templ render hooks onto descriptors.
	_ "tangled.org/arabica.social/arabica/internal/arabica/web/components"
	_ "tangled.org/arabica.social/arabica/internal/oolong/web/components"
)

// teaAppName is the single source of truth for the tea-tracking sister
// app's identity. Changing this constant renames everything: the
// app.Name (used as the env-var prefix and data-dir component), the
// NSIDBase, the brand display name, and the APPS env-var value the
// user sets to select it.
//
// Renaming checklist when teaAppName changes:
//  1. Bump this constant.
//  2. Rename internal/entities/<old>/ if a tea entities tree exists.
//  3. Update the oolong section of nix/module.nix accordingly.
const teaAppName = "oolong"

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

// newTeaApp builds the App value for the tea-tracking sister app.
// Descriptors is intentionally empty: tea lexicons live in a tree that
// hasn't been authored yet (see docs/tea-multitenant-refactor.md).
// Once they exist they slot in via a registry import here, identical
// to how arabica wires entities/arabica.
func newTeaApp() *domain.App {
	return &domain.App{
		Name:        teaAppName,
		NSIDBase:    oolong.NSIDBase,
		Descriptors: entities.AllForApp(oolong.NSIDBase),
		Brand: domain.BrandConfig{
			DisplayName: strings.ToUpper(teaAppName[:1]) + teaAppName[1:],
			Tagline:     "Your tea, your data",
		},
	}
}
