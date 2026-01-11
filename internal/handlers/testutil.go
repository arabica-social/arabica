package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"arabica/internal/database"
	"arabica/internal/models"
)

// TestFixtures contains sample data for testing
type TestFixtures struct {
	Bean    *models.Bean
	Roaster *models.Roaster
	Grinder *models.Grinder
	Brewer  *models.Brewer
	Brew    *models.Brew
}

// NewTestFixtures creates a set of sample test data
func NewTestFixtures() *TestFixtures {
	now := time.Now()

	roaster := &models.Roaster{
		RKey:      "test-roaster-rkey",
		Name:      "Test Roaster",
		Location:  "Test City",
		Website:   "https://test-roaster.com",
		CreatedAt: now,
	}

	bean := &models.Bean{
		RKey:        "test-bean-rkey",
		Name:        "Test Bean",
		Origin:      "Ethiopia",
		RoastLevel:  "Medium",
		Process:     "Washed",
		Description: "Test description",
		RoasterRKey: roaster.RKey,
		Roaster:     roaster,
		CreatedAt:   now,
	}

	grinder := &models.Grinder{
		RKey:        "test-grinder-rkey",
		Name:        "Test Grinder",
		GrinderType: "Hand",
		BurrType:    "Conical",
		Notes:       "Test notes",
		CreatedAt:   now,
	}

	brewer := &models.Brewer{
		RKey:        "test-brewer-rkey",
		Name:        "Test Brewer",
		BrewerType:  "Pour Over",
		Description: "Test brewer description",
		CreatedAt:   now,
	}

	brew := &models.Brew{
		RKey:         "test-brew-rkey",
		BeanRKey:     bean.RKey,
		Method:       "V60",
		Temperature:  93.0,
		WaterAmount:  250,
		CoffeeAmount: 15,
		TimeSeconds:  180,
		GrindSize:    "Medium-Fine",
		GrinderRKey:  grinder.RKey,
		BrewerRKey:   brewer.RKey,
		TastingNotes: "Fruity, bright",
		Rating:       8,
		CreatedAt:    now,
		Bean:         bean,
		GrinderObj:   grinder,
		BrewerObj:    brewer,
	}

	return &TestFixtures{
		Bean:    bean,
		Roaster: roaster,
		Grinder: grinder,
		Brewer:  brewer,
		Brew:    brew,
	}
}

// TestContext contains test dependencies
type TestContext struct {
	Handler   *Handler
	MockStore *database.MockStore
	Fixtures  *TestFixtures
	Request   *http.Request
	Recorder  *httptest.ResponseRecorder
}

// NewTestContext creates a test context with mock dependencies
func NewTestContext() *TestContext {
	mockStore := &database.MockStore{}
	fixtures := NewTestFixtures()

	// Create minimal handler dependencies
	// Note: OAuth and Client are nil in tests - handlers should check for nil
	config := Config{
		SecureCookies: false,
	}

	handler := &Handler{
		oauth:         nil, // Tests will mock auth via context
		atprotoClient: nil,
		sessionCache:  nil,
		config:        config,
		feedService:   nil, // Can be set later if needed
		feedRegistry:  nil, // Can be set later if needed
	}

	return &TestContext{
		Handler:   handler,
		MockStore: mockStore,
		Fixtures:  fixtures,
	}
}

// contextKey type for storing auth info in tests
type contextKey string

const (
	contextKeyUserDID   contextKey = "userDID"
	contextKeySessionID contextKey = "sessionID"
)

// NewAuthenticatedRequest creates a request with authentication context
func NewAuthenticatedRequest(method, path string, body interface{}) *http.Request {
	req := httptest.NewRequest(method, path, nil)

	// Add authenticated DID to context using the same keys as OAuth middleware
	ctx := context.WithValue(req.Context(), contextKeyUserDID, "did:plc:test123456789")
	ctx = context.WithValue(ctx, contextKeySessionID, "test-session-id")

	return req.WithContext(ctx)
}

// NewUnauthenticatedRequest creates a request without authentication context
func NewUnauthenticatedRequest(method, path string) *http.Request {
	return httptest.NewRequest(method, path, nil)
}

// AssertResponseCode checks if the response has the expected status code
func AssertResponseCode(t interface {
	Errorf(format string, args ...interface{})
}, rec *httptest.ResponseRecorder, expected int) {
	if rec.Code != expected {
		t.Errorf("Expected status code %d, got %d. Body: %s", expected, rec.Code, rec.Body.String())
	}
}
