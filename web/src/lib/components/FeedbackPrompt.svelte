<script lang="ts">
  import { app } from "$lib/stores/session";
  import { definitionFor } from "$lib/app/definitions";

  let {
    eyebrow = "Notes from the counter",
    title = "Tell us what needs work.",
    description = "Found something confusing, broken, or missing? Let us know.",
    actionLabel = "Share feedback",
    href,
  }: {
    eyebrow?: string;
    title?: string;
    description?: string;
    actionLabel?: string;
    href?: string;
  } = $props();

  // Feedback lives on the app's external board; the prompt opens it in a new
  // tab. An explicit href override still wins for special placements.
  let feedbackHref = $derived(href ?? definitionFor($app).feedbackUrl);
</script>

<section class="feedback-prompt" aria-label={title}>
  <div class="feedback-prompt-copy">
    <p class="feedback-prompt-eyebrow">{eyebrow}</p>
    <h2>{title}</h2>
    <p>{description}</p>
  </div>
  <a
    class="feedback-prompt-action"
    href={feedbackHref}
    target="_blank"
    rel="noopener noreferrer"
  >
    {actionLabel}
  </a>
</section>

<style>
  .feedback-prompt {
    padding: 1rem 0 1.2rem;
    border: 1px solid var(--card-border);
    border-width: 1px 0 0;
    border-radius: 0;
  }

  .feedback-prompt-copy {
    min-width: 0;
  }

  .feedback-prompt-eyebrow {
    margin: 0 0 0.35rem;
    color: var(--text-muted);
    font-size: 0.65rem;
    font-weight: 700;
    letter-spacing: 0.14em;
    text-transform: uppercase;
  }

  h2 {
    margin: 0;
    color: var(--text-primary);
    font-family: var(--font-display);
    font-size: 1.05rem;
    font-weight: 600;
    line-height: 1.25;
    letter-spacing: -0.01em;
  }

  .feedback-prompt-copy > p:last-child {
    margin: 0.5rem 0 0.75rem;
    color: var(--text-muted);
    font-size: 0.76rem;
    line-height: 1.55;
  }

  .feedback-prompt-action {
    display: inline-flex;
    min-height: 2.75rem;
    align-items: center;
    color: var(--text-emphasis);
    font-size: 0.75rem;
    font-weight: 600;
    text-decoration: underline;
    text-decoration-color: color-mix(in oklch, var(--text-faint) 45%, transparent);
    text-underline-offset: 4px;
  }

  .feedback-prompt-action:hover {
    color: var(--text-primary);
    text-decoration-color: currentColor;
  }

  .feedback-prompt-action:focus-visible {
    outline: 2px solid var(--input-border-focus);
    outline-offset: 3px;
  }
</style>
