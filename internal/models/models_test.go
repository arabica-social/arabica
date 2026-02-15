package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateBeanRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &CreateBeanRequest{Name: "Ethiopian Yirgacheffe"}
		assert.NoError(t, req.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		req := &CreateBeanRequest{Name: ""}
		assert.ErrorIs(t, req.Validate(), ErrNameRequired)
	})

	t.Run("name too long", func(t *testing.T) {
		req := &CreateBeanRequest{Name: strings.Repeat("a", MaxNameLength+1)}
		assert.ErrorIs(t, req.Validate(), ErrNameTooLong)
	})

	t.Run("name at max length", func(t *testing.T) {
		req := &CreateBeanRequest{Name: strings.Repeat("a", MaxNameLength)}
		assert.NoError(t, req.Validate())
	})

	t.Run("origin too long", func(t *testing.T) {
		req := &CreateBeanRequest{
			Name:   "Bean",
			Origin: strings.Repeat("a", MaxOriginLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrOriginTooLong)
	})

	t.Run("roast level too long", func(t *testing.T) {
		req := &CreateBeanRequest{
			Name:       "Bean",
			RoastLevel: strings.Repeat("a", MaxRoastLevelLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrFieldTooLong)
	})

	t.Run("process too long", func(t *testing.T) {
		req := &CreateBeanRequest{
			Name:    "Bean",
			Process: strings.Repeat("a", MaxProcessLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrFieldTooLong)
	})

	t.Run("description too long", func(t *testing.T) {
		req := &CreateBeanRequest{
			Name:        "Bean",
			Description: strings.Repeat("a", MaxDescriptionLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrDescTooLong)
	})

	t.Run("all optional fields populated", func(t *testing.T) {
		req := &CreateBeanRequest{
			Name:        "Ethiopian Yirgacheffe",
			Origin:      "Ethiopia",
			RoastLevel:  "Light",
			Process:     "Washed",
			Description: "Fruity and floral notes",
			RoasterRKey: "abc123",
		}
		assert.NoError(t, req.Validate())
	})
}

func TestUpdateBeanRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &UpdateBeanRequest{Name: "Updated Bean"}
		assert.NoError(t, req.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		req := &UpdateBeanRequest{Name: ""}
		assert.ErrorIs(t, req.Validate(), ErrNameRequired)
	})

	t.Run("name too long", func(t *testing.T) {
		req := &UpdateBeanRequest{Name: strings.Repeat("a", MaxNameLength+1)}
		assert.ErrorIs(t, req.Validate(), ErrNameTooLong)
	})

	t.Run("origin too long", func(t *testing.T) {
		req := &UpdateBeanRequest{
			Name:   "Bean",
			Origin: strings.Repeat("a", MaxOriginLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrOriginTooLong)
	})

	t.Run("description too long", func(t *testing.T) {
		req := &UpdateBeanRequest{
			Name:        "Bean",
			Description: strings.Repeat("a", MaxDescriptionLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrDescTooLong)
	})
}

func TestCreateRoasterRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &CreateRoasterRequest{Name: "Blue Bottle"}
		assert.NoError(t, req.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		req := &CreateRoasterRequest{Name: ""}
		assert.ErrorIs(t, req.Validate(), ErrNameRequired)
	})

	t.Run("name too long", func(t *testing.T) {
		req := &CreateRoasterRequest{Name: strings.Repeat("a", MaxNameLength+1)}
		assert.ErrorIs(t, req.Validate(), ErrNameTooLong)
	})

	t.Run("location too long", func(t *testing.T) {
		req := &CreateRoasterRequest{
			Name:     "Roaster",
			Location: strings.Repeat("a", MaxLocationLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrLocationTooLong)
	})

	t.Run("website too long", func(t *testing.T) {
		req := &CreateRoasterRequest{
			Name:    "Roaster",
			Website: strings.Repeat("a", MaxWebsiteLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrWebsiteTooLong)
	})

	t.Run("all fields at max", func(t *testing.T) {
		req := &CreateRoasterRequest{
			Name:     strings.Repeat("a", MaxNameLength),
			Location: strings.Repeat("a", MaxLocationLength),
			Website:  strings.Repeat("a", MaxWebsiteLength),
		}
		assert.NoError(t, req.Validate())
	})
}

func TestUpdateRoasterRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &UpdateRoasterRequest{Name: "Updated Roaster"}
		assert.NoError(t, req.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		req := &UpdateRoasterRequest{Name: ""}
		assert.ErrorIs(t, req.Validate(), ErrNameRequired)
	})

	t.Run("location too long", func(t *testing.T) {
		req := &UpdateRoasterRequest{
			Name:     "Roaster",
			Location: strings.Repeat("a", MaxLocationLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrLocationTooLong)
	})

	t.Run("website too long", func(t *testing.T) {
		req := &UpdateRoasterRequest{
			Name:    "Roaster",
			Website: strings.Repeat("a", MaxWebsiteLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrWebsiteTooLong)
	})
}

func TestCreateGrinderRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &CreateGrinderRequest{Name: "Comandante C40"}
		assert.NoError(t, req.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		req := &CreateGrinderRequest{Name: ""}
		assert.ErrorIs(t, req.Validate(), ErrNameRequired)
	})

	t.Run("name too long", func(t *testing.T) {
		req := &CreateGrinderRequest{Name: strings.Repeat("a", MaxNameLength+1)}
		assert.ErrorIs(t, req.Validate(), ErrNameTooLong)
	})

	t.Run("grinder type too long", func(t *testing.T) {
		req := &CreateGrinderRequest{
			Name:        "Grinder",
			GrinderType: strings.Repeat("a", MaxGrinderTypeLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrFieldTooLong)
	})

	t.Run("burr type too long", func(t *testing.T) {
		req := &CreateGrinderRequest{
			Name:     "Grinder",
			BurrType: strings.Repeat("a", MaxBurrTypeLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrFieldTooLong)
	})

	t.Run("notes too long", func(t *testing.T) {
		req := &CreateGrinderRequest{
			Name:  "Grinder",
			Notes: strings.Repeat("a", MaxNotesLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrNotesTooLong)
	})
}

func TestUpdateGrinderRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &UpdateGrinderRequest{Name: "Updated Grinder"}
		assert.NoError(t, req.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		req := &UpdateGrinderRequest{Name: ""}
		assert.ErrorIs(t, req.Validate(), ErrNameRequired)
	})

	t.Run("grinder type too long", func(t *testing.T) {
		req := &UpdateGrinderRequest{
			Name:        "Grinder",
			GrinderType: strings.Repeat("a", MaxGrinderTypeLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrFieldTooLong)
	})

	t.Run("notes too long", func(t *testing.T) {
		req := &UpdateGrinderRequest{
			Name:  "Grinder",
			Notes: strings.Repeat("a", MaxNotesLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrNotesTooLong)
	})
}

func TestCreateBrewerRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &CreateBrewerRequest{Name: "V60"}
		assert.NoError(t, req.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		req := &CreateBrewerRequest{Name: ""}
		assert.ErrorIs(t, req.Validate(), ErrNameRequired)
	})

	t.Run("name too long", func(t *testing.T) {
		req := &CreateBrewerRequest{Name: strings.Repeat("a", MaxNameLength+1)}
		assert.ErrorIs(t, req.Validate(), ErrNameTooLong)
	})

	t.Run("brewer type too long", func(t *testing.T) {
		req := &CreateBrewerRequest{
			Name:       "Brewer",
			BrewerType: strings.Repeat("a", MaxBrewerTypeLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrFieldTooLong)
	})

	t.Run("description too long", func(t *testing.T) {
		req := &CreateBrewerRequest{
			Name:        "Brewer",
			Description: strings.Repeat("a", MaxDescriptionLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrDescTooLong)
	})
}

func TestUpdateBrewerRequest_Validate(t *testing.T) {
	t.Run("valid request", func(t *testing.T) {
		req := &UpdateBrewerRequest{Name: "Updated V60"}
		assert.NoError(t, req.Validate())
	})

	t.Run("empty name", func(t *testing.T) {
		req := &UpdateBrewerRequest{Name: ""}
		assert.ErrorIs(t, req.Validate(), ErrNameRequired)
	})

	t.Run("brewer type too long", func(t *testing.T) {
		req := &UpdateBrewerRequest{
			Name:       "Brewer",
			BrewerType: strings.Repeat("a", MaxBrewerTypeLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrFieldTooLong)
	})

	t.Run("description too long", func(t *testing.T) {
		req := &UpdateBrewerRequest{
			Name:        "Brewer",
			Description: strings.Repeat("a", MaxDescriptionLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrDescTooLong)
	})
}

func TestCreateBrewRequest_Validate(t *testing.T) {
	t.Run("valid minimal request", func(t *testing.T) {
		req := &CreateBrewRequest{}
		assert.NoError(t, req.Validate())
	})

	t.Run("valid full request", func(t *testing.T) {
		req := &CreateBrewRequest{
			BeanRKey:     "abc123",
			Method:       "Pour Over",
			Temperature:  93.5,
			WaterAmount:  250,
			CoffeeAmount: 15,
			TimeSeconds:  210,
			GrindSize:    "Medium-Fine",
			GrinderRKey:  "grinder1",
			BrewerRKey:   "brewer1",
			TastingNotes: "Fruity, bright acidity",
			Rating:       8,
		}
		assert.NoError(t, req.Validate())
	})

	t.Run("method too long", func(t *testing.T) {
		req := &CreateBrewRequest{
			Method: strings.Repeat("a", MaxMethodLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrFieldTooLong)
	})

	t.Run("grind size too long", func(t *testing.T) {
		req := &CreateBrewRequest{
			GrindSize: strings.Repeat("a", MaxGrindSizeLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrFieldTooLong)
	})

	t.Run("tasting notes too long", func(t *testing.T) {
		req := &CreateBrewRequest{
			TastingNotes: strings.Repeat("a", MaxTastingNotesLength+1),
		}
		assert.ErrorIs(t, req.Validate(), ErrFieldTooLong)
	})
}
