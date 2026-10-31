<script lang="ts">
  import ChatWorkspace from "#lib/components/ai/ChatWorkspace.svelte";
  import { session } from "#lib/stores/session.js";
  let { data }: { data: { enabled: boolean } } = $props();
</script>

<svelte:head><title>Chat · Arabica</title></svelte:head>
<ChatWorkspace
  accountDid={$session.isAuthenticated ? $session.did : ""}
  enabled={data.enabled}
/>
