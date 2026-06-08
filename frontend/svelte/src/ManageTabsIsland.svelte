<script lang="ts">
  import { onMount } from 'svelte';

  type NudgePayload = {
    name?: string;
    missing?: string;
    entity_type?: string;
    rkey?: string;
  };

  let { target }: { target: HTMLElement } = $props();

  let activeTab = $state('brews');
  let syncing = $state(false);

  function readStoredTab(storageKey: string) {
    try {
      return localStorage.getItem(storageKey) || '';
    } catch {
      return '';
    }
  }

  function writeStoredTab(storageKey: string, tab: string) {
    try {
      localStorage.setItem(storageKey, tab);
    } catch {
      // Device-local preference only.
    }
  }

  function setActiveTab(tab: string) {
    if (!tab) {
      return;
    }
    const tabs = Array.from(target.querySelectorAll<HTMLElement>('[data-manage-tab]')).map(
      (button) => button.dataset.manageTab || ''
    );
    if (!tabs.includes(tab)) {
      tab = target.dataset.initialTab || tabs[0] || 'brews';
    }
    activeTab = tab;
    target.dataset.activeTab = tab;
    const storageKey = target.dataset.tabStorageKey || '';
    if (storageKey) {
      writeStoredTab(storageKey, tab);
    }
    refreshTabButtons();
  }

  function refreshTabButtons() {
    target.querySelectorAll<HTMLElement>('[data-manage-tab]').forEach((button) => {
      const isActive = button.dataset.manageTab === activeTab;
      const activeClass = button.dataset.activeClass || 'tab-row-active';
      const inactiveClass = button.dataset.inactiveClass || 'tab-row-inactive';
      button.classList.toggle(activeClass, isActive);
      button.classList.toggle(inactiveClass, !isActive);
      button.setAttribute('aria-selected', isActive ? 'true' : 'false');
    });
  }

  function showIncompleteNudge() {
    try {
      const raw = sessionStorage.getItem('incompleteNudge');
      if (!raw) {
        return;
      }
      sessionStorage.removeItem('incompleteNudge');
      const nudge = JSON.parse(raw) as NudgePayload;
      if (!nudge.name || !nudge.missing) {
        return;
      }

      const toast = document.createElement('div');
      toast.className = 'nudge-toast';

      const body = document.createElement('div');
      body.className = 'flex-1 text-sm';
      const strong = document.createElement('strong');
      strong.textContent = nudge.name;
      body.append(strong, document.createTextNode(` is missing ${nudge.missing}`));

      const complete = document.createElement('button');
      complete.type = 'button';
      complete.className = 'text-sm font-medium hover:opacity-80 whitespace-nowrap';
      complete.style.color = 'var(--accent-primary, #5d4037)';
      complete.textContent = 'Complete';
      complete.addEventListener('click', () => {
        toast.remove();
        if (!nudge.entity_type || !nudge.rkey) {
          return;
        }
        window.htmx?.ajax?.('GET', `/api/modals/${nudge.entity_type}/${nudge.rkey}`, {
          target: '#modal-container',
          swap: 'innerHTML'
        });
      });

      const dismiss = document.createElement('button');
      dismiss.type = 'button';
      dismiss.className = 'text-brown-400 hover:text-brown-600';
      dismiss.setAttribute('aria-label', 'Dismiss');
      dismiss.innerHTML =
        '<svg class="w-4 h-4" fill="none" stroke="currentColor" stroke-width="2" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12"/></svg>';
      dismiss.addEventListener('click', () => toast.remove());

      toast.append(body, complete, dismiss);
      document.body.appendChild(toast);
      window.setTimeout(() => toast.remove(), 10000);
    } catch {
      // Ignore malformed one-shot session state.
    }
  }

  function refreshSyncButton() {
    target.querySelectorAll<HTMLButtonElement>('[data-manage-refresh]').forEach((button) => {
      button.disabled = syncing;
      button.querySelector('[data-sync-idle]')?.toggleAttribute('hidden', syncing);
      button.querySelector('[data-sync-busy]')?.toggleAttribute('hidden', !syncing);
      button.querySelector('[data-sync-icon]')?.classList.toggle('animate-spin', syncing);
    });
  }

  async function syncFromPDS() {
    syncing = true;
    refreshSyncButton();
    try {
      const refreshURL = target.dataset.refreshUrl || '/api/manage/refresh';
      if (target.dataset.refreshReload === 'true') {
        await fetch(refreshURL, { method: 'POST', credentials: 'same-origin' });
        window.location.reload();
        return;
      }
      await Promise.resolve(
        window.htmx?.ajax?.('POST', refreshURL, {
          target: target.dataset.refreshTarget || '#manage-content',
          swap: 'innerHTML'
        })
      );
    } finally {
      syncing = false;
      refreshSyncButton();
    }
  }

  onMount(() => {
    const storageKey = target.dataset.tabStorageKey || '';
    const initialTab = (storageKey && readStoredTab(storageKey)) || target.dataset.initialTab || 'brews';

    if (target.dataset.initCache === 'true') {
      void window.AppCache?.init?.();
    }
    if (target.dataset.showNudge === 'true') {
      showIncompleteNudge();
    }

    const handleClick = (event: Event) => {
      const node = event.target instanceof Element ? event.target : null;
      const tabButton = node?.closest<HTMLElement>('[data-manage-tab]');
      if (tabButton && target.contains(tabButton)) {
        setActiveTab(tabButton.dataset.manageTab || '');
        return;
      }
      const refreshButton = node?.closest<HTMLButtonElement>('[data-manage-refresh]');
      if (refreshButton && target.contains(refreshButton)) {
        void syncFromPDS();
      }
    };

    target.addEventListener('click', handleClick);
    setActiveTab(initialTab);
    refreshSyncButton();

    return () => {
      target.removeEventListener('click', handleClick);
    };
  });
</script>
