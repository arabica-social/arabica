<script lang="ts">
  let {
    message = "The page you're looking for could not be found.",
  }: { message?: string } = $props();
</script>

<div class="page-container-lg">
  <div class="card p-8 text-center">
    <div class="text-6xl mb-4 font-bold text-secondary">404</div>
    <h1 class="text-2xl font-bold text-primary mb-4">Page Not Found</h1>
    <p class="text-emphasis mb-6">{message}</p>
    <a href="/" class="btn-primary py-3 px-6">Back to Home</a>
  </div>
</div>
