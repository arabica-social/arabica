# Tea Phase G — Templ Shell Extraction (discussion brief)

**Status:** drafted for review (not yet implemented)
**Author:** ptdewey (with Claude)
**Created:** 2026-05-08
**Parent spec:** `docs/tea-multitenant-refactor.md`

---

## Goal

After Phase G, **every page that doesn't render entity-specific
content** is parameterised by `*domain.App` (which carries
`Descriptors`, `Brand`, `Name`, etc.) and renders identically for
arabica and matcha. Per-entity content (record fields, modal bodies,
view pages) stays bespoke per app.

## What's in scope

1. **Modal shell extraction** (ED phase 5'). The dialog mechanics —
   open/close, header bar with title + close button, validation error
   strip, footer with cancel/submit, dirty-tracking — become a single
   `components.EntityModalShell(app, descriptor, …)`. Each entity
   modal still owns its **field body** as a separate templ file; the
   shell wraps that body. ED's earlier full-FieldSpec design was
   rejected (`docs/entity-descriptor-refactor.md:128`); modal shell is
   the chosen replacement.

2. **Manage tab parameterisation**. `manage_partial.templ` currently
   hardcodes one tab per entity. Convert to a loop over
   `app.Descriptors`. Each tab calls into a per-entity table component
   (still bespoke), but the surrounding tab strip + empty states +
   refresh buttons are app-driven.

3. **Feed page parameterisation**. The feed's filter pills
   (`feedFilterTabs()` in `feed.templ`) are a hardcoded list of
   arabica entities. Drive them from `app.Descriptors` (Noun for
   label, NSID for value). Layout, header, footer derive brand
   strings from `app.Brand`.

4. **Profile page parameterisation**. The profile's per-entity
   sections (recent brews, pinned beans, etc.) become a loop over
   descriptors. Per-entity rendering still dispatches to the bespoke
   record component, but the list/empty-state shell is shared.

5. **Layout/header/footer brand strings**. Every literal "Arabica"
   in `layout.templ`, `header.templ`, `footer.templ` reads from
   `app.Brand` instead. CSS palette references stay arabica-specific
   for now (each app embeds its own `static/css/output.css`).

## What's NOT in scope

- **Per-entity record components** (`record_brew.templ`,
  `record_bean.templ`, etc.) — these stay bespoke. Each app owns
  these.
- **Per-entity view pages** (`brew_view.templ`, `bean_view.templ`,
  etc.) — stay bespoke per app.
- **Per-entity modal field bodies** (the inner content of each
  modal) — stay bespoke. Only the surrounding shell extracts.
- **Routing changes** — Phase F handles route registration via
  `app.Descriptors`. Phase G assumes Phase F has landed.
- **CSS/Tailwind tokenization** — each app embeds its own bundle.
  Cross-app shared design tokens are a future concern.

## Key design questions to resolve before implementation

1. **How does `EntityModalShell` accept the entity field body?**

   Two patterns under consideration:

   - **Slot pattern**: shell takes a `templ.Component` for the body.
     Each entity modal is `EntityModalShell(app, desc, beanFields(bean))`.
     Pro: clean composition. Con: templ slot ergonomics can be
     awkward — passing components as parameters loses some scoping.

   - **Children pattern**: shell uses templ's `{ children... }`. Each
     entity modal wraps the shell. Pro: idiomatic templ. Con: harder
     to thread error/dirty state from shell into body without ctx.

   Recommend trying the children pattern first — it composes better
   with templ's existing idioms.

2. **Where do dirty-tracking and validation errors live?**

   Today these are duplicated across each modal. Candidates:

   - Pass an `EntityModalState` struct into the shell with dirty,
     errors, isSubmitting fields. Body reads from same struct.
   - Use Alpine.js `$dispatch` events; shell listens. Looser
     coupling but more JS plumbing.

   Recommend the struct approach — keeps state explicit, server-side.

3. **Can the manage tab strip work without per-entity overrides?**

   Each existing tab has subtle differences (some show counts, some
   don't; brew has no manage tab). Verify the loop covers all current
   variations before committing. If 1-2 entities need overrides, an
   `app.OverridesForTab` escape hatch beats forcing uniformity.

4. **Should `BrandConfig` carry templ fragments or just strings?**

   Strings ("Arabica", "tracking your brews") compose easily. But the
   landing page's hero copy is multi-line with formatting. Risk:
   `BrandConfig` grows to absorb HTML. Counter: keep `BrandConfig`
   pure-string; per-app `landing.templ` stays bespoke.

   Recommend keeping `BrandConfig` strings-only, allowing per-app
   bespoke pages where copy is rich.

## Phased breakdown

| # | Sub-phase | Effort | Deliverable |
|---|---|---|---|
| G.1 | `EntityModalShell` component + migrate one modal (roaster) as proof | ½–1 day | shell exists, one modal uses it |
| G.2 | Migrate remaining modals (bean, grinder, brewer, recipe, brew) | 1 day | all 6 modals use shell |
| G.3 | Manage tab strip → loop over descriptors | ½ day | manage page driven by `app` |
| G.4 | Feed filter pills → loop over descriptors | ½ day | filter bar driven by `app` |
| G.5 | Profile per-entity sections → loop | ½ day | profile driven by `app` |
| G.6 | Layout/header/footer brand strings → `app.Brand` | ½ day | brand strings centralised |
| G.7 | Final verification + spec update | ½ day | green tests, doc updates |

**Total estimate:** 3–4 days. Net LOC: −150.

## Risks / open questions for review

- **Templ children scoping**: if children pattern can't pass dirty/error
  state cleanly into the body, we fall back to slot pattern. Worth a
  quick prototype before committing.
- **Manage tab variations**: I'd want to enumerate every special-case
  in the current `manage_partial.templ` before assuming the loop
  generalises.
- **Phase F dependency**: Phase G assumes route registration is
  app-driven (Phase F). If Phase F isn't fully done, the manage tab
  loop can still ship but won't be wired to actual handlers yet.

## Success criteria

- Adding a new arabica entity adds zero lines to layout/header/footer,
  manage tab strip, feed filter bar, profile sections.
- Modal shell is one templ component used by all 6 modals; the LOC
  reduction is real, not just relocation.
- Brand strings live exactly once per app (in the `App` constructor in
  `cmd/{arabica,matcha}/main.go`).
- All tests green; integration suite preserves behaviour.
- A hypothetical matcha modal needs only its own field-body templ; the
  shell, header, footer, error strip, and submit button come for free.
