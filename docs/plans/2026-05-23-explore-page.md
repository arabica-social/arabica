# Arabica Explore Page Plan

Date: 2026-05-23

## Goal

Add an authenticated Explore page for Arabica that lets users discover public
records from the community through structured filters, faceted counts, search,
and social ranking.

Explore is a separate discovery surface, not a replacement for the feed. The
feed remains temporal and activity-oriented; Explore is for finding reusable
coffee objects such as beans, roasters, grinders, brewers, and recipes.

## Scope

V1 is Arabica-only.

Included record types:

- Beans
- Roasters
- Grinders
- Brewers
- Recipes

Excluded from V1:

- Brews
- Likes and comments as result types
- Arabica cafes/drinks, until those records exist
- Oolong records

Brews are intentionally excluded from Explore because they are more like
activity/history than reusable discovery objects. Related brew information still
matters, but should appear later on entity view pages, for example “this bean has
12 brews.”

## Non-goals

- Do not replace the existing community feed.
- Do not change `/recipes` in this pass. The current recipe explore page stays
  as-is and can be merged into this backend later.
- Do not expose Explore to unauthenticated users in V1.
- Do not add Oolong Explore configuration in V1.
- Do not implement arbitrary graph search or arbitrary field/operator query
  builders.
- Do not infer duplicate clusters by fuzzy matching names or normalized fields.
- Do not make the Explore index a source of truth. It is a rebuildable witness
  cache derivative.

## Product behavior

### Access

Explore is authenticated-only in V1, primarily for compute and anti-scraper
reasons. Public Explore can be reconsidered later if the app has stronger rate
limits, caching, and/or funding for the extra load.

Explore should appear in the authenticated primary navigation as **Explore**.
Unauthenticated navigation should not expose it as a public browsing surface.

### Result ownership

Explore includes all matching records, including the viewer’s own records. Own
records should not be filtered out because they are still relevant to discovery
queries and counts.

Cards should show author identity in the same style used by feed/profile cards:
avatar, display name, and handle. This should use a reusable component rather
than reimplementing author rows for Explore.

### Cards

Explore cards should feel visually close to feed/profile cards, but not use
activity copy such as “new bean by …”. They are discovery cards, not feed events.

Cards should show useful scan metadata where available:

- record or community rating
- like count
- comment count
- subtle created date when it helps, especially under recent sort
- later: related brew counts and other cluster/user-version details

For clustered results, card-level social metadata should describe the visible
canonical record in V1, not the whole cluster. Adoption/reuse is still reflected
through the popularity score and can be shown later with product copy such as
“used by 2 users.” This avoids implying that likes/comments on different users’
copies all belong to the canonical author’s record.

The sourceRef mechanism should not be named directly in the UI. It may affect
popularity and clustering, but users should not see copy like “sourceRefs.” If a
future card needs to communicate the idea, use product language such as “used by
2 users,” tuned per record type.

## Filters and search

Explore uses a faceted UI with normal query parameters. The page should render
server-side first and may use HTMX later to enhance filter/result updates.

Global controls:

- `type`
- `q`
- `sort`

Arabica V1 type-specific controls:

- Beans
  - origin
  - variety
  - process
  - roast level
  - roaster name
  - minimum rating
  - closed/open
- Roasters
  - location
- Grinders
  - grinder type
  - burr type
- Brewers
  - brewer type
- Recipes
  - brewer type
  - ratio minimum
  - ratio maximum

Deferred filters:

- author filter
- coffee/water amount ranges
- richer first-hop reference filters
- arbitrary field/operator query builder

Freeform-ish filters such as origin, variety, location, and roaster name should
be exact facet selections with typeahead over known indexed values. This keeps
counts meaningful while still making the UI feel searchable. Filter behavior
should be registry-driven so it can be changed later.

Global `q` search should use a curated `search_text` field, not every raw JSON
field. It should include title/name, description/notes, and key facet text values
so searches like “ethiopia” can find records where Ethiopia appears as an
origin.

## Sorting

V1 sort modes:

- recent
- popular
- rating high

Popular is Explore-specific and should include adoption/reuse signals in
addition to normal social engagement:

```text
popular_score = source_ref_count*5 + like_count*3 + comment_count*2
```

For V1, `like_count` and `comment_count` in this formula are counts for the
individual indexed record. `source_ref_count` is the cluster/adoption signal:
count records whose `sourceRef` points at this record URI, plus cluster members
whose `cluster_key` is this URI where practical. Do not aggregate likes/comments
across all cluster members until the UI has explicit copy for cluster-wide
activity.

Keep these weights as named constants so they can be tuned later.

`rating_high` should use community average rating when available, falling back
to the record’s own rating. Unrated records should sort last. Community rating
should only be presented in the UI when it has enough data to be meaningful and
clearly labeled, for example an average across multiple related brews/users.

## SourceRef clustering and canonical records

Explore should show one result per sourceRef cluster.

Cluster key rules:

```text
cluster_key = sourceRef if sourceRef is present
cluster_key = uri       if sourceRef is absent
```

Records without a sourceRef are singleton clusters. V1 should not attempt fuzzy
or inferred deduplication by entity name, location, roaster, or other fields.

Canonical result selection should be designed for this direction:

1. Future preferred DID owner wins, if configured for the cluster/entity.
2. Otherwise, the original sourceRef target wins if it exists in the witness
   cache.
3. Otherwise, the best-scoring cluster member wins.

The preferred DID seam is important for future roaster onboarding, where a
verified/preferred roaster account may own the canonical copy of roaster or bean
records.

V1 can implement the original-target/best-scoring behavior and leave an explicit
seam/TODO for preferred DID ownership.

Implement canonical selection with deterministic sort keys that are usable from
SQL. A practical V1 ranking is:

1. `canonical_rank = 2` when `uri = cluster_key` (the original sourceRef target
   exists in the index).
2. `canonical_rank = 1` for all other cluster members.
3. Tie-break by `popular_score DESC`, then `created_at DESC`, then `uri ASC`.

Use a window function such as `ROW_NUMBER() OVER (PARTITION BY cluster_key ORDER
BY canonical_rank DESC, popular_score DESC, created_at DESC, uri ASC)` or an
equivalent two-stage query. Sort and paginate only after selecting `rn = 1`, so
pagination operates over clustered results rather than raw records.

TODO: sourceRef clustering may deserve a shared table or service later. It will
likely be useful outside Explore for entity view pages, canonical ownership,
roaster onboarding, and “other users’ versions” views.

Future entity view behavior:

- view pages should show other users’ versions in the same sourceRef cluster
- view pages should show related brews, e.g. “this bean has 12 brews”

Both are out of scope for the V1 Explore implementation.

## Index architecture

Explore should be backed by the existing firehose SQLite witness database, but
not by ad hoc JSON queries over `records` for every request.

Add generic derived tables owned by `FeedIndex`:

```sql
CREATE TABLE explore_documents (
    uri              TEXT PRIMARY KEY,
    did              TEXT NOT NULL,
    app              TEXT NOT NULL,
    record_type      TEXT NOT NULL,
    cluster_key      TEXT NOT NULL,
    canonical_rank   REAL NOT NULL DEFAULT 0,
    title            TEXT,
    summary          TEXT,
    search_text      TEXT,
    own_rating       REAL,
    community_rating REAL,
    rating_count     INTEGER NOT NULL DEFAULT 0,
    like_count       INTEGER NOT NULL DEFAULT 0,
    comment_count    INTEGER NOT NULL DEFAULT 0,
    source_ref_count INTEGER NOT NULL DEFAULT 0,
    popular_score    REAL NOT NULL DEFAULT 0,
    created_at       TEXT NOT NULL,
    FOREIGN KEY (uri) REFERENCES records(uri) ON DELETE CASCADE
);

CREATE TABLE explore_values (
    uri         TEXT NOT NULL,
    did         TEXT NOT NULL,
    app         TEXT NOT NULL,
    record_type TEXT NOT NULL,
    field       TEXT NOT NULL,
    value_text  TEXT,
    value_num   REAL,
    created_at  TEXT NOT NULL,
    FOREIGN KEY (uri) REFERENCES records(uri) ON DELETE CASCADE
);
```

Exact schema can vary during implementation, but keep these principles:

- one generic document table, not one table per record type
- one generic facet/value table, not one table per facet
- values are derived from `records`, likes/comments, sourceRef counts, and a
  small set of explicitly configured first-hop references
- the index is disposable and rebuildable from the witness cache
- the index must not become a second source of truth

Suggested indexes:

```sql
CREATE INDEX idx_explore_documents_app_type_created
    ON explore_documents(app, record_type, created_at DESC);

CREATE INDEX idx_explore_documents_app_cluster
    ON explore_documents(app, cluster_key);

CREATE INDEX idx_explore_documents_popular
    ON explore_documents(app, popular_score DESC, created_at DESC, cluster_key);

CREATE INDEX idx_explore_documents_rating
    ON explore_documents(app, community_rating DESC, own_rating DESC, created_at DESC, cluster_key);

CREATE INDEX idx_explore_values_text
    ON explore_values(app, record_type, field, value_text);

CREATE INDEX idx_explore_values_num
    ON explore_values(app, record_type, field, value_num);

CREATE INDEX idx_explore_values_uri
    ON explore_values(uri);
```

SQLite JSONB is not required for V1. It may be a later performance
optimization, but the main win comes from a derived faceted index and targeted
indexes rather than repeatedly parsing JSON from `records.record` at query time.

## Explore registry

Create a separate `internal/explore` registry for Explore metadata. Do not add
this directly to `entities.Descriptor`; that descriptor already carries feed,
rendering, edit, model conversion, and reference-resolution concerns.

The Explore registry should define:

- which app and record types are explorable
- display labels
- JSON paths or extractor functions
- allowed filters
- filter kind, such as facet text, fixed enum, numeric min/max, boolean
- searchable text composition
- first-hop reference extractors
- sort/stat extraction behavior where needed

Filter definitions should be constrained and explicit. This keeps SQL
generation safe and gives a clear seam for future changes such as text-contains
filters, author pickers, FTS ranking, or Oolong configuration.

V1 first-hop reference use should stay narrow:

- bean → roaster name
- recipe → brewer type/name where available

Reference-derived Explore fields are allowed to be eventually consistent in V1.
On record upsert, index the record being written. If the written record is a
referenced roaster or brewer, reindex direct dependents when this can be done
cheaply with the existing SQLite witness rows; otherwise mark Explore dirty and
let the next rebuild repair derived values. Do not add broad recursive graph
refresh for V1.

Avoid brew-derived first-hop search in V1 because brews are excluded from
Explore results. Brew data may still feed community rating and related-count
stats where existing helper queries make that practical, but any brew-derived
stat must document its refresh trigger.

Add comments documenting that V1 is not arbitrary graph search. Structured
search is intentionally limited to direct fields and selected first-hop
references.

## Index lifecycle

Explore indexing lives in/next to `FeedIndex`, preferably in a separate file
such as `internal/firehose/explore_index.go` to keep feed query code separate.

### Rebuilds

The Explore index should be versioned. Store an index version in `meta`, for
example `explore_index_version`.

On startup or migration:

- if the stored version differs from the current version, clear and rebuild
  `explore_documents` and `explore_values` from `records`
- if rebuild succeeds, update the stored version and clear dirty state
- if rebuild fails, keep the app running and mark Explore dirty

This works because Explore is a cache derivative.

### Incremental maintenance

On `FeedIndex.UpsertRecord`:

1. Write/update `records` first.
2. Best-effort reindex that record into Explore in a separate small transaction.
3. If the record is a narrow first-hop dependency, such as a roaster referenced
   by beans or a brewer referenced by recipes, best-effort reindex direct
   dependents or mark Explore dirty.
4. If Explore indexing fails, keep the record write and mark Explore dirty.

On social index updates:

- `FeedIndex.UpsertLike` and `FeedIndex.DeleteLike` must refresh the target
  record's `like_count` and `popular_score`, or mark Explore dirty if the target
  cannot be refreshed.
- `FeedIndex.UpsertComment` and `FeedIndex.DeleteComment` must refresh the
  target record's `comment_count` and `popular_score`, or mark Explore dirty if
  the target cannot be refreshed.
- These updates should not require re-extracting all facet values. Prefer a
  small stats refresh helper for one subject URI.
- Tests must cover create/delete of likes and comments changing Explore counts
  and popular ordering.

On `FeedIndex.DeleteRecord`:

- rely on `ON DELETE CASCADE` from `records` to Explore tables
- keep explicit cleanup where it improves clarity, but tests should prove the
  cascade path works
- if a deleted record was a canonical sourceRef target, remaining cluster members
  must still be queryable and canonical selection should fall back to the
  best-scoring member

On `DeleteAllByDID`:

- deleting rows from `records` should cascade Explore rows for that DID
- deleting likes/comments by or targeting that DID should refresh affected
  Explore stats where practical, or mark Explore dirty
- tests should cover this because the witness database already uses foreign
  keys

Explore staleness should be rare and mostly caused by derived-index failures,
reference-dependent fields not being refreshed, social stat refresh failures,
failed rebuilds, or bugs. Serve possibly stale Explore results, but expose dirty
state in health/admin.

## Query behavior

Explore queries should operate over `explore_documents` and `explore_values`,
not directly over raw record JSON.

Core query steps:

1. Scope to app = Arabica.
2. Scope to explorable record types.
3. Apply type filter if present.
4. Apply `q` against curated `search_text`.
5. Apply facet/range filters by intersecting or joining matching
   `explore_values` rows.
6. Choose canonical rows with a partition/window query, one row per
   `cluster_key`.
7. Sort canonical rows by the requested sort tuple.
8. Fetch one page plus overfetch buffer for moderation filtering.
9. Apply the same moderation filter used by the feed.

The query implementation should keep SQL generation constrained to registry
fields. Filter names and sort names must be validated against the Explore
registry before they reach SQL fragments.

Facet counts should be shown in V1, but counts may be raw/unmoderated. Add a
TODO explaining that moderation-aware counts can be added later if mismatches
become visible.

### Pagination

Use cursor pagination, not offset pagination.

Use sort-specific cursor tuples:

- recent: `created_at|cluster_key|uri`
- popular: `popular_score|created_at|cluster_key|uri`
- rating: `rating_sort|created_at|cluster_key|uri`
- search/relevance: use recent fallback initially unless FTS ranking is added

The cursor should be stable across ties and clustered results. Include `uri` as
the final tie-breaker because the canonical row can change when the original
sourceRef target appears or disappears.

## Moderation

Visible result lists must reuse the existing feed moderation filter. This keeps
policy centralized and avoids requiring Explore reindexing when moderation state
changes.

Implement this either by routing Explore result items through the same
moderation filter interface used by the feed service, or by extracting the
shared filtering operation into a small reusable helper. Do not duplicate
blacklist/hidden-record lookup logic in the Explore handler.

Facet counts can be raw index counts in V1. This means a count may include a
hidden record or a blacklisted user’s record even though the result list filters
it out. Document this with a TODO near the count query implementation.

## Health/admin behavior

Expose enough state to debug Explore indexing:

- whether the Explore index is present/ready
- current/stored index version
- dirty flag
- last rebuild error, if practical
- document/value row counts, if practical

Dirty Explore should not make the app unhealthy by itself unless the broader
SQLite witness index is unavailable.

Authenticated access should be enforced in the `/explore` handler using the
same page-level pattern as settings/notifications: inspect the request context
with existing auth helpers and redirect unauthenticated users to the login/home
flow. Do not rely only on hiding the navigation link.

## Tests

Add focused tests for:

- schema creation and foreign-key cascade from `records`
- extracting documents and values for each Arabica V1 record type
- filtering by each configured facet/range kind
- curated `q` search text
- type filtering
- raw facet counts
- sourceRef cluster key behavior
- canonical selection fallback behavior
- popular score calculation
- social stat refresh after like/comment create/delete
- rating sort using community rating when available, own rating otherwise
- unrated records sorting after rated records
- delete behavior via `DeleteRecord` and `DeleteAllByDID`
- versioned rebuild from existing `records`
- dependency refresh or dirty marking when referenced roasters/brewers change
- dirty marking when Explore indexing fails, if failure injection is practical

Use the project’s existing test convention: testify/assert rather than manual
`if`/`t.Error` checks.

## Acceptance criteria

- `/explore` exists, requires authentication, and appears in authenticated nav.
- Arabica Explore configuration includes beans, roasters, grinders, brewers, and
  recipes; brews are excluded.
- `FeedIndex` owns generic derived `explore_documents` and `explore_values`
  tables.
- Explore index is versioned, rebuildable from `records`, and incrementally
  maintained after record upserts/deletes and like/comment updates.
- Deletes cascade from `records` into Explore tables and are covered by tests.
- Results are clustered by `sourceRef`/URI, one result per cluster.
- Canonical selection is deterministic and pagination runs over canonical
  clustered rows, not raw records.
- Filters include type, q, sort, and configured Arabica facets/ranges.
- Facet counts are displayed and documented as raw/unmoderated in V1.
- Result lists are moderation-filtered with the same filter used by the feed.
- Cursor pagination works for supported sort modes.
- Explore cards show reusable author identity and discovery-style metadata.
- Existing `/recipes` behavior remains unchanged.

## Future work

- Add Oolong Explore configuration.
- Consider public Explore once scraper/compute concerns are addressed.
- Add author filter.
- Add FTS-backed relevance ranking.
- Add a shared sourceRef/cluster table or service.
- Add preferred DID ownership for roaster onboarding and canonical records.
- Show other users’ versions within an entity’s sourceRef cluster on view pages.
- Show related brews on entity view pages.
- Merge or rebuild the current `/recipes` explore page on top of the new Explore
  backend.
