<script lang="ts">
  import { onMount } from 'svelte';

  function clearStationDrawers(event: MouseEvent) {
    const target = event.target;
    if (!(target instanceof Element)) return;
    const addButton = target.closest(".station-add[hx-target^='#station-drawer-slot-']");
    if (!addButton) return;
    document.querySelectorAll<HTMLElement>('.station-drawer-row').forEach((slot) => {
      slot.innerHTML = '';
    });
  }

  function handleActionClick(event: MouseEvent) {
    const target = event.target;
    if (!(target instanceof Element)) return;
    const element = target.closest('[data-action]');
    if (!(element instanceof HTMLElement)) return;

    switch (element.dataset.action) {
      case 'history-back':
        history.back();
        break;
      case 'close-dialog':
        element.closest<HTMLDialogElement>('dialog')?.close();
        break;
      case 'open-modal': {
        const id = element.dataset.target;
        const dialog = id ? document.getElementById(id) : null;
        if (dialog instanceof HTMLDialogElement) dialog.showModal();
        break;
      }
      case 'dispatch-event': {
        const name = element.dataset.event;
        if (name) window.dispatchEvent(new CustomEvent(name));
        break;
      }
      case 'close-drawer':
        element.closest('[data-drawer]')?.remove();
        break;
    }
  }

  function handleSubmit(event: SubmitEvent) {
    const target = event.target;
    if (!(target instanceof Element)) return;
    if (target.matches('[data-invalidate-app-cache]')) {
      window.AppCache?.invalidateCache?.();
    }
  }

  function triggeringElement(event: Event) {
    const detail = (event as CustomEvent<{ elt?: unknown; requestConfig?: { elt?: unknown } }>).detail;
    const configured = detail?.requestConfig?.elt;
    if (configured instanceof Element) return configured;
    const elt = detail?.elt;
    if (elt instanceof Element) return elt;
    return event.target instanceof Element ? event.target : null;
  }

  function handleHTMXAfterRequest(event: Event) {
    const detail = (event as CustomEvent<{ successful?: boolean }>).detail;
    const trigger = triggeringElement(event);
    const redirect = trigger?.closest<HTMLElement>('[data-delete-redirect]')?.dataset.deleteRedirect;
    if (detail?.successful && redirect) {
      window.location.href = redirect;
    }
  }

  onMount(() => {
    document.addEventListener('click', clearStationDrawers);
    document.addEventListener('click', handleActionClick);
    document.addEventListener('submit', handleSubmit);
    document.body.addEventListener('htmx:afterRequest', handleHTMXAfterRequest);
    return () => {
      document.removeEventListener('click', clearStationDrawers);
      document.removeEventListener('click', handleActionClick);
      document.removeEventListener('submit', handleSubmit);
      document.body.removeEventListener('htmx:afterRequest', handleHTMXAfterRequest);
    };
  });
</script>
