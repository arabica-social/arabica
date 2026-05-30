package teahandlers

import (
	"context"
	"net/http"

	"github.com/rs/zerolog/log"

	"tangled.org/arabica.social/arabica/internal/handlers"
	oolong "tangled.org/arabica.social/arabica/internal/oolong/entities"
	tea "tangled.org/arabica.social/arabica/internal/oolong/web/components"
	teapages "tangled.org/arabica.social/arabica/internal/oolong/web/pages"
	"tangled.org/arabica.social/arabica/internal/records"
)

func (h *Handlers) HandleOolongOnboarding(w http.ResponseWriter, r *http.Request) {
	store, ok := h.RequireRecordStore(w, r)
	if !ok {
		return
	}
	props, err := buildOolongGetStartedCardProps(r.Context(), store)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build oolong onboarding props")
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}
	if props.Readiness.Ready() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	layoutData, _, _ := h.LayoutDataFromRequest(r, "Get Started")
	if err := teapages.Onboarding(layoutData, teapages.OnboardingProps{Card: props}).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render oolong onboarding page")
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

func (h *Handlers) HandleOolongGetStartedCard(w http.ResponseWriter, r *http.Request) {
	store, ok := h.RequireRecordStore(w, r)
	if !ok {
		return
	}
	props, err := buildOolongGetStartedCardProps(r.Context(), store)
	if err != nil {
		log.Error().Err(err).Msg("Failed to build oolong get-started card props")
		http.Error(w, "Failed to load", http.StatusInternalServerError)
		return
	}
	props.Mode = r.URL.Query().Get("mode")
	if err := tea.GetStartedCard(props).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render oolong get-started card")
		http.Error(w, "Failed to render", http.StatusInternalServerError)
	}
}

func (h *Handlers) HandleOolongOnboardingStationForm(w http.ResponseWriter, r *http.Request) {
	store, ok := h.RequireRecordStore(w, r)
	if !ok {
		return
	}
	kind := r.PathValue("kind")
	props, ok := tea.StationDrawerPropsForKind(kind)
	if !ok {
		http.Error(w, "Unknown entity kind", http.StatusBadRequest)
		return
	}
	if kind == "tea" {
		vendors := handlers.ListRecords(r.Context(), store, oolong.NSIDVendor, oolong.RecordToVendor)
		props.Vendors = vendors
	}
	if err := tea.StationFormDrawer(props).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render oolong station form drawer")
		http.Error(w, "Failed to render", http.StatusInternalServerError)
	}
}

func buildOolongGetStartedCardProps(ctx context.Context, store records.Store) (tea.GetStartedCardProps, error) {
	teas := handlers.ListRecords(ctx, store, oolong.NSIDTea, oolong.RecordToTea)
	vendors := handlers.ListRecords(ctx, store, oolong.NSIDVendor, oolong.RecordToVendor)
	vessels := handlers.ListRecords(ctx, store, oolong.NSIDVessel, oolong.RecordToVessel)
	infusers := handlers.ListRecords(ctx, store, oolong.NSIDInfuser, oolong.RecordToInfuser)
	return tea.GetStartedCardProps{
		Readiness: tea.ReadinessStatus{HasTea: len(teas) > 0, HasVendor: len(vendors) > 0, HasVessel: len(vessels) > 0, HasInfuser: len(infusers) > 0},
		Teas:      teas, Vendors: vendors, Vessels: vessels, Infusers: infusers,
	}, nil
}

func (h *Handlers) oolongReadyToBrew(ctx context.Context, store records.Store) (bool, error) {
	props, err := buildOolongGetStartedCardProps(ctx, store)
	if err != nil {
		return false, err
	}
	return props.Readiness.Ready(), nil
}
