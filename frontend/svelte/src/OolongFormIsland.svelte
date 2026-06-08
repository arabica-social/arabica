<script lang="ts">
  import { onMount } from 'svelte';

  let { target }: { target: HTMLFormElement } = $props();

  function setError(message: string) {
    const error = target.querySelector<HTMLElement>('[data-form-error]');
    if (!error) {
      return;
    }
    error.textContent = message;
    error.hidden = !message;
  }

  function setInfusionMethod(method: string) {
    target.dataset.infusionMethod = method;
    target.querySelectorAll<HTMLElement>('[data-infusion-section]').forEach((section) => {
      section.hidden = method !== 'infuser';
    });
  }

  onMount(() => {
    const initialMethod =
      target.dataset.initialInfusionMethod ||
      target.querySelector<HTMLSelectElement>('select[name="infusion_method"]')?.value ||
      '';
    setInfusionMethod(initialMethod);

    const handleChange = (event: Event) => {
      const input = event.target;
      if (input instanceof HTMLSelectElement && input.name === 'infusion_method') {
        setInfusionMethod(input.value);
      }
    };

    const handleAfterRequest = (event: Event) => {
      const detail = (event as CustomEvent).detail;
      if (detail?.successful) {
        setError('');
        return;
      }
      if (detail?.xhr?.status === 401) {
        window.__showSessionExpiredModal?.();
        return;
      }
      setError(
        detail?.xhr?.responseText ||
          'Something went wrong. Please try again.'
      );
    };

    target.addEventListener('change', handleChange);
    target.addEventListener('htmx:afterRequest', handleAfterRequest);

    return () => {
      target.removeEventListener('change', handleChange);
      target.removeEventListener('htmx:afterRequest', handleAfterRequest);
    };
  });
</script>
