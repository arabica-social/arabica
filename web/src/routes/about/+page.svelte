<script lang="ts">
  import BackButton from "$lib/components/BackButton.svelte";
  import { openLoginModal, session } from "$lib/stores/session";

  let isAuthenticated = $derived($session.isAuthenticated);
</script>

<svelte:head>
  <title>About - Arabica</title>
  <meta
    name="description"
    content="Track coffee brews, beans, recipes, and equipment with Arabica."
  />
  <meta property="og:title" content="About Arabica" />
  <meta property="og:type" content="website" />
</svelte:head>

<div class="info-page">
  <div class="info-hero">
    <div class="info-hero-row">
      <div class="info-title-group">
        <div class="info-eyebrow">
          <BackButton />
        </div>
        <h1 class="info-title">About Arabica</h1>
        <p class="info-deck">
          Log your brews, keep track of beans and equipment, and share recipes
          and tasting notes with other coffee drinkers.
        </p>
      </div>
    </div>
  </div>
  <div class="info-alert">
    <p class="text-sm text-secondary">
      <strong>Alpha Software:</strong> Arabica is currently in early development.
      Features may change, and data structures could be modified in future updates.
    </p>
  </div>
  <div class="prose prose-lg max-w-none info-stack">
    <section>
      <h2 class="text-2xl font-semibold text-primary mb-4">
        Accounts and data
      </h2>
      <p class="text-secondary leading-relaxed mb-3">
        Log in with an existing Bluesky or other AT Protocol account. Your
        provider handles authentication through OAuth; Arabica never sees your
        password.
      </p>
      <p class="text-secondary leading-relaxed">
        Arabica saves your coffee records to your account's Personal Data Server
        (PDS). Read <a href="/atproto" class="link-bold"
          >how Arabica uses AT Protocol</a
        >
        for details about storage and public records. If you don't have an
        account, <a href="/join/create" class="link-bold">choose a provider</a> to
        create one.
      </p>
    </section>
    <section>
      <h2 class="text-2xl font-semibold text-primary mb-4">Open Source</h2>
      <p class="text-secondary leading-relaxed">
        Arabica is open source software. You can view the code, contribute, or
        even run your own instance. Visit our <a
          href="https://tangled.org/arabica.social/arabica"
          class="text-emphasis hover:underline font-medium"
          target="_blank"
          rel="noopener noreferrer">Tangled repository</a
        > to learn more.
      </p>
    </section>
    <section id="support">
      <h2 class="text-2xl font-semibold text-primary mb-4">Support Arabica</h2>
      <p class="text-secondary leading-relaxed">
        Buy me a coffee, or help cover infra costs.
      </p>
      <div class="flex flex-wrap gap-4 mt-4">
        <a
          href="https://ko-fi.com/pdewey"
          target="_blank"
          rel="noopener noreferrer"
          class="btn-secondary"
        >
          Ko-fi
        </a>
        <a
          href="https://github.com/sponsors/ptdewey"
          target="_blank"
          rel="noopener noreferrer"
          class="btn-secondary"
        >
          GitHub Sponsors
        </a>
      </div>
    </section>
  </div>
  <div class="info-cta">
    {#if !isAuthenticated}
      <button
        type="button"
        onclick={openLoginModal}
        class="btn-primary px-8 py-3 font-semibold"
      >
        Log in
      </button>
    {:else}
      <a href="/" class="btn-primary px-8 py-3 font-semibold"> Back to Home </a>
    {/if}
  </div>
</div>
