/**
 * Smart back button implementation for Arabica
 * Handles browser history navigation with intelligent fallbacks
 */

/**
 * Initialize a back button with smart navigation
 * @param {HTMLElement} button - The back button element
 */
function initBackButton(button) {
  if (!button) return;

  button.addEventListener("click", function (e) {
    e.preventDefault();
    handleBackNavigation(button);
  });
}

/**
 * Handle back navigation with fallback logic
 * @param {HTMLElement} button - The back button element
 */
function handleBackNavigation(button) {
  const fallbackUrl = button.getAttribute("data-fallback") || "/brews";
  const referrer = document.referrer;
  const currentUrl = window.location.href;

  // Check if there's actual browser history to go back to
  // We use both referrer AND history.length to determine if we can safely go back
  const hasSameOriginReferrer =
    referrer &&
    referrer.startsWith(window.location.origin) &&
    referrer !== currentUrl;

  // history.length > 2 means there's at least one page before the current page
  // (length includes current page + previous pages)
  const hasHistoryDepth = window.history.length > 2;

  // Only use history.back() if we have both a same-origin referrer AND history depth
  // Otherwise, use the fallback URL to prevent blank pages
  if (hasSameOriginReferrer && hasHistoryDepth) {
    window.history.back();
  } else {
    window.location.href = fallbackUrl;
  }
}

/**
 * Initialize all back buttons on the page
 */
function initAllBackButtons() {
  const buttons = document.querySelectorAll("[data-back-button]");
  buttons.forEach(initBackButton);
}

// Initialize on DOM load
if (document.readyState === "loading") {
  document.addEventListener("DOMContentLoaded", initAllBackButtons);
} else {
  initAllBackButtons();
}

// Re-initialize after HTMX swaps (for dynamic content)
if (document.body) {
  document.body.addEventListener("htmx:afterSwap", function () {
    initAllBackButtons();
  });
}
