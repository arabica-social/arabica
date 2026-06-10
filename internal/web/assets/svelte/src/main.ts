import { mount } from "svelte";
import type { AppCacheAPI } from "./appCache";

type IslandModule = {
  default: any;
};

declare global {
  interface Window {
    htmx?: {
      ajax?: (
        method: string,
        url: string,
        options: { target: string; swap: string; select?: string },
      ) => Promise<unknown> | unknown;
      trigger?: (target: string | Element, eventName: string) => void;
      process?: (target: Element | Document) => void;
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

const islandLoaders = new Map<string, Promise<any>>();

function loadIsland(
  id: string,
  importer: () => Promise<IslandModule>,
): Promise<any> {
  const cached = islandLoaders.get(id);
  if (cached) {
    return cached;
  }

  const loader = importer()
    .then((module) => module.default)
    .catch((error) => {
      islandLoaders.delete(id);
      throw error;
    });

  islandLoaders.set(id, loader);
  return loader;
}

let mounted = false;

const comboSelectIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const feedFiltersIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const recipeExploreIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const manageTabsIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const commentSectionIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const entitySuggestIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const modalContainerIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const modalShellIslands = new WeakMap<
  HTMLFormElement,
  ReturnType<typeof mount>
>();
const oolongFormIslands = new WeakMap<
  HTMLFormElement,
  ReturnType<typeof mount>
>();
const beanRoasterPickerIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const beanRatingIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const beanViewActionsIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const disclosureIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const shareButtonIslands = new WeakMap<
  HTMLButtonElement,
  ReturnType<typeof mount>
>();
const actionMoreMenuIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const commentReplyIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const settingsControlsIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const settingsFormIslands = new WeakMap<
  HTMLFormElement,
  ReturnType<typeof mount>
>();
const scrollTopIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const adminDashboardIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const saveAsRecipeIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const recipeForkButtonIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const brewFormIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const recipeFormIslands = new WeakMap<HTMLElement, ReturnType<typeof mount>>();
const handleAutocompleteIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const profileStatsIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
const tasteProfileIslands = new WeakMap<
  HTMLElement,
  ReturnType<typeof mount>
>();
let feedLayoutMounted = false;

async function mountComboSelects() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-brew-combo]",
  );
  if (targets.length === 0) {
    return;
  }
  const EntityCombo = await loadIsland(
    "entity-combo",
    () => import("./EntityCombo.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (comboSelectIslands.has(target)) {
      return;
    }
    const required = target.dataset.required === "true";
    const passthrough = target.dataset.passthrough === "true";
    const allowCreate = target.dataset.allowCreate !== "false";
    target.innerHTML = "";
    comboSelectIslands.set(
      target,
      mount(EntityCombo, {
        target,
        props: {
          entityType: target.dataset.entityType || "",
          apiEndpoint: target.dataset.apiEndpoint || "",
          suggestEndpoint: target.dataset.suggestEndpoint || "",
          inputName: target.dataset.inputName || "",
          placeholder: target.dataset.placeholder || "Search...",
          sectionLabel: target.dataset.sectionLabel || "Your records",
          required,
          passthrough,
          allowCreate,
          rkey: target.dataset.initialRkey || "",
          label: target.dataset.initialLabel || "",
        },
      }),
    );
  });
}

async function mountFeedFilters() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-feed-filters]",
  );
  if (targets.length === 0) {
    return;
  }

  const FeedFiltersIsland = await loadIsland(
    "feed-filters",
    () => import("./FeedFiltersIsland.svelte") as Promise<IslandModule>,
  );

  targets.forEach((target) => {
    if (feedFiltersIslands.has(target)) {
      return;
    }
    target.innerHTML = "";
    let initialTabs: Array<{ label: string; value: string }> = [];
    try {
      const parsed = JSON.parse(target.dataset.tabs || "[]");
      if (Array.isArray(parsed)) {
        initialTabs = parsed
          .map((tab) => {
            const record = tab as Record<string, unknown>;
            return {
              label: String(record.label ?? record.Label ?? ""),
              value: String(record.value ?? record.Value ?? ""),
            };
          })
          .filter((tab) => tab.label);
      }
    } catch {
      initialTabs = [];
    }
    feedFiltersIslands.set(
      target,
      mount(FeedFiltersIsland, {
        target,
        props: {
          initialType: target.dataset.initialType || "",
          initialSort: target.dataset.initialSort || "recent",
          initialTabs,
        },
      }),
    );
  });
}

async function mountRecipeExplore() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-recipe-explore]",
  );
  if (targets.length === 0) {
    return;
  }
  const RecipeExploreIsland = await loadIsland(
    "recipe-explore",
    () => import("./RecipeExploreIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (recipeExploreIslands.has(target)) {
      return;
    }
    target.innerHTML = "";
    recipeExploreIslands.set(
      target,
      mount(RecipeExploreIsland, {
        target,
        props: { target },
      }),
    );
  });
}

async function mountManageTabs() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-manage-tabs]",
  );
  if (targets.length === 0) {
    return;
  }
  const ManageTabsIsland = await loadIsland(
    "manage-tabs",
    () => import("./ManageTabsIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (manageTabsIslands.has(target)) {
      return;
    }
    manageTabsIslands.set(
      target,
      mount(ManageTabsIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountCommentSections() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-comment-section]",
  );
  if (targets.length === 0) {
    return;
  }
  const CommentSectionIsland = await loadIsland(
    "comment-section",
    () => import("./CommentSectionIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (commentSectionIslands.has(target)) {
      return;
    }
    commentSectionIslands.set(
      target,
      mount(CommentSectionIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountEntitySuggests() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-entity-suggest]",
  );
  if (targets.length === 0) {
    return;
  }
  const EntitySuggestIsland = await loadIsland(
    "entity-suggest",
    () => import("./EntitySuggestIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (entitySuggestIslands.has(target)) {
      return;
    }
    target.innerHTML = "";
    entitySuggestIslands.set(
      target,
      mount(EntitySuggestIsland, {
        target,
        props: {
          target,
          endpoint: target.dataset.endpoint || "",
          entityType: target.dataset.entityType || "",
          placeholder: target.dataset.placeholder || "Name *",
        },
      }),
    );
  });
}

async function mountModalShells() {
  const targets = document.querySelectorAll<HTMLFormElement>(
    "form[data-svelte-modal-shell]",
  );
  if (targets.length === 0) {
    return;
  }
  const ModalShellIsland = await loadIsland(
    "modal-shell",
    () => import("./ModalShellIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (modalShellIslands.has(target)) {
      return;
    }
    modalShellIslands.set(
      target,
      mount(ModalShellIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountModalContainers() {
  const targets = document.querySelectorAll<HTMLElement>("#modal-container");
  if (targets.length === 0) {
    return;
  }
  const ModalContainerIsland = await loadIsland(
    "modal-container",
    () => import("./ModalContainerIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (modalContainerIslands.has(target)) {
      return;
    }
    modalContainerIslands.set(
      target,
      mount(ModalContainerIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountOolongForms() {
  const targets = document.querySelectorAll<HTMLFormElement>(
    "form[data-svelte-oolong-form]",
  );
  if (targets.length === 0) {
    return;
  }
  const OolongFormIsland = await loadIsland(
    "oolong-form",
    () => import("./OolongFormIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (oolongFormIslands.has(target)) {
      return;
    }
    oolongFormIslands.set(
      target,
      mount(OolongFormIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountBeanRoasterPickers() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-bean-roaster-picker]",
  );
  if (targets.length === 0) {
    return;
  }
  const BeanRoasterPickerIsland = await loadIsland(
    "bean-roaster-picker",
    () => import("./BeanRoasterPickerIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (beanRoasterPickerIslands.has(target)) {
      return;
    }
    target.innerHTML = "";
    beanRoasterPickerIslands.set(
      target,
      mount(BeanRoasterPickerIsland, {
        target,
        props: { target },
      }),
    );
  });
}

async function mountBeanRatings() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-bean-rating], [data-svelte-rating-toggle]",
  );
  if (targets.length === 0) {
    return;
  }
  const BeanRatingIsland = await loadIsland(
    "bean-rating",
    () => import("./BeanRatingIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (beanRatingIslands.has(target)) {
      return;
    }
    const initialRating = Number(target.dataset.initialRating || "0");
    target.innerHTML = "";
    beanRatingIslands.set(
      target,
      mount(BeanRatingIsland, {
        target,
        props: {
          initialRating: Number.isNaN(initialRating) ? 0 : initialRating,
        },
      }),
    );
  });
}

async function mountBeanViewActions() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-bean-view-actions]",
  );
  if (targets.length === 0) {
    return;
  }
  const BeanViewActionsIsland = await loadIsland(
    "bean-view-actions",
    () => import("./BeanViewActionsIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (beanViewActionsIslands.has(target)) {
      return;
    }
    let baseBean: Record<string, unknown> = {};
    try {
      baseBean = JSON.parse(target.dataset.baseBean || "{}");
    } catch {
      baseBean = {};
    }
    const initialRating = Number(target.dataset.initialRating || "5");
    target.innerHTML = "";
    beanViewActionsIslands.set(
      target,
      mount(BeanViewActionsIsland, {
        target,
        props: {
          beanRKey: target.dataset.beanRkey || "",
          baseBean,
          initialRating: Number.isNaN(initialRating) ? 5 : initialRating,
          hasRating: target.dataset.hasRating === "true",
          closed: target.dataset.closed === "true",
        },
      }),
    );
  });
}

async function mountDisclosures() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-disclosure]",
  );
  if (targets.length === 0) {
    return;
  }
  const DisclosureIsland = await loadIsland(
    "disclosure",
    () => import("./DisclosureIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (disclosureIslands.has(target)) {
      return;
    }
    disclosureIslands.set(
      target,
      mount(DisclosureIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountShareButtons() {
  const targets = document.querySelectorAll<HTMLButtonElement>(
    "button[data-svelte-share]",
  );
  if (targets.length === 0) {
    return;
  }
  const ShareButtonIsland = await loadIsland(
    "share-button",
    () => import("./ShareButtonIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (shareButtonIslands.has(target)) {
      return;
    }
    shareButtonIslands.set(
      target,
      mount(ShareButtonIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountActionMoreMenus() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-action-more-menu]",
  );
  if (targets.length === 0) {
    return;
  }
  const ActionMoreMenuIsland = await loadIsland(
    "action-more-menu",
    () => import("./ActionMoreMenuIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (actionMoreMenuIslands.has(target)) {
      return;
    }
    actionMoreMenuIslands.set(
      target,
      mount(ActionMoreMenuIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountCommentReplies() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-comment-reply]",
  );
  if (targets.length === 0) {
    return;
  }
  const CommentReplyIsland = await loadIsland(
    "comment-reply",
    () => import("./CommentReplyIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (commentReplyIslands.has(target)) {
      return;
    }
    commentReplyIslands.set(
      target,
      mount(CommentReplyIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountSettingsControls() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-settings-controls]",
  );
  if (targets.length === 0) {
    return;
  }
  const SettingsControlsIsland = await loadIsland(
    "settings-controls",
    () => import("./SettingsControlsIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (settingsControlsIslands.has(target)) {
      return;
    }
    settingsControlsIslands.set(
      target,
      mount(SettingsControlsIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountSettingsForms() {
  const targets = document.querySelectorAll<HTMLFormElement>(
    "[data-svelte-settings-form]",
  );
  if (targets.length === 0) {
    return;
  }
  const SettingsFormIsland = await loadIsland(
    "settings-form",
    () => import("./SettingsFormIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (settingsFormIslands.has(target)) {
      return;
    }
    settingsFormIslands.set(
      target,
      mount(SettingsFormIsland, {
        target,
        props: { target },
      }),
    );
  });
}

async function mountScrollTopControls() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-scroll-top]",
  );
  if (targets.length === 0) {
    return;
  }
  const ScrollTopIsland = await loadIsland(
    "scroll-top",
    () => import("./ScrollTopIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (scrollTopIslands.has(target)) {
      return;
    }
    scrollTopIslands.set(
      target,
      mount(ScrollTopIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountAdminDashboards() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-admin-dashboard]",
  );
  if (targets.length === 0) {
    return;
  }
  const AdminDashboardIsland = await loadIsland(
    "admin-dashboard",
    () => import("./AdminDashboardIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (adminDashboardIslands.has(target)) {
      return;
    }
    adminDashboardIslands.set(
      target,
      mount(AdminDashboardIsland, {
        target: document.body,
        props: { target },
      }),
    );
  });
}

async function mountSaveAsRecipeControls() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-save-as-recipe]",
  );
  if (targets.length === 0) {
    return;
  }
  const SaveAsRecipeIsland = await loadIsland(
    "save-as-recipe",
    () => import("./SaveAsRecipeIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (saveAsRecipeIslands.has(target)) {
      return;
    }
    target.innerHTML = "";
    saveAsRecipeIslands.set(
      target,
      mount(SaveAsRecipeIsland, {
        target,
        props: { brewRKey: target.dataset.brewRkey || "" },
      }),
    );
  });
}

async function mountRecipeForkButtons() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-recipe-fork]",
  );
  if (targets.length === 0) {
    return;
  }
  const RecipeForkButtonIsland = await loadIsland(
    "recipe-fork-button",
    () => import("./RecipeForkButtonIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (recipeForkButtonIslands.has(target)) {
      return;
    }
    target.innerHTML = "";
    recipeForkButtonIslands.set(
      target,
      mount(RecipeForkButtonIsland, {
        target,
        props: {
          recipeRKey: target.dataset.recipeRkey || "",
          ownerDID: target.dataset.ownerDid || "",
        },
      }),
    );
  });
}

async function mountHandleAutocompletes() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-handle-autocomplete]",
  );
  if (targets.length === 0) {
    return;
  }
  const HandleAutocompleteIsland = await loadIsland(
    "handle-autocomplete",
    () => import("./HandleAutocompleteIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (handleAutocompleteIslands.has(target)) {
      return;
    }
    const container =
      target.closest<HTMLElement>("[data-handle-autocomplete-root]") ||
      target.parentElement;
    const input = container?.querySelector<HTMLInputElement>(
      'input[name="handle"]',
    );
    if (!input) {
      return;
    }
    target.innerHTML = "";
    target.classList.remove("hidden");
    handleAutocompleteIslands.set(
      target,
      mount(HandleAutocompleteIsland, {
        target,
        props: { input, target },
      }),
    );
  });
}

async function mountProfileStats() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-profile-stats]",
  );
  if (targets.length === 0) {
    return;
  }
  const ProfileStatsIsland = await loadIsland(
    "profile-stats",
    () => import("./ProfileStatsIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (profileStatsIslands.has(target)) {
      return;
    }
    target.innerHTML = "";
    profileStatsIslands.set(
      target,
      mount(ProfileStatsIsland, {
        target,
        props: { target },
      }),
    );
  });
}

async function mountTasteProfiles() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-taste-profile]",
  );
  if (targets.length === 0) {
    return;
  }
  const TasteProfileIsland = await loadIsland(
    "taste-profile",
    () => import("./TasteProfileIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (tasteProfileIslands.has(target)) {
      return;
    }
    target.innerHTML = "";
    tasteProfileIslands.set(
      target,
      mount(TasteProfileIsland, {
        target,
        props: { target },
      }),
    );
  });
}

async function mountBrewForms() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-brew-form]",
  );
  if (targets.length === 0) {
    return;
  }
  const BrewFormIsland = await loadIsland(
    "brew-form",
    () => import("./BrewFormIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (brewFormIslands.has(target)) {
      return;
    }
    target.innerHTML = "";
    brewFormIslands.set(
      target,
      mount(BrewFormIsland, {
        target,
        props: { target },
      }),
    );
  });
}

async function mountRecipeForms() {
  const targets = document.querySelectorAll<HTMLElement>(
    "[data-svelte-recipe-form]",
  );
  if (targets.length === 0) {
    return;
  }
  const RecipeFormIsland = await loadIsland(
    "recipe-form",
    () => import("./RecipeFormIsland.svelte") as Promise<IslandModule>,
  );
  targets.forEach((target) => {
    if (recipeFormIslands.has(target)) {
      return;
    }
    target.innerHTML = "";
    recipeFormIslands.set(
      target,
      mount(RecipeFormIsland, {
        target,
        props: { target },
      }),
    );
  });
}

async function mountFeedLayout() {
  if (feedLayoutMounted) {
    return;
  }
  const hasFeedLayout =
    document.querySelector(
      "[data-feed-masonry], #feed-items, [data-svelte-feed-filters]",
    ) !== null;
  if (!hasFeedLayout) {
    return;
  }

  const FeedMasonryIsland = await loadIsland(
    "feed-masonry",
    () => import("./FeedMasonryIsland.svelte") as Promise<IslandModule>,
  );
  mount(FeedMasonryIsland, { target: document.body });
  feedLayoutMounted = true;
}

async function mountCoreIslands() {
  if (mounted) {
    return;
  }

  const [
    AppCacheRuntimeIsland,
    GlobalActionsIsland,
    LayoutRuntimeIsland,
    ThemeRuntimeIsland,
    TransitionRuntimeIsland,
    ServiceWorkerIsland,
  ] = await Promise.all([
    loadIsland(
      "app-cache-runtime",
      () => import("./AppCacheRuntimeIsland.svelte") as Promise<IslandModule>,
    ),
    loadIsland(
      "global-actions",
      () => import("./GlobalActionsIsland.svelte") as Promise<IslandModule>,
    ),
    loadIsland(
      "layout-runtime",
      () => import("./LayoutRuntimeIsland.svelte") as Promise<IslandModule>,
    ),
    loadIsland(
      "theme-runtime",
      () => import("./ThemeRuntimeIsland.svelte") as Promise<IslandModule>,
    ),
    loadIsland(
      "transition-runtime",
      () => import("./TransitionRuntimeIsland.svelte") as Promise<IslandModule>,
    ),
    loadIsland(
      "service-worker",
      () => import("./ServiceWorkerIsland.svelte") as Promise<IslandModule>,
    ),
  ]);

  mount(AppCacheRuntimeIsland, { target: document.body });
  mount(GlobalActionsIsland, { target: document.body });
  mount(LayoutRuntimeIsland, { target: document.body });
  mount(ThemeRuntimeIsland, { target: document.body });
  mount(TransitionRuntimeIsland, { target: document.body });
  mount(ServiceWorkerIsland, { target: document.body });

  mounted = true;
}

async function mountAll() {
  await mountCoreIslands();
  await Promise.all([
    mountComboSelects(),
    mountFeedFilters(),
    mountRecipeExplore(),
    mountManageTabs(),
    mountCommentSections(),
    mountEntitySuggests(),
    mountModalContainers(),
    mountModalShells(),
    mountOolongForms(),
    mountBeanRoasterPickers(),
    mountBeanRatings(),
    mountBeanViewActions(),
    mountDisclosures(),
    mountShareButtons(),
    mountActionMoreMenus(),
    mountCommentReplies(),
    mountSettingsControls(),
    mountSettingsForms(),
    mountScrollTopControls(),
    mountAdminDashboards(),
    mountSaveAsRecipeControls(),
    mountRecipeForkButtons(),
    mountHandleAutocompletes(),
    mountProfileStats(),
    // TODO: enable this once its ready
    // mountTasteProfiles(),
    mountBrewForms(),
    mountRecipeForms(),
    mountFeedLayout(),
  ]);

  window.__arabicaApplyFeedMasonry?.();
}

window.__arabicaSvelteIslands = {
  mountAll: () => {
    void mountAll();
  },
  applyFeedMasonry: () => window.__arabicaApplyFeedMasonry?.(),
};

if (document.readyState === "loading") {
  document.addEventListener(
    "DOMContentLoaded",
    () => {
      void mountAll();
    },
    { once: true },
  );
} else {
  void mountAll();
}

document.addEventListener("htmx:afterSwap", () => {
  void mountAll();
});

document.addEventListener("htmx:load", () => {
  void mountAll();
});
