<script lang="ts">
  interface Props {
    target: HTMLElement;
  }

  let { target }: Props = $props();
  let open = false;
  let button: HTMLButtonElement | null = null;
  let panel: HTMLElement | null = null;

  function applyState() {
    button?.setAttribute('aria-expanded', open ? 'true' : 'false');
    panel?.classList.toggle('is-open', open);
  }

  function toggle() {
    open = !open;
    applyState();
  }

  function close() {
    open = false;
    applyState();
  }

  function handleClick(event: MouseEvent) {
    const eventTarget = event.target;
    if (!(eventTarget instanceof Element)) {
      return;
    }

    const toggleButton = eventTarget.closest<HTMLButtonElement>('[data-reply-toggle]');
    if (toggleButton && target.contains(toggleButton)) {
      toggle();
      return;
    }

    const closeButton = eventTarget.closest('[data-reply-close]');
    if (closeButton && target.contains(closeButton)) {
      close();
    }
  }

  $effect(() => {
    button = target.querySelector<HTMLButtonElement>('[data-reply-toggle]');
    panel = target.querySelector<HTMLElement>('[data-reply-panel]');

    target.addEventListener('click', handleClick);
    applyState();

    return () => {
      target.removeEventListener('click', handleClick);
    };
  });
</script>
