const DESKTOP_QUERY = '(min-width: 768px)';
const CARD_ROTATIONS = [-0.8, -0.5, -0.3, 0, 0.3, 0.5, 0.7] as const;

declare global {
  interface Window {
    __arabicaApplyFeedMasonry?: () => void;
  }
}

let mediaQuery: MediaQueryList | undefined;
let installed = false;

function getContainers(): HTMLElement[] {
  const marked = Array.from(document.querySelectorAll<HTMLElement>('[data-feed-masonry]'));
  const feed = document.getElementById('feed-items');
  if (feed && !marked.includes(feed)) {
    marked.unshift(feed);
  }
  return marked;
}

function getCardSelector(container: HTMLElement): string {
  return container.dataset.masonryCard || '.feed-card';
}

function isCard(container: HTMLElement, element: Element): element is HTMLElement {
  return element instanceof HTMLElement && element.matches(getCardSelector(container));
}

function getLooseCards(container: HTMLElement): HTMLElement[] {
  return Array.from(container.children).filter((child): child is HTMLElement => {
    return isCard(container, child);
  });
}

function assignRotation(card: HTMLElement) {
  if (card.style.getPropertyValue('--card-rotate')) {
    return;
  }
  const deg = CARD_ROTATIONS[Math.floor(Math.random() * CARD_ROTATIONS.length)];
  card.style.setProperty('--card-rotate', `${deg}deg`);
}

function ensureColumns(container: HTMLElement, firstCard: HTMLElement): [HTMLElement, HTMLElement] {
  let columns = Array.from(container.querySelectorAll<HTMLElement>(':scope > .feed-masonry-col'));
  for (let index = columns.length; index < 2; index += 1) {
    const column = document.createElement('div');
    column.className = 'feed-masonry-col';
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
  const columns = Array.from(container.querySelectorAll<HTMLElement>(':scope > .feed-masonry-col'));
  if (columns.length === 0) {
    return;
  }

  const merged: HTMLElement[] = [];
  const left = Array.from(columns[0]?.children ?? []).filter((child): child is HTMLElement => child instanceof HTMLElement);
  const right = Array.from(columns[1]?.children ?? []).filter((child): child is HTMLElement => child instanceof HTMLElement);
  const max = Math.max(left.length, right.length);

  for (let index = 0; index < max; index += 1) {
    if (left[index]) merged.push(left[index]);
    if (right[index]) merged.push(right[index]);
  }

  const ref = Array.from(container.children).find((child) => {
    return !child.classList.contains('feed-masonry-col') && !isCard(container, child);
  }) ?? null;

  for (const card of merged) {
    container.insertBefore(card, ref);
  }
  for (const column of columns) {
    column.remove();
  }
}

export function applyFeedMasonry() {
  mediaQuery ??= window.matchMedia(DESKTOP_QUERY);
  for (const container of getContainers()) {
    if (mediaQuery.matches) {
      masonryLayout(container);
    } else {
      flattenLayout(container);
    }
  }
}

function scheduleLayout() {
  requestAnimationFrame(applyFeedMasonry);
}

export function installFeedMasonry() {
  mediaQuery ??= window.matchMedia(DESKTOP_QUERY);
  window.__arabicaApplyFeedMasonry = applyFeedMasonry;

  if (installed) {
    return () => {};
  }
  installed = true;
  mediaQuery.addEventListener('change', applyFeedMasonry);
  document.addEventListener('htmx:afterSettle', scheduleLayout);

  return () => {
    mediaQuery?.removeEventListener('change', applyFeedMasonry);
    document.removeEventListener('htmx:afterSettle', scheduleLayout);
    installed = false;
  };
}
