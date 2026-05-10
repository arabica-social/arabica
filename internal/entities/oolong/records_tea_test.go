package oolong

import (
	"testing"
	"time"

	"github.com/ptdewey/shutter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeaToRecord(t *testing.T) {
	createdAt := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)

	t.Run("full tea with processing", func(t *testing.T) {
		rating := 9
		tea := &Tea{
			Name:        "Long Jing 2024 Spring",
			Category:    CategoryGreen,
			SubStyle:    "Xi Hu Long Jing",
			Origin:      "Hangzhou, Zhejiang",
			Cultivar:    "Qun Ti",
			Farm:        "Mei Jia Wu village",
			HarvestYear: 2024,
			Processing: []ProcessingStep{
				{Step: ProcessingPanFired, Detail: "Hand-fired in iron wok"},
			},
			Description: "Pre-Qingming pluck",
			Rating:      &rating,
			CreatedAt:   createdAt,
		}
		vendorURI := "at://did:plc:test/social.oolong.alpha.vendor/v1"
		rec, err := TeaToRecord(tea, vendorURI)
		require.NoError(t, err)
		shutter.Snap(t, "TeaToRecord/full tea with processing", rec)
	})

	t.Run("minimal tea", func(t *testing.T) {
		tea := &Tea{Name: "Generic green", CreatedAt: createdAt}
		rec, err := TeaToRecord(tea, "")
		require.NoError(t, err)
		shutter.Snap(t, "TeaToRecord/minimal tea", rec)
	})

	t.Run("tea with multiple processing steps", func(t *testing.T) {
		tea := &Tea{
			Name:     "Gyokuro",
			Category: CategoryGreen,
			SubStyle: "Gyokuro",
			Origin:   "Yame",
			Processing: []ProcessingStep{
				{Step: ProcessingShaded, Detail: "20 days under kanreisha"},
				{Step: ProcessingSteamed},
				{Step: ProcessingRolled},
			},
			CreatedAt: createdAt,
		}
		rec, err := TeaToRecord(tea, "")
		require.NoError(t, err)
		shutter.Snap(t, "TeaToRecord/multiple processing steps", rec)
	})
}

func TestRecordToTea(t *testing.T) {
	t.Run("full record with processing", func(t *testing.T) {
		rec := map[string]any{
			"$type":       NSIDTea,
			"name":        "Long Jing 2024 Spring",
			"category":    "green",
			"subStyle":    "Xi Hu Long Jing",
			"origin":      "Hangzhou, Zhejiang",
			"cultivar":    "Qun Ti",
			"farm":        "Mei Jia Wu",
			"harvestYear": float64(2024),
			"processing": []any{
				map[string]any{"step": "pan-fired", "detail": "Hand-fired"},
				map[string]any{"step": "rolled"},
			},
			"description": "Pre-Qingming",
			"vendorRef":   "at://did:plc:test/social.oolong.alpha.vendor/v1",
			"rating":      float64(9),
			"closed":      false,
			"createdAt":   "2026-05-10T12:00:00Z",
		}
		tea, err := RecordToTea(rec, "at://did:plc:test/social.oolong.alpha.tea/tea1")
		require.NoError(t, err)
		assert.Equal(t, "tea1", tea.RKey)
		assert.Equal(t, CategoryGreen, tea.Category)
		assert.Equal(t, 2024, tea.HarvestYear)
		assert.Len(t, tea.Processing, 2)
		assert.Equal(t, ProcessingPanFired, tea.Processing[0].Step)
		assert.Equal(t, "Hand-fired", tea.Processing[0].Detail)
		assert.Equal(t, ProcessingRolled, tea.Processing[1].Step)
		assert.Empty(t, tea.Processing[1].Detail)
		require.NotNil(t, tea.Rating)
		assert.Equal(t, 9, *tea.Rating)
		assert.NotEmpty(t, tea.VendorRKey)
	})

	t.Run("missing name returns error", func(t *testing.T) {
		_, err := RecordToTea(map[string]any{"createdAt": "2026-05-10T12:00:00Z"}, "")
		assert.ErrorIs(t, err, ErrNameRequired)
	})
}

func TestTeaRoundTrip(t *testing.T) {
	rating := 8
	original := &Tea{
		Name:        "Da Hong Pao",
		Category:    CategoryOolong,
		SubStyle:    "Wuyi rock tea",
		Origin:      "Wuyi Mountains",
		Cultivar:    "Da Hong Pao",
		HarvestYear: 2024,
		Processing: []ProcessingStep{
			{Step: ProcessingOxidized, Detail: "60-70%"},
			{Step: ProcessingRoasted, Detail: "Charcoal, medium"},
		},
		Rating:    &rating,
		Closed:    false,
		CreatedAt: time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC),
	}
	rec, err := TeaToRecord(original, "")
	require.NoError(t, err)
	round, err := RecordToTea(rec, "at://did:plc:test/social.oolong.alpha.tea/abc")
	require.NoError(t, err)
	round.RKey = ""
	assert.Equal(t, original.Name, round.Name)
	assert.Equal(t, original.Processing, round.Processing)
	assert.Equal(t, *original.Rating, *round.Rating)
}
