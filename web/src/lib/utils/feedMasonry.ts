// Feed masonry layout for the SvelteKit SPA.
//
// Ported from the legacy HTMX island (internal/web/assets/svelte/src/feedMasonry.ts)
// with the HTMX/`window.__arabicaApplyFeedMasonry` bridge removed — the SPA drives
// re-layout through Svelte reactivity instead of `htmx:afterSettle`.
//
// The layout is a two-column "corkboard": at desktop width (≥768px) loose
// `.feed-card` children are distributed into two `.feed-masonry-col` flex
// columns by shortest-column height, and each card gets a subtle random
// rotation via the `--card-rotate` custom property. Below the breakpoint the
// layout flattens back to the grid's single column. See
// internal/web/assets/css/components/10-feed.css for the matching styles.

const DESKTOP_QUERY = "(min-width: 768px)";
const CARD_ROTATIONS = [-0.8, -0.5, -0.3, 0, 0.3, 0.5, 0.7] as const;

/** Returns the desktop media query list, or null if matchMedia is absent
 *  (e.g. in jsdom test environments without a matchMedia polyfill). */
function desktopQuery(): MediaQueryList | null {
  if (typeof window === "undefined" || !window.matchMedia) return null;
  return window.matchMedia(DESKTOP_QUERY);
}

function getContainers(): HTMLElement[] {
  const marked = Array.from(
    document.querySelectorAll<HTMLElement>("[data-feed-masonry]"),
  );
  const feed = document.getElementById("feed-items");
  if (feed && !marked.includes(feed)) {
    marked.unshift(feed);
  }
  return marked;
}

function getCardSelector(container: HTMLElement): string {
  return container.dataset.masonryCard || ".feed-card";
}

function isCard(
  container: HTMLElement,
  element: Element,
): element is HTMLElement {
  return (
    element instanceof HTMLElement &&
    element.matches(getCardSelector(container))
  );
}

function getLooseCards(container: HTMLElement): HTMLElement[] {
  return Array.from(container.children).filter(
    (child): child is HTMLElement => {
      return isCard(container, child);
    },
  );
}

function assignRotation(card: HTMLElement) {
  if (card.style.getPropertyValue("--card-rotate")) {
    return;
  }
  const deg = CARD_ROTATIONS[Math.floor(Math.random() * CARD_ROTATIONS.length)];
  card.style.setProperty("--card-rotate", `${deg}deg`);
}

function ensureColumns(
  container: HTMLElement,
  firstCard: HTMLElement,
): [HTMLElement, HTMLElement] {
  let columns = Array.from(
    container.querySelectorAll<HTMLElement>(":scope > .feed-masonry-col"),
  );
  for (let index = columns.length; index < 2; index += 1) {
    const column = document.createElement("div");
    column.className = "feed-masonry-col";
    container.insertBefore(column, firstCard);
    columns.push(column);
  }
  return [columns[0], columns[1]];
}

function masonryLayout(container: HTMLElement) {
  const cards = getLooseCards(container);
  const firstCard = cards[0];
  if (!firstCard) {
    return;
  }

  const columns = ensureColumns(container, firstCard);
  const heights = columns.map((column) => column.offsetHeight);

  for (const card of cards) {
    assignRotation(card);
    const index = heights[0] <= heights[1] ? 0 : 1;
    columns[index].appendChild(card);
    heights[index] += card.offsetHeight + 20;
  }
}

function flattenLayout(container: HTMLElement) {
  const columns = Array.from(
    container.querySelectorAll<HTMLElement>(":scope > .feed-masonry-col"),
  );
  if (columns.length === 0) {
    return;
  }

  const merged: HTMLElement[] = [];
  const left = Array.from(columns[0]?.children ?? []).filter(
    (child): child is HTMLElement => child instanceof HTMLElement,
  );
  const right = Array.from(columns[1]?.children ?? []).filter(
    (child): child is HTMLElement => child instanceof HTMLElement,
  );
  const max = Math.max(left.length, right.length);

  for (let index = 0; index < max; index += 1) {
    if (left[index]) merged.push(left[index]);
    if (right[index]) merged.push(right[index]);
  }

  // Restore the merged cards to the position the column block occupied
  // (i.e. before the first column). Anchoring on the first column — rather
  // than on the first trailing non-card scaffolding (e.g. the "Load more"
  // button) — keeps newly appended cards after the originals. Otherwise a
  // load-more that adds cards past the columns would leave the merged
  // originals behind the new cards, surfacing older items at the top.
  const ref = columns[0] ?? null;

  for (const card of merged) {
    container.insertBefore(card, ref);
  }
  for (const column of columns) {
    column.remove();
  }
}

/**
 * Applies the masonry layout to every feed container. Call after the cards
 * have been painted (e.g. inside `requestAnimationFrame` or after Svelte's
 * `tick()`), since it measures `offsetHeight`.
 */
export function applyFeedMasonry(mediaQuery?: MediaQueryList) {
  const mq = mediaQuery ?? desktopQuery();
  if (!mq) return;
  for (const container of getContainers()) {
    if (mq.matches) {
      // Start from a clean slate on every re-layout. If the container already
      // contains masonry columns (e.g. after a filter/sort swap replaced the
      // cards), leaving them in place skews the height measurements and can
      // pile every new card into the first column. Flatten first, then repack.
      flattenLayout(container);
      masonryLayout(container);
    } else {
      flattenLayout(container);
    }
  }
}

/**
 * Installs a media-query listener that re-runs `applyFeedMasonry` on
 * viewport/orientation changes across the breakpoint. Returns a teardown
 * that removes the listener. Re-layout on data changes is the caller's
 * responsibility (the SPA drives it via a Svelte `$effect`).
 */
export function installFeedMasonry(): () => void {
  const mediaQuery = desktopQuery();
  if (!mediaQuery) return () => {};
  const onChange = () => applyFeedMasonry(mediaQuery);
  mediaQuery.addEventListener("change", onChange);
  return () => {
    mediaQuery.removeEventListener("change", onChange);
  };
}
