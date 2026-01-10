# AT Protocol Integration

## Overview

Arabica uses the Bluesky indigo SDK for AT Protocol integration.

**Package:** `github.com/bluesky-social/indigo`

## Key Components

### OAuth Authentication

- Public OAuth client with PKCE
- DPOP-bound access tokens
- Scopes: `atproto`, `transition:generic`
- Session persistence via BoltDB

### Record Operations

Standard AT Protocol record CRUD operations:
- `com.atproto.repo.createRecord`
- `com.atproto.repo.getRecord`
- `com.atproto.repo.listRecords`
- `com.atproto.repo.putRecord`
- `com.atproto.repo.deleteRecord`

### Client Implementation

See `internal/atproto/client.go` for the XRPC client wrapper.

## References

- indigo SDK: https://github.com/bluesky-social/indigo
- AT Protocol docs: https://atproto.com
