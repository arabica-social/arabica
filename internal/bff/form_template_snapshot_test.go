package bff

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"arabica/internal/models"

	"github.com/ptdewey/shutter"
)

func TestBrewForm_NewBrew_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "new brew with populated selects",
			data: map[string]interface{}{
				"Brew": nil,
				"Beans": []*models.Bean{
					{RKey: "bean1", Name: "Ethiopian Yirgacheffe", Origin: "Ethiopia", RoastLevel: "Light"},
					{RKey: "bean2", Name: "", Origin: "Colombia", RoastLevel: "Medium"},
				},
				"Grinders": []*models.Grinder{
					{RKey: "grinder1", Name: "Baratza Encore"},
					{RKey: "grinder2", Name: "Comandante C40"},
				},
				"Brewers": []*models.Brewer{
					{RKey: "brewer1", Name: "Hario V60"},
					{RKey: "brewer2", Name: "AeroPress"},
				},
				"Roasters": []*models.Roaster{
					{RKey: "roaster1", Name: "Blue Bottle"},
					{RKey: "roaster2", Name: "Counter Culture"},
				},
			},
		},
		{
			name: "new brew with empty selects",
			data: map[string]interface{}{
				"Brew":     nil,
				"Beans":    []*models.Bean{},
				"Grinders": []*models.Grinder{},
				"Brewers":  []*models.Brewer{},
				"Roasters": []*models.Roaster{},
			},
		},
		{
			name: "new brew with nil collections",
			data: map[string]interface{}{
				"Brew":     nil,
				"Beans":    nil,
				"Grinders": nil,
				"Brewers":  nil,
				"Roasters": nil,
			},
		},
	}

	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/brew_form.tmpl",
		"../../templates/partials/new_bean_form.tmpl",
		"../../templates/partials/new_grinder_form.tmpl",
		"../../templates/partials/new_brewer_form.tmpl",
	))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "content", tt.data)
			if err != nil {
				t.Fatalf("template execution failed: %v", err)
			}
			shutter.SnapString(t, tt.name, formatHTML(buf.String()))
		})
	}
}

func TestBrewForm_EditBrew_Snapshot(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "edit brew with complete data",
			data: map[string]interface{}{
				"Brew": &BrewData{
					Brew: &models.Brew{
						RKey:          "brew123",
						BeanRKey:      "bean1",
						GrinderRKey:   "grinder1",
						BrewerRKey:    "brewer1",
						CoffeeAmount:  18,
						WaterAmount:   300,
						GrindSize:     "18",
						Temperature:   93.5,
						TimeSeconds:   180,
						TastingNotes:  "Bright citrus notes with floral aroma. Clean finish.",
						Rating:        8,
						CreatedAt:     timestamp,
						Pours: []*models.Pour{
							{PourNumber: 1, WaterAmount: 50, TimeSeconds: 30},
							{PourNumber: 2, WaterAmount: 100, TimeSeconds: 45},
							{PourNumber: 3, WaterAmount: 150, TimeSeconds: 60},
						},
					},
					PoursJSON: `[{"pourNumber":1,"waterAmount":50,"timeSeconds":30},{"pourNumber":2,"waterAmount":100,"timeSeconds":45},{"pourNumber":3,"waterAmount":150,"timeSeconds":60}]`,
				},
				"Beans": []*models.Bean{
					{RKey: "bean1", Name: "Ethiopian Yirgacheffe", Origin: "Ethiopia", RoastLevel: "Light"},
					{RKey: "bean2", Name: "Colombian Supremo", Origin: "Colombia", RoastLevel: "Medium"},
				},
				"Grinders": []*models.Grinder{
					{RKey: "grinder1", Name: "Baratza Encore"},
					{RKey: "grinder2", Name: "Comandante C40"},
				},
				"Brewers": []*models.Brewer{
					{RKey: "brewer1", Name: "Hario V60"},
					{RKey: "brewer2", Name: "AeroPress"},
				},
				"Roasters": []*models.Roaster{
					{RKey: "roaster1", Name: "Blue Bottle"},
				},
			},
		},
		{
			name: "edit brew with minimal data",
			data: map[string]interface{}{
				"Brew": &BrewData{
					Brew: &models.Brew{
						RKey:        "brew456",
						BeanRKey:    "bean1",
						Rating:      5,
						CreatedAt:   timestamp,
						Pours:       nil,
					},
					PoursJSON: "",
				},
				"Beans": []*models.Bean{
					{RKey: "bean1", Name: "House Blend", Origin: "Brazil", RoastLevel: "Medium"},
				},
				"Grinders": nil,
				"Brewers":  nil,
				"Roasters": nil,
			},
		},
		{
			name: "edit brew with pours json",
			data: map[string]interface{}{
				"Brew": &BrewData{
					Brew: &models.Brew{
						RKey:        "brew789",
						BeanRKey:    "bean1",
						Rating:      7,
						CreatedAt:   timestamp,
						Pours: []*models.Pour{
							{PourNumber: 1, WaterAmount: 60, TimeSeconds: 30},
							{PourNumber: 2, WaterAmount: 120, TimeSeconds: 60},
						},
					},
					PoursJSON: `[{"pourNumber":1,"waterAmount":60,"timeSeconds":30},{"pourNumber":2,"waterAmount":120,"timeSeconds":60}]`,
				},
				"Beans": []*models.Bean{
					{RKey: "bean1", Origin: "Kenya", RoastLevel: "Light"},
				},
				"Grinders": []*models.Grinder{},
				"Brewers":  []*models.Brewer{},
				"Roasters": []*models.Roaster{},
			},
		},
		{
			name: "edit brew without loaded collections",
			data: map[string]interface{}{
				"Brew": &BrewData{
					Brew: &models.Brew{
						RKey:        "brew999",
						BeanRKey:    "bean1",
						GrinderRKey: "grinder1",
						BrewerRKey:  "brewer1",
						Rating:      6,
						CreatedAt:   timestamp,
					},
					PoursJSON: "",
				},
				"Beans":    nil,
				"Grinders": nil,
				"Brewers":  nil,
				"Roasters": nil,
			},
		},
	}

	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/brew_form.tmpl",
		"../../templates/partials/new_bean_form.tmpl",
		"../../templates/partials/new_grinder_form.tmpl",
		"../../templates/partials/new_brewer_form.tmpl",
	))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "content", tt.data)
			if err != nil {
				t.Fatalf("template execution failed: %v", err)
			}
			shutter.SnapString(t, tt.name, formatHTML(buf.String()))
		})
	}
}

func TestNewBeanForm_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "bean form with roasters",
			data: map[string]interface{}{
				"Roasters": []*models.Roaster{
					{RKey: "roaster1", Name: "Blue Bottle Coffee"},
					{RKey: "roaster2", Name: "Counter Culture Coffee"},
					{RKey: "roaster3", Name: "Stumptown Coffee Roasters"},
				},
			},
		},
		{
			name: "bean form without roasters",
			data: map[string]interface{}{
				"Roasters": []*models.Roaster{},
			},
		},
		{
			name: "bean form with nil roasters",
			data: map[string]interface{}{
				"Roasters": nil,
			},
		},
	}

	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/partials/new_bean_form.tmpl",
	))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "new_bean_form", tt.data)
			if err != nil {
				t.Fatalf("template execution failed: %v", err)
			}
			shutter.SnapString(t, tt.name, formatHTML(buf.String()))
		})
	}
}

func TestNewGrinderForm_Snapshot(t *testing.T) {
	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/partials/new_grinder_form.tmpl",
	))

	t.Run("grinder form renders", func(t *testing.T) {
		var buf bytes.Buffer
		err := tmpl.ExecuteTemplate(&buf, "new_grinder_form", nil)
		if err != nil {
			t.Fatalf("template execution failed: %v", err)
		}
		shutter.SnapString(t, "grinder_form_renders", formatHTML(buf.String()))
	})
}

func TestNewBrewerForm_Snapshot(t *testing.T) {
	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/partials/new_brewer_form.tmpl",
	))

	t.Run("brewer form renders", func(t *testing.T) {
		var buf bytes.Buffer
		err := tmpl.ExecuteTemplate(&buf, "new_brewer_form", nil)
		if err != nil {
			t.Fatalf("template execution failed: %v", err)
		}
		shutter.SnapString(t, "brewer_form_renders", formatHTML(buf.String()))
	})
}

func TestNewRoasterForm_Snapshot(t *testing.T) {
	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/partials/new_roaster_form.tmpl",
	))

	t.Run("roaster form renders", func(t *testing.T) {
		var buf bytes.Buffer
		err := tmpl.ExecuteTemplate(&buf, "new_roaster_form", nil)
		if err != nil {
			t.Fatalf("template execution failed: %v", err)
		}
		shutter.SnapString(t, "roaster_form_renders", formatHTML(buf.String()))
	})
}

func TestBrewForm_SpecialCharacters_Snapshot(t *testing.T) {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "brew with html in tasting notes",
			data: map[string]interface{}{
				"Brew": &BrewData{
					Brew: &models.Brew{
						RKey:         "brew1",
						BeanRKey:     "bean1",
						TastingNotes: "<script>alert('xss')</script>Bright & fruity, \"amazing\" taste",
						Rating:       8,
						CreatedAt:    timestamp,
					},
					PoursJSON: "",
				},
				"Beans": []*models.Bean{
					{RKey: "bean1", Name: "Test <strong>Bean</strong>", Origin: "Ethiopia", RoastLevel: "Light"},
				},
				"Grinders": []*models.Grinder{},
				"Brewers":  []*models.Brewer{},
				"Roasters": []*models.Roaster{},
			},
		},
		{
			name: "brew with unicode characters",
			data: map[string]interface{}{
				"Brew": &BrewData{
					Brew: &models.Brew{
						RKey:         "brew2",
						BeanRKey:     "bean1",
						TastingNotes: "日本のコーヒー 🇯🇵 - フルーティーで酸味が強い\n\nЯркий вкус с цитрусовыми нотами\n\nCafé con notas de caramelo",
						GrindSize:    "中挽き (medium)",
						Rating:       9,
						CreatedAt:    timestamp,
					},
					PoursJSON: "",
				},
				"Beans": []*models.Bean{
					{RKey: "bean1", Name: "Café Especial™", Origin: "Costa Rica", RoastLevel: "Medium"},
				},
				"Grinders": []*models.Grinder{
					{RKey: "grinder1", Name: "Comandante® C40 MK3"},
				},
				"Brewers": []*models.Brewer{
					{RKey: "brewer1", Name: "Hario V60 (02)"},
				},
				"Roasters": []*models.Roaster{},
			},
		},
	}

	tmpl := template.Must(template.New("test").Funcs(getTemplateFuncs()).ParseFiles(
		"../../templates/brew_form.tmpl",
		"../../templates/partials/new_bean_form.tmpl",
		"../../templates/partials/new_grinder_form.tmpl",
		"../../templates/partials/new_brewer_form.tmpl",
	))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "content", tt.data)
			if err != nil {
				t.Fatalf("template execution failed: %v", err)
			}
			shutter.SnapString(t, tt.name, formatHTML(buf.String()))
		})
	}
}
