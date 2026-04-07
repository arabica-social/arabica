package atproto

import (
	"testing"
	"time"

	"arabica/internal/models"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrewToRecord(t *testing.T) {
	createdAt := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)

	t.Run("full brew with all fields", func(t *testing.T) {
		brew := &models.Brew{
			Method:       "V60",
			Temperature:  93.5,
			WaterAmount:  300,
			TimeSeconds:  180,
			GrindSize:    "Medium",
			TastingNotes: "Fruity and bright",
			Rating:       8,
			CreatedAt:    createdAt,
			Pours: []*models.Pour{
				{WaterAmount: 50, TimeSeconds: 30},
				{WaterAmount: 100, TimeSeconds: 60},
			},
		}

		beanURI := "at://did:plc:test/social.arabica.alpha.bean/bean123"
		grinderURI := "at://did:plc:test/social.arabica.alpha.grinder/grinder123"
		brewerURI := "at://did:plc:test/social.arabica.alpha.brewer/brewer123"

		record, err := BrewToRecord(brew, beanURI, grinderURI, brewerURI, "")
		require.NoError(t, err)
		shutter.Snap(t, "BrewToRecord/full brew", record)
	})

	t.Run("minimal brew", func(t *testing.T) {
		brew := &models.Brew{
			CreatedAt: createdAt,
		}

		record, err := BrewToRecord(brew, "at://did:plc:test/social.arabica.alpha.bean/bean123", "", "", "")
		require.NoError(t, err)
		shutter.Snap(t, "BrewToRecord/minimal brew", record)
	})

	t.Run("error without beanURI", func(t *testing.T) {
		brew := &models.Brew{CreatedAt: createdAt}
		_, err := BrewToRecord(brew, "", "", "", "")
		assert.Error(t, err)
	})
}

func TestRecordToBrew(t *testing.T) {
	t.Run("full record", func(t *testing.T) {
		record := map[string]any{
			"$type":        NSIDBrew,
			"beanRef":      "at://did:plc:test/social.arabica.alpha.bean/bean123",
			"createdAt":    "2025-01-10T12:00:00Z",
			"method":       "V60",
			"temperature":  float64(935),
			"waterAmount":  float64(300),
			"timeSeconds":  float64(180),
			"grindSize":    "Medium",
			"grinderRef":   "at://did:plc:test/social.arabica.alpha.grinder/grinder123",
			"brewerRef":    "at://did:plc:test/social.arabica.alpha.brewer/brewer123",
			"tastingNotes": "Fruity",
			"rating":       float64(8),
			"pours": []any{
				map[string]any{"waterAmount": float64(50), "timeSeconds": float64(30)},
				map[string]any{"waterAmount": float64(100), "timeSeconds": float64(60)},
			},
		}

		brew, err := RecordToBrew(record, "at://did:plc:test/social.arabica.alpha.brew/brew123")
		require.NoError(t, err)
		shutter.Snap(t, "RecordToBrew/full record", brew)
	})

	t.Run("error without beanRef", func(t *testing.T) {
		record := map[string]any{
			"$type":     NSIDBrew,
			"createdAt": "2025-01-10T12:00:00Z",
		}
		_, err := RecordToBrew(record, "at://did:plc:test/social.arabica.alpha.brew/brew123")
		assert.Error(t, err)
	})

	t.Run("error without createdAt", func(t *testing.T) {
		record := map[string]any{
			"$type":   NSIDBrew,
			"beanRef": "at://did:plc:test/social.arabica.alpha.bean/bean123",
		}
		_, err := RecordToBrew(record, "at://did:plc:test/social.arabica.alpha.brew/brew123")
		assert.Error(t, err)
	})

	t.Run("error with invalid AT-URI", func(t *testing.T) {
		record := map[string]any{
			"$type":     NSIDBrew,
			"beanRef":   "at://did:plc:test/social.arabica.alpha.bean/bean123",
			"createdAt": "2025-01-10T12:00:00Z",
		}
		_, err := RecordToBrew(record, "invalid-uri")
		assert.Error(t, err)
	})
}

func TestBeanToRecord(t *testing.T) {
	createdAt := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)

	t.Run("full bean", func(t *testing.T) {
		bean := &models.Bean{
			Name:        "Ethiopian Yirgacheffe",
			Origin:      "Ethiopia",
			RoastLevel:  "Light",
			Process:     "Washed",
			Description: "Fruity and floral notes",
			CreatedAt:   createdAt,
		}
		record, err := BeanToRecord(bean, "at://did:plc:test/social.arabica.alpha.roaster/roaster123")
		require.NoError(t, err)
		shutter.Snap(t, "BeanToRecord/full bean", record)
	})

	t.Run("bean without roaster", func(t *testing.T) {
		bean := &models.Bean{
			Name:      "Generic Coffee",
			CreatedAt: createdAt,
		}
		record, err := BeanToRecord(bean, "")
		require.NoError(t, err)
		shutter.Snap(t, "BeanToRecord/no roaster", record)
	})
}

func TestRecordToBean(t *testing.T) {
	t.Run("full record", func(t *testing.T) {
		record := map[string]any{
			"$type":       NSIDBean,
			"name":        "Ethiopian Yirgacheffe",
			"origin":      "Ethiopia",
			"roastLevel":  "Light",
			"process":     "Washed",
			"description": "Fruity notes",
			"createdAt":   "2025-01-10T12:00:00Z",
		}
		bean, err := RecordToBean(record, "at://did:plc:test/social.arabica.alpha.bean/bean123")
		require.NoError(t, err)
		shutter.Snap(t, "RecordToBean/full record", bean)
	})

	t.Run("error without name", func(t *testing.T) {
		record := map[string]any{
			"$type":     NSIDBean,
			"createdAt": "2025-01-10T12:00:00Z",
		}
		_, err := RecordToBean(record, "at://did:plc:test/social.arabica.alpha.bean/bean123")
		assert.Error(t, err)
	})
}

func TestRoasterToRecord(t *testing.T) {
	createdAt := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)

	t.Run("full roaster", func(t *testing.T) {
		roaster := &models.Roaster{
			Name:      "Counter Culture",
			Location:  "Durham, NC",
			Website:   "https://counterculturecoffee.com",
			CreatedAt: createdAt,
		}
		record, err := RoasterToRecord(roaster)
		require.NoError(t, err)
		shutter.Snap(t, "RoasterToRecord/full roaster", record)
	})
}

func TestRecordToRoaster(t *testing.T) {
	t.Run("full record", func(t *testing.T) {
		record := map[string]any{
			"$type":     NSIDRoaster,
			"name":      "Counter Culture",
			"location":  "Durham, NC",
			"website":   "https://counterculturecoffee.com",
			"createdAt": "2025-01-10T12:00:00Z",
		}
		roaster, err := RecordToRoaster(record, "at://did:plc:test/social.arabica.alpha.roaster/roaster123")
		require.NoError(t, err)
		shutter.Snap(t, "RecordToRoaster/full record", roaster)
	})

	t.Run("error without name", func(t *testing.T) {
		record := map[string]any{
			"$type":     NSIDRoaster,
			"createdAt": "2025-01-10T12:00:00Z",
		}
		_, err := RecordToRoaster(record, "at://did:plc:test/social.arabica.alpha.roaster/roaster123")
		assert.Error(t, err)
	})
}

func TestGrinderToRecord(t *testing.T) {
	createdAt := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)

	t.Run("full grinder", func(t *testing.T) {
		grinder := &models.Grinder{
			Name:        "Comandante C40",
			GrinderType: "Hand",
			BurrType:    "Conical",
			Notes:       "Great for travel",
			CreatedAt:   createdAt,
		}
		record, err := GrinderToRecord(grinder)
		require.NoError(t, err)
		shutter.Snap(t, "GrinderToRecord/full grinder", record)
	})
}

func TestRecordToGrinder(t *testing.T) {
	t.Run("full record", func(t *testing.T) {
		record := map[string]any{
			"$type":       NSIDGrinder,
			"name":        "Comandante C40",
			"grinderType": "Hand",
			"burrType":    "Conical",
			"notes":       "Great for travel",
			"createdAt":   "2025-01-10T12:00:00Z",
		}
		grinder, err := RecordToGrinder(record, "at://did:plc:test/social.arabica.alpha.grinder/grinder123")
		require.NoError(t, err)
		shutter.Snap(t, "RecordToGrinder/full record", grinder)
	})
}

func TestBrewerToRecord(t *testing.T) {
	createdAt := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)

	t.Run("full brewer", func(t *testing.T) {
		brewer := &models.Brewer{
			Name:        "Hario V60",
			Description: "Pour-over dripper",
			CreatedAt:   createdAt,
		}
		record, err := BrewerToRecord(brewer)
		require.NoError(t, err)
		shutter.Snap(t, "BrewerToRecord/full brewer", record)
	})
}

func TestRecordToBrewer(t *testing.T) {
	t.Run("full record", func(t *testing.T) {
		record := map[string]any{
			"$type":       NSIDBrewer,
			"name":        "Hario V60",
			"description": "Pour-over dripper",
			"createdAt":   "2025-01-10T12:00:00Z",
		}
		brewer, err := RecordToBrewer(record, "at://did:plc:test/social.arabica.alpha.brewer/brewer123")
		require.NoError(t, err)
		shutter.Snap(t, "RecordToBrewer/full record", brewer)
	})
}

func TestRoundTrip(t *testing.T) {
	createdAt := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)

	t.Run("bean round trip", func(t *testing.T) {
		original := &models.Bean{
			Name:        "Ethiopian Yirgacheffe",
			Origin:      "Ethiopia",
			RoastLevel:  "Light",
			Process:     "Washed",
			Description: "Fruity notes",
			CreatedAt:   createdAt,
		}
		record, err := BeanToRecord(original, "")
		require.NoError(t, err)
		restored, err := RecordToBean(record, "at://did:plc:test/social.arabica.alpha.bean/bean123")
		require.NoError(t, err)
		shutter.Snap(t, "RoundTrip/bean", restored)
	})

	t.Run("roaster round trip", func(t *testing.T) {
		original := &models.Roaster{
			Name:      "Counter Culture",
			Location:  "Durham, NC",
			Website:   "https://counterculturecoffee.com",
			CreatedAt: createdAt,
		}
		record, err := RoasterToRecord(original)
		require.NoError(t, err)
		restored, err := RecordToRoaster(record, "at://did:plc:test/social.arabica.alpha.roaster/roaster123")
		require.NoError(t, err)
		shutter.Snap(t, "RoundTrip/roaster", restored)
	})

	t.Run("grinder round trip", func(t *testing.T) {
		original := &models.Grinder{
			Name:        "Comandante C40",
			GrinderType: "Hand",
			BurrType:    "Conical",
			Notes:       "Great for travel",
			CreatedAt:   createdAt,
		}
		record, err := GrinderToRecord(original)
		require.NoError(t, err)
		restored, err := RecordToGrinder(record, "at://did:plc:test/social.arabica.alpha.grinder/grinder123")
		require.NoError(t, err)
		shutter.Snap(t, "RoundTrip/grinder", restored)
	})

	t.Run("brewer round trip", func(t *testing.T) {
		original := &models.Brewer{
			Name:        "Hario V60",
			BrewerType:  "Pour-Over",
			Description: "Pour-over dripper",
			CreatedAt:   createdAt,
		}
		record, err := BrewerToRecord(original)
		require.NoError(t, err)
		restored, err := RecordToBrewer(record, "at://did:plc:test/social.arabica.alpha.brewer/brewer123")
		require.NoError(t, err)
		shutter.Snap(t, "RoundTrip/brewer", restored)
	})
}

func TestTemperatureConversion(t *testing.T) {
	tests := []struct {
		name        string
		tempFloat   float64
		tempEncoded int
	}{
		{"zero", 0, 0},
		{"room temp", 20.0, 200},
		{"hot coffee", 93.5, 935},
		{"boiling", 100.0, 1000},
		{"fahrenheit range", 200.0, 2000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			brew := &models.Brew{
				Temperature: tt.tempFloat,
				CreatedAt:   time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC),
			}

			record, err := BrewToRecord(brew, "at://did:plc:test/social.arabica.alpha.bean/bean123", "", "", "")
			require.NoError(t, err)

			if tt.tempFloat > 0 {
				encoded, ok := record["temperature"].(int)
				require.True(t, ok, "temperature should be int in record")
				assert.Equal(t, tt.tempEncoded, encoded)

				record["temperature"] = float64(tt.tempEncoded)
				restored, err := RecordToBrew(record, "at://did:plc:test/social.arabica.alpha.brew/brew123")
				require.NoError(t, err)
				assert.InDelta(t, tt.tempFloat, restored.Temperature, 0.001)
			}
		})
	}
}

func TestBrewRoundTrip_EspressoParams(t *testing.T) {
	original := &models.Brew{
		BeanRKey:    "abc123",
		Temperature: 93.5,
		Rating:      8,
		CreatedAt:   time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC),
		EspressoParams: &models.EspressoParams{
			YieldWeight:        36.0,
			Pressure:           9.0,
			PreInfusionSeconds: 5,
		},
	}

	record, err := BrewToRecord(original, "at://did:plc:test/social.arabica.alpha.bean/abc123", "", "", "")
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/espresso params", record)

	restored, err := RecordToBrew(record, "at://did:plc:test/social.arabica.alpha.brew/tid123")
	require.NoError(t, err)
	shutter.Snap(t, "RecordToBrew/espresso params", restored)
}

func TestBrewRoundTrip_PouroverParams(t *testing.T) {
	original := &models.Brew{
		BeanRKey:  "abc123",
		CreatedAt: time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC),
		PouroverParams: &models.PouroverParams{
			BloomWater:      50,
			BloomSeconds:    45,
			DrawdownSeconds: 30,
			BypassWater:     100,
			Filter:          "paper",
		},
	}

	record, err := BrewToRecord(original, "at://did:plc:test/social.arabica.alpha.bean/abc123", "", "", "")
	require.NoError(t, err)
	shutter.Snap(t, "BrewToRecord/pourover params", record)

	restored, err := RecordToBrew(record, "at://did:plc:test/social.arabica.alpha.brew/tid123")
	require.NoError(t, err)
	shutter.Snap(t, "RecordToBrew/pourover params", restored)
}
