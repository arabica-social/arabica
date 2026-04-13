// Feed masonry layout — measures card heights and sets grid-row spans
// so items flow left-to-right while packing tightly (no vertical gaps).
// Only activates on 2-column layout (md+ breakpoint).
(function () {
  var ROW_HEIGHT = 8;
  var GAP = 20; // visual gap between cards (matches column-gap)

  function layoutFeed() {
    var grid = document.getElementById('feed-items');
    if (!grid) return;

    // Only apply masonry on 2-column layout
    var cols = getComputedStyle(grid).gridTemplateColumns.split(' ').length;
    if (cols < 2) {
      grid.classList.remove('masonry-ready');
      var items = grid.querySelectorAll('.feed-card');
      for (var i = 0; i < items.length; i++) {
        items[i].style.gridRowEnd = '';
      }
      return;
    }

    // Step 1: remove masonry class so cards render at natural height
    grid.classList.remove('masonry-ready');
    var items = grid.querySelectorAll('.feed-card');
    for (var i = 0; i < items.length; i++) {
      items[i].style.gridRowEnd = '';
    }

    // Step 2: wait for reflow, measure natural heights, set spans
    requestAnimationFrame(function () {
      var items = grid.querySelectorAll('.feed-card');
      if (items.length === 0) return;

      for (var i = 0; i < items.length; i++) {
        var height = items[i].getBoundingClientRect().height;
        var span = Math.ceil((height + GAP) / ROW_HEIGHT);
        items[i].style.gridRowEnd = 'span ' + span;
      }

      // Step 3: activate masonry grid now that spans are set
      grid.classList.add('masonry-ready');
    });
  }

  // Run on initial load (feed loads via HTMX, so script may run inline)
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', layoutFeed);
  } else {
    // Small delay to ensure DOM is settled after HTMX swap
    setTimeout(layoutFeed, 50);
  }

  // Re-layout after HTMX swaps (load more, filter changes)
  document.addEventListener('htmx:afterSettle', function (e) {
    if (e.detail.target && (e.detail.target.id === 'feed-items' || e.detail.target.closest('#feed-items'))) {
      setTimeout(layoutFeed, 50);
    }
  });

  // Re-layout on window resize
  var resizeTimer;
  window.addEventListener('resize', function () {
    clearTimeout(resizeTimer);
    resizeTimer = setTimeout(layoutFeed, 150);
  });

  // Re-layout after images load (avatars can shift height)
  document.addEventListener('load', function (e) {
    if (e.target.tagName === 'IMG' && e.target.closest('.feed-card')) {
      setTimeout(layoutFeed, 50);
    }
  }, true);
})();
