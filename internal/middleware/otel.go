package middleware

import (
	"net/http"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	atpmiddleware "tangled.org/pdewey.com/atp/middleware"
)

// UserDIDSpanMiddleware adds the authenticated user's DID to the active HTTP
// server span. Place it inside the auth middleware so the DID has already been
// added to the request context.
func UserDIDSpanMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if did, ok := atpmiddleware.GetDID(r.Context()); ok {
			trace.SpanFromContext(r.Context()).SetAttributes(attribute.String("user.did", did))
		}

		next.ServeHTTP(w, r)
	})
}
