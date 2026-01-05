# Arabica AT Protocol Migration Plan

## Overview

This document outlines the plan to transform Arabica from a self-hosted SQLite-based coffee tracking application into a decentralized AT Protocol (atproto) application where user data lives in their Personal Data Servers (PDS).

## Project Goals

### Core Principles

- **Decentralized Storage**: User data lives in their own PDS, not our server
- **Public Records**: All coffee tracking records are public via atproto repo exploration
- **Self-hostable AppView**: Our server is the primary AppView, but others can self-host
- **OAuth Authentication**: Users authenticate via atproto OAuth with scopes (not app passwords)
- **Backward Compatible UX**: Preserve the user-friendly interface and workflow

### Architecture Vision

```
┌─────────────────────────────────────────────────────────────┐
│                  Arabica AppView (Go Server)                │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐   │
│  │   Web UI     │  │   OAuth      │  │   atproto        │   │
│  │ (HTMX/HTML)  │  │   (indigo)   │  │   Client         │   │
│  └──────────────┘  └──────────────┘  └──────────────────┘   │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐   │
│  │     Future: AppView Index & Discovery                │   │
│  │     - Firehose subscriber                            │   │
│  │     - Cross-user search & discovery                  │   │
│  │     - Social features                                │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                          │
                          ↓ OAuth + XRPC
              ┌───────────────────────────┐
              │   User's PDS (any PDS)    │
              │   at://did:plc:abc123/    │
              │   ├── com.arabica.brew    │
              │   ├── com.arabica.bean    │
              │   ├── com.arabica.roaster │
              │   ├── com.arabica.grinder │
              │   └── com.arabica.brewer  │
              └───────────────────────────┘
```

---

## Phase 0: Research & Validation (2-3 days)

### Goals

- Understand `indigo` SDK capabilities
- Validate lexicon schemas
- Set up development environment
- Test basic atproto operations

### Tasks

#### 0.1: Research `indigo` SDK

- [ ] Review `github.com/bluesky-social/indigo` documentation
- [ ] Study OAuth implementation in indigo (with scopes)
- [ ] Understand XRPC client usage
- [ ] Review record CRUD operations API
- [ ] Find example applications using indigo
- [ ] Document findings in `docs/indigo-research.md`

**Key Questions:**

- Does indigo provide complete OAuth client with DPOP support?
- What's the API for repo operations (createRecord, getRecord, listRecords, etc.)?
- Are there session management helpers?
- How are AT-URIs parsed and resolved?

#### 0.2: Design & Validate Lexicons

- [ ] Write lexicon JSON files for all record types:
  - `com.arabica.bean`
  - `com.arabica.roaster`
  - `com.arabica.grinder`
  - `com.arabica.brewer`
  - `com.arabica.brew`
- [ ] Validate lexicons against atproto schema validator
- [ ] Document schema decisions in `docs/schema-design.md`
- [ ] Review field mappings from current SQLite schema

**Key Decisions:**

- Reference format: AT-URIs vs embedded objects
- Pours: Embedded in brew record vs separate collection
- Enums vs free text for fields like `method`, `roastLevel`, etc.
- String length limits for all text fields

#### 0.3: Set up Development Environment

- [ ] Create/use Bluesky account for testing (or run local PDS)
- [ ] Install atproto development tools (CLI, validators)
- [ ] Set up test DID for development
- [ ] Configure environment variables for PDS endpoints

**Options:**

- **Option A:** Use Bluesky PDS (bsky.social) - easiest, most realistic
- **Option B:** Run local PDS via Docker - more control, offline dev
- **Recommended:** Option A for initial development

#### 0.4: Manual Record Creation Test

- [ ] Manually create test records using atproto CLI or tools
- [ ] Test all 5 record types (bean, roaster, grinder, brewer, brew)
- [ ] Test cross-references (brew → bean → roaster)
- [ ] Verify records appear in repo explorer
- [ ] Query records back via API
- [ ] Document any issues or schema adjustments needed

**Deliverables:**

- Working lexicon files in `lexicons/` directory
- Test records in development PDS
- Research notes documenting indigo SDK usage

---

## Phase 1: Core atproto Client Layer (3-4 days)

### Goals

- Integrate `indigo` SDK
- Create atproto client wrapper
- Implement new Store interface using PDS operations
- Replace SQLite dependency with atproto records

### Tasks

#### 1.1: Project Structure Setup

- [ ] Add `github.com/bluesky-social/indigo` to go.mod
- [ ] Create new package structure:
  ```
  internal/
  ├── atproto/
  │   ├── client.go      # XRPC client wrapper
  │   ├── records.go     # Record CRUD operations
  │   ├── resolver.go    # AT-URI and reference resolution
  │   └── store.go       # Store interface implementation
  ```
- [ ] Keep `internal/database/store.go` interface (for compatibility)
- [ ] Move lexicon files to `lexicons/` at project root

#### 1.2: atproto Client Wrapper (`internal/atproto/client.go`)

Create a wrapper around indigo's XRPC client with high-level operations:

```go
type Client struct {
    xrpc     *xrpc.Client
    pdsURL   string
}

func NewClient(pdsURL string) (*Client, error)
```

**Core Methods:**

- [ ] `CreateRecord(did, collection, record)` → (rkey, error)
- [ ] `GetRecord(did, collection, rkey)` → (record, error)
- [ ] `ListRecords(did, collection, limit, cursor)` → (records, cursor, error)
- [ ] `PutRecord(did, collection, rkey, record)` → error
- [ ] `DeleteRecord(did, collection, rkey)` → error

**Helper Methods:**

- [ ] `ResolveATURI(uri string)` → (did, collection, rkey, error)
- [ ] `BuildATURI(did, collection, rkey)` → string
- [ ] Error handling and retry logic

#### 1.3: Record Type Mapping (`internal/atproto/records.go`)

Map Go structs to/from atproto records:

```go
// Convert domain models to atproto records
func BrewToRecord(brew *models.Brew) (map[string]interface{}, error)
func RecordToBrew(record map[string]interface{}) (*models.Brew, error)

// Similar for Bean, Roaster, Grinder, Brewer
```

**Considerations:**

- [ ] Handle time.Time → RFC3339 string conversion
- [ ] Build AT-URI references for foreign keys
- [ ] Validate required fields before sending to PDS
- [ ] Handle optional fields (nil pointers)

#### 1.4: Reference Resolution (`internal/atproto/resolver.go`)

Handle fetching referenced records:

```go
// Resolve a single reference
func (c *Client) ResolveReference(atURI string, target interface{}) error

// Batch resolve multiple references (optimization for later)
func (c *Client) BatchResolve(atURIs []string) (map[string]interface{}, error)
```

**Strategy (Phase 1):**

- Start with simple lazy loading (one request per reference)
- Document optimization opportunities for later

#### 1.5: Store Implementation (`internal/atproto/store.go`)

Implement the existing `database.Store` interface using atproto operations:

```go
type AtprotoStore struct {
    client *Client
    did    string  // Current authenticated user's DID
}

func NewAtprotoStore(client *Client, did string) *AtprotoStore
```

**Implement all Store methods:**

- [ ] Brew operations (CreateBrew, GetBrew, ListBrews, UpdateBrew, DeleteBrew)
- [ ] Bean operations
- [ ] Roaster operations
- [ ] Grinder operations
- [ ] Brewer operations
- [ ] Pour operations (embedded in brews, special handling)

**Key Changes from SQLite:**

- No user_id field (implicit from DID)
- References stored as AT-URIs
- Need to resolve references when displaying
- List operations may need pagination handling

#### 1.6: Testing

- [ ] Unit tests for client operations (against test PDS)
- [ ] Test record conversion functions
- [ ] Test reference resolution
- [ ] Test Store interface implementation
- [ ] Integration test: full CRUD cycle for each record type

**Deliverables:**

- Working atproto client layer
- Store interface implemented via PDS
- SQLite can be completely removed (but keep for sessions - see Phase 2)

---

## Phase 2: Authentication & OAuth (3-4 days)

### Goals

- Implement atproto OAuth flow using indigo (with scopes, not app passwords)
- Add session management
- Add authentication middleware
- Update UI for login/logout

### Tasks

#### 2.1: OAuth Server Setup (`internal/atproto/oauth.go`)

Implement OAuth using **indigo's OAuth client** (preferred approach):

**IMPORTANT:** Use OAuth scopes, NOT app passwords. This provides proper security and user control.

```go
type OAuthHandler struct {
    // Use indigo's OAuth client
    oauthClient  *indigo_oauth.Client
    clientID     string
    redirectURI  string
    scopes       []string  // Standard: ["atproto", "transition:generic"]
}
```

**OAuth Client Metadata (Required for Registration):**
You'll need to register your OAuth client with atproto. Required metadata:

- `client_id`: Your application identifier (e.g., `https://arabica.example.com/client-metadata.json`)
- `client_name`: "Arabica Coffee Tracker"
- `client_uri`: Your app homepage URL
- `redirect_uris`: Array of callback URLs (e.g., `["https://arabica.example.com/oauth/callback"]`)
- `scope`: Space-separated scopes (e.g., `"atproto transition:generic"`)
- `grant_types`: `["authorization_code", "refresh_token"]`
- `response_types`: `["code"]`
- `token_endpoint_auth_method`: `"none"` (for public clients)
- `application_type`: `"web"`
- `dpop_bound_access_tokens`: `true` (REQUIRED - enables DPOP)

**Client Metadata Hosting:**

- [ ] Serve client metadata at `/.well-known/oauth-client-metadata` or at your `client_id` URL
- [ ] Must be publicly accessible HTTPS endpoint
- [ ] Content-Type: `application/json`

**Required Endpoints:**

- [ ] `GET /login` - Initiate OAuth flow (handle → PDS discovery → auth endpoint)
- [ ] `GET /oauth/callback` - Handle OAuth callback with authorization code
- [ ] `POST /logout` - Clear session and revoke tokens

**OAuth Flow with indigo:**

1. User enters their handle (e.g., `alice.bsky.social`)
2. Resolve handle → DID → PDS URL
3. Discover PDS OAuth endpoints (authorization_endpoint, token_endpoint)
4. Generate PKCE challenge and state
5. Build authorization URL with:
   - `client_id`
   - `redirect_uri`
   - `scope` (e.g., `"atproto transition:generic"`)
   - `response_type=code`
   - `code_challenge` and `code_challenge_method=S256` (PKCE)
   - `state` (CSRF protection)
6. Redirect user to PDS authorization endpoint
7. User authorizes on their PDS
8. PDS redirects back with authorization code
9. Exchange code for tokens using **DPOP**:
   - Generate DPOP proof JWT
   - POST to token_endpoint with code, PKCE verifier, and DPOP proof
   - Receive access_token, refresh_token (both DPOP-bound)
10. Store session with DID, tokens, and DPOP key

**Key Implementation Details:**

- [ ] Use indigo's OAuth client library (handles DPOP automatically)
- [ ] Generate and store DPOP keypairs per session
- [ ] All PDS API calls must include DPOP proof header
- [ ] Handle token refresh (also requires DPOP)
- [ ] Support multiple PDS providers (not just Bluesky)
- [ ] Handle resolution and DID validation
- [ ] PKCE for additional security

**indigo OAuth Components to Use:**

- Handle → DID resolution
- PDS → OAuth endpoint discovery
- DPOP key generation and proof creation
- Token exchange with DPOP
- Token refresh with DPOP

**Security Considerations:**

- [ ] Validate `state` parameter to prevent CSRF
- [ ] Verify PKCE code_verifier matches challenge
- [ ] Store DPOP private keys securely (encrypted in session)
- [ ] Use HTTP-only, secure cookies for session ID
- [ ] Implement token expiration checking
- [ ] Revoke tokens on logout

#### 2.2: Session Management

Store authenticated sessions with user DID and tokens.

**Options:**

- **Development:** In-memory map (simple, ephemeral)
- **Production:** Redis or SQLite for sessions

**Decision (to be made in implementation):**

- Start with in-memory for development
- Document production session storage strategy
- Use secure HTTP-only cookies for session ID

**Session Structure:**

```go
type Session struct {
    SessionID    string
    DID          string
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
    CreatedAt    time.Time
}
```

**Required Methods:**

- [ ] `CreateSession(did, tokens)` → sessionID
- [ ] `GetSession(sessionID)` → session
- [ ] `DeleteSession(sessionID)`
- [ ] `RefreshSession(sessionID)` → updated session

#### 2.3: Authentication Middleware

Add middleware to extract authenticated user from session:

```go
func AuthMiddleware(next http.Handler) http.Handler

// Context key for authenticated DID
type contextKey string
const userDIDKey contextKey = "userDID"

// Helper to get DID from context
func GetAuthenticatedDID(r *http.Request) (string, error)
```

**Behavior:**

- Extract session cookie
- Validate session exists and not expired
- Add DID to request context
- If invalid/missing: redirect to login (for protected routes)

**Route Protection:**

- [ ] Protected routes: All write operations (POST, PUT, DELETE)
- [ ] Public routes: Home page, static assets
- [ ] Semi-protected: Brew list (show your own if logged in, or empty state)

#### 2.4: UI Updates for Authentication

Update templates to support authentication:

**New Templates:**

- [ ] `login.tmpl` - Login page with OAuth button
- [ ] Update `layout.tmpl` - Add user info header
  - If logged in: Show handle, logout button
  - If logged out: Show login button

**Navigation Updates:**

- [ ] Add user menu/dropdown
- [ ] Link to profile/settings (future)
- [ ] Display current user's handle

**Empty States:**

- [ ] If not logged in: Show welcome page with login prompt
- [ ] If logged in but no data: Show getting started guide

#### 2.5: Handler Updates

Update handlers to use authenticated DID:

```go
func (h *Handler) HandleBrewCreate(w http.ResponseWriter, r *http.Request) {
    // Get authenticated user's DID
    did, err := GetAuthenticatedDID(r)
    if err != nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // Create store scoped to this user
    store := atproto.NewAtprotoStore(h.client, did)

    // Rest of handler logic...
}
```

**All handlers need:**

- [ ] Extract DID from context
- [ ] Create user-scoped store
- [ ] Handle unauthenticated users gracefully

#### 2.6: Configuration

Add environment variables for OAuth:

```bash
# OAuth Configuration
OAUTH_CLIENT_ID=your_client_id
OAUTH_CLIENT_SECRET=your_client_secret
OAUTH_REDIRECT_URI=http://localhost:8080/oauth/callback
OAUTH_SCOPES=atproto,transition:generic

# PDS Configuration
DEFAULT_PDS_URL=https://bsky.social  # or support dynamic discovery

# Session Configuration
SESSION_SECRET=random_secret_key
SESSION_MAX_AGE=86400  # 24 hours
```

**Deliverables:**

- Working OAuth login flow
- Session management
- Protected routes with authentication
- Updated UI with login/logout
- Users can authenticate and access their own data

---

## Phase 3: Handler & UI Refactoring (2-3 days)

### Goals

- Update all handlers to work with atproto store
- Handle reference resolution in UI
- Remove SQLite dependency
- Improve error handling

### Tasks

#### 3.1: Update All Handlers

Modify handlers to use atproto store with authenticated DID:

**Handlers to update:**

- [ ] `HandleBrewList` - List user's brews with resolved references
- [ ] `HandleBrewNew` - Create new brew
- [ ] `HandleBrewEdit` - Edit existing brew
- [ ] `HandleBrewCreate` - Process create form
- [ ] `HandleBrewUpdate` - Process update form
- [ ] `HandleBrewDelete` - Delete brew
- [ ] `HandleBrewExport` - Export from PDS (not SQLite)
- [ ] Bean CRUD handlers (create, update, delete)
- [ ] Roaster CRUD handlers
- [ ] Grinder CRUD handlers
- [ ] Brewer CRUD handlers
- [ ] `HandleManage` - Manage all resources

**Key Changes:**

- Get DID from request context
- Create user-scoped store
- Handle references (AT-URIs) in forms
- Resolve references for display
- Error handling for PDS operations

#### 3.2: Reference Handling in UI

Update forms and displays to handle AT-URI references:

**Brew Form Updates:**

- [ ] Bean selector: Show user's beans, store AT-URI
- [ ] Roaster selector: Show user's roasters (via bean's roasterRef)
- [ ] Grinder selector: Show user's grinders, store AT-URI
- [ ] Brewer selector: Show user's brewers, store AT-URI

**Display Updates:**

- [ ] Resolve bean reference when displaying brew
- [ ] Resolve grinder reference
- [ ] Resolve brewer reference
- [ ] Show resolved data (bean name, roaster name, etc.)
- [ ] Handle broken references gracefully (show "Unknown" or ID)

**New Bean Form:**

- [ ] Roaster selector stores AT-URI reference
- [ ] Create new roaster inline (stores as new record, returns AT-URI)

#### 3.3: Error Handling

Improve error handling for atproto operations:

**PDS Operation Errors:**

- [ ] Network failures - retry with backoff
- [ ] Authentication failures - redirect to login
- [ ] Rate limiting - show user-friendly message
- [ ] Invalid records - show validation errors
- [ ] Missing references - handle gracefully

**User-Friendly Error Pages:**

- [ ] 401 Unauthorized - redirect to login
- [ ] 404 Not Found - "Record not found" page
- [ ] 500 Server Error - "Something went wrong" with error ID
- [ ] PDS Unavailable - "Cannot connect to your PDS" message

**Logging:**

- [ ] Log all PDS operations
- [ ] Log OAuth flow steps
- [ ] Log errors with context (DID, operation, error)
- [ ] Add request ID for tracking

#### 3.4: Remove SQLite Dependency

- [ ] Remove `internal/database/sqlite/` package
- [ ] Remove SQLite imports from main.go
- [ ] Remove database migrations
- [ ] Update go.mod (remove modernc.org/sqlite)
- [ ] Update README (remove SQLite references)

**Note:** May keep SQLite or add Redis for session storage in production (TBD).

#### 3.5: Update PWA/Offline Handling

Since app now requires online access to PDS:

**Options:**

- [ ] **Option A:** Remove service worker and PWA manifest (simple)
- [ ] **Option B:** Keep PWA but update service worker to only cache static assets
- [ ] **Option C:** Add offline queue for writes (complex, future enhancement)

**Recommendation:** Option B - keep PWA for "Add to Home Screen" but require online for data.

#### 3.6: Testing

- [ ] Manual testing of full user flow:
  1. Login with OAuth
  2. Create beans, roasters, grinders, brewers
  3. Create brews with references
  4. Edit brews
  5. Delete brews
  6. Logout and verify data persists (in PDS)
  7. Login again and verify data loads
- [ ] Test error scenarios (network failures, invalid data, etc.)
- [ ] Test with multiple users (different DIDs)

**Deliverables:**

- Fully functional personal coffee tracker using atproto
- All CRUD operations working
- Reference resolution working
- SQLite removed
- Error handling improved

---

## Phase 4: Polish & Documentation (2-3 days)

### Goals

- Production-ready configuration
- Deployment documentation
- User documentation
- Clean up code

### Tasks

#### 4.1: Production Configuration

- [ ] Environment variable validation on startup
- [ ] Configuration file support (optional)
- [ ] Secure session storage (Redis or encrypted SQLite)
- [ ] HTTPS/TLS configuration
- [ ] Rate limiting for API endpoints
- [ ] CORS configuration

#### 4.2: Deployment Preparation

- [ ] Create Dockerfile
- [ ] Docker compose for development (app + Redis)
- [ ] Document environment variables
- [ ] Health check endpoint (`/health`)
- [ ] Graceful shutdown handling
- [ ] Production logging configuration

#### 4.3: Documentation

- [ ] Update README.md:
  - New architecture overview
  - atproto concepts
  - Setup instructions
  - Environment variables
  - Deployment guide
- [ ] Create DEPLOYMENT.md:
  - Server requirements
  - OAuth app registration
  - Domain/DNS setup
  - SSL certificate setup
  - Systemd service example
- [ ] Create SELF-HOSTING.md:
  - Guide for others to run their own AppView
  - Configuration options
  - Maintenance tasks
- [ ] Create docs/LEXICONS.md:
  - Document all lexicon schemas
  - Field descriptions
  - Reference patterns
  - Examples

#### 4.4: Code Cleanup

- [ ] Add/improve code comments
- [ ] Consistent error handling patterns
- [ ] Extract magic constants to config
- [ ] Remove dead code
- [ ] Format all code (gofmt)
- [ ] Lint and fix issues (golangci-lint)

#### 4.5: Testing

- [ ] Write unit tests for critical paths
- [ ] Integration tests for OAuth flow
- [ ] Test deployment in production-like environment
- [ ] Load testing (basic)
- [ ] Security review (OWASP basics)

**Deliverables:**

- Production-ready application
- Complete documentation
- Deployment artifacts (Docker, etc.)
- Ready to launch

---

## Phase 5: Launch & Initial Users (TBD)

### Goals

- Deploy to production
- Onboard initial users
- Monitor and fix issues
- Gather feedback

### Tasks

- [ ] Deploy to production hosting
- [ ] Set up monitoring and alerting
- [ ] Create landing page explaining the project
- [ ] Announce in atproto community
- [ ] Gather user feedback
- [ ] Fix bugs and usability issues
- [ ] Iterate based on feedback

**Success Metrics:**

- Users can successfully authenticate
- Users can create and manage their coffee data
- Data persists in their PDS
- No critical bugs

---

## Future Enhancements (Post-Launch)

### Phase 6: Public Browsing & Discovery (Future)

**Goal:** Allow users to discover and view other users' coffee brews.

**Features:**

- [ ] Public profile pages (`/users/:did`)
- [ ] Browse any user's public brews
- [ ] Manual DID entry for discovery
- [ ] Handle resolution (DID → handle display)
- [ ] Basic search within a user's data

**Technical:**

- [ ] Query any PDS for public records
- [ ] Cache results for performance
- [ ] Handle PDS availability issues

### Phase 7: AppView Indexing (Future)

**Goal:** Build a centralized index for cross-user discovery.

**Features:**

- [ ] Firehose subscription
- [ ] Index public arabica records from all PDSs
- [ ] Cross-user search (by bean, roaster, method, etc.)
- [ ] Trending/popular content
- [ ] User directory

**Technical:**

- [ ] Add PostgreSQL for index storage
- [ ] Firehose consumer using indigo
- [ ] Background indexing jobs
- [ ] Search API

### Phase 8: Social Features (Future)

**Goal:** Add social interactions around coffee.

**Features:**

- [ ] Follow users
- [ ] Like/bookmark brews
- [ ] Comments on brews (atproto replies)
- [ ] Share brews
- [ ] Brew collections/lists

**Technical:**

- [ ] New lexicons for likes, follows, etc.
- [ ] Integration with atproto social graph
- [ ] Notification system

### Phase 9: Advanced Features (Future)

**Ideas for future consideration:**

- [ ] Statistics and analytics (personal insights)
- [ ] Brew recipes and recommendations
- [ ] Photo uploads (blob storage in PDS)
- [ ] Equipment database (community-maintained)
- [ ] Taste profile analysis
- [ ] CSV import/export
- [ ] Mobile native apps (using same lexicons)

### Phase 10: Performance & Scale (Future)

**Optimizations when needed:**

- [ ] Implement caching layer (Redis/SQLite)
- [ ] Batch reference resolution
- [ ] CDN for static assets
- [ ] Optimize PDS queries
- [ ] Background sync for index
- [ ] Horizontal scaling

---

## Open Questions & Decisions

### To Be Decided During Implementation

#### Authentication & Session Management

- **Q:** Use in-memory sessions (dev) or add Redis immediately?
- **Q:** Session timeout duration?
- **Q:** Support "remember me" functionality?
- **Decision:** TBD based on hosting environment

#### Reference Resolution Strategy

- **Q:** Lazy loading (simple) or batch resolution (complex)?
- **Q:** Cache resolved references?
- **Decision:** Start with lazy loading, optimize later if needed

#### PDS Support

- **Q:** Bluesky only, or support any PDS from day 1?
- **Q:** How to handle PDS discovery (handle → PDS URL)?
- **Decision:** Support any PDS, use handle resolution

#### Error Handling Philosophy

- **Q:** Detailed errors for debugging vs user-friendly messages?
- **Q:** Retry strategy for PDS operations?
- **Decision:** User-friendly errors, log details, retry with backoff

#### Lexicon Publishing

- **Q:** Where to host lexicon files publicly?
- **Options:**
  - GitHub repo (easy)
  - `.well-known` on domain (proper)
  - Both
- **Decision:** GitHub for now, add .well-known later

#### Export Functionality

- **Q:** Keep JSON export feature?
- **Q:** Export from PDS or from AppView cache?
- **Decision:** Keep export, fetch from PDS

#### PWA/Offline

- **Q:** Remove service worker entirely?
- **Q:** Keep PWA manifest for "Add to Home Screen"?
- **Decision:** Keep manifest, update service worker for static-only caching

---

## Timeline Summary

| Phase                       | Duration | Key Milestone                               |
| --------------------------- | -------- | ------------------------------------------- |
| 0: Research & Validation    | 2-3 days | Lexicons validated, indigo SDK understood   |
| 1: Core atproto Client      | 3-4 days | PDS operations working, SQLite removed      |
| 2: Authentication & OAuth   | 3-4 days | Users can login with atproto OAuth (scopes) |
| 3: Handler & UI Refactoring | 2-3 days | Full app working with atproto               |
| 4: Polish & Documentation   | 2-3 days | Production ready, documented                |
| 5: Launch                   | Variable | Live with initial users                     |

**Total Estimated Time: 12-17 days** of focused development work

**Future phases:** TBD based on user feedback and priorities

---

## Success Criteria

### Phase 1-4 Complete (Personal Tracker v1)

- [ ] Users can authenticate via atproto OAuth with proper scopes
- [ ] Users can create, edit, delete all coffee tracking entities
- [ ] Data persists in user's PDS (any PDS)
- [ ] References between records work correctly
- [ ] App is self-hostable by others
- [ ] Documentation is complete and accurate
- [ ] No dependency on SQLite for data storage
- [ ] Existing UX/UI is preserved

### Future Success (Discovery & Social)

- [ ] Users can discover other coffee enthusiasts
- [ ] Cross-user search and browsing works
- [ ] Social features enable community building
- [ ] AppView scales to many users

---

## Resources & References

### Documentation

- [AT Protocol Docs](https://atproto.com)
- [Indigo SDK](https://github.com/bluesky-social/indigo)
- [Lexicon Specification](https://atproto.com/specs/lexicon)
- [OAuth DPOP](https://atproto.com/specs/oauth)

### Example Applications

- Bluesky (reference implementation)
- Other atproto apps (TBD - research during Phase 0)

### Tools

- [atproto CLI tools](https://github.com/bluesky-social/atproto)
- PDS explorer tools
- Lexicon validators

---

## Notes

- This plan is a living document and will be updated as we learn more
- Technical decisions may change based on discoveries during implementation
- Timeline estimates are rough and may vary
- Focus is on shipping a working v1 (personal tracker) before adding social features
- OAuth must use scopes, not app passwords, for proper security and user control
