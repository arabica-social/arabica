package handlers

import (
	"testing"

	"github.com/stretchr/testify/assert"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/entities"

	"tangled.org/arabica.social/arabica/internal/lexicons"
)

func TestFeedViews_AllArabicaEntitiesHaveFilterTab(t *testing.T) {
	views := FeedViews()
	want := map[lexicons.RecordType]string{
		lexicons.RecordTypeBean:    "Beans",
		lexicons.RecordTypeRoaster: "",
		lexicons.RecordTypeGrinder: "Grinders",
		lexicons.RecordTypeBrewer:  "Brewers",
		lexicons.RecordTypeRecipe:  "Recipes",
		lexicons.RecordTypeBrew:    "Brews",
	}
	for rt, filterLabel := range want {
		d := entities.Get(rt)
		assert.NotNil(t, d, "descriptor missing for %s", rt)
		if d == nil {
			continue
		}
		assert.Equal(t, filterLabel, views.FilterLabel(rt), "feed filter label for %s", rt)
	}
}

func TestFeedViews_FilterTabsMatchFilterValues(t *testing.T) {
	views := FeedViews()
	for rt, view := range views {
		assert.NotEmpty(t, view.FilterLabel, "%s tab needs a label", rt)
		assert.Equal(t, string(rt), view.FilterValue, "%s tab value should select its record type", rt)
	}
}

func TestFeedViews_FeedTabJSONUnchanged(t *testing.T) {
	tabs := BuildFeedTabs(entities.AllForApp(arabica.NSIDBase), FeedViews())
	assert.Equal(t, []FeedFilterTabJSON{
		{Label: "All", Value: ""},
		{Label: "Beans", Value: "bean"},
		{Label: "Brews", Value: "brew"},
		{Label: "Brewers", Value: "brewer"},
		{Label: "Grinders", Value: "grinder"},
		{Label: "Recipes", Value: "recipe"},
	}, tabs)
}
