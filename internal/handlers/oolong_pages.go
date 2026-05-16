package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/attribute"
	atp "tangled.org/pdewey.com/atp"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"

	"tangled.org/arabica.social/arabica/internal/atproto"
	"tangled.org/arabica.social/arabica/internal/entities/oolong"
	teapages "tangled.org/arabica.social/arabica/internal/oolong/web/pages"
	"tangled.org/arabica.social/arabica/internal/tracing"
	"tangled.org/arabica.social/arabica/internal/web/bff"
)

// HandleTeaRefresh is the oolong equivalent of HandleManageRefresh.
// Invalidates session cache, re-fetches every oolong entity collection
// from the user's PDS, and writes them through to the witness cache.
// Returns 204 — the my-tea page is server-rendered, so the client
// reloads after the POST resolves.
func (h *Handler) HandleTeaRefresh(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}

	sessionID, ok := atpmiddleware.GetSessionID(r.Context())
	if !ok {
		http.Error(w, "Session required", http.StatusUnauthorized)
		return
	}
	didStr, ok := atpmiddleware.GetDID(r.Context())
	if !ok {
		http.Error(w, "Authentication required", http.StatusUnauthorized)
		return
	}
	did, err := syntax.ParseDID(didStr)
	if err != nil {
		http.Error(w, "Invalid DID", http.StatusInternalServerError)
		return
	}

	h.sessionCache.Invalidate(sessionID)

	collections := []string{
		oolong.NSIDTea, oolong.NSIDVendor, oolong.NSIDVessel,
		oolong.NSIDInfuser, oolong.NSIDBrew,
	}

	if h.witnessCache != nil {
		ctx, span := tracing.HandlerSpan(r.Context(), "tea.refresh.witness_sync",
			attribute.String("user.did", didStr),
		)
		defer span.End()
		atpClient, err := h.atprotoClient.AtpClient(ctx, did, sessionID)
		if err != nil {
			log.Warn().Err(err).Msg("tea refresh: failed to get atp client")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		var batch []atproto.WitnessWriteRecord
		for _, collection := range collections {
			records, err := atpClient.ListAllRecords(ctx, collection)
			if err != nil {
				log.Warn().Err(err).Str("collection", collection).Msg("tea refresh: list failed")
				continue
			}
			for _, rec := range records {
				rkey := atp.RKeyFromURI(rec.URI)
				if rkey == "" {
					continue
				}
				recordJSON, jsonErr := json.Marshal(rec.Value)
				if jsonErr != nil {
					continue
				}
				batch = append(batch, atproto.WitnessWriteRecord{
					DID:        didStr,
					Collection: collection,
					RKey:       rkey,
					CID:        rec.CID,
					Record:     recordJSON,
				})
			}
			short := collection[strings.LastIndex(collection, ".")+1:]
			log.Info().Str("collection", short).Int("count", len(records)).Msg("tea refresh: fetched")
		}
		if err := h.witnessCache.UpsertWitnessRecordBatch(ctx, batch); err != nil {
			log.Error().Err(err).Msg("tea refresh: batch upsert failed")
		}
		span.SetAttributes(attribute.Int("refresh.total_records", len(batch)))
	}

	_ = store
	w.WriteHeader(http.StatusNoContent)
}

// HandleMyTea renders /my-tea — the oolong equivalent of arabica's
// /my-coffee. Fetches all 7 entity types the authenticated user owns
// and lists them in collapsible sections. Each section surfaces a
// "+ Add" button that opens the matching dialog modal.
//
// Lists come from store.FetchAllRecords (witness-cache-first) + the
// matching oolong.RecordTo* decoder, not from typed Get* wrappers
// (which don't exist for oolong yet). Reference fields (Vendor on
// Tea, etc.) are not joined here — the row label uses the entity's
// own Name or a sensible fallback.
func (h *Handler) HandleMyTea(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}

	layoutData, did, _ := h.layoutDataFromRequest(r, "My Tea")
	actor := did
	if layoutData != nil && layoutData.UserProfile != nil && layoutData.UserProfile.Handle != "" {
		actor = layoutData.UserProfile.Handle
	}

	teas := listOolong(r.Context(), store, oolong.NSIDTea, oolong.RecordToTea)
	vendors := listOolong(r.Context(), store, oolong.NSIDVendor, oolong.RecordToVendor)
	vessels := listOolong(r.Context(), store, oolong.NSIDVessel, oolong.RecordToVessel)
	infusers := listOolong(r.Context(), store, oolong.NSIDInfuser, oolong.RecordToInfuser)
	brews := listOolong(r.Context(), store, oolong.NSIDBrew, oolong.RecordToBrew)

	// Join references so the steep cards can show tea name, vessel, infuser.
	vendorByRKey := make(map[string]*oolong.Vendor, len(vendors))
	for _, v := range vendors {
		vendorByRKey[v.RKey] = v
	}
	teaByRKey := make(map[string]*oolong.Tea, len(teas))
	for _, t := range teas {
		if t.Vendor == nil && t.VendorRKey != "" {
			t.Vendor = vendorByRKey[t.VendorRKey]
		}
		teaByRKey[t.RKey] = t
	}
	vesselByRKey := make(map[string]*oolong.Vessel, len(vessels))
	for _, v := range vessels {
		vesselByRKey[v.RKey] = v
	}
	infuserByRKey := make(map[string]*oolong.Infuser, len(infusers))
	for _, i := range infusers {
		infuserByRKey[i.RKey] = i
	}
	for _, b := range brews {
		if b.Tea == nil && b.TeaRKey != "" {
			b.Tea = teaByRKey[b.TeaRKey]
		}
		if b.Vessel == nil && b.VesselRKey != "" {
			b.Vessel = vesselByRKey[b.VesselRKey]
		}
		if b.Infuser == nil && b.InfuserRKey != "" {
			b.Infuser = infuserByRKey[b.InfuserRKey]
		}
	}

	props := teapages.MyTeaProps{
		Teas:     teas,
		Vendors:  vendors,
		Vessels:  vessels,
		Infusers: infusers,
		Brews:    brews,
		Actor:    actor,
	}

	if err := teapages.MyTea(layoutData, props).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render my-tea page")
	}
}

// listOolong fetches every record of nsid the user owns and decodes
// each one. Records that fail to decode are logged and skipped — the
// page degrades to "this record is missing" rather than 500ing.
func listOolong[T any](
	ctx context.Context,
	store *atproto.AtprotoStore,
	nsid string,
	decode func(map[string]any, string) (*T, error),
) []*T {
	raw, err := store.FetchAllRecords(ctx, nsid)
	if err != nil {
		log.Warn().Err(err).Str("nsid", nsid).Msg("FetchAllRecords failed; rendering empty list")
		return nil
	}
	out := make([]*T, 0, len(raw))
	for _, r := range raw {
		t, err := decode(r.Record, r.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", r.URI).Msg("decode failed; skipping record")
			continue
		}
		out = append(out, t)
	}
	return out
}

// HandleOolongTeaNew renders the new-tea page.
func (h *Handler) HandleOolongTeaNew(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	props := teapages.TeaFormProps{
		Tea:     nil,
		Vendors: listOolong(r.Context(), store, oolong.NSIDVendor, oolong.RecordToVendor),
	}
	layoutData, _, _ := h.layoutDataFromRequest(r, "New Tea")
	if err := teapages.TeaFormPage(layoutData, props).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render new-tea page")
	}
}

// HandleOolongTeaEdit renders the edit-tea page for an existing tea.
func (h *Handler) HandleOolongTeaEdit(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	rec, uri, _, err := store.FetchRecord(r.Context(), oolong.NSIDTea, rkey)
	if err != nil {
		http.Error(w, "Tea not found", http.StatusNotFound)
		return
	}
	t, err := oolong.RecordToTea(rec, uri)
	if err != nil {
		http.Error(w, "Failed to decode tea", http.StatusInternalServerError)
		return
	}
	t.RKey = rkey
	props := teapages.TeaFormProps{
		Tea:     t,
		Vendors: listOolong(r.Context(), store, oolong.NSIDVendor, oolong.RecordToVendor),
	}
	layoutData, _, _ := h.layoutDataFromRequest(r, "Edit Tea")
	if err := teapages.TeaFormPage(layoutData, props).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render edit-tea page")
	}
}

// HandleOolongSteepNew renders the new-steep page (oolong brew create form).
// Mirrors arabica's /brews/new, but for tea: full page form instead of
// modal so the form can grow without crowding a dialog.
func (h *Handler) HandleOolongSteepNew(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	props := teapages.SteepFormProps{
		Brew:     nil,
		Teas:     listOolong(r.Context(), store, oolong.NSIDTea, oolong.RecordToTea),
		Vessels:  listOolong(r.Context(), store, oolong.NSIDVessel, oolong.RecordToVessel),
		Infusers: listOolong(r.Context(), store, oolong.NSIDInfuser, oolong.RecordToInfuser),
	}
	layoutData, _, _ := h.layoutDataFromRequest(r, "New Steep")
	if err := teapages.SteepFormPage(layoutData, props).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render new-steep page")
	}
}

// HandleOolongSteepEdit renders the edit-steep page for an existing brew.
func (h *Handler) HandleOolongSteepEdit(w http.ResponseWriter, r *http.Request) {
	store, ok := h.requireOolongStore(w, r)
	if !ok {
		return
	}
	rkey := validateRKey(w, r.PathValue("id"))
	if rkey == "" {
		return
	}
	rec, uri, _, err := store.FetchRecord(r.Context(), oolong.NSIDBrew, rkey)
	if err != nil {
		http.Error(w, "Brew not found", http.StatusNotFound)
		return
	}
	b, err := oolong.RecordToBrew(rec, uri)
	if err != nil {
		http.Error(w, "Failed to decode brew", http.StatusInternalServerError)
		return
	}
	b.RKey = rkey
	props := teapages.SteepFormProps{
		Brew:     b,
		Teas:     listOolong(r.Context(), store, oolong.NSIDTea, oolong.RecordToTea),
		Vessels:  listOolong(r.Context(), store, oolong.NSIDVessel, oolong.RecordToVessel),
		Infusers: listOolong(r.Context(), store, oolong.NSIDInfuser, oolong.RecordToInfuser),
	}
	layoutData, _, _ := h.layoutDataFromRequest(r, "Edit Steep")
	if err := teapages.SteepFormPage(layoutData, props).Render(r.Context(), w); err != nil {
		log.Error().Err(err).Msg("Failed to render edit-steep page")
	}
}

// HandleOolongProfile renders /profile/{actor} for the oolong app.
// Intentionally smaller surface than arabica's HandleProfile: it
// resolves the actor → DID, fetches their public profile + oolong
// records (brews, teas, vendors), and renders a minimal page. The
// owner-only tabs, equipment management, and modal-driven editing
// flows live on /my-tea instead — public profiles stay read-only.
func (h *Handler) HandleOolongProfile(w http.ResponseWriter, r *http.Request) {
	actor := r.PathValue("actor")
	if actor == "" {
		http.Error(w, "Actor parameter is required", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	publicClient := atproto.NewPublicClient()

	// Resolve actor → DID (handle or did:plc:...).
	var did string
	var err error
	if strings.HasPrefix(actor, "did:") {
		did = actor
	} else {
		if h.feedIndex != nil {
			did, _ = h.feedIndex.GetDIDByHandle(ctx, actor)
		}
		if did == "" {
			did, err = publicClient.ResolveHandle(ctx, actor)
			if err != nil {
				http.Error(w, "User not found", http.StatusNotFound)
				return
			}
		}
	}

	// Fetch atproto profile (display name, avatar, handle).
	var profile *atproto.Profile
	if h.feedIndex != nil {
		profile, _ = h.feedIndex.GetProfile(ctx, did)
	}
	if profile == nil {
		profile, err = publicClient.GetProfile(ctx, did)
		if err != nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
	}

	// Canonical-handle redirect.
	if strings.HasPrefix(actor, "did:") && profile.Handle != "" {
		http.Redirect(w, r, "/profile/"+profile.Handle, http.StatusFound)
		return
	}

	viewedProfile := &bff.UserProfile{Handle: profile.Handle}
	if profile.DisplayName != nil {
		viewedProfile.DisplayName = *profile.DisplayName
	}
	if profile.Avatar != nil {
		viewedProfile.Avatar = *profile.Avatar
	}

	// Fetch oolong records the user has published. Public reads via
	// the AT Protocol client — viewer doesn't need to be logged in to
	// the owner's PDS. All seven entity types are loaded server-side
	// so tab switching on the rendered page stays instant.
	brews := listOolongPublic(ctx, publicClient, did, oolong.NSIDBrew, oolong.RecordToBrew)
	teas := listOolongPublic(ctx, publicClient, did, oolong.NSIDTea, oolong.RecordToTea)
	vendors := listOolongPublic(ctx, publicClient, did, oolong.NSIDVendor, oolong.RecordToVendor)
	vessels := listOolongPublic(ctx, publicClient, did, oolong.NSIDVessel, oolong.RecordToVessel)
	infusers := listOolongPublic(ctx, publicClient, did, oolong.NSIDInfuser, oolong.RecordToInfuser)

	// Resolve brew → tea/vessel/infuser joins so cards can render the
	// tea name (and gear) without a separate fetch per item.
	teaByRKey := make(map[string]*oolong.Tea, len(teas))
	for _, t := range teas {
		teaByRKey[t.RKey] = t
	}
	vendorByRKey := make(map[string]*oolong.Vendor, len(vendors))
	for _, v := range vendors {
		vendorByRKey[v.RKey] = v
	}
	vesselByRKey := make(map[string]*oolong.Vessel, len(vessels))
	for _, v := range vessels {
		vesselByRKey[v.RKey] = v
	}
	infuserByRKey := make(map[string]*oolong.Infuser, len(infusers))
	for _, i := range infusers {
		infuserByRKey[i.RKey] = i
	}
	for _, t := range teas {
		if t.Vendor == nil && t.VendorRKey != "" {
			t.Vendor = vendorByRKey[t.VendorRKey]
		}
	}
	for _, b := range brews {
		if b.Tea == nil && b.TeaRKey != "" {
			b.Tea = teaByRKey[b.TeaRKey]
		}
		if b.Vessel == nil && b.VesselRKey != "" {
			b.Vessel = vesselByRKey[b.VesselRKey]
		}
		if b.Infuser == nil && b.InfuserRKey != "" {
			b.Infuser = infuserByRKey[b.InfuserRKey]
		}
	}

	_, didStr, isAuthenticated := h.layoutDataFromRequest(r, "Profile")
	isOwn := isAuthenticated && didStr == did

	if len(brews) == 0 && len(teas) == 0 && len(vendors) == 0 &&
		len(vessels) == 0 && len(infusers) == 0 &&
		!h.feedRegistry.IsRegistered(did) {
		layoutData, _, _ := h.layoutDataFromRequest(r, "Profile Not Found")
		w.WriteHeader(http.StatusNotFound)
		if err := teapages.ProfileNotFound(layoutData).Render(ctx, w); err != nil {
			log.Error().Err(err).Msg("Failed to render oolong profile-not-found page")
		}
		return
	}

	pageTitle := "@" + viewedProfile.Handle
	if viewedProfile.DisplayName != "" {
		pageTitle = viewedProfile.DisplayName + " (@" + viewedProfile.Handle + ")"
	}
	layoutData, _, _ := h.layoutDataFromRequest(r, pageTitle)

	props := teapages.ProfileProps{
		Profile:      viewedProfile,
		DID:          did,
		IsOwnProfile: isOwn,
		Brews:        brews,
		Teas:         teas,
		Vendors:      vendors,
		Vessels:      vessels,
		Infusers:     infusers,
	}
	if err := teapages.Profile(layoutData, props).Render(ctx, w); err != nil {
		log.Error().Err(err).Msg("Failed to render oolong profile page")
	}
}

// listOolongPublic mirrors listOolong but reads from an arbitrary DID's
// PDS via the public client rather than the authenticated AtprotoStore.
// Used by HandleOolongProfile to surface another user's records.
func listOolongPublic[T any](
	ctx context.Context,
	client *atp.PublicClient,
	did, nsid string,
	decode func(map[string]any, string) (*T, error),
) []*T {
	records, err := client.ListAllRecords(ctx, did, nsid)
	if err != nil {
		// Not having a record type isn't an error from the user's POV — it
		// just means they haven't created any of that entity. The PDS may
		// return 404 for an empty collection; degrade to empty list.
		return nil
	}
	out := make([]*T, 0, len(records))
	for _, rec := range records {
		t, err := decode(rec.Value, rec.URI)
		if err != nil {
			log.Warn().Err(err).Str("uri", rec.URI).Msg("oolong profile: decode failed; skipping record")
			continue
		}
		out = append(out, t)
	}
	return out
}
