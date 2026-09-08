# Homepage Open Graph contract

`GET /` serves the SvelteKit shell as `text/html; charset=utf-8`. Its initial
HTML head includes the brand title and description, `og:type=website`, a
canonical absolute root `og:url`, and an absolute `og:image` pointing to
`/og-image`. Query parameters do not change these URLs. Twitter metadata uses
`summary_large_image` and the same image URL. Crawlers do not need JavaScript.

URLs use the configured public URL when set, otherwise the existing
`PublicBaseURL` request-origin fallback. Configure `ARABICA_PUBLIC_URL` for the
public origin when deploying behind a proxy.

`GET /og-image` is public. It generates the app's branded 1200×630 PNG with
embedded fonts and logo. Successful responses use `Content-Type: image/png`
and `Cache-Control: public, max-age=86400`. The image endpoint is independent
of SPA page routing.

The HTTP regression in `internal/atplatform/server/og_test.go` connects the
production resolver, embedded shell, Arabica router, and PNG renderer.
