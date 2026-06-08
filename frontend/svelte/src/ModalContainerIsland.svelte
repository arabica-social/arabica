<script lang="ts">
  import { onMount } from 'svelte';

  let { target }: { target: HTMLElement } = $props();

  function openLoadedDialog(event: Event) {
    const detail = (event as CustomEvent<{ target?: Element }>).detail;
    const swapTarget = detail?.target || event.target;
    if (!(swapTarget instanceof Element)) return;
    if (swapTarget !== target && !target.contains(swapTarget)) return;

    const dialog = target.querySelector<HTMLDialogElement>('dialog#entity-modal');
    if (!dialog || typeof dialog.showModal !== 'function') return;

    window.setTimeout(() => {
      if (!dialog.open) dialog.showModal();
    }, 10);
  }

  onMount(() => {
    document.body.addEventListener('htmx:afterSwap', openLoadedDialog);
    return () => {
      document.body.removeEventListener('htmx:afterSwap', openLoadedDialog);
    };
  });
</script>
