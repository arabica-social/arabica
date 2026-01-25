const CACHE_VERSION = "v1";
const CACHE_NAMES = {
  static: `arabica-static-${CACHE_VERSION}`,
  dynamic: `arabica-dynamic-${CACHE_VERSION}`,
  api: `arabica-api-${CACHE_VERSION}`,
};

// Resources to cache on install
const staticAssets = [
  "/",
  "/static/app/index.html",
  "/static/manifest.json",
  "/static/favicon.svg",
  "/static/icon-192.svg",
  "/static/icon-512.svg",
];

// Install service worker - cache static assets
self.addEventListener("install", (event) => {
  console.log("[SW] Installing service worker");
  event.waitUntil(
    caches.open(CACHE_NAMES.static).then((cache) => {
      console.log("[SW] Caching static assets");
      return cache.addAll(staticAssets).catch((err) => {
        console.warn("[SW] Failed to cache some assets:", err);
        // Don't fail install if some assets can't be cached
        return Promise.resolve();
      });
    }),
  );
  self.skipWaiting(); // Activate new service worker immediately
});

// Fetch event - implement cache strategies
self.addEventListener("fetch", (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Skip cross-origin requests
  if (url.origin !== self.location.origin) {
    return;
  }

  // API requests: network-first with cache fallback
  if (url.pathname.startsWith("/api/")) {
    event.respondWith(networkFirstStrategy(request, CACHE_NAMES.api));
    return;
  }

  // Static assets: cache-first with network fallback
  if (
    url.pathname.includes("/assets/") ||
    url.pathname.endsWith(".svg") ||
    url.pathname.endsWith(".css") ||
    url.pathname.endsWith(".js")
  ) {
    event.respondWith(cacheFirstStrategy(request, CACHE_NAMES.static));
    return;
  }

  // HTML documents: network-first with cache fallback
  if (request.method === "GET" && request.headers.get("accept")?.includes("text/html")) {
    event.respondWith(networkFirstStrategy(request, CACHE_NAMES.dynamic));
    return;
  }

  // Default: try network, fallback to cache
  event.respondWith(networkFirstStrategy(request, CACHE_NAMES.dynamic));
});

// Activate service worker - clean up old caches
self.addEventListener("activate", (event) => {
  console.log("[SW] Activating service worker");
  event.waitUntil(
    caches.keys().then((cacheNames) => {
      const cacheWhitelist = Object.values(CACHE_NAMES);
      return Promise.all(
        cacheNames
          .filter((cacheName) => !cacheWhitelist.includes(cacheName))
          .map((cacheName) => {
            console.log("[SW] Deleting old cache:", cacheName);
            return caches.delete(cacheName);
          }),
      );
    }),
  );
  self.clients.claim(); // Take control of all pages immediately
});

// Cache-first strategy: use cache, fallback to network
async function cacheFirstStrategy(request, cacheName) {
  const cache = await caches.open(cacheName);
  const cached = await cache.match(request);

  if (cached) {
    return cached;
  }

  try {
    const response = await fetch(request);
    if (response.ok) {
      cache.put(request, response.clone());
    }
    return response;
  } catch (error) {
    console.warn("[SW] Fetch failed for", request.url, error);
    // Return offline page if available
    return cache.match("/") || new Response("Offline - content not available", {
      status: 503,
      statusText: "Service Unavailable",
      headers: { "Content-Type": "text/plain" },
    });
  }
}

// Network-first strategy: try network, fallback to cache
async function networkFirstStrategy(request, cacheName) {
  const cache = await caches.open(cacheName);

  try {
    const response = await fetch(request);
    if (response.ok) {
      cache.put(request, response.clone());
    }
    return response;
  } catch (error) {
    console.warn("[SW] Network request failed for", request.url, error);
    const cached = await cache.match(request);
    if (cached) {
      return cached;
    }

    // Fallback for failed requests
    return new Response(
      JSON.stringify({ error: "Offline - unable to fetch data" }),
      {
        status: 503,
        statusText: "Service Unavailable",
        headers: { "Content-Type": "application/json" },
      },
    );
  }
}
