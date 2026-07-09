package coffeehandlers

import (
	"net/http"
	"sort"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	"tangled.org/arabica.social/arabica/internal/arabica/onboarding"
	coffee "tangled.org/arabica.social/arabica/internal/arabica/web/components"
	"tangled.org/arabica.social/arabica/internal/handlers"

	"github.com/rs/zerolog/log"
)

// OnboardingResponseJSON is the JSON envelope returned by GET /api/onboarding.
type OnboardingResponseJSON struct {
	Readiness onboarding.ReadinessStatus `json:"readiness"`
	Beans     []*arabica.Bean            `json:"beans"`
	Brewers   []*arabica.Brewer          `json:"brewers"`
	Grinders  []*arabica.Grinder         `json:"grinders"`
	Roasters  []*arabica.Roaster         `json:"roasters"`
}

// IncompleteRecordsResponseJSON is the JSON envelope returned by GET /api/incomplete-records.
type IncompleteRecordsResponseJSON struct {
	Records []coffee.IncompleteRecord `json:"records"`
}

// HandleOnboardingJSON returns the user's onboarding readiness + entity lists
// as JSON for the SvelteKit SPA.
func (h *Handlers) HandleOnboardingJSON(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	status, err := onboarding.CheckBrewReadiness(ctx, store)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build onboarding JSON")
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}

	beans, err := store.ListBeans(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list beans for onboarding JSON")
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}
	brewers, err := listBrewers(ctx, store)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list brewers for onboarding JSON")
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}
	grinders, err := listGrinders(ctx, store)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list grinders for onboarding JSON")
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}
	roasters, err := store.ListRoasters(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to list roasters for onboarding JSON")
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}

	handlers.WriteJSON(w, OnboardingResponseJSON{
		Readiness: status,
		Beans:     beans,
		Brewers:   brewers,
		Grinders:  grinders,
		Roasters:  roasters,
	}, "onboarding")
}

// HandleIncompleteRecordsJSON returns the user's incomplete records as JSON.
func (h *Handlers) HandleIncompleteRecordsJSON(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	ctx := r.Context()
	beans, err := store.ListBeans(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch beans for incomplete records JSON")
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	grinders, err := listGrinders(ctx, store)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch grinders for incomplete records JSON")
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}
	brewers, err := listBrewers(ctx, store)
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch brewers for incomplete records JSON")
		http.Error(w, "Failed to fetch data", http.StatusInternalServerError)
		return
	}

	records := coffee.CollectIncompleteRecords(beans, grinders, brewers, 5)

	handlers.WriteJSON(w, IncompleteRecordsResponseJSON{
		Records: records,
	}, "incomplete-records")
}

// HandlePopularRecipesJSON returns popular recipes as JSON for the SvelteKit SPA.
// Sorted by popularity (brew_count + fork_count, descending), limited to 3.
func (h *Handlers) HandlePopularRecipesJSON(w http.ResponseWriter, r *http.Request) {
	_, authenticated := h.GetArabicaStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	recipes, err := h.listAllRecipesFromIndex(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch recipes for popular JSON")
		http.Error(w, "Failed to fetch recipes", http.StatusInternalServerError)
		return
	}

	sort.Slice(recipes, func(i, j int) bool {
		si := recipes[i].BrewCount + recipes[i].ForkCount
		sj := recipes[j].BrewCount + recipes[j].ForkCount
		return si > sj
	})

	const maxRecipes = 3
	if len(recipes) > maxRecipes {
		recipes = recipes[:maxRecipes]
	}

	handlers.WriteJSON(w, recipes, "popular-recipes")
}
