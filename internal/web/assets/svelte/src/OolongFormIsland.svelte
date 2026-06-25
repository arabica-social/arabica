<script lang="ts">
  import { onMount } from "svelte";

  let { target }: { target: HTMLFormElement } = $props();

  function setError(message: string) {
    const error = target.querySelector<HTMLElement>("[data-form-error]");
    if (!error) {
      return;
    }
    error.textContent = message;
    error.hidden = !message;
  }

  function setInfusionMethod(method: string) {
    target.dataset.infusionMethod = method;
    target
      .querySelectorAll<HTMLElement>("[data-infusion-section]")
      .forEach((section) => {
        section.hidden = method !== "infuser";
      });
  }

  function setMethodMode(method: string, mode: string) {
    target.dataset.method = method;
    target.dataset.mode = mode;
    target
      .querySelectorAll<HTMLElement>("[data-session-fields]")
      .forEach((section) => {
        section.hidden = method !== "gongfu" && mode !== "session";
      });

    const style = target.querySelector<HTMLInputElement>(
      'input[name="style"][data-legacy-style-field]',
    );
    if (style) {
      style.value = method === "cold-brew" ? "coldBrew" : "longSteep";
    }
  }

  onMount(() => {
    const initialMethod =
      target.dataset.initialInfusionMethod ||
      target.querySelector<HTMLSelectElement>('select[name="infusion_method"]')
        ?.value ||
      "";
    setInfusionMethod(initialMethod);

    const initialBrewMethod =
      target.dataset.initialMethod ||
      target.querySelector<HTMLSelectElement>('select[name="method"]')?.value ||
      "";
    const initialMode =
      target.dataset.initialMode ||
      target.querySelector<HTMLSelectElement>('select[name="mode"]')?.value ||
      "";
    setMethodMode(initialBrewMethod, initialMode);

    const handleChange = (event: Event) => {
      const input = event.target;
      if (
        input instanceof HTMLSelectElement &&
        input.name === "infusion_method"
      ) {
        setInfusionMethod(input.value);
      }
      if (input instanceof HTMLSelectElement && input.name === "method") {
        const mode =
          target.querySelector<HTMLSelectElement>('select[name="mode"]')
            ?.value || "";
        setMethodMode(input.value, mode);
      }
      if (input instanceof HTMLSelectElement && input.name === "mode") {
        const method =
          target.querySelector<HTMLSelectElement>('select[name="method"]')
            ?.value || "";
        setMethodMode(method, input.value);
      }
    };

    const handleAfterRequest = (event: Event) => {
      const detail = (event as CustomEvent).detail;
      if (detail?.successful) {
        setError("");
        return;
      }
      if (detail?.xhr?.status === 401) {
        window.__showSessionExpiredModal?.();
        return;
      }
      setError(
        detail?.xhr?.responseText || "Something went wrong. Please try again.",
      );
    };

    target.addEventListener("change", handleChange);
    target.addEventListener("htmx:afterRequest", handleAfterRequest);

    return () => {
      target.removeEventListener("change", handleChange);
      target.removeEventListener("htmx:afterRequest", handleAfterRequest);
    };
  });
</script>
