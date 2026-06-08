<script lang="ts">
  import { onMount } from 'svelte';

  let { target }: { target: HTMLElement } = $props();

  let open = false;

  function button() {
    return target.querySelector<HTMLElement>('[data-disclosure-button]');
  }

  function menu() {
    return target.querySelector<HTMLElement>('[data-disclosure-menu]');
  }

  function setOpen(nextOpen: boolean) {
    open = nextOpen;
    button()?.setAttribute('aria-expanded', open ? 'true' : 'false');
    menu()?.classList.toggle('is-open', open);
    target.querySelectorAll<HTMLElement>('[data-disclosure-rotate]').forEach((item) => {
      item.classList.toggle('rotate-180', open);
    });
    target.querySelectorAll<HTMLElement>('[data-disclosure-close]').forEach((item) => {
      item.dataset.disclosureOpen = open ? 'true' : 'false';
    });
  }

  onMount(() => {
    const handleClick = (event: MouseEvent) => {
      const node = event.target instanceof Element ? event.target : null;
      const trigger = node?.closest('[data-disclosure-button]');
      if (trigger && target.contains(trigger)) {
        event.preventDefault();
        setOpen(!open);
        return;
      }
      if (node?.closest('[data-disclosure-close]') && target.contains(node)) {
        setOpen(false);
      }
    };

    const handleOutside = (event: MouseEvent) => {
      if (open && event.target instanceof Node && !target.contains(event.target)) {
        setOpen(false);
      }
    };

    const handlePageHide = () => setOpen(false);
    const handleKeydown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        setOpen(false);
      }
    };

    target.addEventListener('click', handleClick);
    document.addEventListener('click', handleOutside);
    window.addEventListener('pagehide', handlePageHide);
    target.addEventListener('keydown', handleKeydown);
    setOpen(false);

    return () => {
      target.removeEventListener('click', handleClick);
      document.removeEventListener('click', handleOutside);
      window.removeEventListener('pagehide', handlePageHide);
      target.removeEventListener('keydown', handleKeydown);
    };
  });
</script>
