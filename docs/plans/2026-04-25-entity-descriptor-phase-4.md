# Entity Descriptor — Phase 4: View Handler Unification

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to
> implement this plan task-by-task.

**Goal:** Collapse the 4 near-identical entity view handlers (Bean, Roaster,
Grinder, Brewer) into a shared `handleEntityView` helper driven by a
per-entity config. Unify 3 OG image handlers and 3 OG metadata helpers as
a bonus. Recipe and brew stay bespoke.

**Scope note:** The user wants to differentiate view pages visually later.
This refactor only touches the *handler* layer (auth, cache, fallback, social
data, author profile). The templ view pages themselves are untouched — they
retain their per-entity layouts.

**Parent spec:** `docs/entity-descriptor-refactor.md`

---

## Major Design Changes

### `EntityViewBase` — embedded social fields

All 4 view props structs repeat 19 identical fields. Extract them into:

```go
// EntityViewBase holds the social and auth fields shared by all simple
// entity view pages. Embed this in XxxViewProps to use.
type EntityViewBase struct {
    IsOwnProfile      bool
    IsAuthenticated   bool
    SubjectURI        string
    SubjectCID        string
    IsLiked           bool
    LikeCount         int
    CommentCount      int
    Comments          []firehose.IndexedComment
    CurrentUserDID    string
    ShareURL          string
    IsModerator       bool
    CanHideRecord     bool
    CanBlockUser      bool
    IsRecordHidden    bool
    AuthorDID         string
    AuthorHandle      string
    AuthorDisplayName string
    AuthorAvatar      string
}
```

After embedding, `props.IsAuthenticated` still works in Go and templ via
promotion — zero changes needed in the templ page bodies.

Count fields (`BrewCount`, `BeanCount`) stay per-entity; the label differs
("X brews" vs "X beans") so collapsing them would hurt readability.

### `entityViewConfig` — per-entity closures

A config struct captures per-entity behavior as function fields. Configs are
constructed as methods on `*Handler` so closures can capture `h.witnessCache`
etc. naturally.

```go
type entityViewConfig struct {
    descriptor  *entities.Descriptor
    // fromWitness converts a witness map + rkey to the typed model. Closures
    // handle ref resolution (bean → roaster) where needed.
    fromWitness func(ctx context.Context, m map[string]any, uri, rkey, ownerDID string) (any, error)
    // fromPDS converts a PDS record entry to the typed model.
    fromPDS     func(ctx context.Context, e *atproto.PublicRecordEntry, rkey, ownerDID string) (any, error)
    // fromStore fetches from the authenticated user's AtprotoStore.
    // Returns (record, subjectURI, subjectCID, error).
    fromStore   func(ctx context.Context, s *atproto.AtprotoStore, rkey string) (any, string, string, error)
    // displayName extracts the page title from the record.
    displayName func(record any) string
    // ogSubtitle extracts the OG description subtitle from the record.
    ogSubtitle  func(record any) string
    // countLookup returns the entity-specific count (brews, beans, etc.).
    // Nil is OK — returns 0 if feedIndex is nil or URI is empty.
    countLookup func(ctx context.Context, ownerDID, subjectURI string) int
    // render constructs entity-specific props from the shared base and renders the page.
    render      func(ctx context.Context, w http.ResponseWriter, layoutData *components.LayoutData, record any, base EntityViewBase) error
}
```

### What stays bespoke

- `HandleRecipeView` and `HandleBrewView` — richer ref resolution (brew: bean
  chain, recipe: brewer obj). Don't force through the config.
- Bean's closures do roaster ref resolution; all others are 2-3 lines.
- `HandleBeanOGImage` and `HandleRecipeOGImage` — both have nested ref
  resolution; stay bespoke.

### What gets collapsed in the OG layer

- `populateRoasterOGMetadata`, `populateGrinderOGMetadata`,
  `populateBrewerOGMetadata` all call `populateOGFields(layoutData,
  entity.Name, noun, ...)`. One helper replaces three.
- `HandleRoasterOGImage`, `HandleGrinderOGImage`, `HandleBrewerOGImage` are
  byte-for-byte identical except type names. One `handleSimpleOGImage` helper
  + an `ogImageConfig` struct replaces all three.

---

## Tasks

### Task 1: Define `EntityViewBase` in pages package

**Files:**
- Create: `internal/web/pages/entity_view_base.go`

Define `EntityViewBase` (the struct above). This is a plain Go file, no templ.

### Task 2: Embed `EntityViewBase` in all 4 view props structs

**Files:**
- Modify: `internal/web/pages/bean_view.templ`
- Modify: `internal/web/pages/roaster_view.templ`
- Modify: `internal/web/pages/grinder_view.templ`
- Modify: `internal/web/pages/brewer_view.templ`

For each, replace the 19 repeated fields with `EntityViewBase`. Keep the
entity pointer and count field. Example for bean:

```go
type BeanViewProps struct {
    Bean       *models.Bean
    BrewCount  int
    EntityViewBase
}
```

After this change, the templ page body is unchanged — promoted fields are
accessed identically.

Run `templ generate` for each modified file. Verify `go build ./...` still
passes (promoted field assignments in the existing handlers still work).

### Task 3: Add `entityViewConfig`, `handleEntityView`, per-entity configs

**Files:**
- Modify: `internal/handlers/entity_views.go`

**Step 1: Add `entityViewConfig` struct** (see design above).

**Step 2: Add `handleEntityView` method:**

```go
func (h *Handler) handleEntityView(w http.ResponseWriter, r *http.Request, cfg entityViewConfig) {
    rkey := validateRKey(w, r.PathValue("id"))
    if rkey == "" { return }

    owner := r.URL.Query().Get("owner")
    didStr, _ := atproto.GetAuthenticatedDID(r.Context())
    isAuthenticated := didStr != ""

    var userProfile *bff.UserProfile
    if isAuthenticated {
        userProfile = h.getUserProfile(r.Context(), didStr)
    }

    var record any
    var subjectURI, subjectCID, entityOwnerDID string
    isOwnProfile := false

    if owner != "" {
        var err error
        entityOwnerDID, err = resolveOwnerDID(r.Context(), owner)
        if err != nil {
            http.Error(w, "User not found", http.StatusNotFound)
            return
        }

        entityURI := atproto.BuildATURI(entityOwnerDID, cfg.descriptor.NSID, rkey)
        if h.witnessCache != nil {
            if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), entityURI); wr != nil {
                if m, err := atproto.WitnessRecordToMap(wr); err == nil {
                    if rec, err := cfg.fromWitness(r.Context(), m, wr.URI, rkey, entityOwnerDID); err == nil {
                        metrics.WitnessCacheHitsTotal.WithLabelValues(cfg.descriptor.Noun).Inc()
                        record = rec
                        subjectURI = wr.URI
                        subjectCID = wr.CID
                        isOwnProfile = isAuthenticated && didStr == entityOwnerDID
                    }
                }
            }
        }

        if record == nil {
            metrics.WitnessCacheMissesTotal.WithLabelValues(cfg.descriptor.Noun).Inc()
            pub := atproto.NewPublicClient()
            entry, err := pub.GetRecord(r.Context(), entityOwnerDID, cfg.descriptor.NSID, rkey)
            if err != nil {
                http.Error(w, cfg.descriptor.DisplayName+" not found", http.StatusNotFound)
                return
            }
            rec, err := cfg.fromPDS(r.Context(), entry, rkey, entityOwnerDID)
            if err != nil {
                http.Error(w, "Failed to load "+cfg.descriptor.Noun, http.StatusInternalServerError)
                return
            }
            record = rec
            subjectURI = entry.URI
            subjectCID = entry.CID
            isOwnProfile = isAuthenticated && didStr == entityOwnerDID
        }
    } else {
        store, authenticated := h.getAtprotoStore(r)
        if !authenticated {
            http.Redirect(w, r, "/login", http.StatusFound)
            return
        }
        atprotoStore, ok := store.(*atproto.AtprotoStore)
        if !ok {
            http.Error(w, "Internal error", http.StatusInternalServerError)
            return
        }
        rec, uri, cid, err := cfg.fromStore(r.Context(), atprotoStore, rkey)
        if err != nil {
            http.Error(w, cfg.descriptor.DisplayName+" not found", http.StatusNotFound)
            return
        }
        record, subjectURI, subjectCID = rec, uri, cid
        isOwnProfile = true
    }

    var shareURL string
    if owner != "" {
        shareURL = fmt.Sprintf("/%s/%s?owner=%s", cfg.descriptor.URLPath, rkey, owner)
    } else if userProfile != nil && userProfile.Handle != "" {
        shareURL = fmt.Sprintf("/%s/%s?owner=%s", cfg.descriptor.URLPath, rkey, userProfile.Handle)
    }

    ownerHandle := h.resolveOwnerHandle(r.Context(), owner)
    layoutData := h.buildLayoutData(r, cfg.displayName(record), isAuthenticated, didStr, userProfile)
    populateOGFields(layoutData, cfg.ogSubtitle(record), cfg.descriptor.Noun, ownerHandle, h.publicBaseURL(r), shareURL)

    sd := h.fetchSocialData(r.Context(), subjectURI, didStr, isAuthenticated)

    // Fetch author profile
    authorDID := entityOwnerDID
    if authorDID == "" { authorDID = didStr }
    base := EntityViewBase{
        IsOwnProfile:  isOwnProfile,
        IsAuthenticated: isAuthenticated,
        SubjectURI:    subjectURI,
        SubjectCID:    subjectCID,
        IsLiked:       sd.IsLiked,
        LikeCount:     sd.LikeCount,
        CommentCount:  sd.CommentCount,
        Comments:      sd.Comments,
        CurrentUserDID: didStr,
        ShareURL:      shareURL,
        IsModerator:   sd.IsModerator,
        CanHideRecord: sd.CanHideRecord,
        CanBlockUser:  sd.CanBlockUser,
        IsRecordHidden: sd.IsRecordHidden,
        AuthorDID:     entityOwnerDID,
    }
    if ap := h.getUserProfile(r.Context(), authorDID); ap != nil {
        base.AuthorHandle = ap.Handle
        base.AuthorDisplayName = ap.DisplayName
        base.AuthorAvatar = ap.Avatar
    }

    if err := cfg.render(r.Context(), w, layoutData, record, base); err != nil {
        http.Error(w, "Failed to render page", http.StatusInternalServerError)
        log.Error().Err(err).Msgf("Failed to render %s view", cfg.descriptor.Noun)
    }
}
```

**Step 3: Define per-entity config methods**

```go
func (h *Handler) roasterViewConfig() entityViewConfig { ... }
func (h *Handler) grinderViewConfig() entityViewConfig { ... }
func (h *Handler) brewerViewConfig() entityViewConfig { ... }
func (h *Handler) beanViewConfig() entityViewConfig { ... } // includes roaster ref resolution
```

Each config's `render` closure: looks up its count (via a closure capturing
`h.feedIndex`), constructs the typed props struct embedding `EntityViewBase`,
and calls `pages.XView(layoutData, props).Render(ctx, w)`.

### Task 4: Replace 4 view handlers

Each becomes a one-liner:

```go
func (h *Handler) HandleBeanView(w http.ResponseWriter, r *http.Request) {
    h.handleEntityView(w, r, h.beanViewConfig())
}
```

### Task 5: Collapse OG metadata and OG image handlers

**Collapse 3 OG metadata helpers:**

```go
// Replace populateRoasterOGMetadata, populateGrinderOGMetadata, populateBrewerOGMetadata
func (h *Handler) populateSimpleEntityOGMetadata(
    d *entities.Descriptor,
    layoutData *components.LayoutData,
    name, owner, baseURL, shareURL string,
) {
    populateOGFields(layoutData, name, d.Noun, owner, baseURL, shareURL)
}
```

Call from beanViewConfig's ogSubtitle and from roaster/grinder/brewer render
closures. Or just call `populateOGFields` directly in `handleEntityView` via
the `ogSubtitle` field — which the plan already does.

Actually: with `handleEntityView` calling `populateOGFields(layoutData,
cfg.ogSubtitle(record), ...)` directly, the `populateXOGMetadata` functions
are no longer called at all. Delete them.

Bean's custom subtitle ("Name from Roaster") lives in its `ogSubtitle`
closure.

**Unify 3 OG image handlers:**

```go
type ogImageConfig struct {
    nsid        string
    metricLabel string
    convert     func(m map[string]any, uri string) (any, error)
    drawCard    func(record any) (*ogcard.Card, error)
}

func (h *Handler) handleSimpleOGImage(w http.ResponseWriter, r *http.Request, cfg ogImageConfig) {
    rkey := validateRKey(w, r.PathValue("id"))
    if rkey == "" { return }
    owner := r.URL.Query().Get("owner")
    if owner == "" {
        http.Error(w, "owner parameter required", http.StatusBadRequest)
        return
    }
    ownerDID, err := resolveOwnerDID(r.Context(), owner)
    if err != nil {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    }
    var record any
    entityURI := atproto.BuildATURI(ownerDID, cfg.nsid, rkey)
    if h.witnessCache != nil {
        if wr, _ := h.witnessCache.GetWitnessRecord(r.Context(), entityURI); wr != nil {
            if m, err := atproto.WitnessRecordToMap(wr); err == nil {
                if rec, err := cfg.convert(m, wr.URI); err == nil {
                    metrics.WitnessCacheHitsTotal.WithLabelValues(cfg.metricLabel).Inc()
                    record = rec
                    setRKey(record, rkey)
                }
            }
        }
    }
    if record == nil {
        metrics.WitnessCacheMissesTotal.WithLabelValues(cfg.metricLabel).Inc()
        pub := atproto.NewPublicClient()
        pr, err := pub.GetRecord(r.Context(), ownerDID, cfg.nsid, rkey)
        if err != nil {
            http.Error(w, "Not found", http.StatusNotFound)
            return
        }
        rec, err := cfg.convert(pr.Value, pr.URI)
        if err != nil {
            http.Error(w, "Failed to load record", http.StatusInternalServerError)
            return
        }
        record = rec
        setRKey(record, rkey)
    }
    card, err := cfg.drawCard(record)
    if err != nil {
        log.Error().Err(err).Msgf("Failed to generate %s OG image", cfg.metricLabel)
        http.Error(w, "Failed to generate image", http.StatusInternalServerError)
        return
    }
    writeOGImage(w, card)
}
```

Note: `setRKey` is a small helper that type-switches on record and sets `.RKey`.
Or: change `convert` to also accept and set rkey. Simpler.

The 3 handlers become:

```go
func (h *Handler) HandleRoasterOGImage(w http.ResponseWriter, r *http.Request) {
    h.handleSimpleOGImage(w, r, ogImageConfig{
        nsid: atproto.NSIDRoaster, metricLabel: "roaster_og",
        convert:  func(m map[string]any, uri string) (any, error) { return atproto.RecordToRoaster(m, uri) },
        drawCard: func(rec any) (*ogcard.Card, error) { return ogcard.DrawRoasterCard(rec.(*models.Roaster)) },
    })
}
```

### Task 6: Verify

```bash
go vet ./...
go build ./...
go test ./...
just run
```

Manual smoke test: load `/beans/{rkey}?owner=X`, `/roasters/...`, `/grinders/...`,
`/brewers/...` — all should render identically to before. Check bean view
shows roaster name. Check OG image endpoints.

---

## Expected delta

- 4 view handlers (~180 LOC each): collapsed to 4 one-liners + 4 config methods
  (~25 LOC each) + 1 `handleEntityView` (~80 LOC) ≈ −430 LOC
- `EntityViewBase` embedding: 19 fields × 4 structs → 19-field base struct
  ≈ −55 LOC  
- 3 OG metadata helpers → 0 (folded into config's `ogSubtitle`) ≈ −25 LOC
- 3 OG image handlers (~50 LOC each) → 3 one-liners + 1 helper (~50 LOC)
  ≈ −100 LOC
- **Total: ~−610 LOC** (largest phase by LOC reduction)
