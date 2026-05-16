package handlers

import (
	"context"
	"net/http"

	"github.com/bluesky-social/indigo/atproto/syntax"
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
// model and renders the matching teapages templ.

func (h *Handler) teaViewConfig() entityViewConfig {
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeOolongTea),
		fromWitness: func(ctx context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			t, err := oolong.RecordToTea(m, uri)
			if err != nil {
				return nil, err
			}
			t.RKey = rkey
			h.resolveOolongTeaVendor(ctx, t, m)
			return t, nil
		},
		fromPDS: func(ctx context.Context, e *atp.Record, rkey, _ string) (any, error) {
			t, err := oolong.RecordToTea(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			t.RKey = rkey
			h.resolveOolongTeaVendor(ctx, t, e.Value)
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
			h.resolveOolongTeaVendor(ctx, t, m)
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

func (h *Handler) oolongVesselViewConfig() entityViewConfig {
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeOolongVessel),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			v, err := oolong.RecordToVessel(m, uri)
			if err != nil {
				return nil, err
			}
			v.RKey = rkey
			return v, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			v, err := oolong.RecordToVessel(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			v.RKey = rkey
			return v, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			m, uri, cid, err := s.FetchRecord(ctx, oolong.NSIDVessel, rkey)
			if err != nil {
				return nil, "", "", err
			}
			v, err := oolong.RecordToVessel(m, uri)
			if err != nil {
				return nil, "", "", err
			}
			v.RKey = rkey
			return v, uri, cid, nil
		},
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
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeOolongInfuser),
		fromWitness: func(_ context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			i, err := oolong.RecordToInfuser(m, uri)
			if err != nil {
				return nil, err
			}
			i.RKey = rkey
			return i, nil
		},
		fromPDS: func(_ context.Context, e *atp.Record, rkey, _ string) (any, error) {
			i, err := oolong.RecordToInfuser(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			i.RKey = rkey
			return i, nil
		},
		fromStore: func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error) {
			m, uri, cid, err := s.FetchRecord(ctx, oolong.NSIDInfuser, rkey)
			if err != nil {
				return nil, "", "", err
			}
			i, err := oolong.RecordToInfuser(m, uri)
			if err != nil {
				return nil, "", "", err
			}
			i.RKey = rkey
			return i, uri, cid, nil
		},
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
	return entityViewConfig{
		descriptor: entities.Get(lexicons.RecordTypeOolongBrew),
		fromWitness: func(ctx context.Context, m map[string]any, uri, rkey, _ string) (any, error) {
			b, err := oolong.RecordToBrew(m, uri)
			if err != nil {
				return nil, err
			}
			b.RKey = rkey
			h.resolveOolongBrewReferences(ctx, b, m)
			return b, nil
		},
		fromPDS: func(ctx context.Context, e *atp.Record, rkey, _ string) (any, error) {
			b, err := oolong.RecordToBrew(e.Value, e.URI)
			if err != nil {
				return nil, err
			}
			b.RKey = rkey
			h.resolveOolongBrewReferences(ctx, b, e.Value)
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
			h.resolveOolongBrewReferences(ctx, b, m)
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

// resolveOolongBrewReferences hydrates tea (with nested vendor), vessel and
// infuser on an oolong brew. Tries the witness cache first, falls back to a
// public PDS read for any ref not in the cache.
func (h *Handler) resolveOolongBrewReferences(ctx context.Context, brew *oolong.Brew, record map[string]any) {
	teaRef, _ := record["teaRef"].(string)
	vesselRef, _ := record["vesselRef"].(string)
	infuserRef, _ := record["infuserRef"].(string)

	var pub *atp.PublicClient
	getPub := func() *atp.PublicClient {
		if pub == nil {
			pub = atproto.NewPublicClient()
		}
		return pub
	}

	if teaRef != "" {
		teaMap, teaURI := h.fetchOolongRefMap(ctx, teaRef, getPub)
		if teaMap != nil {
			if tea, err := oolong.RecordToTea(teaMap, teaURI); err == nil {
				tea.RKey = atp.RKeyFromURI(teaURI)
				brew.Tea = tea
				if vendorRef, _ := teaMap["vendorRef"].(string); vendorRef != "" {
					vendorMap, vendorURI := h.fetchOolongRefMap(ctx, vendorRef, getPub)
					if vendorMap != nil {
						if vendor, err := oolong.RecordToVendor(vendorMap, vendorURI); err == nil {
							vendor.RKey = atp.RKeyFromURI(vendorURI)
							brew.Tea.Vendor = vendor
						}
					}
				}
			}
		}
	}

	if vesselRef != "" {
		m, uri := h.fetchOolongRefMap(ctx, vesselRef, getPub)
		if m != nil {
			if v, err := oolong.RecordToVessel(m, uri); err == nil {
				v.RKey = atp.RKeyFromURI(uri)
				brew.Vessel = v
			}
		}
	}

	if infuserRef != "" {
		m, uri := h.fetchOolongRefMap(ctx, infuserRef, getPub)
		if m != nil {
			if i, err := oolong.RecordToInfuser(m, uri); err == nil {
				i.RKey = atp.RKeyFromURI(uri)
				brew.Infuser = i
			}
		}
	}
}

// resolveOolongTeaVendor hydrates the vendor on a tea record from the witness
// cache or PDS.
func (h *Handler) resolveOolongTeaVendor(ctx context.Context, tea *oolong.Tea, record map[string]any) {
	vendorRef, _ := record["vendorRef"].(string)
	if vendorRef == "" {
		return
	}
	pub := atproto.NewPublicClient()
	m, uri := h.fetchOolongRefMap(ctx, vendorRef, func() *atp.PublicClient { return pub })
	if m == nil {
		return
	}
	if v, err := oolong.RecordToVendor(m, uri); err == nil {
		v.RKey = atp.RKeyFromURI(uri)
		tea.Vendor = v
	}
}

// fetchOolongRefMap returns the record map and resolved URI for an AT-URI
// reference, trying the witness cache first and falling back to a public PDS
// read. Returns (nil, "") on any error.
func (h *Handler) fetchOolongRefMap(ctx context.Context, atURI string, getPub func() *atp.PublicClient) (map[string]any, string) {
	if h.witnessCache != nil {
		if wr, _ := h.witnessCache.GetWitnessRecord(ctx, atURI); wr != nil {
			if m, err := atproto.WitnessRecordToMap(wr); err == nil {
				return m, wr.URI
			}
		}
	}
	parsed, err := syntax.ParseATURI(atURI)
	if err != nil {
		return nil, ""
	}
	did := parsed.Authority().String()
	collection := parsed.Collection().String()
	rkey := parsed.RecordKey().String()
	rec, err := getPub().GetPublicRecord(ctx, did, collection, rkey)
	if err != nil || rec == nil {
		return nil, ""
	}
	return rec.Value, rec.URI
}
