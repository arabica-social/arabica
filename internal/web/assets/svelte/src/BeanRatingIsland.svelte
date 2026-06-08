<script lang="ts">
  import { onMount } from "svelte";

  let { initialRating }: { initialRating: number } = $props();

  let showRating = $state(false);
  let rating = $state(0);

  function addRating() {
    showRating = true;
    if (rating === 0) {
      rating = 5;
    }
  }

  function removeRating() {
    showRating = false;
    rating = 0;
  }

  onMount(() => {
    showRating = initialRating > 0;
    rating = initialRating > 0 ? initialRating : 0;
  });
</script>

{#if !showRating}
  <button
    type="button"
    onclick={addRating}
    class="text-sm font-medium text-emphasis hover:text-primary flex items-center gap-1"
  >
    <svg
      aria-hidden="true"
      viewBox="0 0 20 20"
      class="w-4 h-4"
      fill="currentColor"
    >
      <path
        d="m10 1.5 2.56 5.2 5.74.83-4.15 4.05.98 5.72L10 14.6 4.87 17.3l.98-5.72L1.7 7.53l5.74-.83L10 1.5Z"
      />
    </svg>
    Add Rating
  </button>
{:else}
  <div class="space-y-2">
    <div class="flex items-center justify-between">
      <label class="form-label mb-0" for="bean-rating">Rating</label>
      <button
        type="button"
        onclick={removeRating}
        class="text-xs text-faint hover:text-emphasis"
      >
        Remove rating
      </button>
    </div>
    <input
      id="bean-rating"
      type="range"
      min="1"
      max="10"
      bind:value={rating}
      class="w-full accent-brown-700"
    />
    <div class="text-center text-2xl font-bold text-secondary">{rating}/10</div>
    <input type="hidden" name="rating" value={showRating ? rating : ""} />
  </div>
{/if}
