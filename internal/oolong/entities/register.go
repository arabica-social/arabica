package oolong

import (
	"tangled.org/arabica.social/arabica/internal/entities"
	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func init() {
	registerTea()
	registerVendor()
	registerVessel()
	registerInfuser()
	registerBrew()

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

func registerTea() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeOolongTea, NSID: NSIDTea, DisplayName: "Tea"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeOolongTea, &entities.RecordBehavior{
		GetField: teaField,
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
		ResolveRefs: resolveTeaFeedRef,
	})
}

func registerVendor() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeOolongVendor, NSID: NSIDVendor, DisplayName: "Tea Vendor"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeOolongVendor, &entities.RecordBehavior{
		GetField: vendorField,
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
}

func registerVessel() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeOolongVessel, NSID: NSIDVessel, DisplayName: "Vessel"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeOolongVessel, &entities.RecordBehavior{
		GetField: vesselField,
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
}

func registerInfuser() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeOolongInfuser, NSID: NSIDInfuser, DisplayName: "Infuser"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeOolongInfuser, &entities.RecordBehavior{
		GetField: infuserField,
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
}

func registerBrew() {
	entities.Register(&entities.Descriptor{Type: lexicons.RecordTypeOolongBrew, NSID: NSIDBrew, DisplayName: "Tea Brew"})
	entities.RegisterRecordBehavior(lexicons.RecordTypeOolongBrew, &entities.RecordBehavior{
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
		ResolveRefs: resolveBrewFeedRefs,
	})
}
