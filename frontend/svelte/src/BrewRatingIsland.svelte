<script lang="ts">
  import { onMount } from 'svelte';

  let { initialRating = 5 }: { initialRating?: number } = $props();

  let rating = $state(5);

  onMount(() => {
    rating = initialRating;
  });
</script>

<label class="form-label" for="brew-rating">Rating</label>
<input
  id="brew-rating"
  type="range"
  name="rating"
  min="1"
  max="10"
  bind:value={rating}
  class="w-full accent-brown-700"
/>
<div class="text-center text-2xl font-bold text-secondary">{rating}/10</div>
