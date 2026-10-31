<script lang="ts">
  import { tick } from "svelte";
  let {
    value,
    disabled = false,
    onchange,
  }: {
    value?: number | null;
    disabled?: boolean;
    onchange: (value: number | null) => void;
  } = $props();
</script>

<label class="draft-rating">
  <span>Rating (1–10)</span>
  <input
    class="form-input"
    type="number"
    min="1"
    max="10"
    step="1"
    placeholder="1–10"
    {disabled}
    value={value ?? ""}
    oninput={async (event) => {
      const input = event.currentTarget;
      onchange(input.value === "" ? null : input.valueAsNumber);
      await tick();
      input.value = value == null ? "" : String(value);
    }}
  />
</label>

<style>
  .draft-rating {
    display: grid;
    gap: 0.35rem;
    margin-inline-end: auto;
    color: var(--text-muted);
    font-size: 0.75rem;
  }
  .draft-rating input {
    width: 6rem;
    min-height: 2.75rem;
  }
</style>
