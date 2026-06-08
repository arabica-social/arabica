<script lang="ts">
  import { onMount } from 'svelte';

  function applyTheme() {
    try {
      const theme = localStorage.getItem('arabica-theme');
      if (theme === 'dark' || theme === 'light') {
        document.documentElement.setAttribute('data-theme', theme);
      } else {
        document.documentElement.removeAttribute('data-theme');
      }
    } catch {
      document.documentElement.removeAttribute('data-theme');
    }
  }

  onMount(() => {
    window.applyTheme = applyTheme;
    document.addEventListener('htmx:afterSettle', applyTheme);
    document.addEventListener('htmx:historyRestore', applyTheme);
    applyTheme();

    return () => {
      document.removeEventListener('htmx:afterSettle', applyTheme);
      document.removeEventListener('htmx:historyRestore', applyTheme);
      if (window.applyTheme === applyTheme) {
        delete window.applyTheme;
      }
    };
  });
</script>
