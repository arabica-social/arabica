//go:build integration

package integration

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/require"
	"tangled.org/pdewey.com/atp"
)

// scrubAPI extends scrubPDS with additional scrubbers for API JSON responses.
// Handle resolution is non-deterministic in the test harness (the handle may
// be resolved to "alice.test", fall back to the DID, or be empty depending on
// firehose timing), so we ignore the handle field entirely to keep snapshots
// deterministic. CIDs are also non-deterministic (PDS-generated content hashes).
func scrubAPI(did string, rkeys map[string]string) []shutter.Option {
	opts := scrubPDS(did, rkeys)
	// The handle field is non-deterministic — it may be resolved, fall back
	// to the DID, or be empty depending on firehose/profile-watcher timing.
	opts = append(opts, shutter.IgnoreKeyValue("handle", "*"))
	opts = append(opts, shutter.IgnoreKeyValue("actor_handle", "*"))
	opts = append(opts, shutter.IgnoreKeyValue("display_name", "*"))
	opts = append(opts, shutter.IgnoreKeyValue("actor_display_name", "*"))
	// CIDs are PDS-generated content hashes that change every run.
	opts = append(opts, shutter.IgnoreKeyValue("subject_cid", "*"))
	opts = append(opts, shutter.IgnoreKeyValue("cid", "*"))
	opts = append(opts, shutter.IgnoreKeyValue("SubjectCID", "*"))
	// The time_ago field is relative and changes between runs.
	opts = append(opts, shutter.IgnoreKeyValue("time_ago", "*"))
	return opts
}

// snapAPIJSON fetches a JSON API endpoint (with Accept: application/json) and
// snapshots the response body. Dynamic values (DIDs, rkeys, timestamps, handles)
// are scrubbed so snapshots are deterministic across runs.
func snapAPIJSON(t *testing.T, h *Harness, title, path string, rkeys map[string]string) {
	t.Helper()
	resp := getJSON(t, h, path)
	body := ReadBody(t, resp)
	require.Equalf(t, 200, resp.StatusCode, "%s: %s", title, statusErr(resp, body))

	// Re-marshal with indentation for readable snapshots.
	var raw any
	require.NoErrorf(t, json.Unmarshal([]byte(body), &raw), "%s: response is not valid JSON: %s", title, body)
	pretty, err := json.MarshalIndent(raw, "", "  ")
	require.NoError(t, err)

	shutter.SnapJSON(t, title, string(pretty), scrubAPI(h.PrimaryAccount.DID, rkeys)...)
}

func TestSnap_API_FeedJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Snap Feed Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Snap Feed Bean", "origin", "Ethiopia", "roaster_rkey", roasterRKey)), "bean")

	roasterURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", roasterRKey)
	h.WaitForRecord(roasterURI, firehoseWait)
	beanURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.bean", beanRKey)
	h.WaitForRecord(beanURI, firehoseWait)

	snapAPIJSON(t, h, "feed json", "/api/feed?sort=recent", map[string]string{
		"roaster": roasterRKey,
		"bean":    beanRKey,
	})
}

func TestSnap_API_EntityViewBeanJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Snap Bean Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form(
		"name", "Snap Bean",
		"origin", "Colombia",
		"roast_level", "Medium",
		"process", "Washed",
		"variety", "Caturra",
		"roaster_rkey", roasterRKey,
	)), "bean")

	roasterURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", roasterRKey)
	h.WaitForRecord(roasterURI, firehoseWait)

	actor := h.PrimaryAccount.DID
	snapAPIJSON(t, h, "entity view bean json", "/api/beans/"+actor+"/"+beanRKey, map[string]string{
		"roaster": roasterRKey,
		"bean":    beanRKey,
	})
}

func TestSnap_API_EntityViewRoasterJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form(
		"name", "Snap Roaster",
		"location", "Portland, OR",
		"website", "https://snap.example.com",
	)), "roaster")

	roasterURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.roaster", roasterRKey)
	h.WaitForRecord(roasterURI, firehoseWait)

	actor := h.PrimaryAccount.DID
	snapAPIJSON(t, h, "entity view roaster json", "/api/roasters/"+actor+"/"+roasterRKey, map[string]string{
		"roaster": roasterRKey,
	})
}

func TestSnap_API_ManageJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Snap Manage Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Snap Manage Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Snap Manage Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Snap Manage Brewer")), "brewer")

	brewRKey := mustBrewRKey(t, h, form(
		"bean_rkey", beanRKey,
		"grinder_rkey", grinderRKey,
		"brewer_rkey", brewerRKey,
		"method", "Pour Over",
		"water_amount", "300",
		"coffee_amount", "18",
		"rating", "8",
	))

	beanURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.bean", beanRKey)
	h.WaitForRecord(beanURI, firehoseWait)

	snapAPIJSON(t, h, "manage json", "/api/manage", map[string]string{
		"roaster": roasterRKey,
		"bean":    beanRKey,
		"grinder": grinderRKey,
		"brewer":  brewerRKey,
		"brew":    brewRKey,
	})
}

// mustBrewRKey creates a brew via POST /brews (which returns HX-Redirect, not
// JSON) and extracts the rkey by fetching the brew list.
func mustBrewRKey(t *testing.T, h *Harness, brewForm url.Values) string {
	t.Helper()
	brewResp := h.PostForm("/brews", brewForm)
	require.Equal(t, 200, brewResp.StatusCode, statusErr(brewResp, ReadBody(t, brewResp)))

	listResp := h.Get("/api/data")
	listBody := ReadBody(t, listResp)
	require.Equal(t, 200, listResp.StatusCode, statusErr(listResp, listBody))
	var data struct {
		Brews []struct {
			RKey string `json:"rkey"`
		} `json:"brews"`
	}
	require.NoError(t, json.Unmarshal([]byte(listBody), &data))
	require.Len(t, data.Brews, 1, "expected one brew")
	require.NotEmpty(t, data.Brews[0].RKey)
	return data.Brews[0].RKey
}

func TestSnap_API_BrewListJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Snap BrewList Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Snap BrewList Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Snap BrewList Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Snap BrewList Brewer")), "brewer")

	brewRKey := mustBrewRKey(t, h, form(
		"bean_rkey", beanRKey,
		"grinder_rkey", grinderRKey,
		"brewer_rkey", brewerRKey,
		"method", "Pour Over",
		"water_amount", "250",
		"coffee_amount", "15",
		"rating", "7",
	))

	beanURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.bean", beanRKey)
	h.WaitForRecord(beanURI, firehoseWait)

	snapAPIJSON(t, h, "brew list json", "/api/brews", map[string]string{
		"roaster": roasterRKey,
		"bean":    beanRKey,
		"grinder": grinderRKey,
		"brewer":  brewerRKey,
		"brew":    brewRKey,
	})
}

func TestSnap_API_ProfileJSON(t *testing.T) {
	h := StartHarness(t, &HarnessOptions{EnableFirehose: true})

	roasterRKey := mustRKey(t, h.PostForm("/api/roasters", form("name", "Snap Profile Roaster")), "roaster")
	beanRKey := mustRKey(t, h.PostForm("/api/beans", form("name", "Snap Profile Bean", "roaster_rkey", roasterRKey)), "bean")
	grinderRKey := mustRKey(t, h.PostForm("/api/grinders", form("name", "Snap Profile Grinder")), "grinder")
	brewerRKey := mustRKey(t, h.PostForm("/api/brewers", form("name", "Snap Profile Brewer")), "brewer")

	brewRKey := mustBrewRKey(t, h, form(
		"bean_rkey", beanRKey,
		"grinder_rkey", grinderRKey,
		"brewer_rkey", brewerRKey,
		"method", "Pour Over",
		"water_amount", "300",
		"coffee_amount", "18",
		"rating", "9",
	))

	beanURI := atp.BuildATURI(h.PrimaryAccount.DID, "social.arabica.alpha.bean", beanRKey)
	h.WaitForRecord(beanURI, firehoseWait)

	actor := h.PrimaryAccount.DID
	snapAPIJSON(t, h, "profile json", "/api/profile/"+actor, map[string]string{
		"roaster": roasterRKey,
		"bean":    beanRKey,
		"grinder": grinderRKey,
		"brewer":  brewerRKey,
		"brew":    brewRKey,
	})
}

func TestSnap_API_SettingsJSON(t *testing.T) {
	h := StartHarness(t, nil)

	// Set non-default preferences so the snapshot captures meaningful values.
	formData := form("temperature_unit", "fahrenheit")
	h.PostForm("/api/settings/preferences", formData)

	visForm := form("bean_avg_rating", "private", "roaster_avg_rating", "public")
	h.PostForm("/api/settings/profile-visibility", visForm)

	snapAPIJSON(t, h, "settings json", "/api/settings", nil)
}
