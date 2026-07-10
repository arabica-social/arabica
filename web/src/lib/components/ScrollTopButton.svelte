<script lang="ts">
  // Floating "back to top" button. Mirrors the SSR home page's
  // .scroll-top-btn markup (internal/web/pages/home.templ) and the
  // ScrollTopIsland scroll behavior (internal/web/assets/svelte/src/
  // ScrollTopIsland.svelte). The .scroll-top-btn + .is-visible styles
  // live in the shared global CSS bundle (components/18-misc.css) that
  // the SPA loads via the Go server's SPA shell, so no scoped CSS is
  // needed here.
  let visible = $state(false);

  function applyVisibility() {
    visible = window.scrollY > 400;
  }

  function scrollTop() {
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  $effect(() => {
    applyVisibility();
    window.addEventListener("scroll", applyVisibility, { passive: true });
    return () => {
      window.removeEventListener("scroll", applyVisibility);
    };
  });
</script>

<button
  type="button"
  class="scroll-top-btn fixed bottom-6 right-6 z-50 bg-brown-800 text-brown-50 rounded-full shadow-lg hover:bg-brown-700 transition-colors cursor-pointer p-3 sm:px-4 sm:py-2.5 sm:rounded-lg flex items-center gap-2"
  class:is-visible={visible}
  aria-label="Return to top"
  onclick={scrollTop}
>
  <svg
    xmlns="http://www.w3.org/2000/svg"
    class="h-5 w-5"
    viewBox="0 0 20 20"
    fill="currentColor"
  >
    <path
      fill-rule="evenodd"
      d="M14.707 12.707a1 1 0 01-1.414 0L10 9.414l-3.293 3.293a1 1 0 01-1.414-1.414l4-4a1 1 0 011.414 0l4 4a1 1 0 010 1.414z"
      clip-rule="evenodd"
    ></path>
  </svg>
  <span class="hidden sm:inline text-sm font-medium">Back to top</span>
</button>
