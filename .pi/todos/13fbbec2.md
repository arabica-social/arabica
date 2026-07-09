{
  "id": "13fbbec2",
  "title": "P1.12 — Admin JSON",
  "tags": [
    "stream-a",
    "p1.12",
    "admin"
  ],
  "status": "done",
  "created_at": "2026-07-09T02:14:32.749Z"
}

GET /api/_mod -> AdminProps as JSON. GET /api/_mod/stats -> AdminStats + backups. POST /_mod/* (hide, unhide, dismiss-report, block, unblock, reset-autohide, label add/remove) -> {ok: true, action: "..."} via content negotiation. File: internal/handlers/admin.go.
