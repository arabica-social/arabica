// Package lexicon_test validates that records produced by the production
// XToRecord converters conform to the lexicon JSON schemas in lexicons/.
//
// These tests catch drift between Go models, converter functions, and
// the published lexicons. Whenever a lexicon is added or changed, add or
// update the corresponding sample(s) below.
package lexicon_test

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/bluesky-social/indigo/atproto/atdata"
	"github.com/bluesky-social/indigo/atproto/lexicon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"tangled.org/arabica.social/arabica/internal/entities/arabica"
	"tangled.org/arabica.social/arabica/internal/entities/oolong"
)

const sampleCID = "bafyreigh2akiscaildcqabsyg3dfr6chu3fgpregiymsck7e7aqa4s52zy"

// loadCatalog reads every lexicon under lexicons/ into a BaseCatalog.
// LoadDirectory walks recursively, so the namespaced subdirs
// (lexicons/social/arabica/alpha/...) are picked up automatically.
func loadCatalog(t *testing.T) *lexicon.BaseCatalog {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	lexiconsDir := filepath.Join(filepath.Dir(thisFile), "..", "..", "lexicons")

	cat := lexicon.NewBaseCatalog()
	require.NoError(t, cat.LoadDirectory(lexiconsDir), "failed to load lexicons from %s", lexiconsDir)
	return cat
}

// sample is one (name, NSID, record) triple to validate.
// Each entity should have at minimum a "minimal" (only required fields)
// and ideally a "full" (every field populated) sample.
type sample struct {
	name   string
	nsid   string
	record map[string]any
}

// wireFormat round-trips a record through JSON via atdata.UnmarshalJSON to
// produce the exact map[string]any shape the validator expects (int64 for
// integers, etc.) — mirrors what a PDS sees on the wire.
func wireFormat(t *testing.T, record map[string]any) map[string]any {
	t.Helper()
	b, err := json.Marshal(record)
	require.NoError(t, err)
	parsed, err := atdata.UnmarshalJSON(b)
	require.NoError(t, err)
	return parsed
}

func runSamples(t *testing.T, cat *lexicon.BaseCatalog, samples []sample) {
	t.Helper()
	for _, s := range samples {
		t.Run(s.name, func(t *testing.T) {
			parsed := wireFormat(t, s.record)
			err := lexicon.ValidateRecord(cat, parsed, s.nsid, lexicon.LenientMode)
			assert.NoError(t, err, "record failed lexicon validation against %s\nrecord: %#v", s.nsid, parsed)
		})
	}
}

func TestArabicaLexicons(t *testing.T) {
	cat := loadCatalog(t)
	createdAt := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)

	beanURI := "at://did:plc:test/social.arabica.alpha.bean/bean123"
	roasterURI := "at://did:plc:test/social.arabica.alpha.roaster/roaster123"
	grinderURI := "at://did:plc:test/social.arabica.alpha.grinder/grinder123"
	brewerURI := "at://did:plc:test/social.arabica.alpha.brewer/brewer123"
	recipeURI := "at://did:plc:test/social.arabica.alpha.recipe/recipe123"
	commentURI := "at://did:plc:test/social.arabica.alpha.comment/comment123"

	var samples []sample

	// Roaster
	{
		minimal, err := arabica.RoasterToRecord(&arabica.Roaster{
			Name:      "Black & White",
			CreatedAt: createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"roaster/minimal", arabica.NSIDRoaster, minimal})

		full, err := arabica.RoasterToRecord(&arabica.Roaster{
			Name:      "Black & White",
			Location:  "Raleigh, NC",
			Website:   "https://blackwhiteroasters.com",
			SourceRef: roasterURI,
			CreatedAt: createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"roaster/full", arabica.NSIDRoaster, full})
	}

	// Bean
	{
		minimal, err := arabica.BeanToRecord(&arabica.Bean{
			Name:      "Ethiopia Yirgacheffe",
			CreatedAt: createdAt,
		}, roasterURI)
		require.NoError(t, err)
		samples = append(samples, sample{"bean/minimal", arabica.NSIDBean, minimal})

		rating := 8
		full, err := arabica.BeanToRecord(&arabica.Bean{
			Name:        "Ethiopia Yirgacheffe",
			Origin:      "Ethiopia",
			Variety:     "Heirloom",
			RoastLevel:  "Light",
			Process:     "Washed",
			Description: "Floral and citrus",
			Rating:      &rating,
			Closed:      false,
			SourceRef:   beanURI,
			CreatedAt:   createdAt,
		}, roasterURI)
		require.NoError(t, err)
		samples = append(samples, sample{"bean/full", arabica.NSIDBean, full})
	}

	// Grinder
	{
		minimal, err := arabica.GrinderToRecord(&arabica.Grinder{
			Name:      "1Zpresso K-Ultra",
			CreatedAt: createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"grinder/minimal", arabica.NSIDGrinder, minimal})

		full, err := arabica.GrinderToRecord(&arabica.Grinder{
			Name:        "1Zpresso K-Ultra",
			GrinderType: "Hand",
			BurrType:    "Conical",
			Notes:       "Best for pourover",
			SourceRef:   grinderURI,
			CreatedAt:   createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"grinder/full", arabica.NSIDGrinder, full})
	}

	// Brewer
	{
		minimal, err := arabica.BrewerToRecord(&arabica.Brewer{
			Name:      "Hario V60",
			CreatedAt: createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"brewer/minimal", arabica.NSIDBrewer, minimal})

		full, err := arabica.BrewerToRecord(&arabica.Brewer{
			Name:        "Hario V60",
			BrewerType:  "Pour Over",
			Description: "Glass dripper",
			SourceRef:   brewerURI,
			CreatedAt:   createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"brewer/full", arabica.NSIDBrewer, full})
	}

	// Recipe
	{
		minimal, err := arabica.RecipeToRecord(&arabica.Recipe{
			Name:      "James Hoffmann V60",
			CreatedAt: createdAt,
		}, brewerURI)
		require.NoError(t, err)
		samples = append(samples, sample{"recipe/minimal", arabica.NSIDRecipe, minimal})
	}

	// Brew
	{
		minimal, err := arabica.BrewToRecord(&arabica.Brew{
			CreatedAt: createdAt,
		}, beanURI, "", "", "")
		require.NoError(t, err)
		samples = append(samples, sample{"brew/minimal", arabica.NSIDBrew, minimal})

		full, err := arabica.BrewToRecord(&arabica.Brew{
			Method:       "V60",
			Temperature:  93.5,
			WaterAmount:  300,
			CoffeeAmount: 18,
			TimeSeconds:  180,
			GrindSize:    "Medium",
			TastingNotes: "Fruity and bright",
			Rating:       8,
			CreatedAt:    createdAt,
			Pours: []*arabica.Pour{
				{WaterAmount: 50, TimeSeconds: 30},
				{WaterAmount: 100, TimeSeconds: 60},
			},
		}, beanURI, grinderURI, brewerURI, recipeURI)
		require.NoError(t, err)
		samples = append(samples, sample{"brew/full", arabica.NSIDBrew, full})
	}

	// Like
	{
		like, err := arabica.LikeToRecord(&arabica.Like{
			SubjectURI: beanURI,
			SubjectCID: sampleCID,
			CreatedAt:  createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"like/full", arabica.NSIDLike, like})
	}

	// Comment
	{
		minimal, err := arabica.CommentToRecord(&arabica.Comment{
			SubjectURI: beanURI,
			SubjectCID: sampleCID,
			Text:       "Great bean!",
			CreatedAt:  createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"comment/minimal", arabica.NSIDComment, minimal})

		full, err := arabica.CommentToRecord(&arabica.Comment{
			SubjectURI: beanURI,
			SubjectCID: sampleCID,
			ParentURI:  commentURI,
			ParentCID:  sampleCID,
			Text:       "Reply to a comment",
			CreatedAt:  createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"comment/full", arabica.NSIDComment, full})
	}

	runSamples(t, cat, samples)
}

func TestOolongLexicons(t *testing.T) {
	cat := loadCatalog(t)
	createdAt := time.Date(2025, 1, 10, 12, 0, 0, 0, time.UTC)

	teaURI := "at://did:plc:test/social.oolong.alpha.tea/tea123"
	vendorURI := "at://did:plc:test/social.oolong.alpha.vendor/vendor123"
	vesselURI := "at://did:plc:test/social.oolong.alpha.vessel/vessel123"
	infuserURI := "at://did:plc:test/social.oolong.alpha.infuser/infuser123"

	var samples []sample

	// Vendor
	{
		minimal, err := oolong.VendorToRecord(&oolong.Vendor{
			Name:      "White2Tea",
			CreatedAt: createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"vendor/minimal", oolong.NSIDVendor, minimal})
	}

	// Tea
	{
		minimal, err := oolong.TeaToRecord(&oolong.Tea{
			Name:      "Da Hong Pao",
			CreatedAt: createdAt,
		}, vendorURI)
		require.NoError(t, err)
		samples = append(samples, sample{"tea/minimal", oolong.NSIDTea, minimal})
	}

	// Vessel
	{
		minimal, err := oolong.VesselToRecord(&oolong.Vessel{
			Name:      "Glass teapot",
			CreatedAt: createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"oolong-vessel/minimal", oolong.NSIDVessel, minimal})

		full, err := oolong.VesselToRecord(&oolong.Vessel{
			Name:        "Glass teapot",
			Style:       "teapot",
			CapacityMl:  500,
			Material:    "glass",
			Description: "Standard 500ml glass teapot",
			SourceRef:   vesselURI,
			CreatedAt:   createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"oolong-vessel/full", oolong.NSIDVessel, full})
	}

	// Infuser
	{
		minimal, err := oolong.InfuserToRecord(&oolong.Infuser{
			Name:      "Stainless basket",
			CreatedAt: createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"oolong-infuser/minimal", oolong.NSIDInfuser, minimal})

		full, err := oolong.InfuserToRecord(&oolong.Infuser{
			Name:        "Stainless basket",
			Style:       "basket",
			Material:    "stainless steel",
			Description: "Fine mesh basket",
			CreatedAt:   createdAt,
		})
		require.NoError(t, err)
		samples = append(samples, sample{"oolong-infuser/full", oolong.NSIDInfuser, full})
	}

	// Brew
	{
		minimal, err := oolong.BrewToRecord(&oolong.Brew{
			Style:     oolong.StyleLongSteep,
			CreatedAt: createdAt,
		}, teaURI, "", "")
		require.NoError(t, err)
		samples = append(samples, sample{"oolong-brew/minimal", oolong.NSIDBrew, minimal})

		full, err := oolong.BrewToRecord(&oolong.Brew{
			Style:          oolong.StyleColdBrew,
			InfusionMethod: oolong.InfusionMethodInfuser,
			Temperature:    4,
			LeafGrams:      10,
			WaterAmount:    1000,
			TimeSeconds:    43200,
			TastingNotes:   "Smooth, low astringency",
			Rating:         8,
			CreatedAt:      createdAt,
		}, teaURI, vesselURI, infuserURI)
		require.NoError(t, err)
		samples = append(samples, sample{"oolong-brew/full", oolong.NSIDBrew, full})
	}

	runSamples(t, cat, samples)
}
