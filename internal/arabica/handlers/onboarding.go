package coffeehandlers

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"

	arabica "tangled.org/arabica.social/arabica/internal/arabica/entities"
	coffee "tangled.org/arabica.social/arabica/internal/arabica/web/components"
	"tangled.org/arabica.social/arabica/internal/onboarding"
)

// getStartedCardStore is a narrow interface for buildGetStartedCardProps.
// This allows tests to pass *database.MockStore without implementing the
// full database.Store interface.
type getStartedCardStore interface {
	onboarding.BrewPrerequisiteStore // ListBeans + ListBrewers
	ListGrinders(ctx context.Context) ([]*arabica.Grinder, error)
	ListRoasters(ctx context.Context) ([]*arabica.Roaster, error)
}

// HandleGetStartedCard returns the rendered onboarding card. Reloaded via
// HTMX on every refreshManage event so newly-created entities show up.
//
// When the user is fully ready, the card still renders (with the green
// "Log your first brew" CTA). The home template decides whether to mount
// the card slot at all on the initial server render.
func (h *Handlers) HandleGetStartedCard(w http.ResponseWriter, r *http.Request) {
	store, authenticated := h.GetAtprotoStore(r)
	if !authenticated {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	props, err := buildGetStartedCardProps(r.Context(), store)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build get-started card props")
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}

	if err := coffee.GetStartedCard(props).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render get-started card")
		http.Error(w, "Failed to render", http.StatusInternalServerError)
	}
}

func buildGetStartedCardProps(ctx context.Context, store getStartedCardStore) (coffee.GetStartedCardProps, error) {
	status, err := onboarding.CheckBrewReadiness(ctx, store)
	if err != nil {
		return coffee.GetStartedCardProps{}, err
	}
	beans, err := store.ListBeans(ctx)
	if err != nil {
		return coffee.GetStartedCardProps{}, err
	}
	brewers, err := store.ListBrewers(ctx)
	if err != nil {
		return coffee.GetStartedCardProps{}, err
	}
	grinders, err := store.ListGrinders(ctx)
	if err != nil {
		return coffee.GetStartedCardProps{}, err
	}
	roasters, err := store.ListRoasters(ctx)
	if err != nil {
		return coffee.GetStartedCardProps{}, err
	}
	return coffee.GetStartedCardProps{
		Readiness: status,
		Beans:     beans,
		Brewers:   brewers,
		Grinders:  grinders,
		Roasters:  roasters,
	}, nil
}
