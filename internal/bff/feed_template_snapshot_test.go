package bff

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"github.com/ptdewey/shutter"

	"arabica/internal/atproto"
	"arabica/internal/feed"
	"arabica/internal/models"
)

// Helper functions for creating test data

func mockProfile(handle string, displayName string, avatar string) *atproto.Profile {
	var dn *string
	if displayName != "" {
		dn = &displayName
	}
	var av *string
	if avatar != "" {
		av = &avatar
	}
	return &atproto.Profile{
		DID:         "did:plc:" + handle,
		Handle:      handle,
		DisplayName: dn,
		Avatar:      av,
	}
}

func mockBrew(beanName string, roasterName string, rating int) *models.Brew {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	brew := &models.Brew{
		RKey:         "brew123",
		BeanRKey:     "bean123",
		CreatedAt:    testTime,
		Rating:       rating,
		Temperature:  93.0,
		WaterAmount:  250,
		CoffeeAmount: 16,
		TimeSeconds:  180,
		GrindSize:    "Medium-fine",
		Method:       "V60",
		TastingNotes: "Bright citrus notes with floral aroma",
	}

	if beanName != "" {
		brew.Bean = &models.Bean{
			RKey:       "bean123",
			Name:       beanName,
			Origin:     "Ethiopia",
			RoastLevel: "Light",
			Process:    "Washed",
			CreatedAt:  testTime,
		}
		if roasterName != "" {
			brew.Bean.Roaster = &models.Roaster{
				RKey:      "roaster123",
				Name:      roasterName,
				Location:  "Portland, OR",
				Website:   "https://example.com",
				CreatedAt: testTime,
			}
		}
	}

	brew.GrinderObj = &models.Grinder{
		RKey:        "grinder123",
		Name:        "1Zpresso JX-Pro",
		GrinderType: "Hand",
		BurrType:    "Conical",
		CreatedAt:   testTime,
	}

	brew.BrewerObj = &models.Brewer{
		RKey:        "brewer123",
		Name:        "Hario V60",
		BrewerType:  "Pour Over",
		Description: "Ceramic dripper",
		CreatedAt:   testTime,
	}

	brew.Pours = []*models.Pour{
		{PourNumber: 1, WaterAmount: 50, TimeSeconds: 30, CreatedAt: testTime},
		{PourNumber: 2, WaterAmount: 100, TimeSeconds: 45, CreatedAt: testTime},
		{PourNumber: 3, WaterAmount: 100, TimeSeconds: 60, CreatedAt: testTime},
	}

	return brew
}

func mockBean(name string, origin string, hasRoaster bool) *models.Bean {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	bean := &models.Bean{
		RKey:        "bean456",
		Name:        name,
		Origin:      origin,
		RoastLevel:  "Medium",
		Process:     "Natural",
		Description: "Sweet and fruity with notes of blueberry",
		CreatedAt:   testTime,
	}

	if hasRoaster {
		bean.Roaster = &models.Roaster{
			RKey:      "roaster456",
			Name:      "Onyx Coffee Lab",
			Location:  "Bentonville, AR",
			Website:   "https://onyxcoffeelab.com",
			CreatedAt: testTime,
		}
	}

	return bean
}

func mockRoaster() *models.Roaster {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	return &models.Roaster{
		RKey:      "roaster789",
		Name:      "Heart Coffee Roasters",
		Location:  "Portland, OR",
		Website:   "https://heartroasters.com",
		CreatedAt: testTime,
	}
}

func mockGrinder() *models.Grinder {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	return &models.Grinder{
		RKey:        "grinder789",
		Name:        "Comandante C40",
		GrinderType: "Hand",
		BurrType:    "Conical",
		Notes:       "Excellent for pour over",
		CreatedAt:   testTime,
	}
}

func mockBrewer() *models.Brewer {
	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	return &models.Brewer{
		RKey:        "brewer789",
		Name:        "Kalita Wave 185",
		BrewerType:  "Pour Over",
		Description: "Flat-bottom dripper with wave filters",
		CreatedAt:   testTime,
	}
}

// Template execution helper
func execFeedTemplate(feedItems []*feed.FeedItem, isAuthenticated bool) (string, error) {
	tmpl := template.New("test").Funcs(getTemplateFuncs())
	tmpl, err := tmpl.ParseFiles("../../templates/partials/feed.tmpl")
	if err != nil {
		return "", err
	}

	data := map[string]interface{}{
		"FeedItems":       feedItems,
		"IsAuthenticated": isAuthenticated,
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "feed", data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// Test individual record types

func TestFeedTemplate_BrewItem_Snapshot(t *testing.T) {
	tests := []struct {
		name     string
		feedItem *feed.FeedItem
	}{
		{
			name: "complete brew with all fields",
			feedItem: &feed.FeedItem{
				RecordType: "brew",
				Action:     "☕ added a new brew",
				Brew:       mockBrew("Ethiopian Yirgacheffe", "Onyx Coffee Lab", 9),
				Author:     mockProfile("coffee.lover", "Coffee Enthusiast", "https://cdn.bsky.app/avatar.jpg"),
				Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				TimeAgo:    "2 hours ago",
			},
		},
		{
			name: "brew with minimal data",
			feedItem: &feed.FeedItem{
				RecordType: "brew",
				Action:     "☕ added a new brew",
				Brew: &models.Brew{
					RKey:      "brew456",
					CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
					Rating:    0, // no rating
					Bean: &models.Bean{
						Name: "House Blend",
					},
				},
				Author:    mockProfile("newbie", "", ""),
				Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				TimeAgo:   "1 minute ago",
			},
		},
		{
			name: "brew with unicode bean name",
			feedItem: &feed.FeedItem{
				RecordType: "brew",
				Action:     "☕ added a new brew",
				Brew: &models.Brew{
					RKey:      "brew789",
					CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
					Rating:    8,
					Bean: &models.Bean{
						Name:   "コーヒー豆",
						Origin: "日本",
					},
				},
				Author:    mockProfile("japan.coffee", "日本のコーヒー", ""),
				Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				TimeAgo:   "3 hours ago",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := execFeedTemplate([]*feed.FeedItem{tt.feedItem}, true)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}
			shutter.Snap(t, tt.name, result)
		})
	}
}

func TestFeedTemplate_BeanItem_Snapshot(t *testing.T) {
	tests := []struct {
		name     string
		feedItem *feed.FeedItem
	}{
		{
			name: "bean with roaster",
			feedItem: &feed.FeedItem{
				RecordType: "bean",
				Action:     "🫘 added a new bean",
				Bean:       mockBean("Kenya AA", "Kenya", true),
				Author:     mockProfile("roaster.pro", "Pro Roaster", ""),
				Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				TimeAgo:    "5 minutes ago",
			},
		},
		{
			name: "bean without roaster",
			feedItem: &feed.FeedItem{
				RecordType: "bean",
				Action:     "🫘 added a new bean",
				Bean:       mockBean("Colombian Supremo", "Colombia", false),
				Author:     mockProfile("homebrewer", "Home Brewer", ""),
				Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				TimeAgo:    "1 day ago",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := execFeedTemplate([]*feed.FeedItem{tt.feedItem}, true)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}
			shutter.Snap(t, tt.name, result)
		})
	}
}

func TestFeedTemplate_RoasterItem_Snapshot(t *testing.T) {
	feedItem := &feed.FeedItem{
		RecordType: "roaster",
		Action:     "🏪 added a new roaster",
		Roaster:    mockRoaster(),
		Author:     mockProfile("roastmaster", "Roast Master", "https://cdn.bsky.app/avatar2.jpg"),
		Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		TimeAgo:    "10 minutes ago",
	}

	result, err := execFeedTemplate([]*feed.FeedItem{feedItem}, true)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	shutter.Snap(t, "roaster item", result)
}

func TestFeedTemplate_GrinderItem_Snapshot(t *testing.T) {
	feedItem := &feed.FeedItem{
		RecordType: "grinder",
		Action:     "⚙️ added a new grinder",
		Grinder:    mockGrinder(),
		Author:     mockProfile("gearhead", "Coffee Gear Head", ""),
		Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		TimeAgo:    "30 minutes ago",
	}

	result, err := execFeedTemplate([]*feed.FeedItem{feedItem}, true)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	shutter.Snap(t, "grinder item", result)
}

func TestFeedTemplate_BrewerItem_Snapshot(t *testing.T) {
	feedItem := &feed.FeedItem{
		RecordType: "brewer",
		Action:     "☕ added a new brewer",
		Brewer:     mockBrewer(),
		Author:     mockProfile("pourover.fan", "Pour Over Fan", ""),
		Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		TimeAgo:    "2 days ago",
	}

	result, err := execFeedTemplate([]*feed.FeedItem{feedItem}, true)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	shutter.Snap(t, "brewer item", result)
}

// Test mixed feeds and edge cases

func TestFeedTemplate_MixedFeed_Snapshot(t *testing.T) {
	feedItems := []*feed.FeedItem{
		{
			RecordType: "brew",
			Action:     "☕ added a new brew",
			Brew:       mockBrew("Ethiopian Yirgacheffe", "Onyx", 9),
			Author:     mockProfile("user1", "User One", ""),
			Timestamp:  time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC),
			TimeAgo:    "1 hour ago",
		},
		{
			RecordType: "bean",
			Action:     "🫘 added a new bean",
			Bean:       mockBean("Kenya AA", "Kenya", true),
			Author:     mockProfile("user2", "User Two", "https://cdn.bsky.app/avatar.jpg"),
			Timestamp:  time.Date(2024, 1, 15, 11, 30, 0, 0, time.UTC),
			TimeAgo:    "1.5 hours ago",
		},
		{
			RecordType: "roaster",
			Action:     "🏪 added a new roaster",
			Roaster:    mockRoaster(),
			Author:     mockProfile("user3", "User Three", ""),
			Timestamp:  time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			TimeAgo:    "2 hours ago",
		},
		{
			RecordType: "grinder",
			Action:     "⚙️ added a new grinder",
			Grinder:    mockGrinder(),
			Author:     mockProfile("user4", "", ""),
			Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			TimeAgo:    "2.5 hours ago",
		},
		{
			RecordType: "brewer",
			Action:     "☕ added a new brewer",
			Brewer:     mockBrewer(),
			Author:     mockProfile("user5", "User Five", ""),
			Timestamp:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			TimeAgo:    "3 hours ago",
		},
	}

	result, err := execFeedTemplate(feedItems, true)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	shutter.Snap(t, "mixed feed all types", result)
}

func TestFeedTemplate_EmptyFeed_Snapshot(t *testing.T) {
	tests := []struct {
		name            string
		feedItems       []*feed.FeedItem
		isAuthenticated bool
	}{
		{
			name:            "empty feed authenticated",
			feedItems:       []*feed.FeedItem{},
			isAuthenticated: true,
		},
		{
			name:            "empty feed unauthenticated",
			feedItems:       []*feed.FeedItem{},
			isAuthenticated: false,
		},
		{
			name:            "nil feed",
			feedItems:       nil,
			isAuthenticated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := execFeedTemplate(tt.feedItems, tt.isAuthenticated)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}
			shutter.Snap(t, tt.name, result)
		})
	}
}

// Test security (URL sanitization)

func TestFeedTemplate_SecurityURLs_Snapshot(t *testing.T) {
	tests := []struct {
		name     string
		feedItem *feed.FeedItem
	}{
		{
			name: "roaster with unsafe website URL",
			feedItem: &feed.FeedItem{
				RecordType: "roaster",
				Action:     "🏪 added a new roaster",
				Roaster: &models.Roaster{
					RKey:      "roaster999",
					Name:      "Sketchy Roaster",
					Website:   "javascript:alert('xss')", // Should be sanitized
					CreatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				},
				Author:    mockProfile("hacker", "Hacker", ""),
				Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				TimeAgo:   "1 minute ago",
			},
		},
		{
			name: "profile with unsafe avatar URL",
			feedItem: &feed.FeedItem{
				RecordType: "bean",
				Action:     "🫘 added a new bean",
				Bean:       mockBean("Test Bean", "Test Origin", false),
				Author:     mockProfile("badavatar", "Bad Avatar", "javascript:alert('xss')"), // Should be sanitized
				Timestamp:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				TimeAgo:    "2 minutes ago",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := execFeedTemplate([]*feed.FeedItem{tt.feedItem}, true)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}
			shutter.Snap(t, tt.name, result)
		})
	}
}

// Test special characters

func TestFeedTemplate_SpecialCharacters_Snapshot(t *testing.T) {
	feedItem := &feed.FeedItem{
		RecordType: "brew",
		Action:     "☕ added a new brew",
		Brew: &models.Brew{
			RKey:         "brew999",
			CreatedAt:    time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			Rating:       8,
			TastingNotes: "Notes with \"quotes\" and <html>tags</html> and 'single quotes'",
			Bean: &models.Bean{
				Name:        "Bean with & ampersand",
				Description: "Description with <script>alert('xss')</script>",
			},
		},
		Author:    mockProfile("special.chars", "User & Co.", ""),
		Timestamp: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		TimeAgo:   "5 seconds ago",
	}

	result, err := execFeedTemplate([]*feed.FeedItem{feedItem}, true)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	shutter.Snap(t, "special characters in content", result)
}
