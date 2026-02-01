/**
 * Arabica Transitions
 * Provides smooth page transitions using HTMX and View Transitions API
 */

(function () {
  "use strict";

  // Force visibility on page load (handles cache restoration issues)
  function forceContentVisibility() {
    const elementsToShow = [
      document.body,
      document.querySelector("main"),
      document.querySelector("main > *"),
    ].filter(Boolean);

    elementsToShow.forEach((el) => {
      if (el) {
        el.classList.remove(
          "htmx-swapping",
          "htmx-transitioning",
          "htmx-settling",
          "htmx-added",
          "transitioning"
        );
        el.style.opacity = "1";
        el.style.transform = "none";
        el.style.visibility = "visible";
      }
    });
  }

  // Initialize after DOM is ready
  function init() {
    // Force content visibility immediately (in case page loaded from cache)
    forceContentVisibility();

    // Disable View Transitions API to avoid double-fade issues
    // Using component-level animations instead for better control
    if (typeof htmx !== "undefined") {
      htmx.config.globalViewTransitions = false;
    }

    // Add transition classes to HTMX requests (only if body exists)
    if (document.body) {
      document.body.addEventListener("htmx:beforeRequest", function (evt) {
        const target = evt.detail?.target;
        if (target) {
          target.classList.add("htmx-transitioning");
        }
      });

      document.body.addEventListener("htmx:afterSwap", function (evt) {
        const target = evt.detail?.target;
        if (target) {
          target.classList.remove("htmx-transitioning");
        }
      });
    }

    // Handle browser back/forward cache (bfcache) restoration
    window.addEventListener("pageshow", function (evt) {
      // If page is loaded from bfcache (browser cache), force visibility
      if (evt.persisted) {
        forceContentVisibility();
      }
    });

    // Handle back/forward button transitions
    // CRITICAL: HTMX history restoration needs explicit handling
    window.addEventListener("popstate", function () {
      // Force visibility immediately
      forceContentVisibility();
    });

    // Pre-handle before history restoration to prevent content from being hidden
    document.body.addEventListener("htmx:beforeHistoryRestore", function (evt) {
      // Add marker class to disable all transitions during history restore
      document.body.classList.add("htmx-history-restoring");

      // Prevent any transition classes from being applied during restore
      const main = document.querySelector("main");
      if (main) {
        main.style.opacity = "1";
        main.style.transform = "none";
        main.style.visibility = "visible";
      }
    });

    // Handle HTMX history restoration (back/forward navigation)
    document.body.addEventListener("htmx:historyRestore", function (evt) {
      // CRITICAL: Ensure content is visible after history restore
      const target = evt.detail?.target || document.body;

      // Immediately force visibility on all potentially hidden elements
      const elementsToShow = [
        target,
        document.body,
        document.querySelector("main"),
        document.querySelector("main > *"),
      ].filter(Boolean);

      elementsToShow.forEach((el) => {
        if (el) {
          // Remove all transition classes
          el.classList.remove(
            "htmx-swapping",
            "htmx-transitioning",
            "htmx-settling",
            "htmx-added",
            "transitioning"
          );
          // Force visibility
          el.style.opacity = "1";
          el.style.transform = "none";
          el.style.visibility = "visible";
        }
      });

      // Safeguard: Force visibility again after animations complete
      setTimeout(() => {
        elementsToShow.forEach((el) => {
          if (el) {
            el.style.opacity = "1";
            el.style.transform = "none";
            el.style.visibility = "visible";
          }
        });

        // Remove history restoring marker class
        document.body.classList.remove("htmx-history-restoring");
      }, 50);
    });

    // Ensure content is visible after any HTMX swap completes
    document.body.addEventListener("htmx:afterSettle", function (evt) {
      const target = evt.detail?.target;
      if (target) {
        // Clear any transition styles that might hide content
        target.style.opacity = "";
        target.style.transform = "";
      }
    });
  }

  // Wait for DOM to be ready
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }

  // Safety net: Check and fix content visibility periodically after page load
  // This catches any edge cases where content gets stuck hidden
  window.addEventListener("load", function () {
    setTimeout(forceContentVisibility, 100);
    setTimeout(forceContentVisibility, 500);
  });

  // Re-initialize Alpine after HTMX swaps content (needed for profile page)
  if (document.body && typeof Alpine !== 'undefined') {
    document.body.addEventListener('htmx:afterSwap', function(evt) {
      if (evt.target && evt.target.id === 'profile-content') {
        Alpine.initTree(evt.target);
      }
    });
  }
})();
