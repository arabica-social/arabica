package handlers

import (
	"context"
	"net/http"

	atp "tangled.org/pdewey.com/atp"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/entities/oolong"
	"tangled.org/arabica.social/arabica/internal/lexicons"
	teapages "tangled.org/arabica.social/arabica/internal/oolong/web/pages"
	"tangled.org/arabica.social/arabica/internal/web/components"
	"tangled.org/arabica.social/arabica/internal/web/pages"
)

// Oolong entity view handlers. Each builds an entityViewConfig that
// converts witness/PDS/store reads into the appropriate typed oolong
// model and renders the matching teapages templ. fromStore uses the
// generic AtprotoStore.FetchRecord since oolong doesn't have per-entity
// Get*ByRKey wrappers yet — those will land alongside CRUD in a later
// phase.

func (h *Handler) teaViewConfig() entityViewConfig {
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeOolongTea),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			t, err := oolong.RecordToTea(m, uri)
			if err != nil {
				return nil, err
			}
			t.RKey = rkey
			return t, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			t, err := oolong.RecordToTea(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			t.RKey = rkey
			return t, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			m, uri, cid, err := s.FetchRecord(ctx, oolong.NSIDTea, rkey)
			if err != nil {
				return nil, "", "", err
			}
			t, err := oolong.RecordToTea(m, uri)
			if err != nil {
				return nil, "", "", err
			}
			t.RKey = rkey
			return t, uri, cid, nil
		},
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
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeOolongVendor),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			v, err := oolong.RecordToVendor(m, uri)
			if err != nil {
				return nil, err
			}
			v.RKey = rkey
			return v, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			v, err := oolong.RecordToVendor(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			v.RKey = rkey
			return v, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			m, uri, cid, err := s.FetchRecord(ctx, oolong.NSIDVendor, rkey)
			if err != nil {
				return nil, "", "", err
			}
			v, err := oolong.RecordToVendor(m, uri)
			if err != nil {
				return nil, "", "", err
			}
			v.RKey = rkey
			return v, uri, cid, nil
		},
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

func (h *Handler) oolongBrewerViewConfig() entityViewConfig {
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeOolongBrewer),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			b, err := oolong.RecordToBrewer(m, uri)
			if err != nil {
				return nil, err
			}
			b.RKey = rkey
			return b, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			b, err := oolong.RecordToBrewer(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			b.RKey = rkey
			return b, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			m, uri, cid, err := s.FetchRecord(ctx, oolong.NSIDBrewer, rkey)
			if err != nil {
				return nil, "", "", err
			}
			b, err := oolong.RecordToBrewer(m, uri)
			if err != nil {
				return nil, "", "", err
			}
			b.RKey = rkey
			return b, uri, cid, nil
		},
		displayName: func(record any) string { return record.(*oolong.Brewer).Name },
		ogSubtitle:  func(record any) string { return record.(*oolong.Brewer).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.BrewerView(layoutData, teapages.BrewerViewProps{
				Brewer:         record.(*oolong.Brewer),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

func (h *Handler) oolongRecipeViewConfig() entityViewConfig {
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeOolongRecipe),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			r, err := oolong.RecordToRecipe(m, uri)
			if err != nil {
				return nil, err
			}
			r.RKey = rkey
			return r, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			r, err := oolong.RecordToRecipe(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			r.RKey = rkey
			return r, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			m, uri, cid, err := s.FetchRecord(ctx, oolong.NSIDRecipe, rkey)
			if err != nil {
				return nil, "", "", err
			}
			r, err := oolong.RecordToRecipe(m, uri)
			if err != nil {
				return nil, "", "", err
			}
			r.RKey = rkey
			return r, uri, cid, nil
		},
		displayName: func(record any) string { return record.(*oolong.Recipe).Name },
		ogSubtitle:  func(record any) string { return record.(*oolong.Recipe).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.RecipeView(layoutData, teapages.RecipeViewProps{
				Recipe:         record.(*oolong.Recipe),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

func (h *Handler) oolongBrewViewConfig() entityViewConfig {
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeOolongBrew),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			b, err := oolong.RecordToBrew(m, uri)
			if err != nil {
				return nil, err
			}
			b.RKey = rkey
			return b, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			b, err := oolong.RecordToBrew(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			b.RKey = rkey
			return b, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			m, uri, cid, err := s.FetchRecord(ctx, oolong.NSIDBrew, rkey)
			if err != nil {
				return nil, "", "", err
			}
			b, err := oolong.RecordToBrew(m, uri)
			if err != nil {
				return nil, "", "", err
			}
			b.RKey = rkey
			return b, uri, cid, nil
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

func (h *Handler) oolongCafeViewConfig() entityViewConfig {
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeOolongCafe),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			c, err := oolong.RecordToCafe(m, uri)
			if err != nil {
				return nil, err
			}
			c.RKey = rkey
			return c, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			c, err := oolong.RecordToCafe(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			c.RKey = rkey
			return c, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			m, uri, cid, err := s.FetchRecord(ctx, oolong.NSIDCafe, rkey)
			if err != nil {
				return nil, "", "", err
			}
			c, err := oolong.RecordToCafe(m, uri)
			if err != nil {
				return nil, "", "", err
			}
			c.RKey = rkey
			return c, uri, cid, nil
		},
		displayName: func(record any) string { return record.(*oolong.Cafe).Name },
		ogSubtitle:  func(record any) string { return record.(*oolong.Cafe).Name },
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.CafeView(layoutData, teapages.CafeViewProps{
				Cafe:           record.(*oolong.Cafe),
				EntityViewBase: base,
			}).Render(ctx, w)
		},
	}
}

func (h *Handler) oolongDrinkViewConfig() entityViewConfig {
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeOolongDrink),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			d, err := oolong.RecordToDrink(m, uri)
			if err != nil {
				return nil, err
			}
			d.RKey = rkey
			return d, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			d, err := oolong.RecordToDrink(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			d.RKey = rkey
			return d, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			m, uri, cid, err := s.FetchRecord(ctx, oolong.NSIDDrink, rkey)
			if err != nil {
				return nil, "", "", err
			}
			d, err := oolong.RecordToDrink(m, uri)
			if err != nil {
				return nil, "", "", err
			}
			d.RKey = rkey
			return d, uri, cid, nil
		},
		displayName: func(record any) string {
			d := record.(*oolong.Drink)
			if d.Name != "" {
				return d.Name
			}
			if d.Tea != nil && d.Tea.Name != "" {
				return d.Tea.Name
			}
			return "Tea Drink"
		},
		ogSubtitle: func(record any) string {
			d := record.(*oolong.Drink)
			if d.Name != "" {
				return d.Name
			}
			return "Tea Drink"
		},
		render: func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base pages.EntityViewBase) error {
			return teapages.DrinkView(layoutData, teapages.DrinkViewProps{
				Drink:          record.(*oolong.Drink),
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

func (h *Handler) HandleOolongBrewerView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.oolongBrewerViewConfig())
}

func (h *Handler) HandleOolongRecipeView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.oolongRecipeViewConfig())
}

func (h *Handler) HandleOolongBrewView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.oolongBrewViewConfig())
}

func (h *Handler) HandleOolongCafeView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.oolongCafeViewConfig())
}

func (h *Handler) HandleOolongDrinkView(w http.ResponseWriter, r *http.Request) {
	h.handleEntityView(w, r, h.oolongDrinkViewConfig())
}
