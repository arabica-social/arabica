/**
 * Arabica Transitions
 * Provides smooth page transitions using HTMX and View Transitions API
 */

(function () {
  "use strict";

  // Check if View Transitions API is supported
  const supportsViewTransitions = "startViewTransition" in document;

  // Initialize after DOM is ready
  function init() {
    // Disable View Transitions API to avoid double-fade issues
    // Using component-level animations instead for better control
    if (typeof htmx !== 'undefined') {
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

    // Handle back/forward button transitions
    window.addEventListener("popstate", function () {
      if (supportsViewTransitions) {
        // View Transitions API will handle this automatically
        // No need to manually add classes
        return;
      }

      // Fallback for browsers without View Transitions API
      if (document.body) {
        document.body.classList.add("transitioning");
        setTimeout(() => {
          document.body.classList.remove("transitioning");
        }, 300);
      }
    });

    // Add page transition class when links are clicked
    document.addEventListener("click", function (e) {
      const link = e.target.closest("a[href]");
      if (
        link &&
        !link.hasAttribute("hx-get") &&
        !link.hasAttribute("hx-boost") &&
        !link.hasAttribute("target") &&
        link.href.startsWith(window.location.origin)
      ) {
        // Only for same-origin links that aren't HTMX or external
        if (!e.ctrlKey && !e.metaKey && !e.shiftKey && document.body) {
          document.body.classList.add("transitioning");
        }
      }
    });
  }

  // Utility: Add fade transition to an element
  window.fadeIn = function (element, duration = 300) {
    element.style.opacity = "0";
    element.style.transition = `opacity ${duration}ms ease-in-out`;

    requestAnimationFrame(() => {
      element.style.opacity = "1";
    });
  };

  // Utility: Add slide-in transition to an element
  window.slideIn = function (element, direction = "up", duration = 300) {
    const translations = {
      up: "translateY(20px)",
      down: "translateY(-20px)",
      left: "translateX(20px)",
      right: "translateX(-20px)",
    };

    element.style.opacity = "0";
    element.style.transform = translations[direction];
    element.style.transition = `opacity ${duration}ms ease-out, transform ${duration}ms ease-out`;

    requestAnimationFrame(() => {
      element.style.opacity = "1";
      element.style.transform = "translate(0, 0)";
    });
  };

  // Wait for DOM to be ready
  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", init);
  } else {
    init();
  }
})();
