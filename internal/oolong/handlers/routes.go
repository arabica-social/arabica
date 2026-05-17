package teahandlers

import (
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

// EntityRouteBundles returns the per-entity handler bundles for oolong's
// public surface. Mirrors the shared
// (*handlers.Handler).EntityRouteBundles contract; see entity_routes.go
// for the per-field semantics.
//
// Tea uses a full-page form (/teas/new + /teas/{id}/edit) rather than a
// modal partial, so its ModalNew/ModalEdit slots stay nil — the route
// registrar skips nil slots.
func (h *Handlers) EntityRouteBundles() []handlers.EntityRouteBundle {
	return []handlers.EntityRouteBundle{
		{
			RecordType: lexicons.RecordTypeOolongTea,
			Create:     h.HandleTeaCreate,
			Update:     h.HandleTeaUpdate,
			Delete:     h.HandleTeaDelete,
			View:       h.HandleTeaView,
		},
		{
			RecordType: lexicons.RecordTypeOolongVendor,
			Create:     h.HandleOolongVendorCreate,
			Update:     h.HandleOolongVendorUpdate,
			Delete:     h.HandleOolongVendorDelete,
			View:       h.HandleOolongVendorView,
			ModalNew:   h.HandleOolongVendorModalNew,
			ModalEdit:  h.HandleOolongVendorModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeOolongVessel,
			Create:     h.HandleOolongVesselCreate,
			Update:     h.HandleOolongVesselUpdate,
			Delete:     h.HandleOolongVesselDelete,
			View:       h.HandleOolongVesselView,
			ModalNew:   h.HandleOolongVesselModalNew,
			ModalEdit:  h.HandleOolongVesselModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeOolongInfuser,
			Create:     h.HandleOolongInfuserCreate,
			Update:     h.HandleOolongInfuserUpdate,
			Delete:     h.HandleOolongInfuserDelete,
			View:       h.HandleOolongInfuserView,
			ModalNew:   h.HandleOolongInfuserModalNew,
			ModalEdit:  h.HandleOolongInfuserModalEdit,
		},
		{
			RecordType: lexicons.RecordTypeOolongBrew,
			Create:     h.HandleOolongBrewCreate,
			Update:     h.HandleOolongBrewUpdate,
			Delete:     h.HandleOolongBrewDelete,
			View:       h.HandleOolongBrewView,
			ModalNew:   h.HandleOolongBrewModalNew,
			ModalEdit:  h.HandleOolongBrewModalEdit,
		},
	}
}
