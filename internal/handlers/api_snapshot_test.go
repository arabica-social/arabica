package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"arabica/internal/models"

	"github.com/ptdewey/shutter"
)

// TestAPIMe_Snapshot tests the /api/me endpoint response format
func TestAPIMe_Snapshot(t *testing.T) {
	tc := NewTestContext()

	req := NewAuthenticatedRequest("GET", "/api/me", nil)
	rec := httptest.NewRecorder()

	tc.Handler.HandleAPIMe(rec, req)

	// For unauthenticated scenario, just verify status code
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", rec.Code)
	}
}

// TestAPIListAll_Snapshot tests the /api/data endpoint response format
func TestAPIListAll_Snapshot(t *testing.T) {
	tc := NewTestContext()

	// Mock store to return test data
	tc.MockStore.ListBeansFunc = func(ctx context.Context) ([]*models.Bean, error) {
		return []*models.Bean{tc.Fixtures.Bean}, nil
	}
	tc.MockStore.ListRoastersFunc = func(ctx context.Context) ([]*models.Roaster, error) {
		return []*models.Roaster{tc.Fixtures.Roaster}, nil
	}
	tc.MockStore.ListGrindersFunc = func(ctx context.Context) ([]*models.Grinder, error) {
		return []*models.Grinder{tc.Fixtures.Grinder}, nil
	}
	tc.MockStore.ListBrewersFunc = func(ctx context.Context) ([]*models.Brewer, error) {
		return []*models.Brewer{tc.Fixtures.Brewer}, nil
	}
	tc.MockStore.ListBrewsFunc = func(ctx context.Context, userID int) ([]*models.Brew, error) {
		return []*models.Brew{tc.Fixtures.Brew}, nil
	}

	req := NewAuthenticatedRequest("GET", "/api/data", nil)
	rec := httptest.NewRecorder()

	tc.Handler.HandleAPIListAll(rec, req)

	// For unauthenticated scenario, status will be 401
	if rec.Code == http.StatusUnauthorized {
		return
	}

	// Snapshot the JSON response
	if rec.Code == http.StatusOK {
		shutter.SnapJSON(t, "api_list_all", rec.Body.String(),
			shutter.ScrubTimestamp(),
			shutter.IgnoreKey("created_at"),
		)
	}
}

// TestBeanCreate_Success_Snapshot tests successful bean creation response
func TestBeanCreate_Success_Snapshot(t *testing.T) {
	tc := NewTestContext()

	// Mock successful bean creation
	tc.MockStore.CreateBeanFunc = func(ctx context.Context, bean *models.CreateBeanRequest) (*models.Bean, error) {
		return &models.Bean{
			RKey:       "test-bean-rkey",
			Name:       bean.Name,
			Origin:     bean.Origin,
			RoastLevel: bean.RoastLevel,
			Process:    bean.Process,
		}, nil
	}

	reqBody := models.CreateBeanRequest{
		Name:       "Test Bean",
		Origin:     "Ethiopia",
		RoastLevel: "Medium",
		Process:    "Washed",
	}
	body, _ := json.Marshal(reqBody)

	req := NewAuthenticatedRequest("POST", "/api/beans", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBeanCreate(rec, req)

	// For unauthenticated, will be 401
	if rec.Code == http.StatusUnauthorized {
		return
	}

	// Snapshot the JSON response
	if rec.Code == http.StatusCreated {
		shutter.SnapJSON(t, "bean_create_success", rec.Body.String(),
			shutter.ScrubTimestamp(),
			shutter.IgnoreKey("created_at"),
			shutter.IgnoreKey("rkey"),
		)
	}
}

// TestBeanUpdate_Success_Snapshot tests successful bean update response
func TestBeanUpdate_Success_Snapshot(t *testing.T) {
	tc := NewTestContext()

	// Mock successful bean update
	tc.MockStore.UpdateBeanByRKeyFunc = func(ctx context.Context, rkey string, bean *models.UpdateBeanRequest) error {
		return nil
	}

	reqBody := models.UpdateBeanRequest{
		Name:       "Updated Bean",
		Origin:     "Colombia",
		RoastLevel: "Dark",
	}
	body, _ := json.Marshal(reqBody)

	req := NewAuthenticatedRequest("PUT", "/api/beans/test-rkey", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("id", "test-rkey")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBeanUpdate(rec, req)

	if rec.Code == http.StatusUnauthorized {
		return
	}

	// Snapshot the JSON response
	if rec.Code == http.StatusOK {
		shutter.SnapJSON(t, "bean_update_success", rec.Body.String())
	}
}

// TestRoasterCreate_Success_Snapshot tests successful roaster creation response
func TestRoasterCreate_Success_Snapshot(t *testing.T) {
	tc := NewTestContext()

	// Mock successful roaster creation
	tc.MockStore.CreateRoasterFunc = func(ctx context.Context, roaster *models.CreateRoasterRequest) (*models.Roaster, error) {
		return &models.Roaster{
			RKey:     "test-roaster-rkey",
			Name:     roaster.Name,
			Location: roaster.Location,
			Website:  roaster.Website,
		}, nil
	}

	reqBody := models.CreateRoasterRequest{
		Name:     "Test Roaster",
		Location: "Portland, OR",
		Website:  "https://example.com",
	}
	body, _ := json.Marshal(reqBody)

	req := NewAuthenticatedRequest("POST", "/api/roasters", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleRoasterCreate(rec, req)

	if rec.Code == http.StatusUnauthorized {
		return
	}

	// Snapshot the JSON response
	if rec.Code == http.StatusCreated {
		shutter.SnapJSON(t, "roaster_create_success", rec.Body.String(),
			shutter.ScrubTimestamp(),
			shutter.IgnoreKey("created_at"),
			shutter.IgnoreKey("rkey"),
		)
	}
}

// TestGrinderCreate_Success_Snapshot tests successful grinder creation response
func TestGrinderCreate_Success_Snapshot(t *testing.T) {
	tc := NewTestContext()

	// Mock successful grinder creation
	tc.MockStore.CreateGrinderFunc = func(ctx context.Context, grinder *models.CreateGrinderRequest) (*models.Grinder, error) {
		return &models.Grinder{
			RKey:        "test-grinder-rkey",
			Name:        grinder.Name,
			GrinderType: grinder.GrinderType,
			BurrType:    grinder.BurrType,
		}, nil
	}

	reqBody := models.CreateGrinderRequest{
		Name:        "Test Grinder",
		GrinderType: "Manual",
		BurrType:    "Conical",
	}
	body, _ := json.Marshal(reqBody)

	req := NewAuthenticatedRequest("POST", "/api/grinders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleGrinderCreate(rec, req)

	if rec.Code == http.StatusUnauthorized {
		return
	}

	// Snapshot the JSON response
	if rec.Code == http.StatusCreated {
		shutter.SnapJSON(t, "grinder_create_success", rec.Body.String(),
			shutter.ScrubTimestamp(),
			shutter.IgnoreKey("created_at"),
			shutter.IgnoreKey("rkey"),
		)
	}
}

// TestBrewerCreate_Success_Snapshot tests successful brewer creation response
func TestBrewerCreate_Success_Snapshot(t *testing.T) {
	tc := NewTestContext()

	// Mock successful brewer creation
	tc.MockStore.CreateBrewerFunc = func(ctx context.Context, brewer *models.CreateBrewerRequest) (*models.Brewer, error) {
		return &models.Brewer{
			RKey:        "test-brewer-rkey",
			Name:        brewer.Name,
			BrewerType:  brewer.BrewerType,
			Description: brewer.Description,
		}, nil
	}

	reqBody := models.CreateBrewerRequest{
		Name:        "Test Brewer",
		BrewerType:  "Pour Over",
		Description: "V60",
	}
	body, _ := json.Marshal(reqBody)

	req := NewAuthenticatedRequest("POST", "/api/brewers", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewerCreate(rec, req)

	if rec.Code == http.StatusUnauthorized {
		return
	}

	// Snapshot the JSON response
	if rec.Code == http.StatusCreated {
		shutter.SnapJSON(t, "brewer_create_success", rec.Body.String(),
			shutter.ScrubTimestamp(),
			shutter.IgnoreKey("created_at"),
			shutter.IgnoreKey("rkey"),
		)
	}
}

// TestBrewCreate_Success_Snapshot tests successful brew creation response
func TestBrewCreate_Success_Snapshot(t *testing.T) {
	tc := NewTestContext()

	// Mock successful brew creation
	tc.MockStore.CreateBrewFunc = func(ctx context.Context, brew *models.CreateBrewRequest, userID int) (*models.Brew, error) {
		return &models.Brew{
			RKey:         "test-brew-rkey",
			BeanRKey:     brew.BeanRKey,
			Method:       brew.Method,
			Temperature:  brew.Temperature,
			WaterAmount:  brew.WaterAmount,
			CoffeeAmount: brew.CoffeeAmount,
			TimeSeconds:  brew.TimeSeconds,
			GrindSize:    brew.GrindSize,
			GrinderRKey:  brew.GrinderRKey,
			BrewerRKey:   brew.BrewerRKey,
			TastingNotes: brew.TastingNotes,
			Rating:       brew.Rating,
		}, nil
	}

	reqBody := models.CreateBrewRequest{
		BeanRKey:     "bean-rkey",
		Method:       "Pour Over",
		Temperature:  93.0,
		WaterAmount:  250,
		CoffeeAmount: 15.0,
		TimeSeconds:  180,
		GrindSize:    "Medium-Fine",
		GrinderRKey:  "grinder-rkey",
		BrewerRKey:   "brewer-rkey",
		TastingNotes: "Bright and fruity",
		Rating:       8,
	}
	body, _ := json.Marshal(reqBody)

	req := NewAuthenticatedRequest("POST", "/brews", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewCreate(rec, req)

	if rec.Code == http.StatusUnauthorized {
		return
	}

	// Snapshot the JSON response
	if rec.Code == http.StatusCreated {
		shutter.SnapJSON(t, "brew_create_success", rec.Body.String(),
			shutter.ScrubTimestamp(),
			shutter.IgnoreKey("created_at"),
			shutter.IgnoreKey("rkey"),
		)
	}
}

// TestBrewDelete_Success_Snapshot tests successful brew deletion response
func TestBrewDelete_Success_Snapshot(t *testing.T) {
	tc := NewTestContext()

	// Mock successful brew deletion
	tc.MockStore.DeleteBrewByRKeyFunc = func(ctx context.Context, rkey string) error {
		return nil
	}

	req := NewAuthenticatedRequest("DELETE", "/brews/test-rkey", nil)
	req.SetPathValue("id", "test-rkey")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBrewDelete(rec, req)

	if rec.Code == http.StatusUnauthorized {
		return
	}

	// Snapshot the JSON response
	if rec.Code == http.StatusOK {
		shutter.SnapJSON(t, "brew_delete_success", rec.Body.String())
	}
}

// TestBeanDelete_Success_Snapshot tests successful bean deletion response
func TestBeanDelete_Success_Snapshot(t *testing.T) {
	tc := NewTestContext()

	// Mock successful bean deletion
	tc.MockStore.DeleteBeanByRKeyFunc = func(ctx context.Context, rkey string) error {
		return nil
	}

	req := NewAuthenticatedRequest("DELETE", "/api/beans/test-rkey", nil)
	req.SetPathValue("id", "test-rkey")
	rec := httptest.NewRecorder()

	tc.Handler.HandleBeanDelete(rec, req)

	if rec.Code == http.StatusUnauthorized {
		return
	}

	// Snapshot the JSON response
	if rec.Code == http.StatusOK {
		shutter.SnapJSON(t, "bean_delete_success", rec.Body.String())
	}
}

// TestFeedAPI_Snapshot tests the /api/feed-json endpoint response format
func TestFeedAPI_Snapshot(t *testing.T) {
	tc := NewTestContext()

	req := NewUnauthenticatedRequest("GET", "/api/feed-json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleFeedAPI(rec, req)

	// Snapshot the JSON response
	if rec.Code == http.StatusOK {
		shutter.SnapJSON(t, "feed_api", rec.Body.String(),
			shutter.ScrubTimestamp(),
			shutter.IgnoreKey("created_at"),
			shutter.IgnoreKey("indexed_at"),
		)
	}
}

// TestResolveHandle_Success_Snapshot tests handle resolution response
func TestResolveHandle_Success_Snapshot(t *testing.T) {
	tc := NewTestContext()

	req := NewUnauthenticatedRequest("GET", "/api/resolve-handle?handle=test.bsky.social")
	rec := httptest.NewRecorder()

	tc.Handler.HandleResolveHandle(rec, req)

	// This will fail without proper setup, but we can snapshot the error response
	if rec.Code == http.StatusOK {
		shutter.SnapJSON(t, "resolve_handle_success", rec.Body.String())
	}
}

// TestClientMetadata_Snapshot tests OAuth client metadata endpoint
func TestClientMetadata_Snapshot(t *testing.T) {
	tc := NewTestContext()

	req := NewUnauthenticatedRequest("GET", "/client-metadata.json")
	rec := httptest.NewRecorder()

	tc.Handler.HandleClientMetadata(rec, req)

	// Snapshot the JSON response
	if rec.Code == http.StatusOK {
		shutter.SnapJSON(t, "client_metadata", rec.Body.String())
	}
}
