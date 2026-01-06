# Auth-Required UI Changes

## Changes Made

### 1. Home Page - Login Button

**File:** `internal/templates/home.tmpl`

**When Not Authenticated:**
- Shows welcome message explaining AT Protocol
- Large "Log In with AT Protocol" button
- Lists Arabica features (decentralized, portable data, etc.)

**When Authenticated:**
- Shows user's DID
- "Add New Brew" and "View All Brews" buttons
- Full functionality

### 2. Navigation Bar - Conditional Links

**File:** `internal/templates/layout.tmpl`

**When Not Authenticated:**
- Shows: Home, Login button
- Hides: Brews, New Brew, Manage

**When Authenticated:**
- Shows: Home, Brews, New Brew, Manage, Logout button
- Logout button triggers POST to `/logout`

### 3. Removed SQLite Fallback

**File:** `internal/handlers/handlers.go`

**All protected handlers now:**
```go
// Require authentication
store, authenticated := h.getAtprotoStore(r)
if !authenticated {
    http.Redirect(w, r, "/login", http.StatusFound)
    return
}
```

**Affected handlers:**
- `HandleBrewList` - redirects to login
- `HandleBrewNew` - redirects to login
- `HandleBrewEdit` - redirects to login
- `HandleBrewCreate` - redirects to login
- `HandleManage` - redirects to login
- `HandleBeanCreate` - returns 401 (API endpoint)
- `HandleRoasterCreate` - returns 401 (API endpoint)

**SQLite is NO LONGER used** - all data operations require authentication and use PDS storage.

### 4. Template Data Structure

**File:** `internal/templates/render.go`

**Added to PageData:**
```go
type PageData struct {
    // ... existing fields
    IsAuthenticated bool
    UserDID         string
}
```

**All render functions updated** to accept and pass authentication status:
- `RenderHome(w, isAuthenticated, userDID)`
- `RenderBrewList(w, brews, isAuthenticated, userDID)`
- `RenderBrewForm(w, beans, roasters, grinders, brewers, brew, isAuthenticated, userDID)`
- `RenderManage(w, beans, roasters, grinders, brewers, isAuthenticated, userDID)`

## User Experience

### First Visit (Not Logged In)
1. Visit http://localhost:18910
2. See welcome page with "Log In with AT Protocol" button
3. Nav bar only shows "Home" and "Login"
4. Cannot access /brews, /manage, etc. (redirects to /login)

### After Login
1. Click "Log In with AT Protocol"
2. Enter your handle (e.g., `yourname.bsky.social`)
3. Authenticate on your PDS
4. Redirected to home page
5. Nav bar shows all links + Logout button
6. Full app functionality available
7. All data stored in YOUR PDS

### Logout
1. Click "Logout" in nav bar
2. Session cleared
3. Redirected to home page (unauthenticated view)

## Benefits

### Security
- No data leakage between users
- Cannot access app without authentication
- Each user only sees their own data

### Privacy
- Your data lives in YOUR PDS
- Not stored on Arabica server
- You control your data

### User Clarity
- Clear distinction between logged in/out states
- Obvious call-to-action to log in
- Shows which DID you're logged in as

## Testing

1. **Start server:**
```bash
go build ./cmd/server
./server
```

2. **Visit home page (logged out):**
- Go to http://localhost:18910
- Should see login button
- Nav bar should only show Home + Login

3. **Try accessing protected pages:**
- Go to http://localhost:18910/brews
- Should redirect to /login

4. **Log in:**
- Click "Log In with AT Protocol"
- Enter your Bluesky handle
- Authenticate

5. **Verify logged in state:**
- Home page shows your DID
- Nav bar shows all links + Logout
- Can access /brews, /manage, etc.

6. **Test logout:**
- Click "Logout" in nav
- Should return to unauthenticated home page

## Next Steps

Now that auth is working and UI is clean:
1. Test creating a bean from /manage
2. Verify it appears in your PDS
3. Fix the ID/rkey issue so you can create brews that reference beans
