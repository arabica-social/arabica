# FDR-001: Explore

**Status:** Active
**Last reviewed:** 2026-07-16

## Overview

Explore is Arabica's authenticated discovery surface for reusable coffee
records contributed by the community. It complements the activity-oriented
Community Feed: the feed answers what people are doing, while Explore helps a
user find beans, roasters, gear, and recipes worth examining or reusing.

## Behavior

- Explore requires authentication. An unauthenticated user is prompted to log
  in rather than receiving a public discovery surface.
- Users can discover beans, roasters, grinders, brewers, and recipes.
- Brews are excluded because they represent activity and history rather than a
  reusable discovery object.
- Results can be searched, filtered by record type and type-specific facets,
  and sorted by recent, popular, or highest-rated.
- Results load in cursor-based pages.
- The viewer's own records remain in results and retain ownership affordances.
- Cards show author identity and applicable rating, like, comment, and reuse
  context. The viewer's like state is reflected in the card actions.
- Records hidden by the viewer's moderation settings are excluded.
- Explicit copies or forks are grouped into one visible canonical result. Reuse
  is described with product language such as “Used by N”; the internal
  `sourceRef` field name is not shown to users.
- If the local Explore projection is rebuilding or dirty, the page remains
  available and warns that results may be slightly stale.

## Design Decisions

### 1. Discovery complements the feed

**Decision:** Explore indexes reusable coffee objects rather than brew activity.

**Why:** A community activity stream and a reusable-record catalog answer
different user needs. Keeping the distinction lets Explore provide structured
facets without turning every brew event into a search result.

**Tradeoff:** Brew history and related-brew counts need separate feed, record,
or future discovery surfaces.

### 2. Authentication is required

**Decision:** Explore is available only to authenticated users.

**Why:** Authentication limits scraper and compute exposure while the service
and its operational model are still young, and gives the page viewer context
for likes, ownership, and moderation.

**Tradeoff:** Public visitors cannot browse community records through Explore.

### 3. The index is a rebuildable projection

**Decision:** Explore reads a local firehose-derived projection rather than
querying users' PDSs for every discovery request or owning record state itself.

**Why:** Search, facets, global sorting, duplicate grouping, and social
aggregation require local indexed data. PDS authority is preserved by making
the projection disposable and recoverable. See ADR-003.

**Tradeoff:** Results can temporarily lag. The product must communicate stale
or rebuilding state rather than presenting the projection as perfectly current.

### 4. Duplicate grouping requires explicit provenance

**Decision:** Copies and forks cluster through explicit source references, not
through fuzzy matching of names or fields.

**Why:** Similar names do not prove common authorship, identity, or provenance.
Explicit links support deterministic grouping without silently merging
independent records.

**Tradeoff:** Independently entered duplicates remain separate unless a real
source relationship is recorded.

### 5. Own records remain discoverable

**Decision:** Explore does not filter out records owned by the viewer.

**Why:** A user's records remain relevant to searches, comparisons, community
counts, and reuse relationships. Explore is a catalog, not solely a surface for
“other people's records.”

**Tradeoff:** Users may see records they can already reach through My Coffee.

## Related

- **ADRs:** [ADR-001](../adr/ADR-001-pds-records-are-authoritative.md), [ADR-003](../adr/ADR-003-local-indexes-are-rebuildable-projections.md)
- **FDRs:** None
- **API:** [`docs/api/explore.md`](../api/explore.md)

## Open Questions

- Whether Explore should eventually be available without authentication.
- Whether related brew counts belong on discovery cards or only on record pages.
- Whether future recommendations should remain part of Explore or become a
  separate personalized surface.
