<script lang="ts">
  import { goto } from "$app/navigation";
  import { entryKey } from "#lib/api/ai.js";
  import SendButton from "./SendButton.svelte";
  import { autosizeTextarea } from "./autosizeTextarea";
  import WorkingIndicator from "./WorkingIndicator.svelte";
  let { accountDid }: { accountDid: string } = $props();
  let text = $state("");
  let error = $state("");
  let navigating = $state(false);
  let pendingOwner: object | undefined;
  $effect(() => {
    accountDid;
    text = "";
    error = "";
    navigating = false;
    pendingOwner = undefined;
  });
  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (navigating || !text.trim()) return;
    const owner = {};
    const did = accountDid;
    pendingOwner = owner;
    navigating = true;
    error = "";
    try {
      sessionStorage.setItem(entryKey(did), text.trim());
      await goto("/chat");
    } catch {
      if (pendingOwner === owner && accountDid === did) {
        error = "Could not open chat. Your message is still here.";
      }
    } finally {
      if (pendingOwner === owner && accountDid === did) {
        navigating = false;
        pendingOwner = undefined;
      }
    }
  }
</script>

<form class="entry" onsubmit={submit} aria-busy={navigating}>
  <label for="ai-home-entry" class="sr-only">Describe a brew or bean</label>
  <textarea
    id="ai-home-entry"
    rows="1"
    maxlength="4000"
    disabled={navigating}
    placeholder="Describe a brew or bean..."
    bind:value={text}
    use:autosizeTextarea={text}
    onkeydown={(e) => {
      if (
        e.key === "Enter" &&
        !e.shiftKey &&
        !e.isComposing &&
        e.keyCode !== 229
      ) {
        e.preventDefault();
        if (!e.repeat) e.currentTarget.form?.requestSubmit();
      }
    }}
  ></textarea>
  <div class="send-slot">
    <SendButton disabled={navigating || !text.trim()} />
  </div>
</form>
{#if navigating}<WorkingIndicator label="Opening chat…" />{/if}
{#if error}<p role="alert">{error}</p>{/if}

<style>
  .entry {
    view-transition-name: ai-composer;
    position: relative;
    margin: 0;
  }
  .entry textarea {
    display: block;
    width: 100%;
    min-height: 3rem;
    max-height: 15rem;
    resize: none;
    outline-offset: -3px;
    overflow-y: hidden;
    border: 1px solid var(--surface-border);
    border-radius: 1.25rem;
    background: var(--surface-bg);
    color: var(--text-primary);
    font: inherit;
    font-size: 0.9rem;
    line-height: 1.5;
    padding: 0.75rem 3rem 0.75rem 1rem;
  }
  .send-slot {
    position: absolute;
    right: 0.15rem;
    top: 50%;
    transform: translateY(-50%);
    display: flex;
  }
</style>
