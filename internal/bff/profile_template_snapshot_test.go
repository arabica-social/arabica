package bff

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"github.com/ptdewey/shutter"

	"arabica/internal/models"
)

// Test profile content partial rendering

func TestProfileContent_BeansTab_Snapshot(t *testing.T) {
	tmpl := template.New("test").Funcs(getTemplateFuncs())
	tmpl, err := tmpl.ParseFiles(
		"../../templates/partials/profile_content.tmpl",
		"../../templates/partials/brew_list_content.tmpl",
	)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "profile with multiple beans",
			data: map[string]interface{}{
				"Beans": []*models.Bean{
					{
						RKey:        "bean1",
						Name:        "Ethiopian Yirgacheffe",
						Origin:      "Ethiopia",
						RoastLevel:  "Light",
						Process:     "Washed",
						Description: "Bright and floral with citrus notes",
						Roaster: &models.Roaster{
							RKey:     "roaster1",
							Name:     "Onyx Coffee Lab",
							Location: "Arkansas",
							Website:  "https://onyxcoffeelab.com",
						},
						CreatedAt: testTime,
					},
					{
						RKey:        "bean2",
						Name:        "Colombia Supremo",
						Origin:      "Colombia",
						RoastLevel:  "Medium",
						Process:     "Natural",
						Description: "",
						CreatedAt:   testTime,
					},
				},
				"Roasters": []*models.Roaster{},
				"Grinders": []*models.Grinder{},
				"Brewers":  []*models.Brewer{},
				"Brews":    []*models.Brew{},
				"IsOwnProfile": true,
			},
		},
		{
			name: "profile with empty beans",
			data: map[string]interface{}{
				"Beans":        []*models.Bean{},
				"Roasters":     []*models.Roaster{},
				"Grinders":     []*models.Grinder{},
				"Brewers":      []*models.Brewer{},
				"Brews":        []*models.Brew{},
				"IsOwnProfile": false,
			},
		},
		{
			name: "bean with missing optional fields",
			data: map[string]interface{}{
				"Beans": []*models.Bean{
					{
						RKey:      "bean3",
						Name:      "Mystery Bean",
						CreatedAt: testTime,
					},
				},
				"Roasters":     []*models.Roaster{},
				"Grinders":     []*models.Grinder{},
				"Brewers":      []*models.Brewer{},
				"Brews":        []*models.Brew{},
				"IsOwnProfile": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "profile_content", tt.data)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}
			shutter.Snap(t, tt.name, buf.String())
		})
	}
}

func TestProfileContent_GearTabs_Snapshot(t *testing.T) {
	tmpl := template.New("test").Funcs(getTemplateFuncs())
	tmpl, err := tmpl.ParseFiles(
		"../../templates/partials/profile_content.tmpl",
		"../../templates/partials/brew_list_content.tmpl",
	)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	data := map[string]interface{}{
		"Beans": []*models.Bean{},
		"Roasters": []*models.Roaster{
			{
				RKey:      "roaster1",
				Name:      "Heart Coffee",
				Location:  "Portland, OR",
				Website:   "https://heartroasters.com",
				CreatedAt: testTime,
			},
		},
		"Grinders": []*models.Grinder{
			{
				RKey:        "grinder1",
				Name:        "Comandante C40",
				GrinderType: "Hand",
				BurrType:    "Conical",
				Notes:       "Perfect for pour over",
				CreatedAt:   testTime,
			},
			{
				RKey:        "grinder2",
				Name:        "Niche Zero",
				GrinderType: "Electric",
				BurrType:    "Conical",
				CreatedAt:   testTime,
			},
		},
		"Brewers": []*models.Brewer{
			{
				RKey:        "brewer1",
				Name:        "Hario V60",
				BrewerType:  "Pour Over",
				Description: "Classic pour over cone",
				CreatedAt:   testTime,
			},
		},
		"Brews":        []*models.Brew{},
		"IsOwnProfile": true,
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "profile_content", data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	shutter.Snap(t, "profile with gear collection", buf.String())
}

func TestProfileContent_URLSecurity_Snapshot(t *testing.T) {
	tmpl := template.New("test").Funcs(getTemplateFuncs())
	tmpl, err := tmpl.ParseFiles(
		"../../templates/partials/profile_content.tmpl",
		"../../templates/partials/brew_list_content.tmpl",
	)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "profile roaster with unsafe website URL",
			data: map[string]interface{}{
				"Beans": []*models.Bean{},
				"Roasters": []*models.Roaster{
					{
						RKey:      "roaster1",
						Name:      "Sketchy Roaster",
						Location:  "Unknown",
						Website:   "javascript:alert('xss')", // Should be sanitized
						CreatedAt: testTime,
					},
				},
				"Grinders":     []*models.Grinder{},
				"Brewers":      []*models.Brewer{},
				"Brews":        []*models.Brew{},
				"IsOwnProfile": false,
			},
		},
		{
			name: "profile roaster with invalid URL protocol",
			data: map[string]interface{}{
				"Beans": []*models.Bean{},
				"Roasters": []*models.Roaster{
					{
						RKey:      "roaster2",
						Name:      "FTP Roaster",
						Website:   "ftp://example.com", // Should be rejected
						CreatedAt: testTime,
					},
				},
				"Grinders":     []*models.Grinder{},
				"Brewers":      []*models.Brewer{},
				"Brews":        []*models.Brew{},
				"IsOwnProfile": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "profile_content", tt.data)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}
			shutter.Snap(t, tt.name, buf.String())
		})
	}
}

func TestProfileContent_SpecialCharacters_Snapshot(t *testing.T) {
	tmpl := template.New("test").Funcs(getTemplateFuncs())
	tmpl, err := tmpl.ParseFiles(
		"../../templates/partials/profile_content.tmpl",
		"../../templates/partials/brew_list_content.tmpl",
	)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	data := map[string]interface{}{
		"Beans": []*models.Bean{
			{
				RKey:        "bean1",
				Name:        "Bean with <html> & \"quotes\"",
				Origin:      "Colombia & Peru",
				Description: "Description with 'single' and \"double\" quotes",
				CreatedAt:   testTime,
			},
		},
		"Roasters": []*models.Roaster{},
		"Grinders": []*models.Grinder{
			{
				RKey:        "grinder1",
				Name:        "Grinder & Co.",
				Notes:       "Notes with <script>alert('xss')</script>",
				CreatedAt:   testTime,
			},
		},
		"Brewers":      []*models.Brewer{},
		"Brews":        []*models.Brew{},
		"IsOwnProfile": true,
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "profile_content", data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	shutter.Snap(t, "profile with special characters", buf.String())
}

func TestProfileContent_Unicode_Snapshot(t *testing.T) {
	tmpl := template.New("test").Funcs(getTemplateFuncs())
	tmpl, err := tmpl.ParseFiles(
		"../../templates/partials/profile_content.tmpl",
		"../../templates/partials/brew_list_content.tmpl",
	)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	data := map[string]interface{}{
		"Beans": []*models.Bean{
			{
				RKey:        "bean1",
				Name:        "エチオピア イルガチェフェ", // Japanese
				Origin:      "日本",
				Description: "明るく花のような香り",
				CreatedAt:   testTime,
			},
			{
				RKey:        "bean2",
				Name:        "Café de Colombia",
				Origin:      "Bogotá",
				Description: "Suave y aromático",
				CreatedAt:   testTime,
			},
		},
		"Roasters": []*models.Roaster{
			{
				RKey:      "roaster1",
				Name:      "Кофейня Москва", // Russian
				Location:  "Москва, Россия",
				CreatedAt: testTime,
			},
		},
		"Grinders":     []*models.Grinder{},
		"Brewers":      []*models.Brewer{},
		"Brews":        []*models.Brew{},
		"IsOwnProfile": false,
	}

	var buf bytes.Buffer
	err = tmpl.ExecuteTemplate(&buf, "profile_content", data)
	if err != nil {
		t.Fatalf("Failed to execute template: %v", err)
	}
	shutter.Snap(t, "profile with unicode content", buf.String())
}
