package handlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// EntityRouteBundle groups the HTTP handlers that comprise one entity's
// public surface (CRUD, view page, OG image, modal partials). routing.go
// loops over the bundle returned by EntityRouteBundles to register the
// per-entity routes uniformly, making it cheap for a sister app like
// oolong to ship its own bundle without duplicating the route-wiring
// logic in cmd/{arabica,oolong}/routing.
//
// Bundles cover the entities whose routes are structurally identical:
// bean, roaster, grinder, brewer. Recipe and brew have additional
// endpoints (recipes/from-brew, recipes/fork, brews/{id}/edit,
// brews/{id}/export) and stay registered explicitly.
//
// A nil handler in a bundle field means "no route for this slot" — the
// router skips it. Today every simple-entity bundle populates every
// slot, but the optional shape lets oolong (or future arabica work)
// declare entities without an OG image or without modal partials.
type EntityRouteBundle struct {
	// RecordType identifies the entity. Routing combines this with the active
	// app's EntityRoute metadata to choose URL and modal path segments.
	RecordType lexicons.RecordType

	// CRUD over JSON. Create has no rkey path-parameter; Update/Delete
	// take {id} from the path.
	Create http.HandlerFunc
	Update http.HandlerFunc
	Delete http.HandlerFunc

	// View renders the public entity detail page.
	View http.HandlerFunc

	// Backlinks renders the community backlinks detail page for this entity.
	Backlinks http.HandlerFunc

	// OGImage serves the entity's OpenGraph image.
	OGImage http.HandlerFunc

	// Modal partials return dialog HTML for create / edit flows.
	ModalNew  http.HandlerFunc
	ModalEdit http.HandlerFunc
}

// Per-app entity bundles live in their own packages:
//   - arabica: coffeehandlers.(*Handlers).EntityRouteBundles
//   - oolong:  teahandlers.(*Handlers).EntityRouteBundles
//
// Cafe and Drink bundles are deferred for v1; their descriptors are not
// registered so registerEntityRoutes skips them.
