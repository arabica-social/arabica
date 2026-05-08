package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities/arabica"

	"github.com/bluesky-social/indigo/atproto/atclient"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"tangled.org/pdewey.com/atp"
	"tangled.org/pdewey.com/chrysalis/testpds"
)

// testAccount holds credentials for a test PDS account.
type testAccount struct {
	DID       string
	Handle    string
	AccessJwt string
}

// createAccount registers a new account on the test PDS and returns its credentials.
func createAccount(t *testing.T, pdsURL, email, handle, password string) testAccount {
	t.Helper()

	body, err := json.Marshal(map[string]string{
		"email":    email,
		"handle":   handle,
		"password": password,
	})
	require.NoError(t, err)

	resp, err := http.Post(pdsURL+"/xrpc/com.atproto.server.createAccount", "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode, "createAccount failed: %s", string(respBody))

	var result struct {
		AccessJwt string `json:"accessJwt"`
		Handle    string `json:"handle"`
		Did       string `json:"did"`
	}
	require.NoError(t, json.Unmarshal(respBody, &result))

	return testAccount{
		DID:       result.Did,
		Handle:    result.Handle,
		AccessJwt: result.AccessJwt,
	}
}

// newTestStore creates an AtprotoStore backed by a real test PDS using password auth.
func newTestStore(t *testing.T, pdsURL string, acct testAccount) *atproto.AtprotoStore {
	t.Helper()

	did := syntax.DID(acct.DID)

	apiClient, err := atclient.LoginWithPasswordHost(
		context.Background(),
		pdsURL,
		acct.Handle,
		"hunter2", // matches the password used in createAccount
		"",        // no auth token
		nil,       // no refresh callback
	)
	require.NoError(t, err)

	client := atproto.NewClientWithProvider(func(ctx context.Context, d syntax.DID, _ string) (*atp.Client, error) {
		return atp.NewClient(apiClient, d), nil
	})

	cache := atproto.NewSessionCache()
	store := atproto.NewAtprotoStore(client, did, "test-session", cache)

	return store.(*atproto.AtprotoStore)
}

func TestPDS_RoasterCRUD(t *testing.T) {
	pds := testpds.StartT(t, nil)
	acct := createAccount(t, pds.URL, "alice@test.com", "alice.test", "hunter2")
	store := newTestStore(t, pds.URL, acct)
	ctx := context.Background()

	// Create
	roaster, err := store.CreateRoaster(ctx, &arabica.CreateRoasterRequest{
		Name:     "Counter Culture",
		Location: "Durham, NC",
		Website:  "https://counterculturecoffee.com",
	})
	require.NoError(t, err)
	assert.Equal(t, "Counter Culture", roaster.Name)
	assert.Equal(t, "Durham, NC", roaster.Location)
	assert.NotEmpty(t, roaster.RKey)

	// Read
	fetched, err := store.GetRoasterByRKey(ctx, roaster.RKey)
	require.NoError(t, err)
	assert.Equal(t, "Counter Culture", fetched.Name)
	assert.Equal(t, "Durham, NC", fetched.Location)
	assert.Equal(t, "https://counterculturecoffee.com", fetched.Website)

	// Update
	err = store.UpdateRoasterByRKey(ctx, roaster.RKey, &arabica.UpdateRoasterRequest{
		Name:     "Counter Culture Coffee",
		Location: "Durham, NC",
		Website:  "https://counterculturecoffee.com",
	})
	require.NoError(t, err)

	updated, err := store.GetRoasterByRKey(ctx, roaster.RKey)
	require.NoError(t, err)
	assert.Equal(t, "Counter Culture Coffee", updated.Name)

	// List
	roasters, err := store.ListRoasters(ctx)
	require.NoError(t, err)
	assert.Len(t, roasters, 1)
	assert.Equal(t, "Counter Culture Coffee", roasters[0].Name)

	// Delete
	err = store.DeleteRoasterByRKey(ctx, roaster.RKey)
	require.NoError(t, err)

	roasters, err = store.ListRoasters(ctx)
	require.NoError(t, err)
	assert.Len(t, roasters, 0)
}

func TestPDS_BeanWithRoasterRef(t *testing.T) {
	pds := testpds.StartT(t, nil)
	acct := createAccount(t, pds.URL, "bob@test.com", "bob.test", "hunter2")
	store := newTestStore(t, pds.URL, acct)
	ctx := context.Background()

	// Create a roaster first
	roaster, err := store.CreateRoaster(ctx, &arabica.CreateRoasterRequest{
		Name: "Sweet Maria's",
	})
	require.NoError(t, err)

	// Create a bean referencing the roaster
	bean, err := store.CreateBean(ctx, &arabica.CreateBeanRequest{
		Name:        "Ethiopia Yirgacheffe",
		Origin:      "Ethiopia",
		RoastLevel:  "Light",
		RoasterRKey: roaster.RKey,
	})
	require.NoError(t, err)
	assert.Equal(t, "Ethiopia Yirgacheffe", bean.Name)
	assert.NotEmpty(t, bean.RKey)

	// Fetch and verify roaster reference is intact
	fetched, err := store.GetBeanByRKey(ctx, bean.RKey)
	require.NoError(t, err)
	assert.Equal(t, "Ethiopia Yirgacheffe", fetched.Name)
	assert.Equal(t, "Ethiopia", fetched.Origin)
	assert.Equal(t, "Light", fetched.RoastLevel)
	assert.Equal(t, roaster.RKey, fetched.RoasterRKey)
}

func TestPDS_GrinderCRUD(t *testing.T) {
	pds := testpds.StartT(t, nil)
	acct := createAccount(t, pds.URL, "carol@test.com", "carol.test", "hunter2")
	store := newTestStore(t, pds.URL, acct)
	ctx := context.Background()

	grinder, err := store.CreateGrinder(ctx, &arabica.CreateGrinderRequest{
		Name:        "Comandante C40",
		GrinderType: "hand",
		BurrType:    "conical",
	})
	require.NoError(t, err)
	assert.Equal(t, "Comandante C40", grinder.Name)

	fetched, err := store.GetGrinderByRKey(ctx, grinder.RKey)
	require.NoError(t, err)
	assert.Equal(t, "hand", fetched.GrinderType)
	assert.Equal(t, "conical", fetched.BurrType)
}

func TestPDS_BrewerCRUD(t *testing.T) {
	pds := testpds.StartT(t, nil)
	acct := createAccount(t, pds.URL, "dave@test.com", "dave.test", "hunter2")
	store := newTestStore(t, pds.URL, acct)
	ctx := context.Background()

	brewer, err := store.CreateBrewer(ctx, &arabica.CreateBrewerRequest{
		Name:       "Hario V60",
		BrewerType: "pourover",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hario V60", brewer.Name)

	fetched, err := store.GetBrewerByRKey(ctx, brewer.RKey)
	require.NoError(t, err)
	assert.Equal(t, "pourover", fetched.BrewerType)
}

func TestPDS_FullBrewSession(t *testing.T) {
	pds := testpds.StartT(t, nil)
	acct := createAccount(t, pds.URL, "eve@test.com", "eve.test", "hunter2")
	store := newTestStore(t, pds.URL, acct)
	ctx := context.Background()

	// Set up entities
	roaster, err := store.CreateRoaster(ctx, &arabica.CreateRoasterRequest{Name: "Onyx"})
	require.NoError(t, err)

	bean, err := store.CreateBean(ctx, &arabica.CreateBeanRequest{
		Name:        "Monarch",
		RoasterRKey: roaster.RKey,
		RoastLevel:  "Medium",
	})
	require.NoError(t, err)

	grinder, err := store.CreateGrinder(ctx, &arabica.CreateGrinderRequest{
		Name:        "1Zpresso JX-Pro",
		GrinderType: "hand",
		BurrType:    "conical",
	})
	require.NoError(t, err)

	brewer, err := store.CreateBrewer(ctx, &arabica.CreateBrewerRequest{
		Name:       "V60",
		BrewerType: "pourover",
	})
	require.NoError(t, err)

	// Create a brew referencing all entities
	brew, err := store.CreateBrew(ctx, &arabica.CreateBrewRequest{
		BeanRKey:    bean.RKey,
		GrinderRKey: grinder.RKey,
		BrewerRKey:  brewer.RKey,
		Method:      "V60",
		GrindSize:   "22 clicks",
		WaterAmount: 250,
		Temperature: 93,
		TimeSeconds: 195,
		Rating:      8,
	}, 0)
	require.NoError(t, err)
	assert.Equal(t, "V60", brew.Method)
	assert.Equal(t, 8, brew.Rating)
	assert.NotEmpty(t, brew.RKey)

	// Verify the brew can be fetched back
	fetched, err := store.GetBrewByRKey(ctx, brew.RKey)
	require.NoError(t, err)
	assert.InDelta(t, 250, fetched.WaterAmount, 0.01)
	assert.InDelta(t, 93, fetched.Temperature, 0.01)
	assert.Equal(t, 195, fetched.TimeSeconds)
	assert.Equal(t, bean.RKey, fetched.BeanRKey)
	assert.Equal(t, grinder.RKey, fetched.GrinderRKey)
	assert.Equal(t, brewer.RKey, fetched.BrewerRKey)

	// List brews
	brews, err := store.ListBrews(ctx, 0)
	require.NoError(t, err)
	assert.Len(t, brews, 1)
}
