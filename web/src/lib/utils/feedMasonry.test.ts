import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { applyFeedMasonry, installFeedMasonry } from "./feedMasonry";

// Builds a fake MediaQueryList whose `.matches` we can toggle.
function makeMediaQuery(matches: boolean): MediaQueryList {
  const listeners = new Set<(e: MediaQueryListEvent) => void>();
  return {
    matches,
    media: "(min-width: 768px)",
    onchange: null,
    addEventListener: (
      _type: string,
      listener: (e: MediaQueryListEvent) => void,
    ) => listeners.add(listener),
    removeEventListener: (
      _type: string,
      listener: (e: MediaQueryListEvent) => void,
    ) => listeners.delete(listener),
    addListener: (l: (e: MediaQueryListEvent) => void) => listeners.add(l),
    removeListener: (l: (e: MediaQueryListEvent) => void) => listeners.delete(l),
    dispatchEvent: () => false,
  } as unknown as MediaQueryList;
}

// jsdom does not implement layout, so offsetHeight is always 0. Override it
// so the shortest-column algorithm makes deterministic choices.
function setHeights(map: Record<string, number>) {
  Object.defineProperty(Element.prototype, "offsetHeight", {
    configurable: true,
    get(this: HTMLElement) {
      // Columns report their accumulated height; cards report a fixed height
      // so the left/right balance is driven by card count.
      if (this.classList.contains("feed-masonry-col")) {
        return Number(this.dataset.height ?? "0");
      }
      return map[this.dataset.cardId ?? ""] ?? 40;
    },
  });
}

function makeFeedGrid(cardIds: string[]): HTMLElement {
  const grid = document.createElement("div");
  grid.id = "feed-items";
  grid.className = "feed-grid";
  for (const id of cardIds) {
    const card = document.createElement("div");
    card.className = "feed-card";
    card.dataset.cardId = id;
    grid.appendChild(card);
  }
  document.body.innerHTML = "";
  document.body.appendChild(grid);
  return grid;
}

describe("feedMasonry", () => {
  beforeEach(() => {
    vi.stubGlobal("matchMedia", () => makeMediaQuery(true));
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    document.body.innerHTML = "";
  });

  it("packs cards into two masonry columns at desktop width", () => {
    setHeights({});
    const grid = makeFeedGrid(["a", "b", "c", "d"]);

    applyFeedMasonry();

    const columns = grid.querySelectorAll<HTMLElement>(":scope > .feed-masonry-col");
    expect(columns.length).toBe(2);
    // All four cards should have moved into the two columns (none left loose).
    const looseCards = Array.from(grid.children).filter(
      (c) => c.classList.contains("feed-card"),
    );
    expect(looseCards.length).toBe(0);
    expect(columns[0].children.length + columns[1].children.length).toBe(4);
  });

  it("assigns a --card-rotate custom property to each card", () => {
    setHeights({});
    makeFeedGrid(["a", "b"]);

    applyFeedMasonry();

    const cards = document.querySelectorAll<HTMLElement>(".feed-card");
    for (const card of cards) {
      expect(card.style.getPropertyValue("--card-rotate")).toMatch(/deg$/);
    }
  });

  it("flattens back to loose cards when below the breakpoint", () => {
    vi.stubGlobal("matchMedia", () => makeMediaQuery(false));
    setHeights({});
    const grid = makeFeedGrid(["a", "b", "c"]);
    // First apply at desktop to build columns, then flatten.
    applyFeedMasonry(makeMediaQuery(true));
    expect(grid.querySelectorAll(":scope > .feed-masonry-col").length).toBe(2);

    applyFeedMasonry(makeMediaQuery(false));

    expect(grid.querySelectorAll(":scope > .feed-masonry-col").length).toBe(0);
    expect(grid.querySelectorAll(":scope > .feed-card").length).toBe(3);
  });

  it("rebalances cards into both columns when re-laying out with existing columns", () => {
    setHeights({ a: 40, b: 40, c: 40, d: 40 });
    const grid = makeFeedGrid(["a", "b"]);
    applyFeedMasonry();
    expect(grid.querySelectorAll(":scope > .feed-masonry-col").length).toBe(2);

    // Simulate a filter/sort swap: remove the old cards from the DOM and
    // leave the empty masonry columns behind, then inject a fresh set of cards.
    grid.querySelectorAll(".feed-card").forEach((card) => card.remove());
    for (const id of ["c", "d"]) {
      const card = document.createElement("div");
      card.className = "feed-card";
      card.dataset.cardId = id;
      grid.appendChild(card);
    }

    applyFeedMasonry();

    const columns = grid.querySelectorAll<HTMLElement>(
      ":scope > .feed-masonry-col",
    );
    expect(columns.length).toBe(2);
    expect(columns[0].children.length).toBeGreaterThan(0);
    expect(columns[1].children.length).toBeGreaterThan(0);
  });

  it("keeps newly appended cards after originals when re-laying out (load more)", () => {
    // Reproduces the home-feed "Load more" bug: the page packs originals
    // into two masonry columns, then Svelte appends older cards after the
    // columns. Re-laying out must keep the originals first; the old code
    // anchored merged cards on the trailing "Load more" button and so
    // surfaced older items at the top.
    setHeights({ a: 40, b: 40, c: 40, d: 40, e: 40, f: 40 });
    makeFeedGrid(["a", "b"]);
    applyFeedMasonry();

    // Simulate Svelte appending older cards after the masonry columns,
    // followed by the "Load more" scaffolding element.
    const grid = document.getElementById("feed-items") as HTMLElement;
    for (const id of ["c", "d"]) {
      const card = document.createElement("div");
      card.className = "feed-card";
      card.dataset.cardId = id;
      grid.appendChild(card);
    }
    const loadMore = document.createElement("div");
    loadMore.className = "load-more-btn";
    grid.appendChild(loadMore);

    applyFeedMasonry();

    const columns = grid.querySelectorAll<HTMLElement>(
      ":scope > .feed-masonry-col",
    );
    expect(columns.length).toBe(2);

    // The invariant: originals (a, b) sit at the top of each column (row 0),
    // and the newly appended older cards (c, d) land below them (row 1).
    // The buggy flattenLayout anchored merged cards on the trailing
    // "Load more" button, which left appended cards at the top instead.
    const originals = new Set(["a", "b"]);
    for (const col of columns) {
      const ids = Array.from(col.querySelectorAll<HTMLElement>(".feed-card")).map(
        (card) => card.dataset.cardId ?? "",
      );
      // The first card in each column must be an original.
      expect(ids.length).toBeGreaterThan(0);
      expect(originals.has(ids[0])).toBe(true);
      // Every appended card must appear after at least one original.
      const firstOriginal = ids.findIndex((id) => originals.has(id));
      for (const appended of ["c", "d"]) {
        const idx = ids.indexOf(appended);
        if (idx !== -1) {
          expect(idx).toBeGreaterThan(firstOriginal);
        }
      }
    }
    // The "Load more" scaffolding stays at the end, outside the columns.
    expect(grid.querySelector(".load-more-btn")?.parentElement).toBe(grid);
  });

  it("installFeedMasonry returns a teardown that removes the listener", () => {
    const mq = makeMediaQuery(true);
    vi.stubGlobal("matchMedia", () => mq);
    const removeSpy = vi.spyOn(mq, "removeEventListener");
    const teardown = installFeedMasonry();
    teardown();
    expect(removeSpy).toHaveBeenCalledWith("change", expect.any(Function));
  });
});
