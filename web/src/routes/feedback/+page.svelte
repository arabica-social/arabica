<script lang="ts">
  import BackButton from "$lib/components/BackButton.svelte";
  import Icon from "$lib/components/Icon.svelte";

  const feedbackAddress =
    "mailto:mail@arabica.systems?subject=Arabica%20feedback";
</script>

<svelte:head>
  <title>Submit Feedback - Arabica</title>
  <meta
    name="description"
    content="Send a note, report a rough edge, or suggest what Arabica should do next."
  />
  <meta property="og:title" content="Submit Feedback - Arabica" />
  <meta property="og:type" content="website" />
</svelte:head>

<div class="feedback-page">
  <header class="feedback-hero">
    <div class="feedback-eyebrow">
      <BackButton />
      <span>Notes from the counter</span>
    </div>
    <h1>Submit feedback</h1>
    <p>
      Tell us what is working, what got in your way, or what you want Arabica to
      make room for next.
    </p>
  </header>

  <form
    class="feedback-form"
    action={feedbackAddress}
    method="post"
    enctype="text/plain"
  >
    <div class="feedback-form-heading">
      <div>
        <p class="feedback-kicker">Your note</p>
        <h2>A little context goes a long way.</h2>
      </div>
      <span class="feedback-required">required</span>
    </div>

    <div class="feedback-fields">
      <div>
        <label class="form-label" for="feedback-title">
          Title <span aria-hidden="true">*</span>
        </label>
        <input
          id="feedback-title"
          class="form-input-lg w-full"
          name="title"
          type="text"
          required
          maxlength="160"
          autocomplete="off"
          placeholder="A short summary of your note"
          aria-describedby="feedback-title-help"
        />
        <p id="feedback-title-help" class="feedback-help">
          A sentence is plenty. It helps us find the thread quickly.
        </p>
      </div>

      <div>
        <label class="form-label" for="feedback-contact"
          >Contact (optional)</label
        >
        <input
          id="feedback-contact"
          class="form-input-lg w-full"
          name="contact"
          type="text"
          maxlength="320"
          autocomplete="email"
          placeholder="you@example.com or @you.bsky.social"
          aria-describedby="feedback-contact-help"
        />
        <p id="feedback-contact-help" class="feedback-help">
          Leave an email address or AT Protocol handle if you would like a
          reply.
        </p>
      </div>

      <div>
        <label class="form-label" for="feedback-description">Description</label>
        <textarea
          id="feedback-description"
          class="form-textarea w-full"
          name="description"
          rows="8"
          maxlength="5000"
          placeholder="What happened? What did you expect? Screenshots, brew details, and little examples all help."
        ></textarea>
      </div>
    </div>

    <div class="feedback-submit-row">
      <button class="btn-primary feedback-submit" type="submit">
        <Icon name="fileText" class="w-4 h-4 text-white" />
        Open feedback email
      </button>
      <p>
        This opens a draft addressed to Arabica in your email app. This form
        does not store your note in Arabica or your PDS.
      </p>
    </div>
  </form>
</div>

<style>
  .feedback-page {
    width: min(100%, 46rem);
    margin-inline: auto;
    padding-inline: 0.75rem;
  }

  .feedback-hero {
    position: relative;
    overflow: hidden;
    margin-bottom: 1.5rem;
    padding: clamp(1.25rem, 4vw, 2rem);
    border: 1px solid var(--card-border);
    border-radius: 1.35rem;
    background: var(--texture-kraft), var(--card-bg);
    background-blend-mode: multiply, normal;
    box-shadow: var(--shadow-md);
  }

  .feedback-hero::after {
    content: "";
    position: absolute;
    right: -2.5rem;
    bottom: -3rem;
    width: 13rem;
    height: 13rem;
    border: 1px dashed
      color-mix(in oklch, var(--brand-amber-400) 35%, transparent);
    border-radius: 50%;
    opacity: 0.55;
    pointer-events: none;
  }

  .feedback-hero > * {
    position: relative;
    z-index: 1;
  }

  .feedback-eyebrow {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    margin-bottom: 0.75rem;
    color: var(--text-muted);
    font-size: 0.73rem;
    font-weight: 700;
    letter-spacing: 0.16em;
    text-transform: uppercase;
  }

  .feedback-hero h1 {
    margin: 0;
    color: var(--text-primary);
    font-family: var(--font-display);
    font-size: clamp(2rem, 7vw, 3.4rem);
    font-weight: 600;
    line-height: 0.98;
    letter-spacing: -0.025em;
  }

  .feedback-hero p {
    max-width: 57ch;
    margin: 1rem 0 0;
    color: var(--text-secondary);
    font-size: 1rem;
    line-height: 1.7;
  }

  .feedback-form {
    padding: clamp(1rem, 3vw, 1.5rem);
    border: 1px solid var(--card-border);
    border-radius: 1rem;
    background: var(--card-bg);
    box-shadow: var(--shadow-sm);
  }

  .feedback-form-heading {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: 1rem;
    padding-bottom: 1rem;
    border-bottom: 1px dotted var(--card-border);
  }

  .feedback-kicker {
    margin: 0 0 0.3rem;
    color: var(--text-muted);
    font-size: 0.65rem;
    font-weight: 700;
    letter-spacing: 0.13em;
    text-transform: uppercase;
  }

  .feedback-form-heading h2 {
    margin: 0;
    color: var(--text-primary);
    font-family: var(--font-display);
    font-size: 1.35rem;
    font-weight: 600;
    line-height: 1.2;
    letter-spacing: -0.01em;
  }

  .feedback-required {
    display: inline-flex;
    flex: 0 0 auto;
    align-items: center;
    min-height: 1.6rem;
    padding: 0.2rem 0.45rem;
    border-radius: 999px;
    background: var(--text-emphasis);
    color: var(--card-bg);
    font-size: 0.6rem;
    font-weight: 700;
    letter-spacing: 0.1em;
    text-transform: uppercase;
  }

  .feedback-fields {
    display: grid;
    gap: 1.2rem;
    margin-top: 1.25rem;
  }

  .feedback-help {
    margin: 0.4rem 0 0;
    color: var(--text-faint);
    font-size: 0.72rem;
    line-height: 1.5;
  }

  .feedback-submit-row {
    display: flex;
    align-items: center;
    gap: 1rem;
    margin-top: 1.5rem;
    padding-top: 1rem;
    border-top: 1px dotted var(--card-border);
  }

  .feedback-submit {
    display: inline-flex;
    flex: 0 0 auto;
    min-height: 2.75rem;
    align-items: center;
    gap: 0.45rem;
  }

  .feedback-submit-row p {
    margin: 0;
    color: var(--text-muted);
    font-size: 0.7rem;
    line-height: 1.55;
  }

  @media (max-width: 640px) {
    .feedback-page {
      padding-inline: 0.5rem;
    }

    .feedback-hero {
      border-radius: 1rem;
    }

    .feedback-submit-row {
      align-items: stretch;
      flex-direction: column;
    }

    .feedback-submit {
      justify-content: center;
    }
  }

  @media (prefers-reduced-transparency: reduce) {
    .feedback-hero::after {
      display: none;
    }
  }
</style>
