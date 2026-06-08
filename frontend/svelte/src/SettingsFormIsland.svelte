<script lang="ts">
  interface Props {
    target: HTMLFormElement;
  }

  let { target }: Props = $props();

  let isSaving = $state(false);
  let statusMessage = $state('');
  let statusType = $state<'success' | 'error' | ''>('');
  let statusNode = $state<HTMLElement | null>(null);

  const endpoint = target.dataset.settingsEndpoint || target.action || '';
  const method = (target.method || 'POST').toUpperCase();

  const classes = {
    success: 'text-sm text-green-700 dark:text-green-400',
    error: 'text-sm text-danger'
  };

  function updateStatusUI() {
    if (!statusNode) {
      return;
    }
    statusNode.textContent = statusMessage;
    statusNode.className = statusType ? classes[statusType] : '';
  }

  function setStatus(message: string, type: 'success' | 'error') {
    statusMessage = message;
    statusType = type;
    updateStatusUI();
  }

  function clearStatus() {
    statusMessage = '';
    statusType = '';
    updateStatusUI();
  }

  function setSubmitting(value: boolean) {
    target.querySelectorAll<HTMLButtonElement>('button[type="submit"]').forEach((button) => {
      button.disabled = value;
    });
  }

  async function handleSubmit(event: SubmitEvent) {
    event.preventDefault();

    if (isSaving || !endpoint) {
      if (!endpoint) {
        setStatus('Missing settings endpoint', 'error');
      }
      return;
    }

    isSaving = true;
    clearStatus();
    setSubmitting(true);

    try {
      const response = await fetch(endpoint, {
        method,
        credentials: 'same-origin',
        body: new FormData(target)
      });

      const text = await response.text();
      if (!response.ok) {
        setStatus(text.trim() || 'Save failed', 'error');
        if (response.status === 401) {
          window.__showSessionExpiredModal?.();
        }
        return;
      }

      setStatus(text.trim() || 'Saved', 'success');
    } catch (error) {
      console.error('Failed to save settings form:', error);
      setStatus('Save failed', 'error');
    } finally {
      isSaving = false;
      setSubmitting(false);
    }
  }

  $effect(() => {
    statusNode = target.querySelector<HTMLElement>('[data-settings-save-status]');
    target.addEventListener('submit', handleSubmit);
    return () => {
      target.removeEventListener('submit', handleSubmit);
      setSubmitting(false);
    };
  });
</script>
