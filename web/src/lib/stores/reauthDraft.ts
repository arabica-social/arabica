import { onMount, tick } from "svelte";

export const reauthLanding = "/?resume_reauth=1";
const storageKey = "arabica:reauth-draft:v1";
const maxAge = 30 * 60 * 1000;
const maxSize = 128 * 1024;

type Recovery = {
  did: string;
  path: string;
  expires: number;
  drafts: Record<string, unknown>;
};

type Draft = {
  capture: () => unknown;
  restore: (value: unknown) => void;
};

const mounted = new Map<string, Draft>();
let recovery: Recovery | undefined;
let leavingForReauth = false;

export function currentPath(): string {
  const { pathname, search, hash } = window.location;
  return pathname + search + hash;
}

export function registerReauthDraft<T extends Record<string, unknown>>(
  key: string,
  read: () => T,
  apply: (value: T) => void,
  changed?: () => boolean,
) {
  onMount(() => {
    const initial = changed ? undefined : JSON.stringify(read());
    const draft: Draft = {
      capture() {
        if (changed && !changed()) return undefined;
        const value = read();
        return JSON.stringify(value) !== initial ? value : undefined;
      },
      restore(value) {
        if (value && typeof value === "object" && !Array.isArray(value))
          apply(value as T);
      },
    };
    mounted.set(key, draft);
    restoreDraft(key, draft);
    return () => {
      if (mounted.get(key) === draft) mounted.delete(key);
    };
  });
}

function restoreDraft(key: string, draft: Draft) {
  if (recovery?.path !== currentPath() || !Object.hasOwn(recovery.drafts, key))
    return;
  const value = recovery.drafts[key];
  delete recovery.drafts[key];
  draft.restore(value);
}

export function captureReauth(did: string): {
  stored: boolean;
  hasEdits: boolean;
} {
  const drafts = Object.fromEntries(
    [...mounted]
      .map(([key, draft]) => [key, draft.capture()])
      .filter(([, value]) => value !== undefined),
  );
  const hasEdits = Object.keys(drafts).length > 0;
  try {
    sessionStorage.removeItem(storageKey);
    if (did) {
      const raw = JSON.stringify({
        did,
        path: currentPath(),
        expires: Date.now() + maxAge,
        drafts,
      } satisfies Recovery);
      if (raw.length <= maxSize) {
        sessionStorage.setItem(storageKey, raw);
        return { stored: true, hasEdits };
      }
    }
  } catch {
    return { stored: false, hasEdits };
  }
  return { stored: false, hasEdits };
}

function takeRecovery(did: string): Recovery | undefined {
  try {
    const raw = sessionStorage.getItem(storageKey);
    sessionStorage.removeItem(storageKey);
    if (!did || !raw || raw.length > maxSize) return;
    const saved = JSON.parse(raw) as Recovery;
    if (
      saved?.did !== did ||
      !Number.isFinite(saved.expires) ||
      saved.expires <= Date.now() ||
      typeof saved.path !== "string" ||
      !saved.path.startsWith("/") ||
      saved.path.startsWith("//") ||
      saved.path === reauthLanding ||
      !saved.drafts ||
      typeof saved.drafts !== "object" ||
      Array.isArray(saved.drafts)
    )
      return;
    const url = new URL(saved.path, window.location.origin);
    if (url.origin !== window.location.origin) return;
    if (url.pathname + url.search + url.hash !== saved.path) return;
    return saved;
  } catch {
    return;
  }
}

export async function resumeReauth(
  did: string,
  navigate: (path: string) => Promise<void>,
) {
  if (currentPath() !== reauthLanding) return;
  recovery = takeRecovery(did);
  try {
    await navigate(recovery?.path ?? "/");
    for (const [key, draft] of mounted) restoreDraft(key, draft);
    await tick();
  } finally {
    recovery = undefined;
  }
}

export function allowReauthUnload() {
  leavingForReauth = true;
}

export function consumeReauthUnload(): boolean {
  const leaving = leavingForReauth;
  leavingForReauth = false;
  return leaving;
}
