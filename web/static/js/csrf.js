/**
 * CSRF Token Helper
 * 
 * Provides functions to get the CSRF token from the cookie and
 * automatically configures HTMX to include the token on all requests.
 * 
 * Usage:
 *   // Get token manually for fetch calls
 *   const token = getCSRFToken();
 *   
 *   // Manual fetch with CSRF header
 *   fetch('/api/beans', {
 *       method: 'POST',
 *       headers: { 
 *           'Content-Type': 'application/json',
 *           'X-CSRF-Token': getCSRFToken() 
 *       },
 *       body: JSON.stringify(data)
 *   });
 */

/**
 * Get CSRF token from cookie
 * @returns {string} The CSRF token or empty string if not found
 */
function getCSRFToken() {
    const name = 'csrf_token=';
    const decodedCookie = decodeURIComponent(document.cookie);
    const cookies = decodedCookie.split(';');
    
    for (let cookie of cookies) {
        cookie = cookie.trim();
        if (cookie.indexOf(name) === 0) {
            return cookie.substring(name.length);
        }
    }
    return '';
}

/**
 * Configure HTMX to automatically include CSRF token on all requests
 * This handles all HTMX requests (hx-get, hx-post, hx-put, hx-delete, etc.)
 */
document.addEventListener('DOMContentLoaded', function() {
    // Add CSRF token header to all HTMX requests
    document.body.addEventListener('htmx:configRequest', function(event) {
        const token = getCSRFToken();
        if (token) {
            event.detail.headers['X-CSRF-Token'] = token;
        }
    });
    
    // Populate hidden CSRF token fields in forms
    const token = getCSRFToken();
    if (token) {
        document.querySelectorAll('.csrf-token-field').forEach(function(field) {
            field.value = token;
        });
    }
});

// Export for use in other modules (if using module system)
// For non-module scripts, getCSRFToken is available as a global
if (typeof window !== 'undefined') {
    window.getCSRFToken = getCSRFToken;
}
