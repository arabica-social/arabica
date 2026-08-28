package arabicastore

import (
	"testing"

	"github.com/stretchr/testify/assert"
	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
)

func TestLinkBeansToRoasters(t *testing.T) {
	t.Run("links beans to matching roasters", func(t *testing.T) {
		roasters := []*arabica.Roaster{
			{RKey: "roaster1", Name: "Roaster One"},
			{RKey: "roaster2", Name: "Roaster Two"},
			{RKey: "roaster3", Name: "Roaster Three"},
		}

		beans := []*arabica.Bean{
			{RKey: "bean1", Name: "Bean One", RoasterRKey: "roaster1"},
			{RKey: "bean2", Name: "Bean Two", RoasterRKey: "roaster2"},
			{RKey: "bean3", Name: "Bean Three", RoasterRKey: ""},
		}

		LinkBeansToRoasters(beans, roasters)

		assert.NotNil(t, beans[0].Roaster)
		assert.Equal(t, "Roaster One", beans[0].Roaster.Name)
		assert.NotNil(t, beans[1].Roaster)
		assert.Equal(t, "Roaster Two", beans[1].Roaster.Name)
		assert.Nil(t, beans[2].Roaster)
	})

	t.Run("handles missing roaster gracefully", func(t *testing.T) {
		roasters := []*arabica.Roaster{
			{RKey: "roaster1", Name: "Roaster One"},
		}

		beans := []*arabica.Bean{
			{RKey: "bean1", Name: "Bean One", RoasterRKey: "nonexistent"},
		}

		LinkBeansToRoasters(beans, roasters)

		assert.Nil(t, beans[0].Roaster)
	})

	t.Run("handles empty slices", func(t *testing.T) {
		assert.NotPanics(t, func() {
			LinkBeansToRoasters(nil, nil)
			LinkBeansToRoasters([]*arabica.Bean{}, []*arabica.Roaster{})
		})
	})
}

func TestExtractBrewRefRKeys(t *testing.T) {
	t.Run("derives recipe owner DID from the recipeRef authority", func(t *testing.T) {
		brew := &arabica.Brew{}
		record := map[string]any{
			"beanRef":    "at://did:plc:brewer123/social.arabica.alpha.bean/bean1",
			"grinderRef": "at://did:plc:brewer123/social.arabica.alpha.grinder/grinder1",
			"recipeRef":  "at://did:plc:chef456/social.arabica.alpha.recipe/recipe9",
		}

		ExtractBrewRefRKeys(brew, record)

		assert.Equal(t, "bean1", brew.BeanRKey)
		assert.Equal(t, "grinder1", brew.GrinderRKey)
		assert.Equal(t, "recipe9", brew.RecipeRKey)
		assert.Equal(t, "did:plc:chef456", brew.RecipeOwnerDID)
	})

	t.Run("no recipeRef leaves recipe fields empty", func(t *testing.T) {
		brew := &arabica.Brew{}
		record := map[string]any{
			"beanRef": "at://did:plc:brewer123/social.arabica.alpha.bean/bean1",
		}

		ExtractBrewRefRKeys(brew, record)

		assert.Equal(t, "bean1", brew.BeanRKey)
		assert.Empty(t, brew.RecipeRKey)
		assert.Empty(t, brew.RecipeOwnerDID)
	})

	t.Run("invalid recipeRef is ignored", func(t *testing.T) {
		brew := &arabica.Brew{}
		record := map[string]any{
			"recipeRef": "not-an-at-uri",
		}

		ExtractBrewRefRKeys(brew, record)

		assert.Empty(t, brew.RecipeRKey)
		assert.Empty(t, brew.RecipeOwnerDID)
	})
}
