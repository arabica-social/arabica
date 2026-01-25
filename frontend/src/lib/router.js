import navaid from 'navaid';

/**
 * Simple client-side router using navaid
 * Handles browser history and navigation
 */
const router = navaid('/');

/**
 * Navigate to a route programmatically
 * @param {string} path - Route path
 */
export function navigate(path) {
  router.route(path);
}

/**
 * Navigate back in history
 */
export function back() {
  window.history.back();
}

export default router;
