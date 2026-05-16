package atproto

import (
	"context"

	atp "tangled.org/pdewey.com/atp"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"
)

// Codec registrations for arabica entities riding the generic CRUD
// helpers in store_entity.go. Brew keeps a bespoke implementation
// because its List path uses bulk witness-batch ref resolution that
// the per-item PostList hook can't express.

var (
	roasterCodec = &EntityCodec[arabica.Roaster]{
		NSID:       arabica.NSIDRoaster,
		ToRecord:   func(_ *AtprotoStore, m *arabica.Roaster) (any, error) { return arabica.RoasterToRecord(m) },
		FromRecord: arabica.RecordToRoaster,
		SetRKey:    func(m *arabica.Roaster, rkey string) { m.RKey = rkey },
	}

	grinderCodec = &EntityCodec[arabica.Grinder]{
		NSID:       arabica.NSIDGrinder,
		ToRecord:   func(_ *AtprotoStore, m *arabica.Grinder) (any, error) { return arabica.GrinderToRecord(m) },
		FromRecord: arabica.RecordToGrinder,
		SetRKey:    func(m *arabica.Grinder, rkey string) { m.RKey = rkey },
	}

	brewerCodec = &EntityCodec[arabica.Brewer]{
		NSID:       arabica.NSIDBrewer,
		ToRecord:   func(_ *AtprotoStore, m *arabica.Brewer) (any, error) { return arabica.BrewerToRecord(m) },
		FromRecord: arabica.RecordToBrewer,
		SetRKey:    func(m *arabica.Brewer, rkey string) { m.RKey = rkey },
	}

	beanCodec = &EntityCodec[arabica.Bean]{
		NSID: arabica.NSIDBean,
		ToRecord: func(s *AtprotoStore, m *arabica.Bean) (any, error) {
			var roasterURI string
			if m.RoasterRKey != "" {
				roasterURI = atp.BuildATURI(s.did.String(), arabica.NSIDRoaster, m.RoasterRKey)
			}
			return arabica.BeanToRecord(m, roasterURI)
		},
		FromRecord: arabica.RecordToBean,
		SetRKey:    func(m *arabica.Bean, rkey string) { m.RKey = rkey },
		PostGet: func(ctx context.Context, s *AtprotoStore, m *arabica.Bean, rec map[string]any) {
			s.resolveBeanRefs(ctx, m, rec)
		},
		PostList: extractBeanRoasterRKey,
	}

	recipeCodec = &EntityCodec[arabica.Recipe]{
		NSID: arabica.NSIDRecipe,
		ToRecord: func(s *AtprotoStore, m *arabica.Recipe) (any, error) {
			var brewerURI string
			if m.BrewerRKey != "" {
				brewerURI = atp.BuildATURI(s.did.String(), arabica.NSIDBrewer, m.BrewerRKey)
			}
			return arabica.RecipeToRecord(m, brewerURI)
		},
		FromRecord: arabica.RecordToRecipe,
		SetRKey:    func(m *arabica.Recipe, rkey string) { m.RKey = rkey },
		PostGet: func(ctx context.Context, s *AtprotoStore, m *arabica.Recipe, rec map[string]any) {
			s.resolveRecipeRefs(ctx, m, rec)
		},
		PostList: extractRecipeBrewerRKey,
	}
)

