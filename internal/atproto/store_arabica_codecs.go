package atproto

import (
	"tangled.org/arabica.social/arabica/internal/entities/arabica"
)

// Codec registrations for arabica entities that ride the generic CRUD
// helpers in store_entity.go. Entities with cross-record reference
// resolution (Bean, Brew, Recipe) keep bespoke methods because their
// Get/List paths interleave witness lookups for nested refs.

var (
	roasterCodec = &EntityCodec[arabica.Roaster]{
		NSID:       arabica.NSIDRoaster,
		ToRecord:   func(m *arabica.Roaster) (any, error) { return arabica.RoasterToRecord(m) },
		FromRecord: arabica.RecordToRoaster,
		SetRKey:    func(m *arabica.Roaster, rkey string) { m.RKey = rkey },
	}

	grinderCodec = &EntityCodec[arabica.Grinder]{
		NSID:       arabica.NSIDGrinder,
		ToRecord:   func(m *arabica.Grinder) (any, error) { return arabica.GrinderToRecord(m) },
		FromRecord: arabica.RecordToGrinder,
		SetRKey:    func(m *arabica.Grinder, rkey string) { m.RKey = rkey },
	}

	brewerCodec = &EntityCodec[arabica.Brewer]{
		NSID:       arabica.NSIDBrewer,
		ToRecord:   func(m *arabica.Brewer) (any, error) { return arabica.BrewerToRecord(m) },
		FromRecord: arabica.RecordToBrewer,
		SetRKey:    func(m *arabica.Brewer, rkey string) { m.RKey = rkey },
	}
)
