package handlers

import (
	"context"
	"net/http"

	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/entities/oolong"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	teapages "tangled.org/arabica.social/arabica/internal/oolong/web/pages"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/arabica.social/arabica/internal/web/pages"
)

// Oolong entity view handlers. Each builds an entityViewConfig that
// converts witness/PDS/store reads into the appropriate typed oolong
// model and renders the matching teapages templ. The shared from*
// triple is produced by standardViewTriple; entity-specific ref
// resolution lives in resolveRefs.

func (h *Handler) teaViewConfig() entityViewConfig {
	fromWitness, fromPDS, fromStore := standardViewTriple(
		oolong.NSIDTea, oolong.RecordToTea,
		func(t *oolong.Tea, r string) { t.RKey = r },
	)
	return entityViewConfig{
		descriptor:  entities.Get(lexicons.RecordTypeOolongTea),
		fromWitness: fromWitness,
		fromPDS:     fromPDS,
		fromStore:   fromStore,
		displayName: func(record any) string { return record.(*oolong.Tea).Name },
		ogSubtitle:  func(record any) string { return record.(*oolong.Tea).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.TeaView(layoutData, teapages.TeaViewProps{
				Tea:            record.(*oolong.Tea),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

func (h *Handler) oolongVendorViewConfig() entityViewConfig {
	fromWitness, fromPDS, fromStore := standardViewTriple(
		oolong.NSIDVendor, oolong.RecordToVendor,
		func(v *oolong.Vendor, r string) { v.RKey = r },
	)
	return entityViewConfig{
		descriptor:  entities.Get(lexicons.RecordTypeOolongVendor),
		fromWitness: fromWitness,
		fromPDS:     fromPDS,
		fromStore:   fromStore,
		displayName: func(record any) string { return record.(*oolong.Vendor).Name },
		ogSubtitle:  func(record any) string { return record.(*oolong.Vendor).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.VendorView(layoutData, teapages.VendorViewProps{
				Vendor:         record.(*oolong.Vendor),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

func (h *Handler) oolongVesselViewConfig() entityViewConfig {
	fromWitness, fromPDS, fromStore := standardViewTriple(
		oolong.NSIDVessel, oolong.RecordToVessel,
		func(v *oolong.Vessel, r string) { v.RKey = r },
	)
	return entityViewConfig{
		descriptor:  entities.Get(lexicons.RecordTypeOolongVessel),
		fromWitness: fromWitness,
		fromPDS:     fromPDS,
		fromStore:   fromStore,
		displayName: func(record any) string { return record.(*oolong.Vessel).Name },
		ogSubtitle:  func(record any) string { return record.(*oolong.Vessel).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.VesselView(layoutData, teapages.VesselViewProps{
				Vessel:         record.(*oolong.Vessel),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

func (h *Handler) oolongInfuserViewConfig() entityViewConfig {
	fromWitness, fromPDS, fromStore := standardViewTriple(
		oolong.NSIDInfuser, oolong.RecordToInfuser,
		func(i *oolong.Infuser, r string) { i.RKey = r },
	)
	return entityViewConfig{
		descriptor:  entities.Get(lexicons.RecordTypeOolongInfuser),
		fromWitness: fromWitness,
		fromPDS:     fromPDS,
		fromStore:   fromStore,
		displayName: func(record any) string { return record.(*oolong.Infuser).Name },
		ogSubtitle:  func(record any) string { return record.(*oolong.Infuser).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.InfuserView(layoutData, teapages.InfuserViewProps{
				Infuser:        record.(*oolong.Infuser),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

func (h *Handler) oolongBrewViewConfig() entityViewConfig {
	fromWitness, fromPDS, fromStore := standardViewTriple(
		oolong.NSIDBrew, oolong.RecordToBrew,
		func(b *oolong.Brew, r string) { b.RKey = r },
	)
	return entityViewConfig{
		descriptor:  entities.Get(lexicons.RecordTypeOolongBrew),
		fromWitness: fromWitness,
		fromPDS:     fromPDS,
		fromStore:   fromStore,
		resolveRefs: func(_ context.Context, model any, raw map[string]any, lookup func(string) (map[string]any, bool)) {
			b := model.(*oolong.Brew)
			if b.Tea != nil {
				return
			}
			teaRef, _ := raw["teaRef"].(string)
			if teaRef == "" {
				return
			}
			m, ok := lookup(teaRef)
			if !ok {
				return
			}
			if tea, err := oolong.RecordToTea(m, teaRef); err == nil {
				b.Tea = tea
			}
		},
		displayName: func(record any) string {
			b := record.(*oolong.Brew)
			if b.Tea != nil && b.Tea.Name != "" {
				return b.Tea.Name
			}
			return "Tea Brew"
		},
		ogSubtitle: func(record any) string {
			b := record.(*oolong.Brew)
			if b.Tea != nil && b.Tea.Name != "" {
				return b.Tea.Name
			}
			return "Tea Brew"
		},
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.BrewView(layoutData, teapages.BrewViewProps{
				Brew:           record.(*oolong.Brew),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

// HTTP entry points.

func (h *Handler) HandleTeaView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.teaViewConfig())
}

func (h *Handler) HandleOolongVendorView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.oolongVendorViewConfig())
}

func (h *Handler) HandleOolongVesselView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.oolongVesselViewConfig())
}

func (h *Handler) HandleOolongInfuserView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.oolongInfuserViewConfig())
}

func (h *Handler) HandleOolongBrewView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.oolongBrewViewConfig())
}
