<script lang="ts">
  import { onMount } from "svelte";
  import {
    changeAIAccess,
    getAIAccess,
    type AIAccess,
  } from "#lib/api/aiAccess.js";

  let access = $state<AIAccess | null>(null);
  let did = $state("");
  let busy = $state(false);
  let error = $state("");
  let result = $state("");

  async function refresh() {
    busy = true;
    error = "";
    result = "";
    try {
      access = await getAIAccess(fetch);
    } catch (cause) {
      access = null;
      error =
        cause instanceof Error ? cause.message : "Failed to load AI access.";
    } finally {
      busy = false;
    }
  }

  async function change(method: "PUT" | "DELETE", target: string) {
    if (busy || !target) return;
    busy = true;
    error = "";
    result = "";
    try {
      await changeAIAccess(fetch, method, target);
    } catch (cause) {
      error =
        cause instanceof Error ? cause.message : "Failed to change AI access.";
      busy = false;
      return;
    }
    if (method === "PUT") did = "";
    try {
      access = await getAIAccess(fetch);
      result = method === "PUT" ? "Access granted." : "Explicit grant removed.";
    } catch {
      access = null;
      error =
        "Access changed, but the list could not be refreshed. Reload access to check the current grants.";
    } finally {
      busy = false;
    }
  }

  onMount(() => {
    void refresh();
  });
</script>

<section aria-labelledby="ai-access-title" aria-busy={busy}>
  <h2 id="ai-access-title" class="section-title">AI chat access</h2>
  <p class="text-muted">
    Admins and moderators have access automatically. Removing an explicit grant
    does not remove staff access. Chat also requires server AI configuration.
  </p>
  {#if error}<p role="alert" class="access-error">{error}</p>{/if}
  <p role="status">{busy ? "Loading AI access…" : result}</p>
  {#if access}
    <form
      onsubmit={(event) => {
        event.preventDefault();
        void change("PUT", did.trim());
      }}
    >
      <div class="did-field">
        <label for="ai-access-did">Account DID</label>
        <input
          id="ai-access-did"
          class="form-input"
          bind:value={did}
          placeholder="did:plc:…"
          required
          disabled={busy}
          autocapitalize="none"
          spellcheck="false"
        />
      </div>
      <button class="btn-primary" type="submit" disabled={busy || !did.trim()}
        >Grant access</button
      >
    </form>
    <h3>Explicit grants</h3>
    {#if access.entries.length === 0}
      <p class="text-muted">No explicit grants.</p>
    {:else}
      <ul class="access-list">
        {#each access.entries as entry (entry.did)}
          <li>
            <div>
              <p class="account-did">{entry.did}</p>
              <p class="text-muted grant-detail">
                Granted by {entry.granted_by} ·
                <time datetime={entry.created_at}
                  >{new Date(entry.created_at).toLocaleString()}</time
                >
              </p>
            </div>
            <button
              class="btn-secondary"
              type="button"
              disabled={busy}
              aria-label={`Remove access for ${entry.did}`}
              onclick={() => change("DELETE", entry.did)}>Remove</button
            >
          </li>
        {/each}
      </ul>
    {/if}
    <h3>Staff access</h3>
    <ul class="access-list">
      {#each access.moderators as moderator (moderator.did)}
        <li class="account-did">{moderator.did}</li>
      {/each}
    </ul>
  {/if}
  <button class="btn-secondary" type="button" disabled={busy} onclick={refresh}
    >Reload access</button
  >
</section>

<style>
  section {
    display: grid;
    gap: 1rem;
  }
  form {
    display: flex;
    align-items: end;
    gap: 0.75rem;
    flex-wrap: wrap;
  }
  .did-field {
    flex: 1 1 20rem;
    display: grid;
    gap: 0.5rem;
  }
  input {
    width: 100%;
  }
  h3 {
    font-weight: 600;
  }
  .access-error {
    color: var(--text-danger);
  }
  .access-list {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  li {
    display: flex;
    align-items: center;
    justify-content: space-between;
    flex-wrap: wrap;
    gap: 0.75rem;
    padding: 0.75rem 0;
    border-bottom: 1px solid var(--surface-border);
  }
  li > div {
    min-width: 0;
    flex: 1 1 18rem;
  }
  .account-did,
  .grant-detail {
    overflow-wrap: anywhere;
  }
  .grant-detail {
    font-size: 0.875rem;
    margin-top: 0.25rem;
  }
  button {
    min-height: 44px;
  }
  section > button {
    justify-self: start;
  }
</style>
