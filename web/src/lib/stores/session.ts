// Session store for the SvelteKit SPA.
//
// Reads the data-* attributes the Go shell injects on <body>:
//   data-user-did, data-user-handle, data-user-display, data-user-avatar,
//   data-is-moderator, data-unread-notifications, data-app
//
// These are populated server-side by the ShellHandler (with a
// SessionResolver) so the header renders the authenticated state on first
// paint without an extra API call. The values are static for the lifetime
// of the page — a full reload refreshes them (e.g. after login/logout).

import { writable, derived } from "svelte/store";

declare global {
  interface Window {
    /** Exposed by +layout.svelte to open the login modal from anywhere. */
    __showLoginModal?: () => void;
  }
}

export type Session = {
  did: string;
  handle: string;
  displayName: string;
  avatar: string;
  isAuthenticated: boolean;
  isModerator: boolean;
  unreadNotifications: number;
  /** Preferred brew temperature display unit ("recorded", "celsius", "fahrenheit"). */
  temperatureUnit: string;
};

export type AppName = "arabica" | "oolong";

function readBody(): HTMLElement {
  if (typeof document === "undefined") {
    return { dataset: {} } as unknown as HTMLElement;
  }
  return document.body;
}

function readSession(): Session {
  const body = readBody();
  const ds = body.dataset;
  const did = ds.userDid ?? "";
  return {
    did,
    handle: ds.userHandle ?? "",
    displayName: ds.userDisplay ?? "",
    avatar: ds.userAvatar ?? "",
    isAuthenticated: did !== "",
    isModerator: ds.isModerator === "true",
    unreadNotifications: Number(ds.unreadNotifications ?? "0") || 0,
    temperatureUnit: ds.temperatureUnit ?? "recorded",
  };
}

function readApp(): AppName {
  const body = readBody();
  if (body.dataset.app === "arabica" || body.dataset.app === "oolong") {
    return body.dataset.app;
  }
  return import.meta.env.VITE_APP === "oolong" ? "oolong" : "arabica";
}

export const session = writable<Session>(readSession());
export const app = writable<AppName>(readApp());

type SessionResponse = {
  did?: string;
  handle?: string;
  display_name?: string;
  avatar?: string;
  is_authenticated?: boolean;
  is_moderator?: boolean;
  unread_notifications?: number;
  temperature_unit?: string;
  app?: AppName;
};

/**
 * Re-read the body data attributes, or fetch their API equivalent in Vite
 * development. Production keeps the server-injected values and avoids an
 * additional request on the initial paint.
 */
export async function refreshSession() {
  const bodySession = readSession();
  const bodyApp = readApp();
  const hasServerShellData =
    typeof document !== "undefined" &&
    (document.body.dataset.app !== undefined || bodySession.did !== "");
  if (hasServerShellData || !import.meta.env.DEV) {
    session.set(bodySession);
    app.set(bodyApp);
    return;
  }

  try {
    const response = await fetch("/api/session", { credentials: "same-origin" });
    if (!response.ok) throw new Error(`Session request failed: ${response.status}`);
    const data = (await response.json()) as SessionResponse;
    session.set({
      did: data.did ?? "",
      handle: data.handle ?? "",
      displayName: data.display_name ?? "",
      avatar: data.avatar ?? "",
      isAuthenticated: data.is_authenticated ?? false,
      isModerator: data.is_moderator ?? false,
      unreadNotifications: data.unread_notifications ?? 0,
      temperatureUnit: data.temperature_unit || "recorded",
    });
    app.set(data.app ?? bodyApp);
  } catch (error) {
    console.warn("Unable to load development session state:", error);
    session.set(bodySession);
    app.set(bodyApp);
  }
}

/** Returns the profile identifier (handle preferred, DID fallback) for URLs. */
export function profileIdentifier(s: Session = readSession()): string {
  if (s.handle) return s.handle;
  return s.did;
}

/**
 * Sanitizes avatar URLs client-side, mirroring bff.SafeAvatarURL on the
 * server. Only allows HTTPS URLs from the Bluesky CDN, or relative
 * /static/ paths. Returns "" for anything else (the avatar component
 * renders a placeholder in that case).
 */
export function safeAvatarURL(avatarURL: string): string {
  if (!avatarURL) return "";
  if (avatarURL.startsWith("/static/")) return avatarURL;
  if (avatarURL.startsWith("/")) return "";
  let parsed: URL;
  try {
    parsed = new URL(avatarURL);
  } catch {
    return "";
  }
  if (parsed.protocol !== "https:") {
    return "";
  }
  const trusted = ["cdn.bsky.app", "av-cdn.bsky.app"];
  const host = parsed.hostname.toLowerCase();
  for (const domain of trusted) {
    if (host === domain || host.endsWith(`.${domain}`)) return avatarURL;
  }
  return "";
}

/** Formats a handle for display, stripping a leading @ and lowercasing. */
export function displayHandle(handle: string): string {
  if (!handle) return "";
  let h = handle;
  if (h.startsWith("@")) h = h.slice(1);
  return h;
}

/** Formats an unread notification count as "99+" for large numbers. */
export function formatNotificationCount(count: number): string {
  if (count > 99) return "99+";
  return String(count);
}

/** Opens the global login modal if the layout has mounted it. */
export function openLoginModal() {
  if (typeof window !== "undefined" && typeof window.__showLoginModal === "function") {
    window.__showLoginModal();
  }
}
