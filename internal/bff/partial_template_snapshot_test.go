package bff

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"arabica/internal/models"

	"github.com/ptdewey/shutter"
)

func TestBrewListContent_Snapshot(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "empty brew list own profile",
			data: map[string]interface{}{
				"Brews":        []*models.Brew{},
				"IsOwnProfile": true,
			},
		},
		{
			name: "empty brew list other profile",
			data: map[string]interface{}{
				"Brews":        []*models.Brew{},
				"IsOwnProfile": false,
			},
		},
		{
			name: "brew list with complete data",
			data: map[string]interface{}{
				"Brews": []*models.Brew{
					{
						RKey:         "brew1",
						BeanRKey:     "bean1",
						CoffeeAmount: 18,
						WaterAmount:  250,
						Temperature:  93.0,
						TimeSeconds:  180,
						GrindSize:    "Medium-fine",
						Rating:       8,
						TastingNotes: "Bright citrus notes with floral aroma. Clean finish.",
						CreatedAt:    timestamp,
						Bean: &models.Bean{
							Name:       "Ethiopian Yirgacheffe",
							Origin:     "Ethiopia",
							RoastLevel: "Light",
							Roaster: &models.Roaster{
								Name: "Onyx Coffee Lab",
							},
						},
						GrinderObj: &models.Grinder{
							Name: "Comandante C40",
						},
						BrewerObj: &models.Brewer{
							Name: "Hario V60",
						},
						Pours: []*models.Pour{
							{PourNumber: 1, WaterAmount: 50, TimeSeconds: 30},
							{PourNumber: 2, WaterAmount: 100, TimeSeconds: 45},
							{PourNumber: 3, WaterAmount: 100, TimeSeconds: 60},
						},
					},
					{
						RKey:        "brew2",
						BeanRKey:    "bean2",
						Rating:      6,
						CreatedAt:   timestamp.Add(-24 * time.Hour),
						Bean: &models.Bean{
							Origin:     "Colombia",
							RoastLevel: "Medium",
						},
						Method: "AeroPress",
					},
				},
				"IsOwnProfile": true,
			},
		},
		{
			name: "brew list minimal data",
			data: map[string]interface{}{
				"Brews": []*models.Brew{
					{
						RKey:      "brew3",
						BeanRKey:  "bean3",
						CreatedAt: timestamp,
					},
				},
				"IsOwnProfile": false,
			},
		},
	}

	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/partials/brew_list_content.tmpl",
	))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "brew_list_content", tt.data)
			if err != nil {
				t.Fatalf("template execution failed: %v", err)
			}
			shutter.SnapString(t, tt.name, formatHTML(buf.String()))
		})
	}
}

func TestManageContent_BeansTab_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "beans empty",
			data: map[string]interface{}{
				"Beans":    []*models.Bean{},
				"Roasters": []*models.Roaster{},
			},
		},
		{
			name: "beans with roaster",
			data: map[string]interface{}{
				"Beans": []*models.Bean{
					{
						RKey:        "bean1",
						Name:        "Ethiopian Yirgacheffe",
						Origin:      "Ethiopia",
						RoastLevel:  "Light",
						Process:     "Washed",
						Description: "Bright and fruity with notes of blueberry",
						RoasterRKey: "roaster1",
						Roaster: &models.Roaster{
							RKey: "roaster1",
							Name: "Onyx Coffee Lab",
						},
					},
					{
						RKey:       "bean2",
						Origin:     "Colombia",
						RoastLevel: "Medium",
					},
				},
				"Roasters": []*models.Roaster{
					{RKey: "roaster1", Name: "Onyx Coffee Lab"},
					{RKey: "roaster2", Name: "Counter Culture"},
				},
			},
		},
	}

	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/partials/manage_content.tmpl",
	))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "manage_content", tt.data)
			if err != nil {
				t.Fatalf("template execution failed: %v", err)
			}
			shutter.SnapString(t, tt.name, formatHTML(buf.String()))
		})
	}
}

func TestManageContent_RoastersTab_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "roasters empty",
			data: map[string]interface{}{
				"Roasters": []*models.Roaster{},
			},
		},
		{
			name: "roasters with data",
			data: map[string]interface{}{
				"Roasters": []*models.Roaster{
					{
						RKey:     "roaster1",
						Name:     "Onyx Coffee Lab",
						Location: "Bentonville, AR",
						Website:  "https://onyxcoffeelab.com",
					},
					{
						RKey: "roaster2",
						Name: "Counter Culture Coffee",
					},
				},
			},
		},
		{
			name: "roasters with unsafe url",
			data: map[string]interface{}{
				"Roasters": []*models.Roaster{
					{
						RKey:     "roaster1",
						Name:     "Test Roaster",
						Location: "Test Location",
						Website:  "javascript:alert('xss')",
					},
				},
			},
		},
	}

	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/partials/manage_content.tmpl",
	))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "manage_content", tt.data)
			if err != nil {
				t.Fatalf("template execution failed: %v", err)
			}
			shutter.SnapString(t, tt.name, formatHTML(buf.String()))
		})
	}
}

func TestManageContent_GrindersTab_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "grinders empty",
			data: map[string]interface{}{
				"Grinders": []*models.Grinder{},
			},
		},
		{
			name: "grinders with data",
			data: map[string]interface{}{
				"Grinders": []*models.Grinder{
					{
						RKey:        "grinder1",
						Name:        "Comandante C40 MK3",
						GrinderType: "Hand",
						BurrType:    "Conical",
						Notes:       "Excellent consistency, great for pour-over",
					},
					{
						RKey:        "grinder2",
						Name:        "Baratza Encore",
						GrinderType: "Electric",
						BurrType:    "Conical",
					},
				},
			},
		},
	}

	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/partials/manage_content.tmpl",
	))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "manage_content", tt.data)
			if err != nil {
				t.Fatalf("template execution failed: %v", err)
			}
			shutter.SnapString(t, tt.name, formatHTML(buf.String()))
		})
	}
}

func TestManageContent_BrewersTab_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "brewers empty",
			data: map[string]interface{}{
				"Brewers": []*models.Brewer{},
			},
		},
		{
			name: "brewers with data",
			data: map[string]interface{}{
				"Brewers": []*models.Brewer{
					{
						RKey:        "brewer1",
						Name:        "Hario V60",
						BrewerType:  "Pour-Over",
						Description: "Cone-shaped dripper for clean, bright brews",
					},
					{
						RKey: "brewer2",
						Name: "AeroPress",
					},
				},
			},
		},
	}

	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/partials/manage_content.tmpl",
	))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "manage_content", tt.data)
			if err != nil {
				t.Fatalf("template execution failed: %v", err)
			}
			shutter.SnapString(t, tt.name, formatHTML(buf.String()))
		})
	}
}

func TestManageContent_SpecialCharacters_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "beans with special characters and html",
			data: map[string]interface{}{
				"Beans": []*models.Bean{
					{
						RKey:        "bean1",
						Name:        "Café <script>alert('xss')</script> Especial",
						Origin:      "Costa Rica™",
						RoastLevel:  "Medium",
						Process:     "Honey & Washed",
						Description: "\"Amazing\" coffee with <strong>bold</strong> flavor",
					},
				},
				"Roasters": []*models.Roaster{},
			},
		},
		{
			name: "grinders with unicode",
			data: map[string]interface{}{
				"Grinders": []*models.Grinder{
					{
						RKey:        "grinder1",
						Name:        "手動コーヒーミル Comandante® C40",
						GrinderType: "Hand",
						BurrType:    "Conical",
						Notes:       "日本語のノート - Отличная кофемолка 🇯🇵",
					},
				},
			},
		},
	}

	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/partials/manage_content.tmpl",
	))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "manage_content", tt.data)
			if err != nil {
				t.Fatalf("template execution failed: %v", err)
			}
			shutter.SnapString(t, tt.name, formatHTML(buf.String()))
		})
	}
}
