const CACHE_NAME = "arabica-v2";
const STATIC_ASSETS = ["/static/js/htmx.min.js"];

self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(CACHE_NAME).then((cache) => cache.addAll(STATIC_ASSETS)),
  );
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(
    caches
      .keys()
      .then((cacheNames) =>
        Promise.all(
          cacheNames
            .filter((name) => name !== CACHE_NAME)
            .map((name) => caches.delete(name)),
        ),
      )
      .then(() => self.clients.claim()),
  );
});

// Fetch strategy:
// - Navigation requests (HTML pages): network-first with no cache fallback
// - Static assets: stale-while-revalidate
// - API/other: network only (pass through)
self.addEventListener("fetch", (event) => {
  const url = new URL(event.request.url);

  if (url.origin !== self.location.origin) return;

  // This ensures authenticated pages always get the latest head scripts
  if (event.request.mode === "navigate") {
    event.respondWith(fetch(event.request));
    return;
  }

  if (url.pathname.startsWith("/static/")) {
    event.respondWith(
      caches.match(event.request).then((cached) => {
        const fetchPromise = fetch(event.request).then((response) => {
          if (response.ok) {
            const clone = response.clone();
            caches
              .open(CACHE_NAME)
              .then((cache) => cache.put(event.request, clone));
          }
          return response;
        });
        return cached || fetchPromise;
      }),
    );
    return;
  }
});
