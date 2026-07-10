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

  it("installFeedMasonry returns a teardown that removes the listener", () => {
    const mq = makeMediaQuery(true);
    vi.stubGlobal("matchMedia", () => mq);
    const removeSpy = vi.spyOn(mq, "removeEventListener");
    const teardown = installFeedMasonry();
    teardown();
    expect(removeSpy).toHaveBeenCalledWith("change", expect.any(Function));
  });
});
