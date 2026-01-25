import "./styles.css";
import App from "./App.svelte";

// Register service worker for PWA functionality
// Note: Service workers require a secure context (HTTPS) or localhost/127.0.0.1 for development
if ("serviceWorker" in navigator) {
  // Wait for page load before registering
  window.addEventListener("load", () => {
    navigator.serviceWorker
      .register("/static/service-worker.js", { scope: "/" })
      .then((registration) => {
        console.log("[App] Service Worker registered:", registration);

        // Listen for updates
        registration.addEventListener("updatefound", () => {
          const newWorker = registration.installing;
          newWorker.addEventListener("statechange", () => {
            if (
              newWorker.state === "installed" &&
              navigator.serviceWorker.controller
            ) {
              // New service worker available - notify user
              console.log("[App] New service worker available");
              // Dispatch event to show update notification
              window.dispatchEvent(new CustomEvent("sw-update-available"));
            }
          });
        });

        // Check for updates periodically (every 60 seconds)
        setInterval(() => {
          registration.update();
        }, 60000);
      })
      .catch((err) => {
        console.error("[App] Service Worker registration failed:", err);
        // Log additional context for debugging
        if (err instanceof DOMException && err.name === "SecurityError") {
          console.warn(
            "[App] Service Worker requires a secure context (HTTPS) or localhost",
          );
          console.warn(
            "[App] For development, access via http://localhost:18910 instead of 127.0.0.1",
          );
        }
      });
  });
} else {
  console.warn("[App] Service Workers not supported in this browser");
}

const app = new App({
  target: document.getElementById("app"),
});

export default app;
