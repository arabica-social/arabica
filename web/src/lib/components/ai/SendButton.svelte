<script lang="ts">
  let { disabled = false }: { disabled?: boolean } = $props();
</script>

<button
  class="ai-send-button"
  type="submit"
  aria-label="Send"
  title="Send"
  {disabled}
>
  <span aria-hidden="true">↑</span>
</button>

<style>
  .ai-send-button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 2.75rem;
    height: 2.75rem;
    padding: 0;
    border: 0;
    border-radius: 50%;
    background: transparent;
    color: var(--btn-primary-text);
    font: inherit;
    font-size: 1rem;
    line-height: 1;
    cursor: pointer;
  }
  .ai-send-button span {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 1.75rem;
    height: 1.75rem;
    border-radius: 50%;
    background: var(--btn-primary-bg);
  }
  .ai-send-button:hover:not(:disabled) span {
    background: var(--btn-primary-bg-hover);
  }
  .ai-send-button:disabled {
    opacity: 0.4;
    cursor: default;
  }
  .ai-send-button:focus-visible {
    outline: 2px solid var(--btn-primary-bg);
    outline-offset: 1px;
  }
</style>
