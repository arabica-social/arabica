// Service Worker Registration
// This is loaded as an external script to comply with strict CSP
// Note: Service workers require a secure context (HTTPS) or localhost/127.0.0.1 for development

if ('serviceWorker' in navigator) {
  window.addEventListener('load', () => {
    navigator.serviceWorker
      .register('/static/service-worker.js', { scope: '/' })
      .then((registration) => {
        console.log('[SW] Service Worker registered:', registration);
        
        // Check for updates periodically (every 60 seconds)
        setInterval(() => {
          registration.update();
        }, 60000);
      })
      .catch((err) => {
        console.error('[SW] Service Worker registration failed:', err);
        
        // Log additional context for debugging
        if (err instanceof DOMException && err.name === 'SecurityError') {
          console.warn('[SW] Service Worker requires a secure context (HTTPS) or localhost');
          console.warn('[SW] For development, access via http://localhost:18910 instead of 127.0.0.1');
        }
      });
  });
} else {
  console.warn('[SW] Service Workers not supported in this browser');
}
