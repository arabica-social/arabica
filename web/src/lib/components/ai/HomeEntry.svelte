<script lang="ts">
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";
  import {
    prepareImage,
    MAX_IMAGES,
    type PreparedImage,
  } from "#lib/ai/images.js";
  import {
    stageHomeImages,
    cancelHomeImages,
    clearHomeImages,
  } from "#lib/ai/homeImages.js";
  import Icon from "#lib/components/Icon.svelte";
  import { entryKey } from "#lib/api/ai.js";
  import SendButton from "./SendButton.svelte";
  import { autosizeTextarea } from "./autosizeTextarea";
  import WorkingIndicator from "./WorkingIndicator.svelte";
  let { accountDid }: { accountDid: string } = $props();
  let text = $state("");
  let error = $state("");
  let navigating = $state(false);
  let pendingOwner: object | undefined;
  let images = $state<PreparedImage[]>([]);
  let preparing = $state(false);
  let generation = 0;
  let handoff: ReturnType<typeof stageHomeImages> | undefined;
  function releaseImages() {
    generation++;
    images = [];
    preparing = false;
    if (handoff) cancelHomeImages(handoff);
    handoff = undefined;
  }
  onMount(() => {
    const leave = () => {
      releaseImages();
      clearHomeImages();
    };
    window.addEventListener("pagehide", leave);
    return () => {
      window.removeEventListener("pagehide", leave);
      generation++; // Ignore a decoder finishing after route teardown.
      images = [];
    };
  });
  async function pasteImages(event: ClipboardEvent) {
    const selected = Array.from(event.clipboardData?.items ?? [])
      .filter((item) => item.kind === "file" && item.type.startsWith("image/"))
      .map((item) => item.getAsFile())
      .filter((file): file is File => !!file);
    if (!selected.length) return;
    event.preventDefault();
    if (navigating || preparing) return;
    error = "";
    if (images.length + selected.length > MAX_IMAGES) {
      error = `Up to ${MAX_IMAGES} images can be sent. Remove an image before pasting more.`;
      return;
    }
    const target = event.currentTarget as HTMLTextAreaElement;
    const pasted = event.clipboardData?.getData("text/plain") ?? "";
    const nextText =
      text.slice(0, target.selectionStart) +
      pasted +
      text.slice(target.selectionEnd);
    if (nextText.length > 4000) {
      error = "Pasted text is too long. Paste the images and text separately.";
      return;
    }
    if (pasted) text = nextText;
    const currentGeneration = generation;
    const did = accountDid;
    preparing = true;
    try {
      const prepared: PreparedImage[] = [];
      for (const file of selected) {
        prepared.push(await prepareImage(file));
        if (generation !== currentGeneration || did !== accountDid) return;
      }
      images = [...images, ...prepared];
    } catch (e) {
      if (generation === currentGeneration && did === accountDid)
        error =
          e instanceof Error ? e.message : "Could not prepare that image.";
    } finally {
      if (generation === currentGeneration) preparing = false;
    }
  }
  $effect(() => {
    accountDid;
    releaseImages();
    clearHomeImages();
    text = "";
    error = "";
    navigating = false;
    pendingOwner = undefined;
  });
  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (navigating || preparing || (!text.trim() && !images.length)) return;
    const owner = {};
    const did = accountDid;
    pendingOwner = owner;
    navigating = true;
    error = "";
    try {
      if (images.length) {
        handoff = stageHomeImages(did, {
          text: text.trim(),
          images: [...images],
        });
        // No image handoff is ever serialized, including an image-only message.
        try {
          sessionStorage.removeItem(entryKey(did));
        } catch {
          /* Memory handoff works without storage. */
        }
      } else sessionStorage.setItem(entryKey(did), text.trim());
      await goto("/chat");
    } catch {
      if (pendingOwner === owner && accountDid === did) {
        if (handoff) cancelHomeImages(handoff);
        handoff = undefined;
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
  {#if images.length}
    <div class="home-images" aria-label="Attached images">
      {#each images as image, index}
        <div class="home-image">
          <img src={image.preview} alt={`Image ${index + 1} to send`} />
          <button
            type="button"
            aria-label={`Remove image ${index + 1}`}
            disabled={navigating || preparing}
            onclick={() => {
              images = images.filter((_, i) => i !== index);
              error = "";
            }}><Icon name="x" class="h-4 w-4" /></button
          >
        </div>
      {/each}
    </div>
  {/if}
  <div class="input-row">
    <label for="ai-home-entry" class="sr-only">Describe a brew or bean</label>
    <textarea
      id="ai-home-entry"
      rows="1"
      maxlength="4000"
      disabled={navigating}
      placeholder="Describe a brew or bean..."
      bind:value={text}
      use:autosizeTextarea={text}
      onpaste={pasteImages}
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
      <SendButton
        disabled={navigating || preparing || (!text.trim() && !images.length)}
      />
    </div>
  </div>
</form>
{#if preparing}<WorkingIndicator label="Preparing images…" />{/if}
{#if images.length}<p class="image-notice">
    Images are sent to the AI provider, not saved in Arabica.
  </p>{/if}
{#if navigating}<WorkingIndicator label="Opening chat…" />{/if}
{#if error}<p role="alert">{error}</p>{/if}

<style>
  .entry {
    view-transition-name: ai-composer;
    position: relative;
    margin: 0;
    border: 1px solid var(--surface-border);
    border-radius: 1.25rem;
    background: var(--surface-bg);
  }
  .input-row {
    position: relative;
  }
  .home-images {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    padding: 0.75rem 0.75rem 0;
  }
  .home-image {
    position: relative;
  }
  .home-image img {
    width: 80px;
    height: 80px;
    object-fit: cover;
    border-radius: 0.5rem;
  }
  .home-image button {
    position: absolute;
    top: 0;
    right: 0;
    width: 44px;
    height: 44px;
    display: grid;
    place-items: center;
    border-radius: 0.5rem;
    border: 1px solid var(--surface-border);
    background: var(--surface-bg);
    color: var(--text-primary);
  }
  .home-image button:focus-visible {
    outline: 2px solid var(--text-primary);
    outline-offset: 2px;
  }
  .image-notice {
    margin: 0.5rem 0 0;
    font-size: 0.75rem;
    color: var(--text-muted);
  }
  .entry textarea {
    display: block;
    width: 100%;
    min-height: 3rem;
    max-height: 15rem;
    resize: none;
    outline-offset: -3px;
    overflow-y: hidden;
    border: none;
    border-radius: 1.25rem;
    background: transparent;
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
