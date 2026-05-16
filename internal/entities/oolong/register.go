package oolong

import (
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func init() {
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongTea,
		NSID:            NSIDTea,
		DisplayName:     "Tea",
		Noun:            "tea",
		URLPath:         "teas",
		FeedFilterLabel: "Teas",
		GetField:        teaField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToTea(rec, uri)
		},
		RKey: func(rec any) string {
			t, _ := rec.(*Tea)
			if t == nil {
				return ""
			}
			return t.RKey
		},
		DisplayTitle: func(rec any) string {
			t, _ := rec.(*Tea)
			if t == nil {
				return ""
			}
			return t.Name
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongVendor,
		NSID:            NSIDVendor,
		DisplayName:     "Tea Vendor",
		Noun:            "vendor",
		URLPath:         "vendors",
		FeedFilterLabel: "Vendors",
		GetField:        vendorField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToVendor(rec, uri)
		},
		RKey: func(rec any) string {
			v, _ := rec.(*Vendor)
			if v == nil {
				return ""
			}
			return v.RKey
		},
		DisplayTitle: func(rec any) string {
			v, _ := rec.(*Vendor)
			if v == nil {
				return ""
			}
			return v.Name
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongVessel,
		NSID:            NSIDVessel,
		DisplayName:     "Vessel",
		Noun:            "vessel",
		URLPath:         "vessels",
		FeedFilterLabel: "Vessels",
		GetField:        vesselField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToVessel(rec, uri)
		},
		RKey: func(rec any) string {
			v, _ := rec.(*Vessel)
			if v == nil {
				return ""
			}
			return v.RKey
		},
		DisplayTitle: func(rec any) string {
			v, _ := rec.(*Vessel)
			if v == nil {
				return ""
			}
			return v.Name
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongInfuser,
		NSID:            NSIDInfuser,
		DisplayName:     "Infuser",
		Noun:            "infuser",
		URLPath:         "infusers",
		FeedFilterLabel: "Infusers",
		GetField:        infuserField,
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToInfuser(rec, uri)
		},
		RKey: func(rec any) string {
			i, _ := rec.(*Infuser)
			if i == nil {
				return ""
			}
			return i.RKey
		},
		DisplayTitle: func(rec any) string {
			i, _ := rec.(*Infuser)
			if i == nil {
				return ""
			}
			return i.Name
		},
	})
	entities.Register(&entities.Descriptor{
		Type:            lexicons.RecordTypeOolongBrew,
		NSID:            NSIDBrew,
		DisplayName:     "Tea Brew",
		Noun:            "brew",
		URLPath:         "brews",
		FeedFilterLabel: "Brews",
		GetField:        nil, // brew has no edit modal that needs prefill
		RecordToModel: func(rec map[string]any, uri string) (any, error) {
			return RecordToBrew(rec, uri)
		},
		RKey: func(rec any) string {
			b, _ := rec.(*Brew)
			if b == nil {
				return ""
			}
			return b.RKey
		},
		DisplayTitle: func(rec any) string {
			b, _ := rec.(*Brew)
			if b == nil {
				return ""
			}
			// Brew has no name; fall back to the associated tea's name.
			if b.Tea != nil && b.Tea.Name != "" {
				return b.Tea.Name
			}
			return "Tea Brew"
		},
	})

	// Cafe and Drink are deferred for the v1 launch. Their models and
	// record conversions remain in tree but are intentionally not
	// registered as descriptors, so they don't appear in the oolong
	// feed, manage UI, or OAuth scopes. Re-enable when the cafe/drink
	// experience is ready to ship.

	// Comment and Like are intentionally NOT registered.
	// App.NSIDs() in internal/atplatform/domain/app.go appends them
	// unconditionally using NSIDBase. Registering them as descriptors
	// would produce duplicates.
}
