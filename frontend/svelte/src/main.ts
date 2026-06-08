import { mount } from 'svelte';
import AdminDashboardIsland from './AdminDashboardIsland.svelte';
import AppCacheRuntimeIsland from './AppCacheRuntimeIsland.svelte';
import type { AppCacheAPI } from './appCache';
import BeanRatingIsland from './BeanRatingIsland.svelte';
import BeanRoasterPickerIsland from './BeanRoasterPickerIsland.svelte';
import BeanViewActionsIsland from './BeanViewActionsIsland.svelte';
import BrewComboSelectIsland from './BrewComboSelectIsland.svelte';
import BrewFormIsland from './BrewFormIsland.svelte';
import BrewMethodSectionIsland from './BrewMethodSectionIsland.svelte';
import BrewPoursIsland from './BrewPoursIsland.svelte';
import BrewRatingIsland from './BrewRatingIsland.svelte';
import BrewRecipeSummaryIsland from './BrewRecipeSummaryIsland.svelte';
import BrewWaterHelperIsland from './BrewWaterHelperIsland.svelte';
import CommentReplyIsland from './CommentReplyIsland.svelte';
import DisclosureIsland from './DisclosureIsland.svelte';
import EntitySuggestIsland from './EntitySuggestIsland.svelte';
import FeedFiltersIsland from './FeedFiltersIsland.svelte';
import FeedMasonryIsland from './FeedMasonryIsland.svelte';
import GlobalActionsIsland from './GlobalActionsIsland.svelte';
import HandleAutocompleteIsland from './HandleAutocompleteIsland.svelte';
import LayoutRuntimeIsland from './LayoutRuntimeIsland.svelte';
import ManageTabsIsland from './ManageTabsIsland.svelte';
import CommentSectionIsland from './CommentSectionIsland.svelte';
import ModalContainerIsland from './ModalContainerIsland.svelte';
import ModalShellIsland from './ModalShellIsland.svelte';
import OolongFormIsland from './OolongFormIsland.svelte';
import ProfileStatsIsland from './ProfileStatsIsland.svelte';
import RecipeExploreIsland from './RecipeExploreIsland.svelte';
import RecipeForkButtonIsland from './RecipeForkButtonIsland.svelte';
import SaveAsRecipeIsland from './SaveAsRecipeIsland.svelte';
import ScrollTopIsland from './ScrollTopIsland.svelte';
import ServiceWorkerIsland from './ServiceWorkerIsland.svelte';
import ShareButtonIsland from './ShareButtonIsland.svelte';
import ActionMoreMenuIsland from './ActionMoreMenuIsland.svelte';
import SettingsControlsIsland from './SettingsControlsIsland.svelte';
import SettingsFormIsland from './SettingsFormIsland.svelte';
import ThemeRuntimeIsland from './ThemeRuntimeIsland.svelte';
import TransitionRuntimeIsland from './TransitionRuntimeIsland.svelte';

declare global {
  interface Window {
    htmx?: {
      ajax?: (
        method: string,
        url: string,
        options: { target: string; swap: string; select?: string }
      ) => Promise<unknown> | unknown;
      trigger?: (target: string | Element, eventName: string) => void;
      config?: {
        globalViewTransitions?: boolean;
      };
    };
    __arabicaSvelteIslands?: {
      mountAll: () => void;
      applyFeedMasonry: () => void;
    };
    __showSessionExpiredModal?: () => void;
    applyTheme?: () => void;
    AppCache?: AppCacheAPI;
  }
}

let mounted = false;
const brewPoursIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const brewComboSelectIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const brewRatingIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const brewWaterHelperIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const brewMethodSectionIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const brewRecipeSummaryIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const feedFiltersIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const recipeExploreIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const manageTabsIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const commentSectionIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const entitySuggestIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const modalContainerIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const modalShellIslands = new WeakMap<HTMLFormElement, ReturnType<typeof mount>>();
const oolongFormIslands = new WeakMap<HTMLFormElement, ReturnType<typeof mount>>();
const beanRoasterPickerIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const beanRatingIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const beanViewActionsIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const disclosureIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const shareButtonIslands = new WeakMap<HTMLButtonElement, ReturnType<typeof mount>>();
const actionMoreMenuIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const commentReplyIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const settingsControlsIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const settingsFormIslands = new WeakMap<HTMLFormElement, ReturnType<typeof mount>>();
const scrollTopIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const adminDashboardIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const saveAsRecipeIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const recipeForkButtonIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const brewFormIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const handleAutocompleteIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const profileStatsIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();

function mountBrewPours() {
  document
    .querySelectorAll<HTMLElement>('[data-svelte-brew-pours], [data-svelte-recipe-pours]')
    .forEach((target) => {
      if (brewPoursIslands.has(target)) {
        return;
      }
      target.innerHTML = '';
      brewPoursIslands.set(
        target,
        mount(BrewPoursIsland, {
          target,
          props: { target }
        })
      );
    });
}

function mountFeedFilters() {
  document.querySelectorAll<HTMLElement>('[data-svelte-feed-filters]').forEach((target) => {
    if (feedFiltersIslands.has(target)) {
      return;
    }
    let initialTabs: Array<{ label: string; value: string }> = [];
    try {
      const parsed = JSON.parse(target.dataset.tabs || '[]');
      if (Array.isArray(parsed)) {
        initialTabs = parsed
          .map((tab) => {
            const record = tab as Record<string, unknown>;
            return {
              label: String(record.label ?? record.Label ?? ''),
              value: String(record.value ?? record.Value ?? '')
            };
          })
          .filter((tab) => tab.label);
      }
    } catch {
      initialTabs = [];
    }
    target.innerHTML = '';
    feedFiltersIslands.set(
      target,
      mount(FeedFiltersIsland, {
        target,
        props: {
          initialType: target.dataset.initialType || '',
          initialSort: target.dataset.initialSort || 'recent',
          initialTabs
        }
      })
    );
  });
}

function mountRecipeExplore() {
  document.querySelectorAll<HTMLElement>('[data-svelte-recipe-explore]').forEach((target) => {
    if (recipeExploreIslands.has(target)) {
      return;
    }
    target.innerHTML = '';
    recipeExploreIslands.set(
      target,
      mount(RecipeExploreIsland, {
        target,
        props: { target }
      })
    );
  });
}

function mountManageTabs() {
  document.querySelectorAll<HTMLElement>('[data-svelte-manage-tabs]').forEach((target) => {
    if (manageTabsIslands.has(target)) {
      return;
    }
    manageTabsIslands.set(
      target,
      mount(ManageTabsIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountCommentSections() {
  document.querySelectorAll<HTMLElement>('[data-svelte-comment-section]').forEach((target) => {
    if (commentSectionIslands.has(target)) {
      return;
    }
    commentSectionIslands.set(
      target,
      mount(CommentSectionIsland, {
        target,
        props: { target }
      })
    );
  });
}

function mountEntitySuggests() {
  document.querySelectorAll<HTMLElement>('[data-svelte-entity-suggest]').forEach((target) => {
    if (entitySuggestIslands.has(target)) {
      return;
    }
    target.innerHTML = '';
    entitySuggestIslands.set(
      target,
      mount(EntitySuggestIsland, {
        target,
        props: {
          target,
          endpoint: target.dataset.endpoint || '',
          entityType: target.dataset.entityType || '',
          placeholder: target.dataset.placeholder || 'Name *'
        }
      })
    );
  });
}

function mountModalShells() {
  document.querySelectorAll<HTMLFormElement>('form[data-svelte-modal-shell]').forEach((target) => {
    if (modalShellIslands.has(target)) {
      return;
    }
    modalShellIslands.set(
      target,
      mount(ModalShellIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountModalContainers() {
  document.querySelectorAll<HTMLElement>('#modal-container').forEach((target) => {
    if (modalContainerIslands.has(target)) {
      return;
    }
    modalContainerIslands.set(
      target,
      mount(ModalContainerIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountOolongForms() {
  document.querySelectorAll<HTMLFormElement>('form[data-svelte-oolong-form]').forEach((target) => {
    if (oolongFormIslands.has(target)) {
      return;
    }
    oolongFormIslands.set(
      target,
      mount(OolongFormIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountBeanRoasterPickers() {
  document.querySelectorAll<HTMLElement>('[data-svelte-bean-roaster-picker]').forEach((target) => {
    if (beanRoasterPickerIslands.has(target)) {
      return;
    }
    target.innerHTML = '';
    beanRoasterPickerIslands.set(
      target,
      mount(BeanRoasterPickerIsland, {
        target,
        props: { target }
      })
    );
  });
}

function mountBeanRatings() {
  document.querySelectorAll<HTMLElement>('[data-svelte-bean-rating], [data-svelte-rating-toggle]').forEach((target) => {
    if (beanRatingIslands.has(target)) {
      return;
    }
    const initialRating = Number(target.dataset.initialRating || '0');
    target.innerHTML = '';
    beanRatingIslands.set(
      target,
      mount(BeanRatingIsland, {
        target,
        props: { initialRating: Number.isNaN(initialRating) ? 0 : initialRating }
      })
    );
  });
}

function mountBeanViewActions() {
  document.querySelectorAll<HTMLElement>('[data-svelte-bean-view-actions]').forEach((target) => {
    if (beanViewActionsIslands.has(target)) {
      return;
    }
    let baseBean: Record<string, unknown> = {};
    try {
      baseBean = JSON.parse(target.dataset.baseBean || '{}');
    } catch {
      baseBean = {};
    }
    const initialRating = Number(target.dataset.initialRating || '5');
    target.innerHTML = '';
    beanViewActionsIslands.set(
      target,
      mount(BeanViewActionsIsland, {
        target,
        props: {
          beanRKey: target.dataset.beanRkey || '',
          baseBean,
          initialRating: Number.isNaN(initialRating) ? 5 : initialRating,
          hasRating: target.dataset.hasRating === 'true',
          closed: target.dataset.closed === 'true'
        }
      })
    );
  });
}

function mountDisclosures() {
  document.querySelectorAll<HTMLElement>('[data-svelte-disclosure]').forEach((target) => {
    if (disclosureIslands.has(target)) {
      return;
    }
    disclosureIslands.set(
      target,
      mount(DisclosureIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountShareButtons() {
  document.querySelectorAll<HTMLButtonElement>('button[data-svelte-share]').forEach((target) => {
    if (shareButtonIslands.has(target)) {
      return;
    }
    shareButtonIslands.set(
      target,
      mount(ShareButtonIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountActionMoreMenus() {
  document.querySelectorAll<HTMLElement>('[data-svelte-action-more-menu]').forEach((target) => {
    if (actionMoreMenuIslands.has(target)) {
      return;
    }
    actionMoreMenuIslands.set(
      target,
      mount(ActionMoreMenuIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountCommentReplies() {
  document.querySelectorAll<HTMLElement>('[data-svelte-comment-reply]').forEach((target) => {
    if (commentReplyIslands.has(target)) {
      return;
    }
    commentReplyIslands.set(
      target,
      mount(CommentReplyIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountSettingsControls() {
  document.querySelectorAll<HTMLElement>('[data-svelte-settings-controls]').forEach((target) => {
    if (settingsControlsIslands.has(target)) {
      return;
    }
    settingsControlsIslands.set(
      target,
      mount(SettingsControlsIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountSettingsForms() {
  document.querySelectorAll<HTMLFormElement>('[data-svelte-settings-form]').forEach((target) => {
    if (settingsFormIslands.has(target)) {
      return;
    }
    settingsFormIslands.set(
      target,
      mount(SettingsFormIsland, {
        target,
        props: { target }
      })
    );
  });
}

function mountScrollTopControls() {
  document.querySelectorAll<HTMLElement>('[data-svelte-scroll-top]').forEach((target) => {
    if (scrollTopIslands.has(target)) {
      return;
    }
    scrollTopIslands.set(
      target,
      mount(ScrollTopIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountAdminDashboards() {
  document.querySelectorAll<HTMLElement>('[data-svelte-admin-dashboard]').forEach((target) => {
    if (adminDashboardIslands.has(target)) {
      return;
    }
    adminDashboardIslands.set(
      target,
      mount(AdminDashboardIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountSaveAsRecipeControls() {
  document.querySelectorAll<HTMLElement>('[data-svelte-save-as-recipe]').forEach((target) => {
    if (saveAsRecipeIslands.has(target)) {
      return;
    }
    target.innerHTML = '';
    saveAsRecipeIslands.set(
      target,
      mount(SaveAsRecipeIsland, {
        target,
        props: { brewRKey: target.dataset.brewRkey || '' }
      })
    );
  });
}

function mountRecipeForkButtons() {
  document.querySelectorAll<HTMLElement>('[data-svelte-recipe-fork]').forEach((target) => {
    if (recipeForkButtonIslands.has(target)) {
      return;
    }
    target.innerHTML = '';
    recipeForkButtonIslands.set(
      target,
      mount(RecipeForkButtonIsland, {
        target,
        props: {
          recipeRKey: target.dataset.recipeRkey || '',
          ownerDID: target.dataset.ownerDid || ''
        }
      })
    );
  });
}

function mountHandleAutocompletes() {
  document.querySelectorAll<HTMLElement>('[data-svelte-handle-autocomplete]').forEach((target) => {
    if (handleAutocompleteIslands.has(target)) {
      return;
    }
    const container = target.closest<HTMLElement>('[data-handle-autocomplete-root]') || target.parentElement;
    const input = container?.querySelector<HTMLInputElement>('input[name="handle"]');
    if (!input) {
      return;
    }
    target.innerHTML = '';
    target.classList.remove('hidden');
    handleAutocompleteIslands.set(
      target,
      mount(HandleAutocompleteIsland, {
        target,
        props: { input, target }
      })
    );
  });
}

function mountProfileStats() {
  document.querySelectorAll<HTMLElement>('[data-svelte-profile-stats]').forEach((target) => {
    if (profileStatsIslands.has(target)) {
      return;
    }
    target.innerHTML = '';
    profileStatsIslands.set(
      target,
      mount(ProfileStatsIsland, {
        target,
        props: { target }
      })
    );
  });
}

function mountBrewComboSelects() {
  document.querySelectorAll<HTMLElement>('[data-svelte-brew-combo]').forEach((target) => {
    if (brewComboSelectIslands.has(target)) {
      return;
    }
    const required = target.dataset.required === 'true';
    const passthrough = target.dataset.passthrough === 'true';
    const allowCreate = target.dataset.allowCreate !== 'false';
    target.innerHTML = '';
    brewComboSelectIslands.set(
      target,
      mount(BrewComboSelectIsland, {
        target,
        props: {
          target,
          entityType: target.dataset.entityType || '',
          apiEndpoint: target.dataset.apiEndpoint || '',
          suggestEndpoint: target.dataset.suggestEndpoint || '',
          inputName: target.dataset.inputName || '',
          placeholder: target.dataset.placeholder || 'Search...',
          sectionLabel: target.dataset.sectionLabel || 'Your records',
          required,
          passthrough,
          allowCreate,
          initialRKey: target.dataset.initialRkey || '',
          initialLabel: target.dataset.initialLabel || ''
        }
      })
    );
  });
}

function mountBrewForms() {
  document.querySelectorAll<HTMLElement>('[data-svelte-brew-form]').forEach((target) => {
    if (brewFormIslands.has(target)) {
      return;
    }
    brewFormIslands.set(
      target,
      mount(BrewFormIsland, {
        target: document.body,
        props: { target }
      })
    );
  });
}

function mountBrewMethodSections() {
  document.querySelectorAll<HTMLElement>('[data-svelte-brew-method-section]').forEach((target) => {
    if (brewMethodSectionIslands.has(target)) {
      return;
    }
    brewMethodSectionIslands.set(
      target,
      mount(BrewMethodSectionIsland, {
        target: document.body,
        props: {
          target,
          method: target.dataset.svelteBrewMethodSection || ''
        }
      })
    );
  });
}

function mountBrewRecipeSummaries() {
  document.querySelectorAll<HTMLElement>('[data-svelte-brew-recipe-summary]').forEach((target) => {
    if (brewRecipeSummaryIslands.has(target)) {
      return;
    }
    target.innerHTML = '';
    brewRecipeSummaryIslands.set(
      target,
      mount(BrewRecipeSummaryIsland, {
        target,
        props: { target }
      })
    );
  });
}

function mountBrewRatings() {
  document.querySelectorAll<HTMLElement>('[data-svelte-brew-rating], [data-svelte-rating-slider]').forEach((target) => {
    if (brewRatingIslands.has(target)) {
      return;
    }
    const initialRating = Number(target.dataset.initialRating || '5');
    target.innerHTML = '';
    brewRatingIslands.set(
      target,
      mount(BrewRatingIsland, {
        target,
        props: { initialRating: Number.isNaN(initialRating) ? 5 : initialRating }
      })
    );
  });
}

function mountBrewWaterHelpers() {
  document.querySelectorAll<HTMLElement>('[data-svelte-brew-water-helper]').forEach((target) => {
    if (brewWaterHelperIslands.has(target)) {
      return;
    }
    target.innerHTML = '';
    brewWaterHelperIslands.set(
      target,
      mount(BrewWaterHelperIsland, {
        target,
        props: { initialShowPours: target.dataset.initialShowPours === 'true' }
      })
    );
  });
}

function mountAll() {
  if (!mounted) {
    mount(FeedMasonryIsland, {
      target: document.body
    });
    mount(AppCacheRuntimeIsland, {
      target: document.body
    });
    mount(GlobalActionsIsland, {
      target: document.body
    });
    mount(LayoutRuntimeIsland, {
      target: document.body
    });
    mount(ThemeRuntimeIsland, {
      target: document.body
    });
    mount(TransitionRuntimeIsland, {
      target: document.body
    });
    mount(ServiceWorkerIsland, {
      target: document.body
    });
    mounted = true;
  }
  mountBrewRatings();
  mountBrewComboSelects();
  mountBrewWaterHelpers();
  mountBrewMethodSections();
  mountBrewRecipeSummaries();
  mountBrewPours();
  mountFeedFilters();
  mountRecipeExplore();
  mountManageTabs();
  mountCommentSections();
  mountEntitySuggests();
  mountModalContainers();
  mountModalShells();
  mountOolongForms();
  mountBeanRoasterPickers();
  mountBeanRatings();
  mountBeanViewActions();
  mountDisclosures();
  mountShareButtons();
  mountActionMoreMenus();
  mountCommentReplies();
  mountSettingsControls();
  mountSettingsForms();
  mountScrollTopControls();
  mountAdminDashboards();
  mountSaveAsRecipeControls();
  mountRecipeForkButtons();
  mountHandleAutocompletes();
  mountProfileStats();
  mountBrewForms();
  window.__arabicaApplyFeedMasonry?.();
}

window.__arabicaSvelteIslands = {
  mountAll,
  applyFeedMasonry: () => window.__arabicaApplyFeedMasonry?.()
};

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', mountAll, { once: true });
} else {
  mountAll();
}

document.addEventListener('htmx:afterSwap', () => {
  mountAll();
});

document.addEventListener('htmx:load', () => {
  mountAll();
});
