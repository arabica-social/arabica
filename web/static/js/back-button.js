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

    button.addEventListener('click', function(e) {
        e.preventDefault();
        handleBackNavigation(button);
    });
}

/**
 * Handle back navigation with fallback logic
 * @param {HTMLElement} button - The back button element
 */
function handleBackNavigation(button) {
    const fallbackUrl = button.getAttribute('data-fallback') || '/brews';
    const referrer = document.referrer;
    const currentUrl = window.location.href;

    // Check if there's actual browser history to go back to
    // We can't directly check history.length in a reliable way across browsers,
    // but we can check if the referrer is from the same origin
    const hasSameOriginReferrer = referrer && 
                                   referrer.startsWith(window.location.origin) &&
                                   referrer !== currentUrl;

    if (hasSameOriginReferrer) {
        // Safe to use history.back() - we came from within the app
        window.history.back();
    } else {
        // No referrer or external referrer - use fallback
        // This handles direct links, external referrers, and bookmarks
        window.location.href = fallbackUrl;
    }
}

/**
 * Initialize all back buttons on the page
 */
function initAllBackButtons() {
    const buttons = document.querySelectorAll('[data-back-button]');
    buttons.forEach(initBackButton);
}

// Initialize on DOM load
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initAllBackButtons);
} else {
    initAllBackButtons();
}

// Re-initialize after HTMX swaps (for dynamic content)
document.body.addEventListener('htmx:afterSwap', function() {
    initAllBackButtons();
});
