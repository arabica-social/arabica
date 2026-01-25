<script>
  import { navigate } from "../lib/router.js";
</script>

<footer class="bg-brown-800 text-brown-100 mt-12">
  <div class="container mx-auto px-4 py-8">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-8">
      <div>
        <h3 class="text-lg font-bold mb-3 flex items-center gap-2">
          <span>☕</span>
          <span>Arabica</span>
        </h3>
        <p class="text-sm text-brown-300">
          Track your coffee brewing journey with decentralized data storage
          powered by AT Protocol.
        </p>
      </div>

      <div>
        <h4 class="font-semibold mb-3">Links</h4>
        <ul class="space-y-2 text-sm">
          <li>
            <a
              href="/about"
              on:click|preventDefault={() => navigate("/about")}
              class="text-brown-300 hover:text-white transition-colors"
            >
              About
            </a>
          </li>
          <li>
            <a
              href="/terms"
              on:click|preventDefault={() => navigate("/terms")}
              class="text-brown-300 hover:text-white transition-colors"
            >
              Terms of Service
            </a>
          </li>
          <li>
            <a
              href="https://tangled.org/arabica.social/arabica"
              target="_blank"
              rel="noopener noreferrer"
              class="text-brown-300 hover:text-white transition-colors"
            >
              Source
            </a>
          </li>
        </ul>
      </div>

      <div>
        <h4 class="font-semibold mb-3">AT Protocol</h4>
        <p class="text-sm text-brown-300">
          Your data lives in your Personal Data Server (PDS), giving you full
          ownership and portability.
        </p>
      </div>
    </div>

    <div
      class="border-t border-brown-700 mt-8 pt-6 text-center text-sm text-brown-400"
    >
      <p>
        &copy; {new Date().getFullYear()} Arabica Social. All rights reserved.
      </p>
    </div>
  </div>
</footer>
