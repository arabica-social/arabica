package teahandlers

import (
	"net/http"

	"github.com/a-h/templ"

	oolong "tangled.org/arabica.social/arabica/internal/oolong/entities"
	tea "tangled.org/arabica.social/arabica/internal/oolong/web/components"
)

// Modal-partial handlers for the oolong entities. New variants render
// an empty modal; Edit variants fetch the existing record via the
// generic AtprotoStore.FetchRecord + oolong.RecordTo* decoder, then
// render the modal pre-filled.
//
// Tea uses a full-page form (HandleOolongTeaNew/Edit) instead of a
// modal — see internal/oolong/web/pages/tea_form.templ.

// --- Vendor ----------------------------------------------------------

func (h *Handlers) HandleOolongVendorModalNew(w http.ResponseWriter, r *http.Request) {
	h.oolongModalNew(w, r, "vendor", func() templ.Component { return tea.VendorDialogModal(nil) })
}

func (h *Handlers) HandleOolongVendorModalEdit(w http.ResponseWriter, r *http.Request) {
	oolongModalEdit(h, w, r, oolong.NSIDVendor, "vendor",
		oolong.RecordToVendor,
		func(v *oolong.Vendor, rkey string) { v.RKey = rkey },
		func(v *oolong.Vendor) templ.Component { return tea.VendorDialogModal(v) },
	)
}

// --- Vessel ----------------------------------------------------------

func (h *Handlers) HandleOolongVesselModalNew(w http.ResponseWriter, r *http.Request) {
	h.oolongModalNew(w, r, "vessel", func() templ.Component { return tea.VesselDialogModal(nil) })
}

func (h *Handlers) HandleOolongVesselModalEdit(w http.ResponseWriter, r *http.Request) {
	oolongModalEdit(h, w, r, oolong.NSIDVessel, "vessel",
		oolong.RecordToVessel,
		func(v *oolong.Vessel, rkey string) { v.RKey = rkey },
		func(v *oolong.Vessel) templ.Component { return tea.VesselDialogModal(v) },
	)
}

// --- Infuser ---------------------------------------------------------

func (h *Handlers) HandleOolongInfuserModalNew(w http.ResponseWriter, r *http.Request) {
	h.oolongModalNew(w, r, "infuser", func() templ.Component { return tea.InfuserDialogModal(nil) })
}

func (h *Handlers) HandleOolongInfuserModalEdit(w http.ResponseWriter, r *http.Request) {
	oolongModalEdit(h, w, r, oolong.NSIDInfuser, "infuser",
		oolong.RecordToInfuser,
		func(i *oolong.Infuser, rkey string) { i.RKey = rkey },
		func(i *oolong.Infuser) templ.Component { return tea.InfuserDialogModal(i) },
	)
}

// --- Brew (steep) ----------------------------------------------------

func (h *Handlers) HandleOolongBrewModalNew(w http.ResponseWriter, r *http.Request) {
	h.oolongModalNew(w, r, "tea brew", func() templ.Component { return tea.BrewDialogModal(nil) })
}

func (h *Handlers) HandleOolongBrewModalEdit(w http.ResponseWriter, r *http.Request) {
	oolongModalEdit(h, w, r, oolong.NSIDBrew, "tea brew",
		oolong.RecordToBrew,
		func(b *oolong.Brew, rkey string) { b.RKey = rkey },
		func(b *oolong.Brew) templ.Component { return tea.BrewDialogModal(b) },
	)
}

// Cafe and Drink modal handlers are deferred for v1.
