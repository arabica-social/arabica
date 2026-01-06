# OAuth Token Fix - DPOP Authentication

## The Problem

Getting error: `XRPC ERROR 400: InvalidToken: Malformed token`

## Root Cause

We were manually creating an XRPC client and setting the auth like this:

```go
client.Auth = &xrpc.AuthInfo{
    AccessJwt:  sessData.AccessToken,
    RefreshJwt: sessData.RefreshToken,
    // ...
}
```

**This doesn't work** because indigo's OAuth uses **DPOP (Demonstrating Proof of Possession)** tokens, which require:
1. Special cryptographic signing of each request
2. Token nonces that rotate
3. Proof-of-possession headers

You can't just pass the access token directly - it needs DPOP signatures!

## The Solution

Use indigo's built-in `ClientSession` which handles DPOP automatically:

```go
// Resume the OAuth session
session, err := c.oauth.app.ResumeSession(ctx, did, sessionID)

// Get the authenticated API client
apiClient := session.APIClient()

// Now make requests - DPOP is handled automatically
apiClient.Post(ctx, "com.atproto.repo.createRecord", body, &result)
```

## What Changed

**File:** `internal/atproto/client.go`

**Before:**
- Manually created `xrpc.Client`
- Set `client.Auth` with raw tokens ❌
- Called `client.Do()` directly

**After:**
- Call `app.ResumeSession()` to get `ClientSession`
- Call `session.APIClient()` to get authenticated client ✅
- Use `apiClient.Post()` and `apiClient.Get()` methods
- DPOP signing happens automatically

## How DPOP Works (Behind the Scenes)

1. Each request gets a unique DPOP proof JWT
2. The proof is signed with a private key stored in the session
3. The proof includes:
   - HTTP method
   - Request URL
   - Current timestamp
   - Nonce from server
4. PDS validates the proof matches the access token

**Why this matters:** Even if someone intercepts your access token, they can't use it without the private key to sign DPOP proofs.

## Testing

Now you should be able to create beans/brews and see them in your PDS!

Try creating a bean from `/manage` - it should work now.

To verify it's actually in your PDS:
```bash
curl "https://bsky.social/xrpc/com.atproto.repo.listRecords?repo=yourhandle.bsky.social&collection=com.arabica.bean"
```
