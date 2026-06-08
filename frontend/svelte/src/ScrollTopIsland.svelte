<script lang="ts">
  interface Props {
    target: HTMLElement;
  }

  let { target }: Props = $props();
  let button: HTMLButtonElement | null = null;

  function applyVisibility() {
    button?.classList.toggle('is-visible', window.scrollY > 400);
  }

  function scrollTop() {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  function handleClick(event: MouseEvent) {
    const eventTarget = event.target;
    if (!(eventTarget instanceof Element)) {
      return;
    }
    const scrollButton = eventTarget.closest<HTMLButtonElement>('[data-scroll-top-button]');
    if (scrollButton && target.contains(scrollButton)) {
      scrollTop();
    }
  }

  $effect(() => {
    button = target.querySelector<HTMLButtonElement>('[data-scroll-top-button]');
    target.addEventListener('click', handleClick);
    window.addEventListener('scroll', applyVisibility, { passive: true });
    applyVisibility();

    return () => {
      target.removeEventListener('click', handleClick);
      window.removeEventListener('scroll', applyVisibility);
    };
  });
</script>
