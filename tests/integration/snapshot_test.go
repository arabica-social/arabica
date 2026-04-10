package integration

import (
	"encoding/json"
	"net/url"
	"testing"

	"tangled.org/arabica.social/arabica/internal/atproto"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/require"
)

// scrubPDS returns shutter options that replace PDS-generated dynamic values
// (AT-URIs containing DIDs and rkeys, timestamps) with stable placeholders so
// snapshots are deterministic across runs.
func scrubPDS(did string, rkeys map[string]string) []shutter.Option {
	opts := []shutter.Option{
		shutter.ScrubTimestamp(),
		shutter.ScrubRegex(`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}[^\s"]*`, "<TIMESTAMP>"),
		shutter.ScrubExact(did, "<DID>"),
	}
	for label, rkey := range rkeys {
		opts = append(opts, shutter.ScrubExact(rkey, "<RKEY:"+label+">"))
	}
	return opts
}

// snapPDSRecord is a helper that fetches a record directly from the PDS and
// snapshots it. This verifies the raw AT Protocol record shape, bypassing all
// arabica caching and model conversion.
func snapPDSRecord(t *testing.T, h *Harness, title, collection, rkey string, rkeys map[string]string) {
	t.Helper()
	raw := h.PDSGetRecord(h.PrimaryAccount, collection, rkey)
	b, err := json.MarshalIndent(raw, "", "  ")
	require.NoError(t, err)
	shutter.SnapJSON(t, title, string(b),
		scrubPDS(h.PrimaryAccount.DID, rkeys)...,
	)
}

// snapPDSCollection is a helper that lists all records in a collection directly
// from the PDS and snapshots them.
func snapPDSCollection(t *testing.T, h *Harness, title, collection string, rkeys map[string]string) {
	t.Helper()
	records := h.PDSListRecords(h.PrimaryAccount, collection)
	b, err := json.MarshalIndent(records, "", "  ")
	require.NoError(t, err)
	shutter.SnapJSON(t, title, string(b),
		scrubPDS(h.PrimaryAccount.DID, rkeys)...,
	)
}

// --- Roaster ---

func TestSnap_PDS_RoasterCreate(t *testing.T) {
	h := StartHarness(t, nil)

	rkey := mustRKey(t, h.PostForm("/api/roasters", form(
		"name", "Counter Culture",
		"location", "Durham, NC",
		"website", "https://counterculturecoffee.com",
	)), "roaster")

	snapPDSRecord(t, h, "roaster record", atproto.NSIDRoaster, rkey, map[string]string{
		"roaster": rkey,
	})
}

func TestSnap_PDS_RoasterUpdate(t *testing.T) {
	h := StartHarness(t, nil)

	rkey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Sey Coffee", "location", "Brooklyn, NY")), "roaster")

	updateResp := h.PutForm("/api/roasters/"+rkey, form(
		"name", "Sey Coffee Roasters",
		"location", "Brooklyn, NY",
		"website", "https://seycoffee.com",
	))
	require.Equal(t, 200, updateResp.StatusCode, statusErr(updateResp, ReadBody(t, updateResp)))

	snapPDSRecord(t, h, "roaster after update", atproto.NSIDRoaster, rkey, map[string]string{
		"roaster": rkey,
	})
}

// --- Bean with roaster reference ---

func TestSnap_PDS_BeanWithRoaster(t *testing.T) {
	h := StartHarness(t, nil)

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Onyx Coffee Lab")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Geometry",
		"origin", "Colombia",
		"roast_level", "Medium",
		"process", "Washed",
		"variety", "Caturra",
		"roaster_rkey", roasterRKey,
	)), "bean")

	snapPDSRecord(t, h, "bean with roaster ref", atproto.NSIDBean, beanRKey, map[string]string{
		"roaster": roasterRKey,
		"bean":    beanRKey,
	})
}

// --- Grinder ---

func TestSnap_PDS_GrinderCreate(t *testing.T) {
	h := StartHarness(t, nil)

	rkey := mustRKey(t, h.PostForm("/api/grinders", form(
		"name", "Comandante C40",
		"grinder_type", "Manual",
		"burr_type", "Conical",
		"notes", "red clix installed",
	)), "grinder")

	snapPDSRecord(t, h, "grinder record", atproto.NSIDGrinder, rkey, map[string]string{
		"grinder": rkey,
	})
}

// --- Brewer ---

func TestSnap_PDS_BrewerCreate(t *testing.T) {
	h := StartHarness(t, nil)

	rkey := mustRKey(t, h.PostForm("/api/brewers", form(
		"name", "Hario V60 02",
		"brewer_type", "Pour Over",
		"description", "Plastic, size 02",
	)), "brewer")

	snapPDSRecord(t, h, "brewer record", atproto.NSIDBrewer, rkey, map[string]string{
		"brewer": rkey,
	})
}

// --- Recipe with pours ---

func TestSnap_PDS_RecipeWithPours(t *testing.T) {
	h := StartHarness(t, nil)

	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "V60", "brewer_type", "Pour Over")), "brewer")
	recipeRKey := mustRKey(t, h.PostForm("/api/recipes", form(
		"name", "4:6 Method",
		"brewer_rkey", brewerRKey,
		"brewer_type", "Pour Over",
		"coffee_amount", "20",
		"water_amount", "300",
		"notes", "Tetsu Kasuya 4:6",
		"pour_water_0", "50",
		"pour_time_0", "0",
		"pour_water_1", "70",
		"pour_time_1", "45",
		"pour_water_2", "60",
		"pour_time_2", "90",
		"pour_water_3", "60",
		"pour_time_3", "120",
		"pour_water_4", "60",
		"pour_time_4", "150",
	)), "recipe")

	snapPDSRecord(t, h, "recipe with pours", atproto.NSIDRecipe, recipeRKey, map[string]string{
		"brewer": brewerRKey,
		"recipe": recipeRKey,
	})
}

// --- Pourover brew (full references + pours + params) ---

func TestSnap_PDS_PouroverBrew(t *testing.T) {
	h := StartHarness(t, nil)

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Snap Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Snap Bean", "roaster_rkey", roasterRKey, "roast_level", "Light",
	)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Snap Grinder", "grinder_type", "Manual")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Snap V60", "brewer_type", "Pour Over")), "brewer")

	brewForm := url.Values{}
	brewForm.Set("bean_rkey", beanRKey)
	brewForm.Set("grinder_rkey", grinderRKey)
	brewForm.Set("brewer_rkey", brewerRKey)
	brewForm.Set("method", "Pour Over")
	brewForm.Set("temperature", "94")
	brewForm.Set("water_amount", "300")
	brewForm.Set("coffee_amount", "18")
	brewForm.Set("time_seconds", "210")
	brewForm.Set("rating", "8")
	brewForm.Set("grind_size", "Medium")
	brewForm.Set("tasting_notes", "bright, floral")
	brewForm.Set("pour_water_0", "60")
	brewForm.Set("pour_time_0", "0")
	brewForm.Set("pour_water_1", "120")
	brewForm.Set("pour_time_1", "45")
	brewForm.Set("pour_water_2", "120")
	brewForm.Set("pour_time_2", "90")
	brewForm.Set("pourover_bloom_water", "60")
	brewForm.Set("pourover_bloom_seconds", "30")
	brewForm.Set("pourover_drawdown_seconds", "45")
	brewForm.Set("pourover_filter", "Hario tabbed")

	resp := h.PostForm("/brews", brewForm)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, ReadBody(t, resp)))

	// Get the brew rkey from the arabica API, then snapshot the raw PDS record.
	data := fetchData(t, h)
	require.Len(t, data.Brews, 1)
	brewRKey := data.Brews[0].RKey

	rkeys := map[string]string{
		"roaster": roasterRKey,
		"bean":    beanRKey,
		"grinder": grinderRKey,
		"brewer":  brewerRKey,
		"brew":    brewRKey,
	}

	snapPDSRecord(t, h, "pourover brew record", atproto.NSIDBrew, brewRKey, rkeys)
}

// --- Espresso brew (method-specific params) ---

func TestSnap_PDS_EspressoBrew(t *testing.T) {
	h := StartHarness(t, nil)

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Snap Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Snap Bean", "roaster_rkey", roasterRKey, "roast_level", "Medium",
	)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Snap Grinder", "grinder_type", "Electric")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Snap Espresso", "brewer_type", "Espresso")), "brewer")

	brewForm := url.Values{}
	brewForm.Set("bean_rkey", beanRKey)
	brewForm.Set("grinder_rkey", grinderRKey)
	brewForm.Set("brewer_rkey", brewerRKey)
	brewForm.Set("method", "Espresso")
	brewForm.Set("water_amount", "36")
	brewForm.Set("coffee_amount", "18")
	brewForm.Set("time_seconds", "28")
	brewForm.Set("rating", "9")
	brewForm.Set("tasting_notes", "syrupy, chocolate")
	brewForm.Set("espresso_yield_weight", "36")
	brewForm.Set("espresso_pressure", "9")
	brewForm.Set("espresso_pre_infusion_seconds", "5")

	resp := h.PostForm("/brews", brewForm)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, ReadBody(t, resp)))

	data := fetchData(t, h)
	require.Len(t, data.Brews, 1)
	brewRKey := data.Brews[0].RKey

	rkeys := map[string]string{
		"roaster": roasterRKey,
		"bean":    beanRKey,
		"grinder": grinderRKey,
		"brewer":  brewerRKey,
		"brew":    brewRKey,
	}

	snapPDSRecord(t, h, "espresso brew record", atproto.NSIDBrew, brewRKey, rkeys)
}

// --- Brew update: pourover → espresso (verify old params removed) ---

func TestSnap_PDS_BrewMethodSwap(t *testing.T) {
	h := StartHarness(t, nil)

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Snap Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Snap Bean", "roaster_rkey", roasterRKey, "roast_level", "Medium",
	)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Snap Grinder", "grinder_type", "Electric")), "grinder")
	pourBrewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Snap V60", "brewer_type", "Pour Over")), "brewer")
	espBrewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Snap Espresso", "brewer_type", "Espresso")), "brewer")

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

	resp := h.PostForm("/brews", createForm)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, ReadBody(t, resp)))

	data := fetchData(t, h)
	require.Len(t, data.Brews, 1)
	brewRKey := data.Brews[0].RKey

	// Update to espresso.
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

	rkeys := map[string]string{
		"roaster":    roasterRKey,
		"bean":       beanRKey,
		"grinder":    grinderRKey,
		"pourBrewer": pourBrewerRKey,
		"espBrewer":  espBrewerRKey,
		"brew":       brewRKey,
	}

	snapPDSRecord(t, h, "brew after pourover to espresso", atproto.NSIDBrew, brewRKey, rkeys)
}

// --- Roaster field permutations ---

func TestSnap_PDS_RoasterPermutations(t *testing.T) {
	cases := []struct {
		name string
		form []string
	}{
		{"name only", []string{"name", "Bare Roaster"}},
		{"name and location", []string{"name", "Located Roaster", "location", "Portland, OR"}},
		{"name and website", []string{"name", "Web Roaster", "website", "https://example.com"}},
		{"all fields", []string{"name", "Full Roaster", "location", "Seattle, WA", "website", "https://full.example.com"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := StartHarness(t, nil)
			rkey := mustRKey(t, h.PostForm("/api/roasters", form(tc.form...)), "roaster")
			snapPDSRecord(t, h, "roaster "+tc.name, atproto.NSIDRoaster, rkey, map[string]string{
				"roaster": rkey,
			})
		})
	}
}

// --- Bean field permutations ---

func TestSnap_PDS_BeanPermutations(t *testing.T) {
	h := StartHarness(t, nil)
	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Perm Roaster")), "roaster")

	cases := []struct {
		name string
		form []string
	}{
		{"name only", []string{"name", "Bare Bean"}},
		{"with origin", []string{"name", "Origin Bean", "origin", "Ethiopia"}},
		{"with roast level", []string{"name", "Roasted Bean", "roast_level", "Dark"}},
		{"with variety and process", []string{
			"name", "Processed Bean", "variety", "Gesha", "process", "Natural",
		}},
		{"with roaster ref only", []string{
			"name", "Sourced Bean", "roaster_rkey", roasterRKey,
		}},
		{"with description", []string{
			"name", "Described Bean", "description", "Juicy and complex",
		}},
		{"all fields", []string{
			"name", "Full Bean",
			"origin", "Colombia",
			"variety", "Caturra",
			"roast_level", "Medium",
			"process", "Washed",
			"description", "Balanced and clean",
			"roaster_rkey", roasterRKey,
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rkey := mustRKey(t, h.PostForm("/api/beans", form(tc.form...)), "bean")
			snapPDSRecord(t, h, "bean "+tc.name, atproto.NSIDBean, rkey, map[string]string{
				"roaster": roasterRKey,
				"bean":    rkey,
			})
		})
	}
}

// --- Grinder field permutations ---

func TestSnap_PDS_GrinderPermutations(t *testing.T) {
	cases := []struct {
		name string
		form []string
	}{
		{"name only", []string{"name", "Bare Grinder"}},
		{"with type", []string{"name", "Typed Grinder", "grinder_type", "Electric"}},
		{"with burr", []string{"name", "Burr Grinder", "burr_type", "Flat"}},
		{"with notes", []string{"name", "Noted Grinder", "notes", "SSP burrs installed"}},
		{"type and burr", []string{
			"name", "Full Manual", "grinder_type", "Manual", "burr_type", "Conical",
		}},
		{"all fields", []string{
			"name", "Full Grinder",
			"grinder_type", "Electric",
			"burr_type", "Flat",
			"notes", "64mm SSP multipurpose",
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := StartHarness(t, nil)
			rkey := mustRKey(t, h.PostForm("/api/grinders", form(tc.form...)), "grinder")
			snapPDSRecord(t, h, "grinder "+tc.name, atproto.NSIDGrinder, rkey, map[string]string{
				"grinder": rkey,
			})
		})
	}
}

// --- Brewer field permutations ---

func TestSnap_PDS_BrewerPermutations(t *testing.T) {
	cases := []struct {
		name string
		form []string
	}{
		{"name only", []string{"name", "Bare Brewer"}},
		{"with type", []string{"name", "Typed Brewer", "brewer_type", "Immersion"}},
		{"with description", []string{"name", "Described Brewer", "description", "12oz capacity"}},
		{"all fields", []string{
			"name", "Full Brewer",
			"brewer_type", "Pour Over",
			"description", "Ceramic, size 02",
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := StartHarness(t, nil)
			rkey := mustRKey(t, h.PostForm("/api/brewers", form(tc.form...)), "brewer")
			snapPDSRecord(t, h, "brewer "+tc.name, atproto.NSIDBrewer, rkey, map[string]string{
				"brewer": rkey,
			})
		})
	}
}

// --- Recipe field permutations ---

func TestSnap_PDS_RecipePermutations(t *testing.T) {
	h := StartHarness(t, nil)
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Recipe Brewer", "brewer_type", "Pour Over")), "brewer")

	cases := []struct {
		name string
		form []string
	}{
		{"name only", []string{"name", "Bare Recipe"}},
		{"with brewer ref", []string{
			"name", "Brewer Recipe", "brewer_rkey", brewerRKey, "brewer_type", "Pour Over",
		}},
		{"with amounts no pours", []string{
			"name", "Amounts Recipe",
			"coffee_amount", "15",
			"water_amount", "250",
		}},
		{"with notes", []string{
			"name", "Noted Recipe",
			"notes", "Hoffmann method",
		}},
		{"with single pour", []string{
			"name", "Single Pour Recipe",
			"brewer_rkey", brewerRKey,
			"brewer_type", "Pour Over",
			"coffee_amount", "18",
			"water_amount", "300",
			"pour_water_0", "300",
			"pour_time_0", "0",
		}},
		{"all fields", []string{
			"name", "Full Recipe",
			"brewer_rkey", brewerRKey,
			"brewer_type", "Pour Over",
			"coffee_amount", "20",
			"water_amount", "300",
			"notes", "Competition recipe",
			"pour_water_0", "60",
			"pour_time_0", "0",
			"pour_water_1", "120",
			"pour_time_1", "45",
			"pour_water_2", "120",
			"pour_time_2", "90",
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rkey := mustRKey(t, h.PostForm("/api/recipes", form(tc.form...)), "recipe")
			snapPDSRecord(t, h, "recipe "+tc.name, atproto.NSIDRecipe, rkey, map[string]string{
				"brewer": brewerRKey,
				"recipe": rkey,
			})
		})
	}
}

// --- Brew field permutations ---

func TestSnap_PDS_BrewPermutations(t *testing.T) {
	h := StartHarness(t, nil)

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Perm Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Perm Bean", "roaster_rkey", roasterRKey, "roast_level", "Light",
	)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Perm Grinder", "grinder_type", "Manual")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Perm V60", "brewer_type", "Pour Over")), "brewer")
	espBrewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Perm Espresso", "brewer_type", "Espresso")), "brewer")
	recipeRKey := mustRKey(t, h.PostForm("/api/recipes", form(
		"name", "Perm Recipe", "brewer_rkey", brewerRKey, "brewer_type", "Pour Over",
		"coffee_amount", "18", "water_amount", "300",
	)), "recipe")

	rkeys := map[string]string{
		"roaster":   roasterRKey,
		"bean":      beanRKey,
		"grinder":   grinderRKey,
		"brewer":    brewerRKey,
		"espBrewer": espBrewerRKey,
		"recipe":    recipeRKey,
	}

	cases := []struct {
		name   string
		fields url.Values
	}{
		{"minimal bean only", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			return f
		}()},
		{"bean with method", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("method", "French Press")
			return f
		}()},
		{"bean with amounts", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("water_amount", "200")
			f.Set("coffee_amount", "13")
			return f
		}()},
		{"with grinder and grind size", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("grinder_rkey", grinderRKey)
			f.Set("grind_size", "Fine")
			return f
		}()},
		{"with brewer no params", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("brewer_rkey", brewerRKey)
			f.Set("method", "Pour Over")
			return f
		}()},
		{"with recipe ref", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("brewer_rkey", brewerRKey)
			f.Set("recipe_rkey", recipeRKey)
			f.Set("method", "Pour Over")
			f.Set("water_amount", "300")
			f.Set("coffee_amount", "18")
			return f
		}()},
		{"with temperature and time", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("temperature", "96")
			f.Set("time_seconds", "240")
			return f
		}()},
		{"with tasting notes and rating", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("tasting_notes", "cherry, jasmine, silky")
			f.Set("rating", "9")
			return f
		}()},
		{"pourover partial params", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("brewer_rkey", brewerRKey)
			f.Set("method", "Pour Over")
			f.Set("pourover_bloom_water", "50")
			f.Set("pourover_bloom_seconds", "45")
			return f
		}()},
		{"pourover with filter only", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("brewer_rkey", brewerRKey)
			f.Set("method", "Pour Over")
			f.Set("pourover_filter", "Sibarist FAST")
			return f
		}()},
		{"espresso partial params", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("brewer_rkey", espBrewerRKey)
			f.Set("method", "Espresso")
			f.Set("coffee_amount", "18")
			f.Set("espresso_yield_weight", "40")
			return f
		}()},
		{"espresso pressure only", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("brewer_rkey", espBrewerRKey)
			f.Set("method", "Espresso")
			f.Set("espresso_pressure", "6")
			return f
		}()},
		{"pours without params", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("method", "Pour Over")
			f.Set("pour_water_0", "60")
			f.Set("pour_time_0", "0")
			f.Set("pour_water_1", "240")
			f.Set("pour_time_1", "30")
			return f
		}()},
		{"single pour", func() url.Values {
			f := url.Values{}
			f.Set("bean_rkey", beanRKey)
			f.Set("method", "Pour Over")
			f.Set("pour_water_0", "300")
			f.Set("pour_time_0", "0")
			return f
		}()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Each subtest needs its own harness to avoid brew accumulation.
			sub := StartHarness(t, nil)
			// Re-create the deps in this harness.
			subRoasterRKey := mustRKey(t, sub.PostForm("/api/roasters", form("name", "Perm Roaster")), "roaster")
			subBeanRKey := mustRKey(t, sub.PostForm("/api/beans", form(
				"name", "Perm Bean", "roaster_rkey", subRoasterRKey, "roast_level", "Light",
			)), "bean")
			subGrinderRKey := mustRKey(t, sub.PostForm("/api/grinders", form("name", "Perm Grinder", "grinder_type", "Manual")), "grinder")
			subBrewerRKey := mustRKey(t, sub.PostForm("/api/brewers", form("name", "Perm V60", "brewer_type", "Pour Over")), "brewer")
			subEspBrewerRKey := mustRKey(t, sub.PostForm("/api/brewers", form("name", "Perm Espresso", "brewer_type", "Espresso")), "brewer")
			subRecipeRKey := mustRKey(t, sub.PostForm("/api/recipes", form(
				"name", "Perm Recipe", "brewer_rkey", subBrewerRKey, "brewer_type", "Pour Over",
				"coffee_amount", "18", "water_amount", "300",
			)), "recipe")

			// Remap the form values to use this harness's rkeys.
			remapped := url.Values{}
			for k, vs := range tc.fields {
				for _, v := range vs {
					switch v {
					case beanRKey:
						v = subBeanRKey
					case grinderRKey:
						v = subGrinderRKey
					case brewerRKey:
						v = subBrewerRKey
					case espBrewerRKey:
						v = subEspBrewerRKey
					case recipeRKey:
						v = subRecipeRKey
					}
					remapped.Set(k, v)
				}
			}

			resp := sub.PostForm("/brews", remapped)
			require.Equal(t, 200, resp.StatusCode, statusErr(resp, ReadBody(t, resp)))

			data := fetchData(t, sub)
			require.Len(t, data.Brews, 1)
			brewRKey := data.Brews[0].RKey

			subRkeys := map[string]string{
				"roaster":   subRoasterRKey,
				"bean":      subBeanRKey,
				"grinder":   subGrinderRKey,
				"brewer":    subBrewerRKey,
				"espBrewer": subEspBrewerRKey,
				"recipe":    subRecipeRKey,
				"brew":      brewRKey,
			}
			snapPDSRecord(t, sub, "brew "+tc.name, atproto.NSIDBrew, brewRKey, subRkeys)
		})
	}

	// Suppress unused variable warnings — these were used to build the test
	// case form values above.
	_ = rkeys
}

// --- Full user repo: create multiple entities, snapshot entire collections ---

func TestSnap_PDS_FullRepo(t *testing.T) {
	h := StartHarness(t, nil)

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form(
		"name", "Onyx", "location", "Rogers, AR",
	)), "roaster")
	roaster2RKey := mustRKey(t, h.PostForm("/api/roasters", form(
		"name", "Sey", "location", "Brooklyn, NY", "website", "https://seycoffee.com",
	)), "roaster")

	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Monarch", "roaster_rkey", roasterRKey, "roast_level", "Medium", "origin", "Blend",
	)), "bean")

	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form(
		"name", "DF64", "grinder_type", "Electric", "burr_type", "Flat",
	)), "grinder")

	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form(
		"name", "V60", "brewer_type", "Pour Over",
	)), "brewer")

	// Create a brew.
	brewForm := url.Values{}
	brewForm.Set("bean_rkey", beanRKey)
	brewForm.Set("grinder_rkey", grinderRKey)
	brewForm.Set("brewer_rkey", brewerRKey)
	brewForm.Set("method", "Pour Over")
	brewForm.Set("water_amount", "250")
	brewForm.Set("coffee_amount", "15")
	brewForm.Set("time_seconds", "180")
	brewForm.Set("rating", "7")

	resp := h.PostForm("/brews", brewForm)
	require.Equal(t, 200, resp.StatusCode, statusErr(resp, ReadBody(t, resp)))

	data := fetchData(t, h)
	require.Len(t, data.Brews, 1)

	// Collect all rkeys for scrubbing.
	rkeys := map[string]string{
		"roaster":  roasterRKey,
		"roaster2": roaster2RKey,
		"bean":     beanRKey,
		"grinder":  grinderRKey,
		"brewer":   brewerRKey,
		"brew":     data.Brews[0].RKey,
	}

	// Snapshot each collection from the PDS.
	snapPDSCollection(t, h, "all roasters", atproto.NSIDRoaster, rkeys)
	snapPDSCollection(t, h, "all beans", atproto.NSIDBean, rkeys)
	snapPDSCollection(t, h, "all grinders", atproto.NSIDGrinder, rkeys)
	snapPDSCollection(t, h, "all brewers", atproto.NSIDBrewer, rkeys)
	snapPDSCollection(t, h, "all brews", atproto.NSIDBrew, rkeys)
}
