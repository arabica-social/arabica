# Osprey Rules Engine Evaluation

Evaluation of [Osprey](https://github.com/roostorg/osprey) (by ROOST /
internet.dev) as a potential replacement or complement to Arabica's moderation
system.

## What Is Osprey?

Osprey is a **real-time safety rules engine** for processing event streams and
making automated decisions about user behavior. Originally built at Discord,
open-sourced through ROOST (Robust Open Online Safety Tools). Tagline: "Automate
the obvious and investigate the ambiguous."

Adopted by **Bluesky, Discord, and Matrix.org**. Apache 2.0 licensed, reached
v1.0.1 (March 2026), actively maintained.

### Core Concepts

**SML (Some Madeup Language)** — A Python-subset DSL for writing rules:

```python
# Models: extract features from event JSON
UserId: Entity[str] = EntityJson(type='User', path='$.user.userId')
PostText: str = JsonData(path='$.text')

# Rules: boolean conditions
SpamLinkRule = Rule(
    when_all=[
        PostCount == 1,
        EmbedLink != None,
        ListLength(list=MentionIds) >= 1,
    ],
    description='First post with link embed',
)

# Effects: actions when rules match
WhenRules(
    rules_any=[SpamLinkRule],
    then=[
        DeclareVerdict(verdict='reject'),
        LabelAdd(entity=UserId, label='likely_spammer'),
    ],
)
```

**Labels** — Stateful tags on entities (users, IPs, etc.) that persist across
evaluations. Support expiry (`expires_after=TimeDelta(days=7)`). Enable stateful
rules like "if this user was flagged as a spammer last week, auto-reject."

**Entities** — Special features (UserID, IP, etc.) that can carry labels and
have effects applied to them.

**File Organization** — Rules compose across files via `Import()` and
conditional `Require(require_if=EventType == 'userPost')`.

### Architecture

Osprey is a **multi-service system**, not an embeddable library:

| Component | Language | Purpose |
|-----------|----------|---------|
| Worker | Python | Core rules engine, consumes Kafka events |
| Coordinator | Rust | Distributed deployment coordination (optional) |
| UI | TypeScript/React | Investigation dashboard, querying, labeling |
| UI API | Python/Flask | Backend for the UI, queries Druid |
| RPC | gRPC/Protobuf | Inter-service communication |

**Infrastructure requirements:**
- Kafka (KRaft mode) — event I/O
- PostgreSQL — labels, execution results
- Apache Druid — OLAP for UI queries
- MinIO — object storage for Druid
- Google Bigtable (optional) — labels at scale

**Data flow:**
1. Events arrive on Kafka input topic
2. Worker evaluates SML rules against events
3. Rules produce verdicts and effects (hide, label, reject)
4. Results dispatched to output sinks (Kafka, Labels, stdout)
5. Execution results flow to Druid for UI querying

### Plugin System

Python `pluggy`-based. Plugins can register:
- **UDFs** — Custom functions for SML rules
- **Output Sinks** — Custom result destinations
- **AST Validators** — Custom rule validation

## Current Arabica Moderation System

For comparison, here's what Arabica has today (~2,100 lines across 3 layers):

### Capabilities

| Feature | Implementation |
|---------|---------------|
| Role-based access | JSON config with admin/moderator roles, 8 granular permissions |
| Record hiding | Manual + automod (3 reports → auto-hide) |
| User blacklisting | Manual ban/unban by moderators |
| Reports | User submission with rate limiting (10/hr), duplicate detection |
| Automod | Threshold-based: 3 reports/URI or 5 reports/user → auto-hide |
| Audit log | All actions logged including automod flag |
| Admin dashboard | HTMX-powered with stats, reports, hidden records, blacklist |
| Feed filtering | Batch-loads hidden URIs for efficient filtering |

### Code Organization

```
internal/moderation/
  models.go          # Roles, permissions, data types
  service.go         # Thread-safe permission checks
  store.go           # 16-method store interface
internal/database/sqlitestore/
  moderation.go      # SQLite implementation
internal/handlers/
  admin.go           # Dashboard + mod actions (662 lines)
  report.go          # Report submission + automod (260 lines)
```

### What Works Well

- Optional design — gracefully degrades without config
- Thread-safe service with RWMutex
- Comprehensive audit trail
- CSRF protection on all mutations
- Efficient batch feed filtering
- Flexible role/permission model

### Pain Points

- Automod thresholds are hardcoded constants
- Permission checks are manual boilerplate in every handler
- Report enrichment makes unbatched PDS calls
- Config changes require server restart
- No rule composition or conditional logic beyond fixed thresholds

## Fit Assessment

### Where Osprey Shines vs Arabica's Needs

**Osprey's strengths:**
- Sophisticated rule composition (AND/OR, conditional loading, labels)
- Stateful rules across evaluations (labels with TTL)
- Investigation UI for T&S teams
- Built for high-throughput event streams
- Plugin system for custom detection logic

**What Arabica could use:**
- More flexible automod rules (not just hardcoded thresholds)
- Rule composition (e.g., "new user + link + mention → suspicious")
- Stateful tracking (e.g., "user had 3 reports dismissed this month")
- Easier rule iteration without code deploys

### Why It Doesn't Fit Today

| Concern | Detail |
|---------|--------|
| **Massive infrastructure overhead** | Kafka + Postgres + Druid + MinIO is orders of magnitude more infra than Arabica's SQLite-based stack. Arabica runs as a single Go binary. |
| **No Go SDK** | Osprey is Python-native. No Go client library exists. Integration would require Kafka as middleware or raw gRPC proto compilation. |
| **Scale mismatch** | Osprey was built for Discord-scale (millions of events/sec). Arabica is a small community app. The operational complexity is not justified. |
| **Python dependency** | Arabica is a pure Go project. Adding a Python rules engine (plus its infra) contradicts the project's preference for stdlib solutions and minimal dependencies. |
| **Overlapping concerns** | Osprey would replace ~2,100 lines of straightforward Go code with a multi-service deployment. The current system is well-structured and maintainable. |

### What Could Be Borrowed (Ideas, Not Code)

Even though deploying Osprey doesn't make sense, some of its concepts are worth
adopting in Arabica's existing moderation code:

1. **Configurable thresholds** — Move automod constants (3 reports/URI, 5
   reports/user) into the moderators JSON config so they're tunable without
   deploys.

2. **Labels / stateful tags** — Add a lightweight label system for users (e.g.,
   `new_account`, `warned`, `trusted`). Labels could influence automod behavior:
   a `warned` user might have a lower auto-hide threshold.

3. **Rule composition** — Express automod rules as config rather than code:
   ```json
   {
     "automod_rules": [
       {
         "name": "high_report_volume",
         "conditions": {"reports_on_uri": {"gte": 3}},
         "action": "hide_record"
       },
       {
         "name": "repeated_offender",
         "conditions": {"reports_on_user": {"gte": 5}, "user_label": "warned"},
         "action": "blacklist_user"
       }
     ]
   }
   ```

4. **Permission middleware** — Replace per-handler permission boilerplate with
   middleware that checks permissions based on route patterns.

5. **TTL-based labels** — Osprey's label expiry is useful for temporary states
   like "under review" or "rate-limited for 24h."

## Recommendation

**Don't integrate Osprey.** The infrastructure and language mismatch is too
large, and Arabica's moderation needs are well-served by its current
~2,100-line Go implementation.

**Do consider** extracting the best ideas (configurable thresholds, labels,
rule composition as config) into the existing system. This would address the
current pain points (hardcoded thresholds, no stateful tracking) without the
operational burden of a multi-service Python deployment.

If Arabica ever grows to need a dedicated T&S team with investigation tooling,
Osprey becomes worth revisiting — but that's a fundamentally different scale
than today.
