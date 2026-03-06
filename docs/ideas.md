# Arabica Feature Ideas

## 1. Recipes

A **recipe** is a reusable, shareable brew procedure — distinct from a brew log, which records a
specific cup. Recipes are the social object people want to discover and re-use.

### New Lexicon: `social.arabica.alpha.recipe`

| Field          | Type              | Description                                      |
| -------------- | ----------------- | ------------------------------------------------ |
| `title`        | string (required) | "My Hario Switch Recipe for Light Roasts"        |
| `description`  | string            | Longer notes, tips, rationale                    |
| `method`       | string            | V60, Aeropress, etc.                             |
| `temperature`  | int (tenths °C)   | Same encoding as brew                            |
| `coffeeAmount` | int (grams)       | Dose                                             |
| `waterAmount`  | int (grams)       | Total water                                      |
| `grindSize`    | string            | "medium-fine"                                    |
| `timeSeconds`  | int               | Total brew time                                  |
| `pours`        | array             | `[{waterAmount, timeSeconds}]` — pour schedule   |
| `tags`         | []string          | Optional: ["fruity", "light roast", "comp-prep"] |
| `beanNotes`    | string            | Optional: "works well with washed Ethiopians"    |
| `createdAt`    | datetime          |                                                  |

### Interaction Flows

- **Likes / Comments** — already works; just point the existing `like` and `comment` records at the
  recipe AT-URI.
- **"Try This"** — button on a recipe opens the brew form pre-populated with the recipe's
  parameters. The resulting brew record optionally stores `basedOn: {uri, cid}` referencing the
  recipe (strong ref).
- **"Save as Recipe"** — button on a completed brew detail page creates a recipe record from that
  brew's parameters (minus the specific bean).
- **"X people tried this"** — count brews across the index whose `basedOn` field references the
  recipe's AT-URI.

### Social Flywheel

Recipe → community tries it → brews reference it → author sees who tried it → ratings provide
feedback → better recipes emerge.

### Feed Integration

Recipes should appear in the community feed as a new record type. The feed already supports
filtering by type, so recipes slot in naturally.

---

## 2. Roaster Analytics

A **public analytics page** for each roaster, aggregated from community brews via the AT Protocol
ref chain: `brew.beanRef → bean.roasterRef → roaster`.

No schema changes required — the firehose index already stores all records and their refs. SQLite's
`json_extract()` lets us follow the ref chain in a single JOIN query.

### URL

`/roasters/{did}/{rkey}` → constructs `at://{did}/social.arabica.alpha.roaster/{rkey}`

### Analytics Data

| Metric              | Query approach                                          |
| ------------------- | ------------------------------------------------------- |
| Total brews         | COUNT brews whose bean references this roaster          |
| Total beans indexed | COUNT distinct beans referencing this roaster           |
| Active brewers      | COUNT DISTINCT brew.did                                 |
| Avg rating          | AVG(json_extract(brew.record, '$.rating'))              |
| Median rating       | Fetch all ratings, sort in Go, pick middle              |
| Top beans           | GROUP BY bean, AVG rating DESC                          |
| Brew method mix     | GROUP BY json_extract(brew.record, '$.method'), COUNT   |
| Rating by month     | GROUP BY strftime('%Y-%m', created_at), AVG rating      |

### Core SQL Pattern

```sql
SELECT
    bean.uri,
    json_extract(bean.record, '$.name')   AS bean_name,
    COUNT(brew.uri)                        AS brew_count,
    AVG(json_extract(brew.record, '$.rating')) AS avg_rating,
    COUNT(DISTINCT brew.did)               AS brewer_count
FROM records brew
JOIN records bean ON bean.uri = json_extract(brew.record, '$.beanRef')
WHERE brew.collection = 'social.arabica.alpha.brew'
  AND bean.collection = 'social.arabica.alpha.bean'
  AND json_extract(bean.record, '$.roasterRef') = ?   -- roaster AT-URI
GROUP BY bean.uri
ORDER BY avg_rating DESC
```

### Page Sections

1. **Header** — Roaster name, location, website, total brews / beans / brewers
2. **Rating summary** — Avg ★, median ★, total rated brews
3. **Top beans** — Table: bean name, brew count, avg rating
4. **Brew method breakdown** — Bar/list showing V60, Aeropress, etc.
5. **Rating trend** — Month-by-month avg rating (simple list or sparkline)

### Linking

Anywhere a roaster name appears in the feed or on a brew/bean detail page, link to
`/roasters/{did}/{rkey}`.

### Public Access

The analytics page is fully public (no auth required) since all data is already public in the
firehose index.

---

## 3. Personal Analytics Dashboard

A private `/me/stats` page showing the authenticated user's own brewing trends, computed from
their PDS records (no cross-user aggregation needed).

### Metrics

| Metric               | Source                                    |
| -------------------- | ----------------------------------------- |
| Brews per week/month | Count brews by created_at bucket          |
| Avg rating over time | Avg rating by month                       |
| Favourite bean       | Most brewed bean (+ highest avg rating)   |
| Favourite method     | Most used brew method                     |
| Equipment usage      | Most used grinder / brewer                |
| Taste evolution      | Rating trend over time                    |
| Bags opened/closed   | Count beans by `closed` flag              |

### Implementation Notes

- Query user's own PDS via `store.ListBrews()` — no firehose needed.
- Aggregate in Go (small data set per user, no need for SQL aggregation).
- Cache in `SessionCache` to avoid repeated PDS fetches.
- No new lexicons required.

---

## Priority

1. **Roaster analytics** — immediate value, no schema changes, pure SQL over existing indexed refs
2. **Recipes** — high social value, new lexicon + feed integration required
3. **Personal stats** — lower complexity, pure client-side aggregation, quality-of-life feature
