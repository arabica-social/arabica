package main

import (
	"fmt"
	"os"

	"arabica/internal/models"
	"arabica/internal/ogcard"
)

func main() {
	type testCase struct {
		path string
		gen  func() (*ogcard.Card, error)
	}

	cases := []testCase{
		// Brew - full V60 pourover with pours + pourover params
		{"/tmp/og-brew-pourover.png", func() (*ogcard.Card, error) {
			return ogcard.DrawBrewCard(&models.Brew{
				Rating: 8, Temperature: 93, WaterAmount: 250, CoffeeAmount: 15,
				TimeSeconds: 195, GrindSize: "22 clicks",
				Bean: &models.Bean{
					Name: "Ethiopia Yirgacheffe", Origin: "Ethiopia", RoastLevel: "Light",
					Roaster: &models.Roaster{Name: "Sweet Maria's"},
				},
				BrewerObj:  &models.Brewer{Name: "V60", BrewerType: "pourover"},
				GrinderObj: &models.Grinder{Name: "Comandante C40"},
				Pours: []*models.Pour{
					{WaterAmount: 50, TimeSeconds: 0},
					{WaterAmount: 100, TimeSeconds: 45},
					{WaterAmount: 100, TimeSeconds: 75},
				},
				PouroverParams: &models.PouroverParams{
					BloomWater: 45, BloomSeconds: 30, DrawdownSeconds: 90, Filter: "paper",
				},
				TastingNotes: "Bright and fruity with prominent blueberry and dark chocolate notes. Clean finish with a pleasant lingering sweetness.",
			})
		}},

		// Brew - espresso shot
		{"/tmp/og-brew-espresso.png", func() (*ogcard.Card, error) {
			return ogcard.DrawBrewCard(&models.Brew{
				Rating: 9, Temperature: 94, WaterAmount: 36, CoffeeAmount: 18,
				TimeSeconds: 28, GrindSize: "8",
				Bean: &models.Bean{
					Name: "Colombia Huila Supremo", Origin: "Colombia", RoastLevel: "Medium",
					Roaster: &models.Roaster{Name: "Onyx Coffee Lab"},
				},
				BrewerObj:  &models.Brewer{Name: "Linea Mini", BrewerType: "espresso"},
				GrinderObj: &models.Grinder{Name: "Niche Zero"},
				EspressoParams: &models.EspressoParams{
					YieldWeight: 36.5, Pressure: 9, PreInfusionSeconds: 5,
				},
				TastingNotes: "Rich caramel, walnut, plum acidity",
			})
		}},

		// Brew - minimal (just name + rating)
		{"/tmp/og-brew-minimal.png", func() (*ogcard.Card, error) {
			return ogcard.DrawBrewCard(&models.Brew{
				Rating: 6,
				Bean:   &models.Bean{Name: "House Blend"},
			})
		}},

		// Brew - French press immersion
		{"/tmp/og-brew-immersion.png", func() (*ogcard.Card, error) {
			return ogcard.DrawBrewCard(&models.Brew{
				Rating: 7, Temperature: 100, WaterAmount: 350, CoffeeAmount: 22,
				TimeSeconds: 300, GrindSize: "30 clicks",
				Bean: &models.Bean{
					Name: "Kenya AA", Origin: "Kenya",
				},
				BrewerObj:    &models.Brewer{Name: "French Press", BrewerType: "immersion"},
				GrinderObj:   &models.Grinder{Name: "Timemore C2"},
				TastingNotes: "Bold, juicy, black currant",
			})
		}},

		// Brew - no rating, long name, lots of tasting notes
		{"/tmp/og-brew-long.png", func() (*ogcard.Card, error) {
			return ogcard.DrawBrewCard(&models.Brew{
				Temperature: 96, WaterAmount: 400, CoffeeAmount: 25, TimeSeconds: 240,
				Bean: &models.Bean{
					Name: "Finca El Paraiso Double Anaerobic Gesha Lot #147", Origin: "Colombia", RoastLevel: "Light",
					Roaster: &models.Roaster{Name: "Manhattan Coffee Roasters"},
				},
				BrewerObj:    &models.Brewer{Name: "Chemex", BrewerType: "pourover"},
				TastingNotes: "Incredibly complex. Layers of tropical fruit, jasmine, bergamot, and a wine-like body. The anaerobic process adds a distinctive fermented fruit character that evolves as the cup cools. One of the most memorable coffees this year.",
			})
		}},

		// Bean - full
		{"/tmp/og-bean.png", func() (*ogcard.Card, error) {
			rating := 9
			return ogcard.DrawBeanCard(&models.Bean{
				Name: "Gesha Village Lot #74", Origin: "Ethiopia", Variety: "Gesha",
				RoastLevel: "Light", Process: "Washed", Rating: &rating,
				Roaster:     &models.Roaster{Name: "Onyx Coffee Lab"},
				Description: "Exceptional lot from the birthplace of Gesha. Floral jasmine and bergamot with layers of tropical fruit. Delicate tea-like body.",
			})
		}},

		// Bean - minimal
		{"/tmp/og-bean-minimal.png", func() (*ogcard.Card, error) {
			return ogcard.DrawBeanCard(&models.Bean{
				Name: "Colombia Supremo", Origin: "Colombia",
			})
		}},

		// Roaster
		{"/tmp/og-roaster.png", func() (*ogcard.Card, error) {
			return ogcard.DrawRoasterCard(&models.Roaster{
				Name: "Onyx Coffee Lab", Location: "Rogers, Arkansas",
				Website: "https://onyxcoffeelab.com",
			})
		}},

		// Grinder
		{"/tmp/og-grinder.png", func() (*ogcard.Card, error) {
			return ogcard.DrawGrinderCard(&models.Grinder{
				Name: "Comandante C40 MK4", GrinderType: "Hand Grinder", BurrType: "Steel",
				Notes: "Outstanding grind consistency across all settings. The MK4 burrs are a significant improvement with faster grinding and better particle distribution.",
			})
		}},

		// Brewer
		{"/tmp/og-brewer.png", func() (*ogcard.Card, error) {
			return ogcard.DrawBrewerCard(&models.Brewer{
				Name: "Hario V60 02", BrewerType: "pourover",
				Description: "Classic cone-shaped dripper with spiral ribs. Produces a clean, bright cup that highlights origin characteristics.",
			})
		}},

		// Recipe - full with pours
		{"/tmp/og-recipe.png", func() (*ogcard.Card, error) {
			return ogcard.DrawRecipeCard(&models.Recipe{
				Name: "James Hoffmann V60 Method", CoffeeAmount: 15, WaterAmount: 250,
				BrewerType: "pourover", Ratio: 16.7,
				BrewerObj: &models.Brewer{Name: "Hario V60 02"},
				Pours: []*models.Pour{
					{WaterAmount: 50, TimeSeconds: 0},
					{WaterAmount: 100, TimeSeconds: 45},
					{WaterAmount: 100, TimeSeconds: 75},
				},
				Notes: "Start with a bloom, then two even pours. Gentle swirl after each pour for even extraction.",
			})
		}},

		// Recipe - minimal
		{"/tmp/og-recipe-minimal.png", func() (*ogcard.Card, error) {
			return ogcard.DrawRecipeCard(&models.Recipe{
				Name: "Quick Aeropress", CoffeeAmount: 15, WaterAmount: 200,
				BrewerType: "immersion",
			})
		}},

		// Bean - Greek text (non-ASCII unicode)
		{"/tmp/og-bean-greek.png", func() (*ogcard.Card, error) {
			rating := 7
			return ogcard.DrawBeanCard(&models.Bean{
				Name: "Λουμίδης Παπαγάλος Παραδοσιακός Ελληνικός Καφές 100 gr", Origin: "Unspecified Latin American",
				RoastLevel: "Medium", Process: "Unknown", Rating: &rating, Variety: "Unspecified Arabica",
				Roaster:     &models.Roaster{Name: "Λουμίδης Παπαγάλος"},
				Description: "A tasty blend with a balanced roast and earthy notes.",
			})
		}},

		// Site card
		{"/tmp/og-site.png", func() (*ogcard.Card, error) {
			return ogcard.DrawSiteCard()
		}},
	}

	for _, tc := range cases {
		card, err := tc.gen()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error generating %s: %v\n", tc.path, err)
			os.Exit(1)
		}
		f, err := os.Create(tc.path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating %s: %v\n", tc.path, err)
			os.Exit(1)
		}
		if err := card.EncodePNG(f); err != nil {
			fmt.Fprintf(os.Stderr, "Error encoding %s: %v\n", tc.path, err)
			os.Exit(1)
		}
		f.Close()
		fmt.Printf("Generated %s\n", tc.path)
	}
}
