package bff

import (
	"bytes"
	"html/template"
	"testing"
	"time"

	"github.com/ptdewey/shutter"

	"arabica/internal/models"
)

func TestDict_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		args []interface{}
	}{
		{
			name: "empty dict",
			args: []interface{}{},
		},
		{
			name: "single key-value",
			args: []interface{}{"key1", "value1"},
		},
		{
			name: "multiple key-values",
			args: []interface{}{"name", "Ethiopian", "roast", "Light", "rating", 9},
		},
		{
			name: "nested values",
			args: []interface{}{
				"bean", map[string]interface{}{"name": "Ethiopian", "origin": "Yirgacheffe"},
				"brew", map[string]interface{}{"method": "V60", "temp": 93.0},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Dict(tt.args...)
			if err != nil {
				shutter.Snap(t, tt.name+"_error", err.Error())
			} else {
				shutter.Snap(t, tt.name, result)
			}
		})
	}
}

func TestDict_ErrorCases_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		args []interface{}
	}{
		{
			name: "odd number of arguments",
			args: []interface{}{"key1", "value1", "key2"},
		},
		{
			name: "non-string key",
			args: []interface{}{123, "value1"},
		},
		{
			name: "mixed valid and invalid",
			args: []interface{}{"key1", "value1", 456, "value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Dict(tt.args...)
			if err != nil {
				shutter.Snap(t, tt.name, err.Error())
			} else {
				shutter.Snap(t, tt.name, "no error")
			}
		})
	}
}

func TestTemplateRendering_BeanCard_Snapshot(t *testing.T) {
	// Create a minimal template with the bean_card template
	tmplStr := `{{define "bean_card"}}
<div class="bean-card">
  <h3>{{.Bean.Name}}</h3>
  {{if .Bean.Origin}}<p>Origin: {{.Bean.Origin}}</p>{{end}}
  {{if .Bean.RoastLevel}}<p>Roast: {{.Bean.RoastLevel}}</p>{{end}}
</div>
{{end}}`

	tmpl := template.New("test").Funcs(getTemplateFuncs())
	tmpl, err := tmpl.Parse(tmplStr)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "full bean data",
			data: map[string]interface{}{
				"Bean": &models.Bean{
					Name:       "Ethiopian Yirgacheffe",
					Origin:     "Ethiopia",
					RoastLevel: "Light",
					Process:    "Washed",
				},
				"IsOwnProfile": true,
			},
		},
		{
			name: "minimal bean data",
			data: map[string]interface{}{
				"Bean": &models.Bean{
					Name: "House Blend",
				},
				"IsOwnProfile": false,
			},
		},
		{
			name: "bean with only origin",
			data: map[string]interface{}{
				"Bean": &models.Bean{
					Origin: "Colombia",
				},
				"IsOwnProfile": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "bean_card", tt.data)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}
			shutter.Snap(t, tt.name, buf.String())
		})
	}
}

func TestTemplateRendering_BrewCard_Snapshot(t *testing.T) {
	// Simplified brew card template for testing
	tmplStr := `{{define "brew_card"}}
<div class="brew-card">
  <div class="date">{{.Brew.CreatedAt.Format "Jan 2, 2006"}}</div>
  {{if .Brew.Bean}}
  <div class="bean">{{.Brew.Bean.Name}}</div>
  {{end}}
  {{if hasValue .Brew.Rating}}
  <div class="rating">{{formatRating .Brew.Rating}}</div>
  {{end}}
  {{if .Brew.TastingNotes}}
  <div class="notes">{{.Brew.TastingNotes}}</div>
  {{end}}
</div>
{{end}}`

	tmpl := template.New("test").Funcs(getTemplateFuncs())
	tmpl, err := tmpl.Parse(tmplStr)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	testTime := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "complete brew",
			data: map[string]interface{}{
				"Brew": &models.Brew{
					CreatedAt: testTime,
					Bean: &models.Bean{
						Name:   "Ethiopian Yirgacheffe",
						Origin: "Ethiopia",
					},
					Rating:       9,
					TastingNotes: "Bright citrus notes with floral aroma",
				},
				"IsOwnProfile": true,
			},
		},
		{
			name: "minimal brew",
			data: map[string]interface{}{
				"Brew": &models.Brew{
					CreatedAt: testTime,
				},
				"IsOwnProfile": false,
			},
		},
		{
			name: "brew with zero rating",
			data: map[string]interface{}{
				"Brew": &models.Brew{
					CreatedAt: testTime,
					Bean: &models.Bean{
						Name: "House Blend",
					},
					Rating: 0,
				},
				"IsOwnProfile": true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "brew_card", tt.data)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}
			shutter.Snap(t, tt.name, buf.String())
		})
	}
}

func TestTemplateRendering_GearCards_Snapshot(t *testing.T) {
	// Simplified grinder card template
	tmplStr := `{{define "grinder_card"}}
<div class="grinder-card">
  <h3>{{.Grinder.Name}}</h3>
  {{if .Grinder.GrinderType}}<p>Type: {{.Grinder.GrinderType}}</p>{{end}}
  {{if .Grinder.BurrType}}<p>Burr: {{.Grinder.BurrType}}</p>{{end}}
</div>
{{end}}`

	tmpl := template.New("test").Funcs(getTemplateFuncs())
	tmpl, err := tmpl.Parse(tmplStr)
	if err != nil {
		t.Fatalf("Failed to parse template: %v", err)
	}

	tests := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "full grinder data",
			data: map[string]interface{}{
				"Grinder": &models.Grinder{
					Name:        "1Zpresso JX-Pro",
					GrinderType: "Hand",
					BurrType:    "Conical",
				},
				"IsOwnProfile": true,
			},
		},
		{
			name: "minimal grinder data",
			data: map[string]interface{}{
				"Grinder": &models.Grinder{
					Name: "Generic Grinder",
				},
				"IsOwnProfile": false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tmpl.ExecuteTemplate(&buf, "grinder_card", tt.data)
			if err != nil {
				t.Fatalf("Failed to execute template: %v", err)
			}
			shutter.Snap(t, tt.name, buf.String())
		})
	}
}

func TestFormatHelpers_Snapshot(t *testing.T) {
	t.Run("temperature formatting", func(t *testing.T) {
		temps := []float64{0, 20.5, 93.0, 100.0, 200.5, 212.0}
		results := make([]string, len(temps))
		for i, temp := range temps {
			results[i] = FormatTemp(temp)
		}
		shutter.Snap(t, "temperature_formatting", results)
	})

	t.Run("time formatting", func(t *testing.T) {
		times := []int{0, 15, 60, 90, 180, 245}
		results := make([]string, len(times))
		for i, sec := range times {
			results[i] = FormatTime(sec)
		}
		shutter.Snap(t, "time_formatting", results)
	})

	t.Run("rating formatting", func(t *testing.T) {
		ratings := []int{0, 1, 5, 7, 10}
		results := make([]string, len(ratings))
		for i, rating := range ratings {
			results[i] = FormatRating(rating)
		}
		shutter.Snap(t, "rating_formatting", results)
	})
}

func TestSafeURL_Snapshot(t *testing.T) {
	t.Run("avatar URLs", func(t *testing.T) {
		urls := []string{
			"",
			"/static/icon-placeholder.svg",
			"https://cdn.bsky.app/avatar.jpg",
			"https://av-cdn.bsky.app/img/avatar/plain/did:plc:test/abc@jpeg",
			"http://cdn.bsky.app/avatar.jpg",
			"https://evil.com/xss.jpg",
			"/../../etc/passwd",
			"javascript:alert('xss')",
		}
		results := make([]string, len(urls))
		for i, url := range urls {
			results[i] = SafeAvatarURL(url)
		}
		shutter.Snap(t, "avatar_urls", results)
	})

	t.Run("website URLs", func(t *testing.T) {
		urls := []string{
			"",
			"https://example.com",
			"http://example.com",
			"https://roastery.coffee/beans",
			"javascript:alert('xss')",
			"ftp://example.com",
			"https://",
			"example.com",
		}
		results := make([]string, len(urls))
		for i, url := range urls {
			results[i] = SafeWebsiteURL(url)
		}
		shutter.Snap(t, "website_urls", results)
	})
}

func TestEscapeJS_Snapshot(t *testing.T) {
	inputs := []string{
		"simple string",
		"string with 'single quotes'",
		"string with \"double quotes\"",
		"line1\nline2",
		"tab\there",
		"backslash\\test",
		"mixed: 'quotes', \"quotes\", \n newlines \t tabs",
		"",
	}

	results := make([]string, len(inputs))
	for i, input := range inputs {
		results[i] = EscapeJS(input)
	}

	shutter.Snap(t, "escape_js", results)
}

func TestPoursToJSON_Snapshot(t *testing.T) {
	tests := []struct {
		name  string
		pours []*models.Pour
	}{
		{
			name:  "empty pours",
			pours: []*models.Pour{},
		},
		{
			name: "single pour",
			pours: []*models.Pour{
				{PourNumber: 1, WaterAmount: 50, TimeSeconds: 30},
			},
		},
		{
			name: "multiple pours",
			pours: []*models.Pour{
				{PourNumber: 1, WaterAmount: 50, TimeSeconds: 30},
				{PourNumber: 2, WaterAmount: 100, TimeSeconds: 45},
				{PourNumber: 3, WaterAmount: 80, TimeSeconds: 60},
			},
		},
		{
			name: "nil pours",
			pours: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PoursToJSON(tt.pours)
			shutter.Snap(t, tt.name, result)
		})
	}
}

func TestIterate_Snapshot(t *testing.T) {
	tests := []struct {
		name  string
		count int
	}{
		{name: "zero count", count: 0},
		{name: "single iteration", count: 1},
		{name: "five iterations", count: 5},
		{name: "ten iterations", count: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Iterate(tt.count)
			shutter.Snap(t, tt.name, result)
		})
	}
}

func TestIterateRemaining_Snapshot(t *testing.T) {
	tests := []struct {
		name  string
		total int
		used  int
	}{
		{name: "all used", total: 10, used: 10},
		{name: "none used", total: 10, used: 0},
		{name: "half used", total: 10, used: 5},
		{name: "rating 7 out of 10", total: 10, used: 7},
		{name: "rating 3 out of 10", total: 10, used: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IterateRemaining(tt.total, tt.used)
			shutter.Snap(t, tt.name, result)
		})
	}
}

func TestHasTemp_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		temp float64
	}{
		{name: "zero temperature", temp: 0},
		{name: "positive temperature", temp: 93.0},
		{name: "negative temperature", temp: -5.0},
		{name: "small positive", temp: 0.1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasTemp(tt.temp)
			shutter.Snap(t, tt.name, result)
		})
	}
}

func TestHasValue_Snapshot(t *testing.T) {
	tests := []struct {
		name  string
		value int
	}{
		{name: "zero value", value: 0},
		{name: "positive value", value: 5},
		{name: "negative value", value: -3},
		{name: "large value", value: 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HasValue(tt.value)
			shutter.Snap(t, tt.name, result)
		})
	}
}

func TestPtrEquals_Snapshot(t *testing.T) {
	str1 := "test"
	str2 := "different"

	tests := []struct {
		name string
		ptr  *string
		val  string
	}{
		{name: "nil pointer", ptr: nil, val: "test"},
		{name: "equal values", ptr: &str1, val: "test"},
		{name: "different values", ptr: &str1, val: str2},
		{name: "pointer to empty vs empty", ptr: &([]string{""}[0]), val: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PtrEquals(tt.ptr, tt.val)
			shutter.Snap(t, tt.name, result)
		})
	}
}

func TestPtrValue_Snapshot(t *testing.T) {
	str1 := "test value"
	num1 := 42

	tests := []struct {
		name string
		ptr  interface{}
	}{
		{name: "nil string pointer", ptr: (*string)(nil)},
		{name: "valid string pointer", ptr: &str1},
		{name: "nil int pointer", ptr: (*int)(nil)},
		{name: "valid int pointer", ptr: &num1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result interface{}
			switch v := tt.ptr.(type) {
			case *string:
				result = PtrValue(v)
			case *int:
				result = PtrValue(v)
			}
			shutter.Snap(t, tt.name, result)
		})
	}
}

func TestFormatTempValue_Snapshot(t *testing.T) {
	tests := []struct {
		name string
		temp float64
	}{
		{name: "zero temp", temp: 0},
		{name: "celsius temp", temp: 93.0},
		{name: "fahrenheit temp", temp: 205.0},
		{name: "decimal celsius", temp: 92.5},
		{name: "decimal fahrenheit", temp: 201.8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatTempValue(tt.temp)
			shutter.Snap(t, tt.name, result)
		})
	}
}
