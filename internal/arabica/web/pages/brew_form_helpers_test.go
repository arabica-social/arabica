package coffeepages

import (
	"bytes"
	"context"
	"testing"

	"github.com/a-h/templ"
	"github.com/stretchr/testify/assert"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

func renderBrewFormTestComponent(t *testing.T, component templ.Component) string {
	t.Helper()
	var buf bytes.Buffer
	err := component.Render(context.Background(), &buf)
	assert.NoError(t, err)
	return buf.String()
}

func TestPoursToJSON(t *testing.T) {
	tests := []struct {
		name     string
		pours    []*arabica.Pour
		expected string
	}{
		{
			name:     "empty pours",
			pours:    []*arabica.Pour{},
			expected: "[]",
		},
		{
			name:     "nil pours",
			pours:    nil,
			expected: "[]",
		},
		{
			name: "single pour",
			pours: []*arabica.Pour{
				{WaterAmount: 50, TimeSeconds: 30},
			},
			expected: `[{"water":50,"time":30}]`,
		},
		{
			name: "multiple pours",
			pours: []*arabica.Pour{
				{WaterAmount: 50, TimeSeconds: 30},
				{WaterAmount: 100, TimeSeconds: 60},
				{WaterAmount: 150, TimeSeconds: 90},
			},
			expected: `[{"water":50,"time":30},{"water":100,"time":60},{"water":150,"time":90}]`,
		},
		{
			name: "zero values",
			pours: []*arabica.Pour{
				{WaterAmount: 0, TimeSeconds: 0},
			},
			expected: `[{"water":0,"time":0}]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PoursToJSON(tt.pours)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestBrewFormIslandMountContractForNewBrew(t *testing.T) {
	props := BrewFormProps{
		RecipeRKey:     "recipe-123",
		RecipeOwnerDID: "did:plc:recipeowner",
		PoursJSON:      `[{"water":50,"time":30}]`,
	}

	html := renderBrewFormTestComponent(t, BrewFormElement(props))

	assert.Contains(t, html, `hx-post="/brews"`)
	assert.Contains(t, html, `data-svelte-brew-form`)
	assert.Contains(t, html, `data-submit-label="Save Brew"`)
	assert.Contains(t, html, `data-recipe-rkey="recipe-123"`)
	assert.Contains(t, html, `data-recipe-owner="did:plc:recipeowner"`)
	assert.Contains(t, html, `data-rating="5"`)
	assert.Contains(t, html, `data-pours="[{&#34;water&#34;:50,&#34;time&#34;:30}]"`)
	assert.Contains(t, html, `JavaScript is required to log brews with the current form.`)
}

func TestBrewFormIslandMountContractForEditBrew(t *testing.T) {
	props := BrewFormProps{
		Brew: &arabica.Brew{
			RKey:           "brew-123",
			RecipeRKey:     "recipe-456",
			BeanRKey:       "bean-123",
			GrinderRKey:    "grinder-123",
			BrewerRKey:     "brewer-123",
			CoffeeAmount:   18,
			WaterAmount:    250,
			GrindSize:      "medium",
			Temperature:    93.5,
			TimeSeconds:    180,
			TastingNotes:   "sweet and bright",
			Rating:         8,
			EspressoParams: &arabica.EspressoParams{YieldWeight: 36, Pressure: 9, PreInfusionSeconds: 5},
			Bean:           &arabica.Bean{RKey: "bean-123", Name: "Chelbesa", Origin: "Ethiopia", RoastLevel: "Light"},
			GrinderObj:     &arabica.Grinder{RKey: "grinder-123", Name: "Comandante"},
			BrewerObj:      &arabica.Brewer{RKey: "brewer-123", Name: "Linea Mini", BrewerType: "espresso"},
			RecipeObj:      &arabica.Recipe{RKey: "recipe-456", Name: "Morning Shot"},
		},
		PoursJSON: `[]`,
		Recipes:   []arabica.Recipe{{RKey: "recipe-456", Name: "Morning Shot"}},
	}

	html := renderBrewFormTestComponent(t, BrewFormElement(props))

	assert.Contains(t, html, `hx-put="/brews/brew-123"`)
	assert.Contains(t, html, `data-editing="true"`)
	assert.Contains(t, html, `data-submit-label="Update Brew"`)
	assert.Contains(t, html, `data-recipe-rkey="recipe-456"`)
	assert.Contains(t, html, `data-recipe-label="Morning Shot"`)
	assert.Contains(t, html, `data-bean-rkey="bean-123"`)
	assert.Contains(t, html, `data-bean-label="Chelbesa (Ethiopia - Light)"`)
	assert.Contains(t, html, `data-grinder-rkey="grinder-123"`)
	assert.Contains(t, html, `data-grinder-label="Comandante"`)
	assert.Contains(t, html, `data-brewer-rkey="brewer-123"`)
	assert.Contains(t, html, `data-brewer-label="Linea Mini"`)
	assert.Contains(t, html, `data-brewer-category="espresso"`)
	assert.Contains(t, html, `data-coffee-amount="18"`)
	assert.Contains(t, html, `data-water-amount="250"`)
	assert.Contains(t, html, `data-grind-size="medium"`)
	assert.Contains(t, html, `data-temperature="93.5"`)
	assert.Contains(t, html, `data-time-seconds="180"`)
	assert.Contains(t, html, `data-tasting-notes="sweet and bright"`)
	assert.Contains(t, html, `data-rating="8"`)
	assert.Contains(t, html, `data-espresso-yield-weight="36.0"`)
	assert.Contains(t, html, `data-espresso-pressure="9.0"`)
	assert.Contains(t, html, `data-espresso-pre-infusion-seconds="5"`)
}
