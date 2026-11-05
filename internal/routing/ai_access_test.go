package routing_test

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
	"tangled.org/arabica.social/arabica/internal/aiaccess"
	arabicaai "tangled.org/arabica.social/arabica/internal/arabica/ai"
	arabicaapp "tangled.org/arabica.social/arabica/internal/arabica/app"
	"tangled.org/arabica.social/arabica/internal/handlers"
	"tangled.org/arabica.social/arabica/internal/moderation"
	"tangled.org/arabica.social/arabica/internal/routing"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

func TestAIAccessRouter(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "roles.json")
	require.NoError(t, os.WriteFile(configPath, []byte(`{"roles":{"admin":{"permissions":[]},"moderator":{"permissions":[]}},"users":[{"did":"did:plc:admin","role":"admin"},{"did":"did:plc:moderator","role":"moderator"}]}`), 0600))
	staff, err := moderation.NewService(configPath)
	require.NoError(t, err)
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "access.db"))
	require.NoError(t, err)
	defer db.Close()
	access, err := aiaccess.NewStore(db)
	require.NoError(t, err)
	app := arabicaapp.New()
	h := handlers.NewHandler(nil, nil, nil, nil, nil, handlers.Config{})
	h.SetApp(app)
	h.SetModeration(staff, nil)
	h.SetAIAccess(access)
	h.SetStoreOverrideForTest(bodyLimitStore{})
	completer := &bodyLimitCompleter{}
	router := routing.SetupRouter(routing.Config{App: app, Handlers: h, AI: arabicaai.NewService(completer), ModerationService: staff, SPAHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("X-Test-SPA", "yes") }), Logger: zerolog.Nop(), DisableRateLimit: true})
	request := func(method, path, did, body, site string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if site != "" {
			req.Header.Set("Sec-Fetch-Site", site)
		}
		if did != "" {
			req = req.WithContext(atpmiddleware.ContextWithAuth(req.Context(), did, "session"))
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec
	}
	checkAccess := func(did string, allowed bool) {
		t.Helper()
		rec := request("GET", "/api/ai/status", did, "", "")
		require.Equal(t, 200, rec.Code)
		require.Equal(t, `{"enabled":`+map[bool]string{true: "true", false: "false"}[allowed]+`,"allowed":`+map[bool]string{true: "true", false: "false"}[allowed]+`}`, strings.TrimSpace(rec.Body.String()))
		for _, method := range []string{"GET", "HEAD"} {
			rec = request(method, "/chat", did, "", "")
			if allowed {
				require.Equal(t, 200, rec.Code)
				require.Equal(t, "yes", rec.Header().Get("X-Test-SPA"))
			} else {
				require.Equal(t, 404, rec.Code)
				require.Equal(t, "yes", rec.Header().Get("X-Test-SPA"))
			}
			require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
		}
		if !allowed {
			for _, site := range []string{"same-origin", "cross-site"} {
				require.Equal(t, 404, request("POST", "/api/ai/chat", did, "not JSON", site).Code)
			}
		}
	}
	checkAccess("", false)
	checkAccess(bodyLimitDID, false)
	for _, did := range []string{"", "did:plc:outsider"} {
		for _, method := range []string{"GET", "PUT", "DELETE"} {
			require.Equal(t, 403, request(method, "/api/_mod/ai-access", did, `{"did":"`+bodyLimitDID+`"}`, "same-origin").Code)
		}
	}
	for _, method := range []string{"PUT", "DELETE"} {
		require.Equal(t, 403, request(method, "/api/_mod/ai-access", "did:plc:admin", `{"did":"`+bodyLimitDID+`"}`, "cross-site").Code)
	}
	require.Equal(t, 400, request("PUT", "/api/_mod/ai-access", "did:plc:admin", `{"did":"not a DID"}`, "same-origin").Code)
	require.Equal(t, 400, request("PUT", "/api/_mod/ai-access", "did:plc:admin", `{"did":"`+bodyLimitDID+`"} {}`, "same-origin").Code)
	for i := 0; i < 2; i++ {
		require.Equal(t, 200, request("PUT", "/api/_mod/ai-access", "did:plc:admin", `{"did":"`+bodyLimitDID+`"}`, "same-origin").Code)
	}
	checkAccess(bodyLimitDID, true)
	require.Equal(t, 200, request("POST", "/api/ai/chat", bodyLimitDID, `{"session_id":"session","request_id":"request","revision":0,"messages":[{"role":"user","content":"18g"}],"draft":{"kind":"brew","fields":{}}}`, "same-origin").Code)
	require.Equal(t, 403, request("POST", "/api/ai/chat", bodyLimitDID, `{}`, "cross-site").Code)
	list := request("GET", "/api/_mod/ai-access", "did:plc:moderator", "", "")
	require.Equal(t, 200, list.Code)
	require.Contains(t, list.Body.String(), `"granted_by":"did:plc:admin"`)
	require.Contains(t, list.Body.String(), `"moderators":`)
	require.Contains(t, list.Body.String(), `"did":"did:plc:moderator"`)
	for i := 0; i < 2; i++ {
		require.Equal(t, 200, request("DELETE", "/api/_mod/ai-access", "did:plc:moderator", `{"did":"`+bodyLimitDID+`"}`, "same-origin").Code)
	}
	checkAccess(bodyLimitDID, false)
	for _, did := range []string{"did:plc:admin", "did:plc:moderator"} {
		require.NoError(t, access.Grant(context.Background(), did, "did:plc:admin"))
		require.Equal(t, 200, request("DELETE", "/api/_mod/ai-access", "did:plc:moderator", `{"did":"`+did+`"}`, "same-origin").Code)
		checkAccess(did, true)
	}
	require.NoError(t, access.Grant(context.Background(), bodyLimitDID, "did:plc:admin"))
	require.NoError(t, db.Close())
	checkAccess(bodyLimitDID, false)
	checkAccess("did:plc:admin", true)
	require.Equal(t, 503, request("GET", "/api/_mod/ai-access", "did:plc:moderator", "", "").Code)
}
