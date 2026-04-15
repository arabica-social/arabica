// Feed masonry — pure client-side two-column layout for desktop.
// Server always renders cards flat in chronological order.
// This script distributes them into two columns on desktop (768px+).
(function () {
  var MQ = window.matchMedia('(min-width: 768px)');

  function getContainer() {
    return document.getElementById('feed-items');
  }

  // Get all .feed-card elements that are direct children of container
  function getLooseCards(container) {
    var cards = [];
    for (var i = 0; i < container.children.length; i++) {
      if (container.children[i].classList.contains('feed-card')) {
        cards.push(container.children[i]);
      }
    }
    return cards;
  }

  // Distribute loose cards into two masonry columns (shortest-first)
  function masonryLayout(container) {
    var cards = getLooseCards(container);
    if (cards.length === 0) return;

    // Get or create columns
    var cols = container.querySelectorAll(':scope > .feed-masonry-col');
    if (cols.length < 2) {
      var ref = cards[0];
      for (var i = cols.length; i < 2; i++) {
        var col = document.createElement('div');
        col.className = 'feed-masonry-col';
        container.insertBefore(col, ref);
      }
      cols = container.querySelectorAll(':scope > .feed-masonry-col');
    }

    var heights = [cols[0].offsetHeight, cols[1].offsetHeight];
    cards.forEach(function (card) {
      var idx = heights[0] <= heights[1] ? 0 : 1;
      cols[idx].appendChild(card);
      heights[idx] += card.offsetHeight + 20;
    });
  }

  // Flatten columns back to chronological order
  function flattenLayout(container) {
    var cols = container.querySelectorAll(':scope > .feed-masonry-col');
    if (cols.length === 0) return;

    // Interleave from both columns to restore chronological order
    var c0 = cols[0] ? Array.from(cols[0].children) : [];
    var c1 = cols[1] ? Array.from(cols[1].children) : [];
    var merged = [];
    var max = Math.max(c0.length, c1.length);
    for (var i = 0; i < max; i++) {
      if (i < c0.length) merged.push(c0[i]);
      if (i < c1.length) merged.push(c1[i]);
    }

    // Find insertion point (before first non-column, non-card child like load-more)
    var ref = null;
    for (var j = 0; j < container.children.length; j++) {
      var ch = container.children[j];
      if (!ch.classList.contains('feed-masonry-col') && !ch.classList.contains('feed-card')) {
        ref = ch;
        break;
      }
    }

    merged.forEach(function (card) {
      container.insertBefore(card, ref);
    });

    cols.forEach(function (col) { col.remove(); });
  }

  function applyLayout() {
    var container = getContainer();
    if (!container) return;
    if (MQ.matches) {
      masonryLayout(container);
    } else {
      flattenLayout(container);
    }
  }

  // Initial layout after DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', applyLayout);
  } else {
    applyLayout();
  }

  // Viewport changes
  MQ.addEventListener('change', applyLayout);

  // After HTMX swaps (load-more, filter/sort changes)
  document.addEventListener('htmx:afterSettle', function (e) {
    var t = e.detail.target;
    if (t && (t.id === 'feed-items' || (t.closest && t.closest('#feed-items')))) {
      applyLayout();
    }
  });
})();
