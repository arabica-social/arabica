/**
 * API client for communicating with Go backend
 * Handles authentication, errors, and JSON serialization
 */

class APIError extends Error {
  constructor(message, status, response) {
    super(message);
    this.name = "APIError";
    this.status = status;
    this.response = response;
  }
}

/**
 * Make an authenticated API request
 * @param {string} endpoint - API endpoint (e.g., '/api/brews')
 * @param {object} options - Fetch options
 * @returns {Promise<any>} Response data
 */
async function request(endpoint, options = {}) {
  const config = {
    credentials: "same-origin", // Send cookies
    headers: {
      "Content-Type": "application/json",
      ...options.headers,
    },
    ...options,
  };

  try {
    const response = await fetch(endpoint, config);

    // Handle 401/403 - but only redirect if not on public endpoints or pages
    if (response.status === 401 || response.status === 403) {
      // Don't redirect if:
      // 1. Already on public pages
      // 2. Calling public API endpoints (feed, resolve-handle, search-actors, me)
      const publicPaths = ["/", "/login", "/about", "/terms"];
      const publicEndpoints = [
        "/api/feed-json",
        "/api/resolve-handle",
        "/api/search-actors",
        "/api/me",
      ];
      const currentPath = window.location.pathname;
      const isPublicEndpoint = publicEndpoints.some((path) =>
        endpoint.includes(path),
      );

      if (!publicPaths.includes(currentPath) && !isPublicEndpoint) {
        window.location.href = "/login";
      }

      throw new APIError("Authentication required", response.status, response);
    }

    // Handle non-OK responses
    if (!response.ok) {
      const text = await response.text();
      throw new APIError(
        text || `Request failed: ${response.statusText}`,
        response.status,
        response,
      );
    }

    // Handle empty responses (e.g., 204 No Content)
    const contentType = response.headers.get("content-type");
    if (!contentType || !contentType.includes("application/json")) {
      return null;
    }

    return await response.json();
  } catch (error) {
    if (error instanceof APIError) {
      throw error;
    }
    throw new APIError(`Network error: ${error.message}`, 0, null);
  }
}

export const api = {
  // GET request
  get: (endpoint) => request(endpoint, { method: "GET" }),

  // POST request
  post: (endpoint, data) =>
    request(endpoint, {
      method: "POST",
      body: JSON.stringify(data),
    }),

  // PUT request
  put: (endpoint, data) =>
    request(endpoint, {
      method: "PUT",
      body: JSON.stringify(data),
    }),

  // DELETE request
  delete: (endpoint) => request(endpoint, { method: "DELETE" }),
};

export { APIError };
