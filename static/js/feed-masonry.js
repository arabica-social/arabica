// Feed masonry — handles distributing dynamically-added cards (load-more)
// into the server-rendered flex columns. Initial render needs no JS.
(function () {
  function distributeLooseCards() {
    var container = document.getElementById('feed-items');
    if (!container) return;

    var cols = container.querySelectorAll('.feed-masonry-col');
    if (cols.length < 2) return;

    // Find cards that are direct children of the container (not in columns)
    var loose = [];
    Array.from(container.children).forEach(function (child) {
      if (child.classList.contains('feed-card')) {
        loose.push(child);
      }
    });
    if (loose.length === 0) return;

    // Measure current column heights
    var colHeights = Array.from(cols).map(function (col) {
      return col.offsetHeight;
    });

    // Distribute loose cards to shortest column
    loose.forEach(function (card) {
      var shortest = colHeights[0] <= colHeights[1] ? 0 : 1;
      cols[shortest].appendChild(card);
      colHeights[shortest] += card.offsetHeight + 20;
    });
  }

  // After HTMX swaps (load more adds loose cards)
  document.addEventListener('htmx:afterSettle', function (e) {
    if (e.detail.target && (e.detail.target.id === 'feed-items' || e.detail.target.closest('#feed-items'))) {
      distributeLooseCards();
    }
  });
})();
