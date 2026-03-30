# Moderation Improvements: Rules as Config, Labels, Permission Middleware

Exploration of three Osprey-inspired ideas adapted for Arabica's single-binary
Go architecture. No new infrastructure required — these build on the existing
SQLite + JSON config system.

## 1. Rules as Config

### Problem

Automod thresholds are hardcoded constants in `internal/handlers/report.go`:

```go
const (
    AutoHideThreshold      = 3   // reports on single record
    AutoHideUserThreshold  = 5   // total reports on user's content
    ReportRateLimitPerHour = 10
    MaxReportReasonLength  = 500
)
```

Changing these requires a code change and redeploy. There's also no way to
express more nuanced rules like "auto-hide from new users at a lower threshold"
or "auto-blacklist after N auto-hides."

### Proposal

Add an `automod` section to the existing moderators JSON config:

```json
{
  "roles": { ... },
  "users": [ ... ],
  "automod": {
    "rules": [
      {
        "name": "high_report_uri",
        "description": "Auto-hide records with 3+ reports",
        "trigger": "report_created",
        "conditions": {
          "reports_on_uri": { "gte": 3 }
        },
        "action": "hide_record"
      },
      {
        "name": "high_report_user",
        "description": "Auto-hide when user has 5+ reported records",
        "trigger": "report_created",
        "conditions": {
          "reports_on_user": { "gte": 5 }
        },
        "action": "hide_record"
      },
      {
        "name": "repeat_offender",
        "description": "Blacklist users auto-hidden 3+ times",
        "trigger": "record_auto_hidden",
        "conditions": {
          "auto_hides_on_user": { "gte": 3 }
        },
        "action": "blacklist_user"
      }
    ],
    "rate_limit_per_hour": 10,
    "max_reason_length": 500
  }
}
```

### Condition Types

Each condition maps to an existing store query or a simple new one:

| Condition | Store Method | Exists? |
|-----------|-------------|---------|
| `reports_on_uri` | `CountReportsForURI()` | Yes |
| `reports_on_user` | `CountReportsForDID()` / `CountReportsForDIDSince()` | Yes |
| `auto_hides_on_user` | Count hidden records where `subject_did = ?` | New query |
| `has_label` | Label lookup (see section 2) | New |
| `account_age_days` | Would need PDS profile data | Deferred |

### Trigger Types

| Trigger | When Evaluated |
|---------|---------------|
| `report_created` | After `CreateReport()` succeeds (replaces current `checkAutomod`) |
| `record_auto_hidden` | After an auto-hide action completes (enables chained rules) |

### Action Types

| Action | Implementation |
|--------|---------------|
| `hide_record` | Existing `HideRecord()` with `AutoHidden: true` |
| `blacklist_user` | Existing `BlacklistUser()` with `BlacklistedBy: "automod"` |
| `add_label` | New — adds a label to user/record (see section 2) |
| `remove_label` | New — removes a label |

### Go Types

```go
// In internal/moderation/models.go

type AutomodConfig struct {
    Rules            []AutomodRule `json:"rules"`
    RateLimitPerHour int           `json:"rate_limit_per_hour"`
    MaxReasonLength  int           `json:"max_reason_length"`
}

type AutomodRule struct {
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Trigger     AutomodTrigger    `json:"trigger"`
    Conditions  []AutomodCondition `json:"conditions"`
    Action      AutomodAction     `json:"action"`
}

type AutomodTrigger string

const (
    TriggerReportCreated    AutomodTrigger = "report_created"
    TriggerRecordAutoHidden AutomodTrigger = "record_auto_hidden"
)

type AutomodCondition struct {
    Type      string `json:"type"`       // "reports_on_uri", "reports_on_user", etc.
    Operator  string `json:"operator"`   // "gte", "eq", "lte"
    Threshold int    `json:"threshold"`
    Label     string `json:"label,omitempty"` // For has_label condition
}

type AutomodAction struct {
    Type  string `json:"type"`            // "hide_record", "blacklist_user", "add_label"
    Label string `json:"label,omitempty"` // For add_label/remove_label actions
}
```

### Evaluation Engine

Replace `checkAutomod()` with a rule evaluator:

```go
// In internal/moderation/automod.go

type RuleEvaluator struct {
    rules []AutomodRule
    store Store
}

func (e *RuleEvaluator) Evaluate(ctx context.Context, trigger AutomodTrigger, event AutomodEvent) []AutomodResult {
    var results []AutomodResult
    for _, rule := range e.rules {
        if rule.Trigger != trigger {
            continue
        }
        if e.allConditionsMet(ctx, rule.Conditions, event) {
            results = append(results, AutomodResult{
                Rule:   rule,
                Action: rule.Action,
            })
        }
    }
    return results
}
```

The handler calls the evaluator instead of checking hardcoded thresholds.
Results chain — if a `hide_record` action fires, it triggers
`record_auto_hidden` rules in the same pass.

### Migration Path

1. Add `AutomodConfig` to `Config` struct with defaults matching current
   constants
2. Move `checkAutomod()` logic into `RuleEvaluator`
3. If no `automod` section in config, use built-in defaults (backward
   compatible)
4. Delete the hardcoded constants

---

## 2. Labels

### Problem

Automod is stateless — it only counts reports at decision time. There's no way
to say "this user was warned" or "this account is new and should have stricter
thresholds." Moderators also can't tag users with notes that affect future
automod behavior.

### Proposal

Add a lightweight label system for entities (users and records). Labels are
key-value tags with optional TTL, stored in SQLite alongside the existing
moderation tables.

### Schema

```sql
CREATE TABLE moderation_labels (
    id          TEXT PRIMARY KEY,       -- TID
    entity_type TEXT NOT NULL,          -- 'user' or 'record'
    entity_id   TEXT NOT NULL,          -- DID or AT-URI
    label       TEXT NOT NULL,          -- e.g. 'warned', 'trusted', 'new_account'
    value       TEXT DEFAULT '',        -- optional value
    created_at  TEXT NOT NULL,          -- RFC3339Nano
    created_by  TEXT NOT NULL,          -- DID or 'automod' or 'system'
    expires_at  TEXT,                   -- RFC3339Nano, NULL = permanent
    UNIQUE(entity_type, entity_id, label)
);

CREATE INDEX idx_labels_entity ON moderation_labels(entity_type, entity_id);
CREATE INDEX idx_labels_expires ON moderation_labels(expires_at) WHERE expires_at IS NOT NULL;
```

### Store Interface Additions

```go
// Added to moderation.Store interface

// Labels
AddLabel(ctx context.Context, label Label) error
RemoveLabel(ctx context.Context, entityType, entityID, labelName string) error
HasLabel(ctx context.Context, entityType, entityID, labelName string) (bool, error)
GetLabel(ctx context.Context, entityType, entityID, labelName string) (*Label, error)
ListLabels(ctx context.Context, entityType, entityID string) ([]Label, error)
CleanExpiredLabels(ctx context.Context) (int, error)  // Periodic cleanup
```

### Label Model

```go
type Label struct {
    ID         string     `json:"id"`
    EntityType string     `json:"entity_type"` // "user" or "record"
    EntityID   string     `json:"entity_id"`   // DID or AT-URI
    Name       string     `json:"label"`
    Value      string     `json:"value,omitempty"`
    CreatedAt  time.Time  `json:"created_at"`
    CreatedBy  string     `json:"created_by"`
    ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

func (l *Label) IsExpired() bool {
    return l.ExpiresAt != nil && time.Now().After(*l.ExpiresAt)
}
```

### Predefined Labels

These would be documented but not hardcoded — any string works as a label name:

| Label | Entity | Meaning | Typical TTL |
|-------|--------|---------|-------------|
| `warned` | user | Moderator issued a warning | 30 days |
| `trusted` | user | Exempt from automod | permanent |
| `under_review` | user | Flagged for manual review | 7 days |
| `rate_limited` | user | Temporarily restricted | 24 hours |
| `spam` | record | Identified as spam | permanent |

### Integration with Automod Rules

Labels become a condition type in automod rules:

```json
{
  "name": "strict_threshold_warned_users",
  "trigger": "report_created",
  "conditions": {
    "reports_on_uri": { "gte": 2 },
    "has_label": { "entity": "subject_user", "label": "warned" }
  },
  "action": "hide_record"
}
```

And an action type:

```json
{
  "name": "warn_on_first_autohide",
  "trigger": "record_auto_hidden",
  "conditions": {
    "auto_hides_on_user": { "gte": 1 },
    "not_has_label": { "entity": "subject_user", "label": "warned" }
  },
  "action": { "type": "add_label", "label": "warned", "expires": "30d" }
}
```

### UI Integration

- Admin dashboard shows labels on users/records
- Moderators with `manage_labels` permission can add/remove labels manually
- Label badges appear on reported content for context

### Cleanup

A goroutine runs `CleanExpiredLabels()` periodically (e.g., every hour). This
is a single DELETE query:

```sql
DELETE FROM moderation_labels WHERE expires_at IS NOT NULL AND expires_at < ?
```

### Relationship to AT Protocol Labels

AT Protocol has its own label system (`com.atproto.label`) designed for
**federated, decentralized moderation**. It's worth understanding how it differs
from the internal labels proposed above, whether Arabica should use both, and
when each applies.

#### How atproto labels work

atproto labels are cryptographically signed metadata tags produced by independent
**labeler services** — third-party identities (with their own DID and signing
key) that publish labels about accounts or records across the network. Key
properties:

- **Signed**: Each label is CBOR-encoded and signed with the labeler's
  `#atproto_label` key. Clients verify authenticity without trusting the
  transport.
- **User-choosable**: Users subscribe to labelers they trust (up to 20). Each
  user independently decides how to handle each label value: hide, warn, or
  ignore. This is fundamentally different from moderator-imposed decisions.
- **Distributed**: Labels are broadcast via
  `com.atproto.label.subscribeLabels` (WebSocket stream) and queryable via
  `com.atproto.label.queryLabels`. AppViews hydrate labels into API responses
  based on the client's `atproto-accept-labelers` header.
- **Graduated actions**: Label values define `severity` (inform/alert/none) and
  `blurs` (content/media/none). The `!hide` and `!warn` system labels are
  non-overridable; content labels like `porn`, `graphic-media` let users choose.
- **Negatable and expirable**: A label is retracted by publishing a negation
  (`neg: true`) with the same src/uri/val. Labels can also carry `exp` for
  automatic expiry.
- **Self-labels**: Record authors can self-apply global label values (e.g.,
  `porn`, `nudity`, `graphic-media`) directly in their records via
  `com.atproto.label.defs#selfLabels`.

The indigo SDK already provides the Go types (`LabelDefs_Label`,
`LabelDefs_SelfLabels`) and signing/verification utilities.

#### How internal labels differ

The labels proposed in this doc are **app-internal operational state** — they
exist only inside Arabica's SQLite database, are not signed or distributed, and
are invisible to the broader AT Protocol network. They serve a fundamentally
different purpose:

| Aspect | Internal Labels | AT Protocol Labels |
|--------|----------------|--------------------|
| **Purpose** | Automod state, moderator notes | Network-wide content classification |
| **Audience** | Arabica's automod engine + moderators | Any atproto client, user-selectable |
| **Storage** | SQLite row | Signed CBOR object, distributed via WebSocket |
| **Who creates** | Automod rules or Arabica moderators | Independent labeler services |
| **Who consumes** | Arabica's rule evaluator | End users via their chosen client |
| **Visibility** | Admin dashboard only | Public (any subscriber can see) |
| **User choice** | None — moderator decisions are authoritative | Users choose labelers and per-label behavior |
| **Examples** | `warned`, `trusted`, `under_review`, `rate_limited` | `porn`, `graphic-media`, `!hide`, `spam` |

These are complementary, not competing. Internal labels are private moderator
bookkeeping. atproto labels are public content classification signals.

#### Should Arabica use both?

**Yes** — but for different things, and not at the same time.

**Phase 1: Internal labels (this proposal)**

Internal labels are the right tool for what's described in this doc: giving
automod memory and letting moderators annotate users. A `warned` label that
expires in 30 days, or a `trusted` label that exempts someone from automod —
these are operational concerns that don't belong on the public network. They're
the equivalent of internal moderation notes, not public content ratings.

Ship internal labels first. They have zero infrastructure requirements, integrate
directly with the automod rule evaluator, and solve the immediate problem of
stateless automod.

**Phase 2: Arabica as an atproto labeler (future)**

Later, Arabica could operate as a labeler service — publishing labels that other
atproto clients can subscribe to. This would be valuable for:

- **Content warnings on brew records**: Labeling brews that contain
  controversial ingredients or methods (e.g., if the community ever needs
  content classification beyond coffee)
- **Quality signals**: A `verified-roaster` or `featured` label that other
  clients could consume to highlight trusted content
- **Spam classification**: Publishing `spam` labels on records Arabica's automod
  has flagged, so other apps indexing `social.arabica.alpha.*` collections can
  benefit from Arabica's moderation work
- **Cross-app moderation**: If other apps build on the arabica lexicons, they
  could subscribe to Arabica's labeler to get moderation decisions without
  running their own moderation

This would require:
1. A signing key (`#atproto_label`) in Arabica's DID document
2. An `app.bsky.labeler.service` declaration record
3. `subscribeLabels` and `queryLabels` endpoints
4. A mapping from internal moderation actions → published atproto labels

#### When an action should become a public label

Not every internal label should be published externally. A rough heuristic:

| Internal action | Publish as atproto label? | Why / why not |
|-----------------|--------------------------|---------------|
| `hide_record` | Yes → `!hide` | Other apps indexing arabica records should respect this |
| `blacklist_user` | Maybe → custom `blocked` | Depends on whether other apps need to know |
| `warned` | No | Private moderator state, not relevant to other clients |
| `trusted` | No | Internal automod bypass, meaningless externally |
| `spam` | Yes → `spam` | Useful for any app consuming arabica records |
| `under_review` | No | Temporary internal state |

The bridge between the two systems would be straightforward: when an internal
moderation action fires that warrants a public label, also sign and publish an
atproto label. The internal label drives automod behavior; the atproto label
communicates the decision to the network.

#### What this means for the current proposal

No changes needed to the internal labels design. The schema, store interface, and
automod integration all remain as specified above. The only future consideration
is adding an optional `publish_label` action type to the automod rules config
(Phase 3), which would sign and emit an atproto label alongside the internal
action:

```json
{
  "name": "publish_spam_label",
  "trigger": "record_auto_hidden",
  "conditions": {
    "has_label": { "entity": "subject_record", "label": "spam" }
  },
  "action": { "type": "publish_label", "val": "spam" }
}
```

This is purely additive and doesn't affect the Phase 2/3 work described in this
doc.

---

## 3. Permission Middleware

### Problem

Every moderation handler repeats the same auth + permission check boilerplate:

```go
userDID, err := atproto.GetAuthenticatedDID(r.Context())
if err != nil || userDID == "" {
    http.Error(w, "Authentication required", http.StatusUnauthorized)
    return
}
if h.moderationService == nil || !h.moderationService.HasPermission(userDID, moderation.PermissionXXX) {
    log.Warn().Str("did", userDID).Msg("Denied: insufficient permissions")
    http.Error(w, "Permission denied", http.StatusForbidden)
    return
}
```

This is ~8 lines repeated in 8 handlers. Changes to auth behavior require
touching every handler.

### Proposal

A `RequirePermission` middleware that wraps handlers with permission checks:

```go
// In internal/middleware/moderation.go

func RequirePermission(
    modService *moderation.Service,
    perm moderation.Permission,
    next http.Handler,
) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userDID := atproto.GetAuthenticatedDIDFromContext(r.Context())
        if userDID == "" {
            http.Error(w, "Authentication required", http.StatusUnauthorized)
            return
        }

        if modService == nil || !modService.HasPermission(userDID, perm) {
            log.Warn().
                Str("did", userDID).
                Str("permission", string(perm)).
                Str("path", r.URL.Path).
                Msg("permission denied")
            http.Error(w, "Permission denied", http.StatusForbidden)
            return
        }

        next.ServeHTTP(w, r)
    })
}

func RequireModerator(
    modService *moderation.Service,
    next http.Handler,
) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        userDID := atproto.GetAuthenticatedDIDFromContext(r.Context())
        if userDID == "" {
            http.Error(w, "Authentication required", http.StatusUnauthorized)
            return
        }

        if modService == nil || !modService.IsModerator(userDID) {
            http.Error(w, "Access denied", http.StatusForbidden)
            return
        }

        next.ServeHTTP(w, r)
    })
}
```

### Routing Changes

Before:
```go
mux.Handle("POST /_mod/hide", cop.Handler(http.HandlerFunc(h.HandleHideRecord)))
mux.Handle("POST /_mod/unhide", cop.Handler(http.HandlerFunc(h.HandleUnhideRecord)))
mux.Handle("POST /_mod/block", cop.Handler(http.HandlerFunc(h.HandleBlockUser)))
// ... each handler checks permissions internally
```

After:
```go
modPerm := middleware.RequirePermission  // alias for readability

mux.Handle("POST /_mod/hide",
    cop.Handler(modPerm(modSvc, moderation.PermissionHideRecord,
        http.HandlerFunc(h.HandleHideRecord))))

mux.Handle("POST /_mod/unhide",
    cop.Handler(modPerm(modSvc, moderation.PermissionUnhideRecord,
        http.HandlerFunc(h.HandleUnhideRecord))))

mux.Handle("POST /_mod/block",
    cop.Handler(modPerm(modSvc, moderation.PermissionBlacklistUser,
        http.HandlerFunc(h.HandleBlockUser))))
```

### What Handlers Lose

Each handler drops the first ~8 lines of boilerplate. They still need the
authenticated DID for audit logging, but can get it from context (it's already
validated by the middleware):

```go
func (h *Handler) HandleHideRecord(w http.ResponseWriter, r *http.Request) {
    // DID is guaranteed valid by middleware
    userDID := atproto.GetAuthenticatedDIDFromContext(r.Context())

    // Straight to business logic
    if err := r.ParseForm(); err != nil {
        http.Error(w, "Invalid request body", http.StatusBadRequest)
        return
    }
    // ...
}
```

### Edge Case: Admin Dashboard

`HandleAdmin` uses `IsModerator()` (not a specific permission) and then
conditionally loads data based on individual permissions. This stays as-is — the
middleware handles the gate, the handler still calls `HasPermission()` for
conditional data loading:

```go
mux.HandleFunc("GET /_mod",
    middleware.RequireModerator(modSvc, http.HandlerFunc(h.HandleAdmin)))

// Inside HandleAdmin, permission checks remain for conditional data:
canHide := h.moderationService.HasPermission(userDID, moderation.PermissionHideRecord)
```

---

## Implementation Order

These three features have natural dependencies:

```
Phase 1: Permission Middleware
  └─ Standalone refactor, no new features
  └─ Reduces boilerplate, makes Phase 2/3 cleaner
  └─ ~1 new file, ~8 handler edits, routing changes

Phase 2: Labels
  └─ New table, store methods, model types
  └─ Admin UI for viewing/managing labels
  └─ ~3 new files, store interface additions

Phase 3: Rules as Config
  └─ Depends on Labels for `has_label` condition
  └─ Config schema additions, rule evaluator
  └─ Replace checkAutomod() with evaluator
  └─ ~2 new files, config changes, handler changes
```

Each phase is independently useful and shippable. Phase 1 is pure cleanup.
Phase 2 gives moderators new capabilities. Phase 3 makes automod flexible.

## What This Doesn't Do

- **No new infrastructure** — everything stays in SQLite + JSON config
- **No DSL** — rules are JSON, not a custom language
- **No real-time streaming** — evaluation happens synchronously in handlers
- **No investigation UI** — the existing admin dashboard gets labels, not a
  query engine
- **No ML/AI integration** — rules are deterministic threshold checks
- **No atproto labeler service** — internal labels are private operational
  state, not published to the network. Becoming an atproto labeler is a natural
  future extension (see "Relationship to AT Protocol Labels" in section 2) but
  is out of scope for this work.

These are deliberate constraints. If the moderation needs outgrow this, that's
the point where Osprey (or a similar system) becomes worth the infrastructure
cost. The atproto labeler path is a lighter lift than Osprey — it reuses the
existing moderation decisions and just adds a signed publication layer — so it's
a reasonable intermediate step if cross-app moderation becomes needed before
full-scale infrastructure is justified.
