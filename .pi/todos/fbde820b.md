{
  "id": "fbde820b",
  "title": "P3.3: Playwright E2E setup + critical paths",
  "tags": [],
  "status": "open",
  "created_at": "2026-07-09T01:49:26.174Z"
}

Set up Playwright E2E tests against the integration test harness.

1. Install Playwright in web/ workspace (or a dedicated e2e/ dir)
2. Configure Playwright to drive a browser against the httptest.Server from tests/integration/harness.go
3. Write a Go test helper that boots the harness and exposes its URL to Playwright
4. Critical paths to cover:
   - Create brew — login → /brews/new → fill form → submit → see in feed/my-coffee
   - View feed — home → feed loads → filter → paginate
   - Manage entities — my-coffee → create roaster → create bean → edit
   - Social — view brew → like → comment → see notification
   - Profile — view own → view other user's
5. Add a justfile target and document in CI

Note: The harness uses custom auth headers (X-Test-Auth-DID) injected by an http.Transport. For Playwright (real browser), need a different auth injection mechanism — either a test login route or a cookie-injecting middleware. Investigate options.
