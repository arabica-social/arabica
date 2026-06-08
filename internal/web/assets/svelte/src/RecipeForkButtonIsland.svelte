<script lang="ts">
  interface Props {
    recipeRKey: string;
    ownerDID: string;
  }

  let { recipeRKey, ownerDID }: Props = $props();
  let forking = $state(false);

  function notify(message: string) {
    window.dispatchEvent(
      new CustomEvent("notify", { detail: { message }, bubbles: true }),
    );
  }

  async function forkRecipe() {
    if (forking) {
      return;
    }
    forking = true;
    try {
      const response = await fetch(
        `/api/recipes/fork/${recipeRKey}?owner=${encodeURIComponent(ownerDID)}`,
        {
          method: "POST",
          credentials: "same-origin",
        },
      );
      if (response.status === 401) {
        window.__showSessionExpiredModal?.();
        return;
      }
      if (!response.ok) {
        throw new Error("Failed to copy recipe");
      }
      await response.json();
      notify("Recipe copied to your library!");
    } catch {
      notify("Failed to copy recipe");
    } finally {
      forking = false;
    }
  }
</script>

<button
  type="button"
  onclick={forkRecipe}
  class="btn-secondary text-sm"
  disabled={forking}
>
  {forking ? "Copying..." : "Copy Recipe"}
</button>
