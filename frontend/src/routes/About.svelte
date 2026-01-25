<script>
  import { navigate } from "../lib/router.js";
</script>

<div class="max-w-4xl mx-auto">
  <div
    class="bg-gradient-to-br from-amber-50 to-brown-100 rounded-xl p-8 border-2 border-brown-300 shadow-lg"
  >
    <h1 class="text-3xl font-bold text-brown-900 mb-6">About Arabica</h1>

    <div class="prose prose-brown max-w-none">
      <p class="text-lg text-brown-800 mb-4">
        Arabica is a coffee brew tracking application that leverages the AT
        Protocol for decentralized data storage.
      </p>

      <h2 class="text-2xl font-bold text-brown-900 mt-8 mb-4">Features</h2>
      <ul class="space-y-2 text-brown-800">
        <li class="flex items-start">
          <span class="mr-2">🔒</span>
          <span
            ><strong>Decentralized:</strong> Your data lives in your Personal Data
            Server (PDS)</span
          >
        </li>
        <li class="flex items-start">
          <span class="mr-2">🚀</span>
          <span><strong>Portable:</strong> Own your coffee brewing history</span
          >
        </li>
        <li class="flex items-start">
          <span class="mr-2">📊</span>
          <span
            >Track brewing variables like temperature, time, and grind size</span
          >
        </li>
        <li class="flex items-start">
          <span class="mr-2">🌍</span>
          <span>Organize beans by origin and roaster</span>
        </li>
        <li class="flex items-start">
          <span class="mr-2">📝</span>
          <span>Add tasting notes and ratings to each brew</span>
        </li>
      </ul>

      <h2 class="text-2xl font-bold text-brown-900 mt-8 mb-4">AT Protocol</h2>
      <p class="text-brown-800 mb-4">
        The Authenticated Transfer Protocol (AT Protocol) is a decentralized
        social networking protocol that gives you full ownership of your data.
        Your brewing records are stored in your own PDS, not in Arabica's
        servers.
      </p>

      <div class="mt-8">
        <button
          on:click={() => navigate("/")}
          class="bg-gradient-to-r from-brown-700 to-brown-800 text-white px-6 py-3 rounded-lg hover:from-brown-800 hover:to-brown-900 transition-all font-semibold shadow-lg"
        >
          Get Started
        </button>
      </div>
    </div>
  </div>
</div>
