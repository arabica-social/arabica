package handlers

import (
	"net/http"

	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// EntityRouteBundle groups the HTTP handlers that comprise one entity's
// public surface (CRUD, view page, OG image, modal partials). routing.go
// loops over the bundle returned by EntityRouteBundles to register the
// per-entity routes uniformly, making it cheap for a sister app like
// matcha to ship its own bundle without duplicating the route-wiring
// logic in cmd/{arabica,matcha}/routing.
//
// Bundles cover the entities whose routes are structurally identical:
// bean, roaster, grinder, brewer. Recipe and brew have additional
// endpoints (recipes/from-brew, recipes/fork, brews/{id}/edit,
// brews/{id}/export) and stay registered explicitly.
//
// A nil handler in a bundle field means "no route for this slot" — the
// router skips it. Today every simple-entity bundle populates every
// slot, but the optional shape lets matcha (or future arabica work)
// declare entities without an OG image or without modal partials.
type EntityRouteBundle struct {
	// RecordType identifies the entity. Used to look up the descriptor
	// when registering routes; the descriptor's URLPath becomes the URL
	// segment (/api/{URLPath}, /{URLPath}/{id}, etc.).
	RecordType lexicons.RecordType

	// CRUD over JSON. Create has no rkey path-parameter; Update/Delete
	// take {id} from the path.
	Create http.HandlerFunc
	Update http.HandlerFunc
	Delete http.HandlerFunc

	// View renders the public entity detail page (/{URLPath}/{id}).
	View http.HandlerFunc

	// OGImage serves the entity's OpenGraph image (/{URLPath}/{id}/og-image).
	OGImage http.HandlerFunc

	// Modal partials return dialog HTML for create / edit flows.
	ModalNew  http.HandlerFunc
	ModalEdit http.HandlerFunc
}

// EntityRouteBundles returns the per-entity handler bundles for the
// simple entities whose routing layout is uniform. See
// EntityRouteBundle for the contract and exclusions.
func (h *Handler) EntityRouteBundles() []EntityRouteBundle {
	return []EntityRouteBundle{
		{
			RecordType: lexicons.RecordTypeBean,
			Create:     h.HandleBeanCreate,
			Update:     h.HandleBeanUpdate,
			Delete:     h.HandleBeanDelete,
			View:       h.HandleBeanView,
			OGImage:    h.HandleBeanOGImage,
			ModalNew:   h.HandleBeanModalNew,
			ModalEdit:  h.HandleBeanModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeRoaster,
			Create:     h.HandleRoasterCreate,
			Update:     h.HandleRoasterUpdate,
			Delete:     h.HandleRoasterDelete,
			View:       h.HandleRoasterView,
			OGImage:    h.HandleRoasterOGImage,
			ModalNew:   h.HandleRoasterModalNew,
			ModalEdit:  h.HandleRoasterModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeGrinder,
			Create:     h.HandleGrinderCreate,
			Update:     h.HandleGrinderUpdate,
			Delete:     h.HandleGrinderDelete,
			View:       h.HandleGrinderView,
			OGImage:    h.HandleGrinderOGImage,
			ModalNew:   h.HandleGrinderModalNew,
			ModalEdit:  h.HandleGrinderModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeBrewer,
			Create:     h.HandleBrewerCreate,
			Update:     h.HandleBrewerUpdate,
			Delete:     h.HandleBrewerDelete,
			View:       h.HandleBrewerView,
			OGImage:    h.HandleBrewerOGImage,
			ModalNew:   h.HandleBrewerModalNew,
			ModalEdit:  h.HandleBrewerModalEdit,
		},
	}
}

