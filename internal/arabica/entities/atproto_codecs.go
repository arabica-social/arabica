package arabica

import (
	"context"
	"fmt"

	"tangled.org/arabica.social/arabica/internal/atproto/storecodec"
	atp "tangled.org/pdewey.com/atp"
)

type didStore interface {
	DID() string
}

type beanRefResolver interface {
	ResolveBeanRefs(context.Context, *Bean, map[string]any)
}

type recipeRefResolver interface {
	ResolveRecipeRefs(context.Context, *Recipe, map[string]any)
}

var (
	AtprotoRoasterCodec = &storecodec.EntityCodec[Roaster]{
		NSID:       NSIDRoaster,
		ToRecord:   func(_ any, m *Roaster) (any, error) { return RoasterToRecord(m) },
		FromRecord: RecordToRoaster,
		SetRKey:    func(m *Roaster, rkey string) { m.RKey = rkey },
	}

	AtprotoGrinderCodec = &storecodec.EntityCodec[Grinder]{
		NSID:       NSIDGrinder,
		ToRecord:   func(_ any, m *Grinder) (any, error) { return GrinderToRecord(m) },
		FromRecord: RecordToGrinder,
		SetRKey:    func(m *Grinder, rkey string) { m.RKey = rkey },
	}

	AtprotoBrewerCodec = &storecodec.EntityCodec[Brewer]{
		NSID:       NSIDBrewer,
		ToRecord:   func(_ any, m *Brewer) (any, error) { return BrewerToRecord(m) },
		FromRecord: RecordToBrewer,
		SetRKey:    func(m *Brewer, rkey string) { m.RKey = rkey },
	}

	AtprotoBeanCodec = &storecodec.EntityCodec[Bean]{
		NSID:       NSIDBean,
		ToRecord:   beanToAtprotoRecord,
		FromRecord: RecordToBean,
		SetRKey:    func(m *Bean, rkey string) { m.RKey = rkey },
		PostGet: func(ctx context.Context, store any, m *Bean, rec map[string]any) {
			if resolver, ok := store.(beanRefResolver); ok {
				resolver.ResolveBeanRefs(ctx, m, rec)
				return
			}
			ExtractBeanRoasterRKey(m, rec)
		},
		PostList: ExtractBeanRoasterRKey,
	}

	AtprotoRecipeCodec = &storecodec.EntityCodec[Recipe]{
		NSID:       NSIDRecipe,
		ToRecord:   recipeToAtprotoRecord,
		FromRecord: RecordToRecipe,
		SetRKey:    func(m *Recipe, rkey string) { m.RKey = rkey },
		PostGet: func(ctx context.Context, store any, m *Recipe, rec map[string]any) {
			if resolver, ok := store.(recipeRefResolver); ok {
				resolver.ResolveRecipeRefs(ctx, m, rec)
				return
			}
			ExtractRecipeBrewerRKey(m, rec)
		},
		PostList: ExtractRecipeBrewerRKey,
	}
)

func beanToAtprotoRecord(store any, m *Bean) (any, error) {
	var roasterURI string
	if m.RoasterRKey != "" {
		did, err := ownerDID(store)
		if err != nil {
			return nil, err
		}
		roasterURI = atp.BuildATURI(did, NSIDRoaster, m.RoasterRKey)
	}
	return BeanToRecord(m, roasterURI)
}

func recipeToAtprotoRecord(store any, m *Recipe) (any, error) {
	var brewerURI string
	if m.BrewerRKey != "" {
		did, err := ownerDID(store)
		if err != nil {
			return nil, err
		}
		brewerURI = atp.BuildATURI(did, NSIDBrewer, m.BrewerRKey)
	}
	return RecipeToRecord(m, brewerURI)
}

func ownerDID(store any) (string, error) {
	didStore, ok := store.(didStore)
	if !ok {
		return "", fmt.Errorf("store does not expose DID")
	}
	return didStore.DID(), nil
}

func ExtractBeanRoasterRKey(bean *Bean, record map[string]any) {
	roasterRef, ok := record["roasterRef"].(string)
	if !ok || roasterRef == "" {
		return
	}
	if rkey := atp.RKeyFromURI(roasterRef); rkey != "" {
		bean.RoasterRKey = rkey
	}
}

func ExtractRecipeBrewerRKey(recipe *Recipe, record map[string]any) {
	brewerRef, ok := record["brewerRef"].(string)
	if !ok || brewerRef == "" {
		return
	}
	if rkey := atp.RKeyFromURI(brewerRef); rkey != "" {
		recipe.BrewerRKey = rkey
	}
}
