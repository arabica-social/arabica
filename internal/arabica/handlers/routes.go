package coffeehandlers

import (
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// EntityRouteBundles returns the per-entity handler bundles for arabica's
// simple entities (bean, roaster, grinder, brewer). Recipe and brew have
// additional endpoints and stay registered explicitly in routing.go.
func (h *Handlers) EntityRouteBundles() []handlers.EntityRouteBundle {
	return []handlers.EntityRouteBundle{
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
