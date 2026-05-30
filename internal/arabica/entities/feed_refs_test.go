package arabica

import (
	"maps"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testDID = "did:plc:abcdefghijklmnopqrstuvwxyz"

func testURI(collection, rkey string) string {
	return "at://" + testDID + "/" + collection + "/" + rkey
}

func createdRecord(fields map[string]any) map[string]any {
	record := map[string]any{"createdAt": time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)}
	maps.Copy(record, fields)
	return record
}

func TestHydrateBeanRefsHydratesRoaster(t *testing.T) {
	roasterURI := testURI(NSIDRoaster, "roaster1")
	bean := &Bean{Name: "Ethiopia"}

	HydrateBeanRefs(bean, createdRecord(map[string]any{"name": "Ethiopia", "roasterRef": roasterURI}), func(ref string) (map[string]any, bool) {
		return createdRecord(map[string]any{"name": "Little Wolf"}), ref == roasterURI
	})

	assert.Equal(t, "roaster1", bean.RoasterRKey)
	assert.NotNil(t, bean.Roaster)
	assert.Equal(t, "Little Wolf", bean.Roaster.Name)
	assert.Equal(t, "roaster1", bean.Roaster.RKey)
}

func TestHydrateRecipeRefsHydratesBrewer(t *testing.T) {
	brewerURI := testURI(NSIDBrewer, "brewer1")
	recipe := &Recipe{Name: "V60"}

	HydrateRecipeRefs(recipe, createdRecord(map[string]any{"name": "V60", "brewerRef": brewerURI}), func(ref string) (map[string]any, bool) {
		return createdRecord(map[string]any{"name": "Hario", "brewerType": BrewerTypePourover}), ref == brewerURI
	})

	assert.Equal(t, "brewer1", recipe.BrewerRKey)
	assert.NotNil(t, recipe.BrewerObj)
	assert.Equal(t, "Hario", recipe.BrewerObj.Name)
	assert.Equal(t, BrewerTypePourover, recipe.BrewerType)
}

func TestHydrateBrewRefsHydratesNestedReferences(t *testing.T) {
	beanURI := testURI(NSIDBean, "bean1")
	roasterURI := testURI(NSIDRoaster, "roaster1")
	grinderURI := testURI(NSIDGrinder, "grinder1")
	brewerURI := testURI(NSIDBrewer, "brewer1")
	recipeURI := testURI(NSIDRecipe, "recipe1")
	recipeBrewerURI := testURI(NSIDBrewer, "brewer2")
	lookupRecords := map[string]map[string]any{
		beanURI:         createdRecord(map[string]any{"name": "Ethiopia", "roasterRef": roasterURI}),
		roasterURI:      createdRecord(map[string]any{"name": "Little Wolf"}),
		grinderURI:      createdRecord(map[string]any{"name": "C40"}),
		brewerURI:       createdRecord(map[string]any{"name": "Origami", "brewerType": BrewerTypePourover}),
		recipeURI:       createdRecord(map[string]any{"name": "Daily", "brewerRef": recipeBrewerURI}),
		recipeBrewerURI: createdRecord(map[string]any{"name": "Kalita", "brewerType": BrewerTypePourover}),
	}
	brew := &Brew{}

	HydrateBrewRefs(brew, createdRecord(map[string]any{
		"beanRef":    beanURI,
		"grinderRef": grinderURI,
		"brewerRef":  brewerURI,
		"recipeRef":  recipeURI,
	}), func(ref string) (map[string]any, bool) {
		record, ok := lookupRecords[ref]
		return record, ok
	})

	assert.Equal(t, "bean1", brew.BeanRKey)
	assert.Equal(t, "grinder1", brew.GrinderRKey)
	assert.Equal(t, "brewer1", brew.BrewerRKey)
	assert.Equal(t, "recipe1", brew.RecipeRKey)
	assert.Equal(t, "Ethiopia", brew.Bean.Name)
	assert.Equal(t, "Little Wolf", brew.Bean.Roaster.Name)
	assert.Equal(t, "C40", brew.GrinderObj.Name)
	assert.Equal(t, "Origami", brew.BrewerObj.Name)
	assert.Equal(t, "Daily", brew.RecipeObj.Name)
	assert.Equal(t, "Kalita", brew.RecipeObj.BrewerObj.Name)
}

func TestHydrateBrewRefsSkipsMissingReferences(t *testing.T) {
	assert.NotPanics(t, func() {
		HydrateBrewRefs(&Brew{}, createdRecord(map[string]any{"beanRef": testURI(NSIDBean, "missing")}), func(string) (map[string]any, bool) {
			return nil, false
		})
	})
}
