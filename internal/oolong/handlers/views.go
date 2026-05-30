package teahandlers

import (
	"context"
	"net/http"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	oolong "tangled.org/arabica.social/arabica/internal/oolong/entities"
	teapages "tangled.org/arabica.social/arabica/internal/oolong/web/pages"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/arabica.social/arabica/internal/web/pages"
)

// Oolong entity view handlers. Each builds an handlers.EntityViewConfig that
// converts witness/PDS/store reads into the appropriate typed oolong
// model and renders the matching teapages templ. The shared from*
// triple is produced by handlers.StandardViewTriple; entity-specific ref
// resolution lives in resolveRefs.

func (h *Handlers) teaViewConfig() handlers.EntityViewConfig {
	fromWitness, fromPDS, fromStore := handlers.StandardViewTriple(
		oolong.NSIDTea, oolong.RecordToTea,
		func(t *oolong.Tea, r string) { t.RKey = r },
	)
	return handlers.EntityViewConfig{
		Descriptor:  entities.Get(lexicons.RecordTypeOolongTea),
		FromWitness: fromWitness,
		FromPDS:     fromPDS,
		FromStore:   fromStore,
		ResolveRefs: func(_ context.Context, model any, raw map[string]any, lookup func(string) (map[string]any, bool)) {
			oolong.HydrateTeaRefs(model.(*oolong.Tea), raw, lookup)
		},
		DisplayName: func(record any) string { return record.(*oolong.Tea).Name },
		OGSubtitle:  func(record any) string { return record.(*oolong.Tea).Name },
		Render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.TeaView(layoutData, teapages.TeaViewProps{
				Tea:            record.(*oolong.Tea),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

func (h *Handlers) oolongVendorViewConfig() handlers.EntityViewConfig {
	fromWitness, fromPDS, fromStore := handlers.StandardViewTriple(
		oolong.NSIDVendor, oolong.RecordToVendor,
		func(v *oolong.Vendor, r string) { v.RKey = r },
	)
	return handlers.EntityViewConfig{
		Descriptor:  entities.Get(lexicons.RecordTypeOolongVendor),
		FromWitness: fromWitness,
		FromPDS:     fromPDS,
		FromStore:   fromStore,
		DisplayName: func(record any) string { return record.(*oolong.Vendor).Name },
		OGSubtitle:  func(record any) string { return record.(*oolong.Vendor).Name },
		Render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.VendorView(layoutData, teapages.VendorViewProps{
				Vendor:         record.(*oolong.Vendor),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

func (h *Handlers) oolongVesselViewConfig() handlers.EntityViewConfig {
	fromWitness, fromPDS, fromStore := handlers.StandardViewTriple(
		oolong.NSIDVessel, oolong.RecordToVessel,
		func(v *oolong.Vessel, r string) { v.RKey = r },
	)
	return handlers.EntityViewConfig{
		Descriptor:  entities.Get(lexicons.RecordTypeOolongVessel),
		FromWitness: fromWitness,
		FromPDS:     fromPDS,
		FromStore:   fromStore,
		DisplayName: func(record any) string { return record.(*oolong.Vessel).Name },
		OGSubtitle:  func(record any) string { return record.(*oolong.Vessel).Name },
		Render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.VesselView(layoutData, teapages.VesselViewProps{
				Vessel:         record.(*oolong.Vessel),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

func (h *Handlers) oolongInfuserViewConfig() handlers.EntityViewConfig {
	fromWitness, fromPDS, fromStore := handlers.StandardViewTriple(
		oolong.NSIDInfuser, oolong.RecordToInfuser,
		func(i *oolong.Infuser, r string) { i.RKey = r },
	)
	return handlers.EntityViewConfig{
		Descriptor:  entities.Get(lexicons.RecordTypeOolongInfuser),
		FromWitness: fromWitness,
		FromPDS:     fromPDS,
		FromStore:   fromStore,
		DisplayName: func(record any) string { return record.(*oolong.Infuser).Name },
		OGSubtitle:  func(record any) string { return record.(*oolong.Infuser).Name },
		Render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.InfuserView(layoutData, teapages.InfuserViewProps{
				Infuser:        record.(*oolong.Infuser),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

func (h *Handlers) oolongBrewViewConfig() handlers.EntityViewConfig {
	fromWitness, fromPDS, fromStore := handlers.StandardViewTriple(
		oolong.NSIDBrew, oolong.RecordToBrew,
		func(b *oolong.Brew, r string) { b.RKey = r },
	)
	return handlers.EntityViewConfig{
		Descriptor:  entities.Get(lexicons.RecordTypeOolongBrew),
		FromWitness: fromWitness,
		FromPDS:     fromPDS,
		FromStore:   fromStore,
		ResolveRefs: func(_ context.Context, model any, raw map[string]any, lookup func(string) (map[string]any, bool)) {
			oolong.HydrateBrewRefs(model.(*oolong.Brew), raw, lookup)
		},
		DisplayName: func(record any) string {
			b := record.(*oolong.Brew)
			if b.Tea != nil && b.Tea.Name != "" {
				return b.Tea.Name
			}
			return "Tea Brew"
		},
		OGSubtitle: func(record any) string {
			b := record.(*oolong.Brew)
			if b.Tea != nil && b.Tea.Name != "" {
				return b.Tea.Name
			}
			return "Tea Brew"
		},
		Render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.BrewView(layoutData, teapages.BrewViewProps{
				Brew:           record.(*oolong.Brew),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

// HTTP entry points.

func (h *Handlers) HandleTeaView(w http.ResponseWriter, r *http.Request) {
	h.RenderEntityView(w, r, h.teaViewConfig())
}

func (h *Handlers) HandleOolongVendorView(w http.ResponseWriter, r *http.Request) {
	h.RenderEntityView(w, r, h.oolongVendorViewConfig())
}

func (h *Handlers) HandleOolongVesselView(w http.ResponseWriter, r *http.Request) {
	h.RenderEntityView(w, r, h.oolongVesselViewConfig())
}

func (h *Handlers) HandleOolongInfuserView(w http.ResponseWriter, r *http.Request) {
	h.RenderEntityView(w, r, h.oolongInfuserViewConfig())
}

func (h *Handlers) HandleOolongBrewView(w http.ResponseWriter, r *http.Request) {
	h.RenderEntityView(w, r, h.oolongBrewViewConfig())
}
