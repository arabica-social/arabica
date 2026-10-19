<script lang="ts">
  import BackButton from "$lib/components/BackButton.svelte";
</script>

<svelte:head>
  <title>The AT Protocol - Arabica</title>
  <meta
    name="description"
    content="How Arabica uses AT Protocol accounts, Personal Data Servers, and public coffee records."
  />
  <meta property="og:title" content="The AT Protocol - Arabica" />
  <meta property="og:type" content="website" />
</svelte:head>

<div class="info-page">
  <div class="info-hero">
    <div class="info-title-group">
      <div class="info-eyebrow">
        <BackButton />
      </div>
      <h1 class="info-title">The AT Protocol</h1>
      <p class="info-deck">
        AT Protocol is the shared account and data system used by Arabica,
        Bluesky, and other apps.
      </p>
    </div>
  </div>
  <div class="prose prose-lg max-w-none info-stack">
    <section>
      <h2 class="text-2xl font-semibold text-primary mb-4">
        Your account and server
      </h2>
      <p class="text-secondary leading-relaxed mb-3">
        You can log in to Arabica with an existing Bluesky or other AT Protocol
        account. Your provider handles login through OAuth. Arabica never sees
        your password.
      </p>
      <p class="text-secondary leading-relaxed mb-3">
        Your Personal Data Server (PDS) stores your coffee records. A provider
        such as
        <a
          href="https://bsky.app"
          class="link-bold"
          target="_blank"
          rel="noopener noreferrer">Bluesky</a
        >,
        <a
          href="https://selfhosted.social"
          class="link-bold"
          target="_blank"
          rel="noopener noreferrer">selfhosted.social</a
        >, or
        <a
          href="https://eurosky.cloud"
          class="link-bold"
          target="_blank"
          rel="noopener noreferrer">Eurosky</a
        >
        can host it, or you can
        <a
          href="https://atproto.com/guides/self-hosting"
          class="link-bold"
          target="_blank"
          rel="noopener noreferrer">host your own</a
        >. AT Protocol supports moving your account between providers.
      </p>
      <p class="text-secondary leading-relaxed">
        Your account has a stable identifier called a DID (Decentralized
        Identifier). Your handle, such as yourname.bsky.social, is its
        human-readable alias.
      </p>
    </section>
    <section>
      <h2 class="text-2xl font-semibold text-primary mb-4">
        Storage and privacy
      </h2>
      <p class="text-secondary leading-relaxed mb-3">
        Arabica writes records to your PDS and uses local indexes and caches to
        make reading them faster. The PDS remains the source of truth; Arabica's
        copies are rebuildable.
      </p>
      <p class="text-secondary leading-relaxed">
        AT Protocol repository records are public. Other apps and services can
        read and copy them, including through the network's public record
        stream. Do not put private information in coffee records. Hiding a
        record from an Arabica feed does not make it private on your PDS or
        remove copies held elsewhere.
      </p>
    </section>
    <section>
      <h2 class="text-2xl font-semibold text-primary mb-4">
        Record formats and references
      </h2>
      <p class="text-secondary leading-relaxed mb-3">
        Lexicons define record formats. Arabica uses coffee-specific lexicons
        for brews, beans, recipes, and equipment. Other apps can read these
        records if they support the same formats.
      </p>
      <p class="text-secondary leading-relaxed">
        Records link to each other with AT-URIs, such as
        <code class="bg-brown-200 px-1 rounded-sm text-xs"
          >at://did:plc:xxx/collection/recordkey</code
        >. A brew uses one of these references to identify its bean record.
      </p>
    </section>
    <section>
      <h2 class="text-2xl font-semibold text-primary mb-4">Learn More</h2>
      <ul class="info-links text-secondary">
        <li>
          <a
            href="https://atproto.com"
            class="link-bold"
            target="_blank"
            rel="noopener noreferrer">atproto.com</a
          >: official documentation
        </li>
        <li>
          <a
            href="https://atproto.com/guides/overview"
            class="link-bold"
            target="_blank"
            rel="noopener noreferrer">Protocol Overview</a
          >: technical introduction
        </li>
        <li>
          <a
            href="https://bsky.app"
            class="link-bold"
            target="_blank"
            rel="noopener noreferrer">Bluesky</a
          >
          and
          <a
            href="https://tangled.sh"
            class="link-bold"
            target="_blank"
            rel="noopener noreferrer">Tangled</a
          >: other apps that use AT Protocol
        </li>
      </ul>
    </section>
  </div>
  <div class="info-cta">
    <a href="/" class="btn-primary px-8 py-3 font-semibold">Back to Home</a>
  </div>
</div>
