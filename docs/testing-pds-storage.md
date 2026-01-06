# Testing PDS Storage

## Current Status

**YES, the app IS trying to write to your PDS!** 

When you're logged in via OAuth, the handlers use `AtprotoStore` which makes XRPC calls to your Personal Data Server.

## How It Works

1. **Login** → OAuth flow → Session with your DID
2. **Create Bean** → `POST /api/beans` → Calls `com.atproto.repo.createRecord` on your PDS
3. **PDS stores the record** → Returns AT-URI like `at://did:plc:abc123/com.arabica.bean/3jxydef789`
4. **App stores the record** → But has a problem with the returned ID...

## The ID Problem

**Current Issue:** The app expects integer IDs but PDS returns string rkeys (TIDs).

### What Happens Now

```
User creates bean "Ethiopian Yirgacheffe"
  ↓
App calls: CreateRecord("com.arabica.bean", {name: "Ethiopian Yirgacheffe", ...})
  ↓
PDS returns: {uri: "at://did:plc:abc/com.arabica.bean/3jxy...", cid: "..."}
  ↓
App extracts rkey: "3jxy..."
  ↓
App sets bean.ID = 0  ← PROBLEM! Should store the rkey somewhere
  ↓
Bean is returned to UI with ID=0
```

### When Creating a Brew

```
User selects bean from dropdown (shows ID=0 or wrong ID)
  ↓
Form submits: {beanID: 0, ...}
  ↓
App constructs: at://did:plc:abc/com.arabica.bean/0  ← INVALID!
  ↓
PDS call fails: "Record not found"
```

## Testing Right Now

### What Works ✅

1. **OAuth login** - You can log in with your atproto handle
2. **List operations** - Will fetch from your PDS (empty initially)
3. **Create operations** - Will call PDS API (but ID handling is broken)

### What Doesn't Work ❌

1. **Creating brews** - Requires valid bean references (IDs don't map correctly)
2. **Editing records** - ID → rkey lookup fails
3. **Deleting records** - ID → rkey lookup fails
4. **References** - Can't construct valid AT-URIs from integer IDs

## How to Test Manually

### Test 1: Create a Bean (Direct API Call)

```bash
# Log in first via browser at http://localhost:18910/login
# Then get your session cookie and...

curl -X POST http://localhost:18910/api/beans \
  -H "Content-Type: application/json" \
  -H "Cookie: account_did=...; session_id=..." \
  -d '{
    "name": "Test Bean",
    "origin": "Ethiopia",
    "roast_level": "Light",
    "process": "Washed"
  }'
```

**Watch the server logs** - you should see:
- XRPC call to your PDS
- Either success with returned URI, or an error

### Test 2: List Beans from PDS

```bash
curl http://localhost:18910/manage \
  -H "Cookie: account_did=...; session_id=..."
```

This will try to list all beans from your PDS.

### Test 3: Check Your PDS Directly

Use the atproto API to see what's actually in your repo:

```bash
# List all records in com.arabica.bean collection
curl "https://bsky.social/xrpc/com.atproto.repo.listRecords?repo=yourhandle.bsky.social&collection=com.arabica.bean"
```

## Debugging

### Enable Verbose Logging

The store already has debug printf statements:

```go
fmt.Printf("Warning: failed to resolve brew references: %v\n", err)
```

Watch your server console for these messages.

### Check XRPC Calls

The `client.go` makes XRPC calls via indigo's `client.Do()`. If there are errors, they'll be returned and logged.

## Next Steps to Fix

### Option 1: Store rkey in models (Quick Fix)

Add a `RKey string` field to all models:

```go
type Bean struct {
    ID    int    `json:"id"`
    RKey  string `json:"rkey"`  // ← Add this
    Name  string `json:"name"`
    // ...
}
```

Then update AtprotoStore to:
1. Store the rkey when creating
2. Use the rkey for updates/deletes
3. Build AT-URIs from stored rkeys

### Option 2: In-Memory Mapping (Temporary)

Keep a map in AtprotoStore:

```go
type AtprotoStore struct {
    // ...
    idToRKey map[string]map[int]string // collection -> id -> rkey
}
```

### Option 3: Use rkeys as IDs (Proper Fix)

Change models to use string IDs everywhere:

```go
type Bean struct {
    ID   string `json:"id"` // Now stores rkey like "3jxy..."
    Name string `json:"name"`
    // ...
}
```

This requires updating:
- All handlers (parse string IDs, not ints)
- Templates (use string IDs in forms)
- Store interface (change signatures)

## Recommended Testing Path

1. **Update models to store rkeys** (Option 1)
2. **Test bean creation** - verify record appears in PDS
3. **Test bean listing** - verify records are fetched from PDS
4. **Test brew creation with valid bean rkey**
5. **Verify end-to-end flow works**

## Current Code Locations

- **Store implementation**: `internal/atproto/store.go`
- **Record conversions**: `internal/atproto/records.go`
- **XRPC client**: `internal/atproto/client.go`
- **Handlers**: `internal/handlers/handlers.go`

## Summary

**You're 90% there!** The OAuth works, the XRPC calls are being made, the record conversions are correct. The only missing piece is proper ID/rkey handling so that references work correctly.

The quickest path forward is to add `RKey` fields to the models and update the store to use them.
