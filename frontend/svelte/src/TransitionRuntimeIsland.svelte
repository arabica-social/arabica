<script lang="ts">
  import { onMount } from 'svelte';

  function visibleElements(extra?: Element | null) {
    return [extra, document.body, document.querySelector('main'), document.querySelector('main > *')].filter(
      (element): element is HTMLElement => element instanceof HTMLElement
    );
  }

  function clearTransitionClasses(element: HTMLElement) {
    element.classList.remove('htmx-swapping', 'htmx-transitioning', 'htmx-settling', 'htmx-added', 'transitioning');
  }

  function forceContentVisibility(extra?: Element | null) {
    visibleElements(extra).forEach((element) => {
      clearTransitionClasses(element);
      element.style.opacity = '1';
      element.style.transform = 'none';
      element.style.visibility = 'visible';
    });
  }

  function eventTarget(event: Event) {
    return (event as CustomEvent<{ target?: Element }>).detail?.target || null;
  }

  onMount(() => {
    forceContentVisibility();
    if (window.htmx?.config) {
      window.htmx.config.globalViewTransitions = false;
    }

    const beforeRequest = (event: Event) => {
      eventTarget(event)?.classList.add('htmx-transitioning');
    };
    const afterSwap = (event: Event) => {
      eventTarget(event)?.classList.remove('htmx-transitioning');
    };
    const pageshow = (event: PageTransitionEvent) => {
      if (event.persisted) forceContentVisibility();
    };
    const popstate = () => {
      forceContentVisibility();
    };
    const beforeHistoryRestore = () => {
      document.body.classList.add('htmx-history-restoring');
      const main = document.querySelector<HTMLElement>('main');
      if (main) {
        main.style.opacity = '1';
        main.style.transform = 'none';
        main.style.visibility = 'visible';
      }
    };
    const historyRestore = (event: Event) => {
      const target = eventTarget(event);
      forceContentVisibility(target);
      window.setTimeout(() => {
        forceContentVisibility(target);
        document.body.classList.remove('htmx-history-restoring');
      }, 50);
    };
    const afterSettle = (event: Event) => {
      const target = eventTarget(event);
      if (target instanceof HTMLElement) {
        target.style.opacity = '';
        target.style.transform = '';
      }
    };
    const load = () => {
      window.setTimeout(forceContentVisibility, 100);
      window.setTimeout(forceContentVisibility, 500);
    };

    document.body.addEventListener('htmx:beforeRequest', beforeRequest);
    document.body.addEventListener('htmx:afterSwap', afterSwap);
    document.body.addEventListener('htmx:beforeHistoryRestore', beforeHistoryRestore);
    document.body.addEventListener('htmx:historyRestore', historyRestore);
    document.body.addEventListener('htmx:afterSettle', afterSettle);
    window.addEventListener('pageshow', pageshow);
    window.addEventListener('popstate', popstate);
    window.addEventListener('load', load);

    return () => {
      document.body.removeEventListener('htmx:beforeRequest', beforeRequest);
      document.body.removeEventListener('htmx:afterSwap', afterSwap);
      document.body.removeEventListener('htmx:beforeHistoryRestore', beforeHistoryRestore);
      document.body.removeEventListener('htmx:historyRestore', historyRestore);
      document.body.removeEventListener('htmx:afterSettle', afterSettle);
      window.removeEventListener('pageshow', pageshow);
      window.removeEventListener('popstate', popstate);
      window.removeEventListener('load', load);
    };
  });
</script>
