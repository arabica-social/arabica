# Indigo SDK Research

## Overview
The `indigo` SDK from Bluesky provides a comprehensive Go implementation for atproto, including:
- Complete OAuth client with DPOP support
- XRPC client for API operations
- Repository operations (create/read/update/delete records)
- DID resolution and handle management
- Lexicon support

**Package:** `github.com/bluesky-social/indigo`  
**Version:** v0.0.0-20260103083015-78a1c1894f36 (pseudo-version, tracks main branch)

---

## Key Packages

### 1. `atproto/auth/oauth` - OAuth Client Implementation
**Location:** `github.com/bluesky-social/indigo/atproto/auth/oauth`

**Features:**
- Complete OAuth 2.0 + DPOP implementation
- PKCE support (Proof Key for Code Exchange)
- Public and confidential client support
- Token refresh handling
- Session management interfaces
- PAR (Pushed Authorization Request) support

**Main Components:**

#### `ClientApp` - High-Level OAuth Client
```go
type ClientApp struct {
    Config ClientConfig
    Store  ClientAuthStore
}

func NewClientApp(config *ClientConfig, store ClientAuthStore) *ClientApp
```
- Manages OAuth flow for multiple users
- Handles session persistence
- Automatic token refresh
- Thread-safe for concurrent use

#### `ClientConfig` - OAuth Configuration
```go
type ClientConfig struct {
    ClientID    string   // URL to client metadata (e.g., "https://arabica.com/client-metadata.json")
    RedirectURI string   // Callback URL
    Scopes      []string // e.g., ["atproto", "transition:generic"]
    // ... other fields
}

func NewPublicConfig(clientID, redirectURI string, scopes []string) ClientConfig
```

#### `ClientAuthStore` - Session Persistence Interface
```go
type ClientAuthStore interface {
    GetSession(ctx context.Context, sessionID string) (*ClientAuth, error)
    PutSession(ctx context.Context, session *ClientAuth) error
    DeleteSession(ctx context.Context, sessionID string) error
}
```

**Built-in Implementations:**
- `MemStore` - In-memory storage (ephemeral, for development)
- Custom implementations needed for production (database-backed)

#### `ClientAuth` - Session State
```go
type ClientAuth struct {
    SessionID    string
    DID          string
    AccessToken  string
    RefreshToken string
    TokenType    string
    ExpiresAt    time.Time
    Scope        string
    // DPOP key material
}
```

**Example OAuth Flow (from doc.go):**
```go
// 1. Create client app (once at startup)
config := oauth.NewPublicConfig(
    "https://arabica.com/client-metadata.json",
    "https://arabica.com/oauth/callback",
    []string{"atproto", "transition:generic"},
)
oauthApp := oauth.NewClientApp(&config, oauth.NewMemStore())

// 2. Initiate login
state, authURL, err := oauthApp.Authorize(ctx, handle, stateParam)
// Redirect user to authURL

// 3. Handle callback
auth, err := oauthApp.HandleCallback(ctx, state, code)
// auth.SessionID, auth.DID now available

// 4. Make authenticated requests
token, err := oauthApp.TokenForSession(ctx, auth.SessionID)
```

**OAuth Demo App:**
- Example implementation: `atproto/auth/oauth/cmd/oauth-web-demo/main.go`
- Shows complete flow with web UI
- Good reference for our implementation

---

### 2. `atproto/atclient` - Repository Operations
**Location:** `github.com/bluesky-social/indigo/atproto/atclient`

Provides high-level client for atproto operations:
- Record CRUD operations
- Repository listing
- Blob uploads
- Identity resolution

**Key Types:**

#### `Client` - XRPC Client
```go
type Client struct {
    // HTTP client with auth
}

// Record operations
func (c *Client) CreateRecord(ctx context.Context, input *CreateRecordInput) (*CreateRecordOutput, error)
func (c *Client) GetRecord(ctx context.Context, input *GetRecordInput) (*GetRecordOutput, error)
func (c *Client) ListRecords(ctx context.Context, input *ListRecordsInput) (*ListRecordsOutput, error)
func (c *Client) PutRecord(ctx context.Context, input *PutRecordInput) (*PutRecordOutput, error)
func (c *Client) DeleteRecord(ctx context.Context, input *DeleteRecordInput) error
```

**Record Input/Output Types:**
```go
type CreateRecordInput struct {
    Repo       string                 // DID
    Collection string                 // e.g., "com.arabica.brew"
    Record     map[string]interface{} // Record data
    RKey       *string                // Optional, uses TID if not provided
}

type CreateRecordOutput struct {
    URI string // AT-URI of created record
    CID string // Content ID
}

type GetRecordInput struct {
    Repo       string // DID
    Collection string // e.g., "com.arabica.brew"
    RKey       string // Record key
}

type GetRecordOutput struct {
    URI    string
    CID    string
    Value  map[string]interface{} // Record data
}

type ListRecordsInput struct {
    Repo       string  // DID
    Collection string  // e.g., "com.arabica.brew"
    Limit      *int64  // Optional
    Cursor     *string // For pagination
}

type ListRecordsOutput struct {
    Records []Record
    Cursor  *string
}
```

---

### 3. `atproto/syntax` - AT-URI Parsing
**Location:** `github.com/bluesky-social/indigo/atproto/syntax`

Parse and construct AT-URIs:
```go
// Parse AT-URI: at://did:plc:abc123/com.arabica.brew/3jxy...
uri, err := syntax.ParseATURI("at://...")
fmt.Println(uri.Authority()) // DID
fmt.Println(uri.Collection()) // Collection name
fmt.Println(uri.RecordKey()) // RKey

// Construct AT-URI
uri := syntax.ATURI{
    Authority:  did,
    Collection: "com.arabica.brew",
    RecordKey:  rkey,
}
```

---

### 4. `atproto/identity` - DID and Handle Resolution
**Location:** `github.com/bluesky-social/indigo/atproto/identity`

Resolve handles to DIDs and discover PDS endpoints:
```go
// Resolve handle to DID
did, err := identity.ResolveHandle(ctx, "alice.bsky.social")

// Resolve DID to PDS URL
pdsURL, err := identity.ResolveDIDToPDS(ctx, did)

// Resolve DID document
didDoc, err := identity.ResolveDID(ctx, did)
```

---

## Implementation Plan for Arabica

### Phase 1: OAuth Integration

**Files to Create:**
- `internal/atproto/oauth.go` - Wrapper around indigo's OAuth
- `internal/atproto/session.go` - Session storage implementation
- `internal/atproto/middleware.go` - Auth middleware

**Key Decisions:**

1. **Session Storage:**
   - Development: Use `oauth.MemStore` (built-in)
   - Production: Implement `ClientAuthStore` with SQLite or Redis

2. **Client Metadata:**
   - Serve at `/.well-known/oauth-client-metadata` or at client_id URL
   - Generate from `config.ClientMetadata()`
   - Must be HTTPS in production

3. **Scopes:**
   - Start with: `["atproto", "transition:generic"]`
   - `atproto` scope gives full repo access
   - `transition:generic` allows legacy operations

4. **Client Type:**
   - Use **public client** (no client secret)
   - Simpler for self-hosted deployments
   - PKCE provides security

**Implementation Steps:**

1. Create OAuth wrapper:
```go
// internal/atproto/oauth.go
type OAuthManager struct {
    app *oauth.ClientApp
}

func NewOAuthManager(clientID, redirectURI string) (*OAuthManager, error) {
    config := oauth.NewPublicConfig(
        clientID,
        redirectURI,
        []string{"atproto", "transition:generic"},
    )
    
    // Use MemStore for development
    store := oauth.NewMemStore()
    
    app := oauth.NewClientApp(&config, store)
    return &OAuthManager{app: app}, nil
}

func (m *OAuthManager) InitiateLogin(ctx context.Context, handle string) (string, error) {
    // Generate state, get auth URL, redirect
}

func (m *OAuthManager) HandleCallback(ctx context.Context, code, state string) (*oauth.ClientAuth, error) {
    // Exchange code for tokens
}

func (m *OAuthManager) GetSession(ctx context.Context, sessionID string) (*oauth.ClientAuth, error) {
    // Retrieve session
}
```

2. Add HTTP handlers:
```go
// internal/handlers/auth.go
func (h *Handler) HandleLogin(w http.ResponseWriter, r *http.Request) {
    handle := r.FormValue("handle")
    authURL, err := h.oauth.InitiateLogin(r.Context(), handle)
    http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *Handler) HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    state := r.URL.Query().Get("state")
    
    auth, err := h.oauth.HandleCallback(r.Context(), code, state)
    
    // Set session cookie
    http.SetCookie(w, &http.Cookie{
        Name:     "session_id",
        Value:    auth.SessionID,
        HttpOnly: true,
        Secure:   true, // HTTPS only in production
        SameSite: http.SameSiteLaxMode,
    })
    
    http.Redirect(w, r, "/", http.StatusFound)
}
```

3. Create auth middleware:
```go
// internal/atproto/middleware.go
func (m *OAuthManager) AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        cookie, err := r.Cookie("session_id")
        if err != nil {
            // No session, continue without auth
            next.ServeHTTP(w, r)
            return
        }
        
        auth, err := m.GetSession(r.Context(), cookie.Value)
        if err != nil {
            // Invalid session
            http.Redirect(w, r, "/login", http.StatusFound)
            return
        }
        
        // Add DID to context
        ctx := context.WithValue(r.Context(), "userDID", auth.DID)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

---

### Phase 2: Repository Operations

**Files to Create:**
- `internal/atproto/client.go` - XRPC client wrapper
- `internal/atproto/records.go` - Record type conversions
- `internal/atproto/store.go` - Store interface implementation

**Implementation:**

1. Create authenticated XRPC client:
```go
// internal/atproto/client.go
type Client struct {
    oauth *OAuthManager
    xrpc  *xrpc.Client
}

func (c *Client) GetAuthenticatedClient(ctx context.Context, sessionID string) (*xrpc.Client, error) {
    auth, err := c.oauth.GetSession(ctx, sessionID)
    if err != nil {
        return nil, err
    }
    
    // Create XRPC client with auth
    client := &xrpc.Client{
        Host: auth.Host, // PDS URL
        Auth: &xrpc.AuthInfo{
            AccessJwt:  auth.AccessToken,
            RefreshJwt: auth.RefreshToken,
            Did:        auth.DID,
            Handle:     auth.Handle,
        },
    }
    
    return client, nil
}
```

2. Record CRUD operations:
```go
// internal/atproto/store.go
type AtprotoStore struct {
    client *Client
    did    string
}

func (s *AtprotoStore) CreateBrew(brew *models.CreateBrewRequest, userID int) (*models.Brew, error) {
    // Convert brew to record
    record := brewToRecord(brew)
    
    // Create record in PDS
    output, err := s.client.CreateRecord(ctx, &atclient.CreateRecordInput{
        Repo:       s.did,
        Collection: "com.arabica.brew",
        Record:     record,
    })
    
    // Convert back to Brew model
    return recordToBrew(output)
}
```

---

## Questions Answered

### ✅ Does indigo provide complete OAuth client with DPOP support?
**YES** - Full implementation in `atproto/auth/oauth` package with:
- DPOP JWT signing and nonces
- Automatic token refresh with DPOP
- Public and confidential client support
- PKCE built-in

### ✅ What's the API for repo operations?
**Answer:** `atproto/atclient` provides:
- `CreateRecord()`, `GetRecord()`, `ListRecords()`, `PutRecord()`, `DeleteRecord()`
- Input/output structs with proper typing
- Handles AT-URIs and CIDs automatically

### ✅ Are there session management helpers?
**YES** - `ClientAuthStore` interface with `MemStore` implementation:
- Define interface for session persistence
- Built-in in-memory store for development
- Can implement custom stores (DB, Redis, etc.)

### ✅ How are AT-URIs parsed and resolved?
**Answer:** `atproto/syntax` package:
- `ParseATURI()` and `ATURI` type
- Extract DID, collection, rkey components
- Construct new AT-URIs

---

## Next Steps

1. ✅ **Understand indigo OAuth** - COMPLETE
2. **Create lexicon files** - Define schemas
3. **Build OAuth wrapper** - Use indigo's ClientApp
4. **Implement Store interface** - Use atclient for PDS operations
5. **Update handlers** - Add auth context

---

## Useful Resources

- **Indigo GitHub:** https://github.com/bluesky-social/indigo
- **OAuth Demo:** `atproto/auth/oauth/cmd/oauth-web-demo/main.go`
- **API Docs:** GoDoc for each package
- **atproto Specs:** https://atproto.com/specs/oauth

---

## Notes

- Indigo is actively developed, uses pseudo-versions (no tags)
- OAuth implementation is production-ready and used by Bluesky
- Good test coverage and examples
- Community support via Bluesky Discord
- Well-architected with clear interfaces
