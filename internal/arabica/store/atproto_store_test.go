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
