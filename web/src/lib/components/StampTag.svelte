<script lang="ts">
  type Props = {
    label: string;
    active?: boolean;
    disabled?: boolean;
    tone?: string;
    onclick: () => void;
  };

  let {
    label,
    active = false,
    disabled = false,
    tone = "",
    onclick,
  }: Props = $props();
</script>

<button
  type="button"
  class:filter-pill={!active}
  class:filter-pill-active={active}
  class:stamp-tag={true}
  class:active
  aria-pressed={active ? "true" : "false"}
  data-tab={tone || undefined}
  {disabled}
  {onclick}
>
  {label}
</button>

<style>
  .stamp-tag {
    position: relative;
    min-height: 2rem;
    padding: 0.32rem 0.68rem 0.32rem 0.68rem;
    border: 1.5px dashed var(--card-border);
    border-radius: var(--radius-paper, 4px);
    background: color-mix(in oklch, var(--surface-bg) 74%, transparent);
    color: var(--text-muted);
    font-size: 0.72rem;
    font-weight: 500;
    letter-spacing: 0.025em;
    line-height: 1;
    box-shadow: none;
    transition:
      color 150ms ease,
      border-color 150ms ease,
      background-color 150ms ease,
      transform 150ms cubic-bezier(0.16, 1, 0.3, 1),
      box-shadow 150ms ease;
  }

  .stamp-tag:hover:not(:disabled) {
    border-color: var(--text-secondary);
    background: var(--card-bg);
    color: var(--text-secondary);
  }

  .stamp-tag.active {
    border-style: solid;
    border-color: var(--text-secondary);
    background: var(--text-secondary);
    color: var(--btn-primary-text);
    font-weight: 600;
    box-shadow: 1px 2px 0
      color-mix(in oklch, var(--text-secondary) 24%, transparent);
    transform: rotate(-1.25deg);
  }

  .stamp-tag.active::before {
    background: var(--brand-amber, var(--rating-star));
  }
  .stamp-tag.active[data-tab="bean"] {
    border-color: var(--type-bean);
    background: var(--type-bean);
  }
  .stamp-tag.active[data-tab="brew"] {
    border-color: var(--type-brew);
    background: var(--type-brew);
  }
  .stamp-tag.active[data-tab="recipe"] {
    border-color: var(--type-recipe);
    background: var(--type-recipe);
  }
  .stamp-tag.active[data-tab="grinder"] {
    border-color: var(--type-grinder);
    background: var(--type-grinder);
  }
  .stamp-tag.active[data-tab="brewer"] {
    border-color: var(--type-brewer);
    background: var(--type-brewer);
  }
  .stamp-tag.active[data-tab="roaster"] {
    border-color: var(--type-roaster);
    background: var(--type-roaster);
  }
  .stamp-tag.active[data-tab="cafe"] {
    border-color: var(--type-cafe);
    background: var(--type-cafe);
  }
  .stamp-tag.active[data-tab="drink"] {
    border-color: var(--type-drink);
    background: var(--type-drink);
  }

  .stamp-tag:focus-visible {
    outline: 2px solid var(--input-border-focus);
    outline-offset: 2px;
  }

  .stamp-tag:disabled {
    cursor: wait;
    opacity: 0.56;
  }

  @media (prefers-reduced-motion: reduce) {
    .stamp-tag {
      transition: none;
    }
    .stamp-tag.active {
      transform: none;
    }
  }
</style>
