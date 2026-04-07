package integration

import (
	"encoding/json"
	"net/url"
	"testing"

	"arabica/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fetchData fetches /api/data and unmarshals into the generic data envelope.
// Used by lifecycle tests to verify a record's post-update or post-delete
// state through the same code path that the JS cache uses.
func fetchData(t *testing.T, h *Harness) listAllResponse {
	t.Helper()
	resp := h.Get("/api/data")
	body := ReadBody(t, resp)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, body))

	var data listAllResponse
	require.NoError(t, json.Unmarshal([]byte(body), &data))
	return data
}

type listAllResponse struct {
	Beans    []models.Bean    `json:"beans"`
	Roasters []models.Roaster `json:"roasters"`
	Grinders []models.Grinder `json:"grinders"`
	Brewers  []models.Brewer  `json:"brewers"`
	Recipes  []models.Recipe  `json:"recipes"`
	Brews    []models.Brew    `json:"brews"`
}

// TestHTTP_BeanLifecycle covers POST → PUT → DELETE for beans, including the
// roaster reference field that's unique to bean update.
func TestHTTP_BeanLifecycle(t *testing.T) {
	h := StartHarness(t, nil)

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Lifecycle Roaster")), "roaster")
	otherRoasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Other Roaster")), "roaster")

	createResp := h.PostForm("/api/beans", form(
		"name", "Lifecycle Bean",
		"origin", "Ethiopia",
		"roaster_rkey", roasterRKey,
		"roast_level", "Light",
	))
	beanRKey := mustRKey(t, createResp, "bean")

	// Update: change name + swap roasters.
	updateResp := h.PutForm("/api/beans/"+beanRKey, form(
		"name", "Lifecycle Bean v2",
		"origin", "Kenya",
		"roaster_rkey", otherRoasterRKey,
		"roast_level", "Medium",
	))
	require.Equal(t, 200, updateResp.StatusCode, statusErr(updateResp, ReadBody(t, updateResp)))

	data := fetchData(t, h)
	var found *models.Bean
	for i := range data.Beans {
		if data.Beans[i].RKey == beanRKey {
			found = &data.Beans[i]
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "Lifecycle Bean v2", found.Name)
	assert.Equal(t, "Kenya", found.Origin)
	assert.Equal(t, otherRoasterRKey, found.RoasterRKey)

	// Delete.
	delResp := h.Delete("/api/beans/" + beanRKey)
	require.Equal(t, 200, delResp.StatusCode, statusErr(delResp, ReadBody(t, delResp)))

	data = fetchData(t, h)
	for _, b := range data.Beans {
		assert.NotEqual(t, beanRKey, b.RKey)
	}
}

// TestHTTP_GrinderLifecycle covers POST → PUT → DELETE for grinders.
func TestHTTP_GrinderLifecycle(t *testing.T) {
	h := StartHarness(t, nil)

	rkey := mustRKey(t, h.PostForm("/api/grinders", form(
		"name", "Original Grinder",
		"grinder_type", "Manual",
		"burr_type", "Conical",
	)), "grinder")

	updateResp := h.PutForm("/api/grinders/"+rkey, form(
		"name", "Updated Grinder",
		"grinder_type", "Electric",
		"burr_type", "Flat",
		"notes", "upgraded",
	))
	require.Equal(t, 200, updateResp.StatusCode, statusErr(updateResp, ReadBody(t, updateResp)))

	data := fetchData(t, h)
	var found *models.Grinder
	for i := range data.Grinders {
		if data.Grinders[i].RKey == rkey {
			found = &data.Grinders[i]
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "Updated Grinder", found.Name)
	assert.Equal(t, "Electric", found.GrinderType)
	assert.Equal(t, "Flat", found.BurrType)

	delResp := h.Delete("/api/grinders/" + rkey)
	require.Equal(t, 200, delResp.StatusCode, statusErr(delResp, ReadBody(t, delResp)))

	data = fetchData(t, h)
	for _, g := range data.Grinders {
		assert.NotEqual(t, rkey, g.RKey)
	}
}

// TestHTTP_BrewerLifecycle covers POST → PUT → DELETE for brewers.
func TestHTTP_BrewerLifecycle(t *testing.T) {
	h := StartHarness(t, nil)

	rkey := mustRKey(t, h.PostForm("/api/brewers", form(
		"name", "V60",
		"brewer_type", "Pour Over",
	)), "brewer")

	updateResp := h.PutForm("/api/brewers/"+rkey, form(
		"name", "V60 Plastic",
		"brewer_type", "Pour Over",
		"description", "02 size, plastic body",
	))
	require.Equal(t, 200, updateResp.StatusCode, statusErr(updateResp, ReadBody(t, updateResp)))

	data := fetchData(t, h)
	var found *models.Brewer
	for i := range data.Brewers {
		if data.Brewers[i].RKey == rkey {
			found = &data.Brewers[i]
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "V60 Plastic", found.Name)
	assert.Equal(t, "02 size, plastic body", found.Description)

	delResp := h.Delete("/api/brewers/" + rkey)
	require.Equal(t, 200, delResp.StatusCode, statusErr(delResp, ReadBody(t, delResp)))

	data = fetchData(t, h)
	for _, b := range data.Brewers {
		assert.NotEqual(t, rkey, b.RKey)
	}
}

// TestHTTP_RecipeLifecycle covers POST → PUT → DELETE for recipes, including
// the brewer reference and pours field.
func TestHTTP_RecipeLifecycle(t *testing.T) {
	h := StartHarness(t, nil)

	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Recipe Brewer", "brewer_type", "Pour Over")), "brewer")

	createResp := h.PostForm("/api/recipes", form(
		"name", "Original Recipe",
		"brewer_rkey", brewerRKey,
		"brewer_type", "Pour Over",
		"coffee_amount", "18",
		"water_amount", "300",
		"notes", "v1",
		"pour_water_0", "60",
		"pour_time_0", "0",
		"pour_water_1", "240",
		"pour_time_1", "45",
	))
	recipeRKey := mustRKey(t, createResp, "recipe")

	updateResp := h.PutForm("/api/recipes/"+recipeRKey, form(
		"name", "Updated Recipe",
		"brewer_rkey", brewerRKey,
		"brewer_type", "Pour Over",
		"coffee_amount", "20",
		"water_amount", "320",
		"notes", "v2",
		"pour_water_0", "60",
		"pour_time_0", "0",
		"pour_water_1", "130",
		"pour_time_1", "45",
		"pour_water_2", "130",
		"pour_time_2", "90",
	))
	require.Equal(t, 200, updateResp.StatusCode, statusErr(updateResp, ReadBody(t, updateResp)))

	data := fetchData(t, h)
	var found *models.Recipe
	for i := range data.Recipes {
		if data.Recipes[i].RKey == recipeRKey {
			found = &data.Recipes[i]
		}
	}
	require.NotNil(t, found)
	assert.Equal(t, "Updated Recipe", found.Name)
	assert.Equal(t, 20.0, found.CoffeeAmount)
	assert.Equal(t, "v2", found.Notes)
	assert.Len(t, found.Pours, 3, "expected three pours after update")

	delResp := h.Delete("/api/recipes/" + recipeRKey)
	require.Equal(t, 200, delResp.StatusCode, statusErr(delResp, ReadBody(t, delResp)))

	data = fetchData(t, h)
	for _, r := range data.Recipes {
		assert.NotEqual(t, recipeRKey, r.RKey)
	}
}

// TestHTTP_BrewLifecycle covers POST → PUT → DELETE for brews. The update path
// is the most complex: it re-marshals references, pours, and method-specific
// params. This test starts with a pourover brew and updates it to espresso to
// exercise method-param swapping (the EspressoParams marshaling path that
// TestHTTP_BrewCreatePourover doesn't reach).
func TestHTTP_BrewLifecycle(t *testing.T) {
	h := StartHarness(t, nil)

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Brew LC Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Brew LC Bean",
		"roaster_rkey", roasterRKey,
		"roast_level", "Medium",
	)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "LC Grinder", "grinder_type", "Electric")), "grinder")
	pourBrewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "LC V60", "brewer_type", "Pour Over")), "brewer")
	espBrewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "LC Espresso", "brewer_type", "Espresso")), "brewer")

	// Create as pourover.
	createForm := url.Values{}
	createForm.Set("bean_rkey", beanRKey)
	createForm.Set("grinder_rkey", grinderRKey)
	createForm.Set("brewer_rkey", pourBrewerRKey)
	createForm.Set("method", "Pour Over")
	createForm.Set("water_amount", "300")
	createForm.Set("coffee_amount", "18")
	createForm.Set("time_seconds", "210")
	createForm.Set("rating", "7")
	createForm.Set("pour_water_0", "60")
	createForm.Set("pour_time_0", "0")
	createForm.Set("pour_water_1", "240")
	createForm.Set("pour_time_1", "45")
	createForm.Set("pourover_bloom_water", "60")
	createForm.Set("pourover_bloom_seconds", "30")

	createResp := h.PostForm("/brews", createForm)
	require.Equal(t, 200, createResp.StatusCode, statusErr(createResp, ReadBody(t, createResp)))

	data := fetchData(t, h)
	require.Len(t, data.Brews, 1)
	brewRKey := data.Brews[0].RKey
	require.NotEmpty(t, brewRKey)

	// Update to espresso: drop pours + pourover params, add espresso params,
	// swap brewer.
	updateForm := url.Values{}
	updateForm.Set("bean_rkey", beanRKey)
	updateForm.Set("grinder_rkey", grinderRKey)
	updateForm.Set("brewer_rkey", espBrewerRKey)
	updateForm.Set("method", "Espresso")
	updateForm.Set("water_amount", "36")
	updateForm.Set("coffee_amount", "18")
	updateForm.Set("time_seconds", "28")
	updateForm.Set("rating", "9")
	updateForm.Set("tasting_notes", "syrupy, chocolate")
	updateForm.Set("espresso_yield_weight", "36")
	updateForm.Set("espresso_pressure", "9")
	updateForm.Set("espresso_pre_infusion_seconds", "5")

	updateResp := h.PutForm("/brews/"+brewRKey, updateForm)
	require.Equal(t, 200, updateResp.StatusCode, statusErr(updateResp, ReadBody(t, updateResp)))

	data = fetchData(t, h)
	require.Len(t, data.Brews, 1)
	updated := data.Brews[0]
	assert.Equal(t, "Espresso", updated.Method)
	assert.Equal(t, espBrewerRKey, updated.BrewerRKey)
	assert.Equal(t, 9, updated.Rating)
	assert.Equal(t, 36, updated.WaterAmount)
	assert.Equal(t, "syrupy, chocolate", updated.TastingNotes)
	require.NotNil(t, updated.EspressoParams, "espresso params should be present after update")
	assert.Equal(t, 36.0, updated.EspressoParams.YieldWeight)
	assert.Equal(t, 9.0, updated.EspressoParams.Pressure)
	assert.Equal(t, 5, updated.EspressoParams.PreInfusionSeconds)

	// Delete.
	delResp := h.Delete("/brews/" + brewRKey)
	require.Equal(t, 200, delResp.StatusCode, statusErr(delResp, ReadBody(t, delResp)))

	data = fetchData(t, h)
	assert.Empty(t, data.Brews, "brew should be gone after delete")
}
