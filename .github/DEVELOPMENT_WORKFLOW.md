# Development Workflow

## Setting Up Firehose Feed with Known DIDs

For development and testing, you can populate your local feed with known Arabica users:

### 1. Create a Known DIDs File

Create `known-dids.txt` in the project root:

```bash
cat > known-dids.txt << 'EOF'
# Known Arabica users for development
# Add one DID per line

# Example (replace with real DIDs):
# did:plc:abc123xyz
# did:plc:def456uvw

EOF
```

### 2. Find DIDs to Add

You can find DIDs of Arabica users in several ways:

**From Bluesky profiles:**
- Visit a user's profile on Bluesky
- Check the URL or profile metadata for their DID

**From authenticated sessions:**
- After logging into Arabica, check your browser cookies
- The `did` cookie contains your DID

**From AT Protocol explorer tools:**
- Use tools like `atproto.blue` to search for users

### 3. Run Server with Backfill

```bash
# Start server with firehose and backfill
go run cmd/server/main.go --firehose --known-dids known-dids.txt

# Or with nix (requires adding flags to flake.nix)
nix run -- --firehose --known-dids known-dids.txt
```

### 4. Monitor Backfill Progress

Watch the logs for backfill activity:

```
{"level":"info","count":3,"file":"known-dids.txt","message":"Loaded known DIDs from file"}
{"level":"info","did":"did:plc:abc123xyz","message":"backfilling user records"}
{"level":"info","total":5,"success":5,"message":"Backfill complete"}
```

### 5. Verify Feed Data

Once backfilled, check:
- Home page feed should show brews from backfilled users
- `/feed` endpoint should return feed items
- Database should contain indexed records

## File Format Notes

The `known-dids.txt` file supports:

- **Comments**: Lines starting with `#`
- **Empty lines**: Ignored
- **Whitespace**: Automatically trimmed
- **Validation**: Non-DID lines logged as warnings

Example valid file:

```
# Coffee enthusiasts to follow
did:plc:user1abc

# Another user
did:plc:user2def

did:web:coffee.example.com  # Web DID example
```

## Security Note

⚠️ **Important**: The `known-dids.txt` file is gitignored by default. Do not commit DIDs unless you have permission from the users.

For production deployments, rely on organic discovery via firehose rather than manual DID lists.
