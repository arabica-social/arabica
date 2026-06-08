<script lang="ts">
  import { onMount } from 'svelte';

  type EntityRecord = Record<string, any>;
  type Pour = {
    water?: number | string;
    time?: number | string;
    water_amount?: number;
    time_seconds?: number;
  };
  type CacheData = {
    recipes?: EntityRecord[];
    brewers?: EntityRecord[];
  };

  let { target }: { target: HTMLElement } = $props();

  let pours: Pour[] = [];
  let brewerCategory = '';
  let activeRecipe: EntityRecord | null = null;
  let recipes: EntityRecord[] = [];
  let brewers: EntityRecord[] = [];
  let recipeOwnerDID = '';

  function applyCacheData(data: CacheData | null | undefined) {
    recipes = data?.recipes || [];
    brewers = data?.brewers || [];
  }

  async function loadCache() {
    const cache = window.AppCache;
    if (!cache) return;

    applyCacheData(cache.getCachedData?.());
    try {
      const data = await cache.getData?.();
      applyCacheData(data);
    } catch (error) {
      console.warn('brew form: failed to load entity cache:', error);
    }
  }

  function combo(type: string) {
    return target.querySelector(`[data-combo-entity-type="${type}"]`);
  }

  function normalizeBrewerCategory(raw: string) {
    if (!raw) return '';
    const lower = raw.toLowerCase().trim();
    if (['pourover', 'espresso', 'immersion', 'mokapot', 'coldbrew', 'cupping', 'other'].includes(lower)) {
      return lower;
    }
    if (['pour-over', 'pour over', 'dripper'].includes(lower)) return 'pourover';
    if (['espresso machine', 'lever espresso', 'lever espresso machine'].includes(lower)) return 'espresso';
    if (['french press', 'aeropress', 'siphon', 'clever', 'clever dripper'].includes(lower)) return 'immersion';
    return '';
  }

  function getRKey(record: EntityRecord) {
    return record.rkey || record.RKey || '';
  }

  function getName(record: EntityRecord) {
    return record.name || record.Name || '';
  }

  function getBrewerType(rkey: string) {
    if (!rkey) return '';
    const brewer = brewers.find((candidate) => getRKey(candidate) === rkey);
    return brewer?.brewer_type || brewer?.BrewerType || '';
  }

  function setBrewerCategory(category: string) {
    brewerCategory = category || '';
    target.dispatchEvent(
      new CustomEvent('brew-method-category-change', {
        detail: { category: brewerCategory },
        bubbles: true
      })
    );
  }

  function dispatchPoursShow() {
    const poursTarget = target.querySelector('[data-svelte-brew-pours]');
    poursTarget?.dispatchEvent(new CustomEvent('brew-pours:show', { bubbles: false }));
  }

  function updatePoursVisibility() {
    if (pours.length > 0 || (activeRecipe?.pours || []).length > 0) {
      dispatchPoursShow();
    }
  }

  function onBrewerChange(rkey: string) {
    setBrewerCategory(normalizeBrewerCategory(getBrewerType(rkey)));
    if (brewerCategory === 'pourover') dispatchPoursShow();
  }

  function recipeSummaryText() {
    if (!activeRecipe) return '';
    const parts: string[] = [];
    if (activeRecipe.coffee_amount > 0) parts.push(`${Math.round(activeRecipe.coffee_amount)}g coffee`);
    if (activeRecipe.water_amount > 0) parts.push(`${Math.round(activeRecipe.water_amount)}g water`);
    if (activeRecipe.brewer_rkey) {
      const brewer = brewers.find((candidate) => getRKey(candidate) === activeRecipe?.brewer_rkey);
      if (brewer) parts.push(getName(brewer));
    }
    if ((activeRecipe.pours || []).length > 0) parts.push(`${activeRecipe.pours.length} pours`);
    return parts.join(' · ');
  }

  function dispatchRecipeState() {
    target.dispatchEvent(
      new CustomEvent('brew-recipe-state-change', {
        detail: {
          active: !!activeRecipe,
          summary: recipeSummaryText()
        },
        bubbles: true
      })
    );
  }

  function setFormField(form: Element, name: string, value: string | number) {
    form.querySelectorAll<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>(`[name="${name}"]`).forEach((field) => {
      field.value = String(value);
      field.dispatchEvent(new Event('input', { bubbles: true }));
    });
  }

  function dispatchPoursChange() {
    const poursTarget = target.querySelector('[data-svelte-brew-pours]');
    if (!poursTarget) return;

    try {
      poursTarget.setAttribute('data-current-pours', JSON.stringify(pours));
    } catch {
      poursTarget.setAttribute('data-current-pours', '[]');
    }
    poursTarget.dispatchEvent(
      new CustomEvent('brew-pours:set', {
        detail: { pours },
        bubbles: false
      })
    );
  }

  function clearRecipeFields(form: Element) {
    setFormField(form, 'coffee_amount', '');
    setFormField(form, 'water_amount', '');
    combo('brewer')?.dispatchEvent(
      new CustomEvent('combo-set', {
        detail: { rkey: '', label: '' },
        bubbles: false
      })
    );
    pours = [];
    dispatchPoursChange();
  }

  async function applyRecipe(rkey: string) {
    const form = target.querySelector('form');
    if (!form) return;

    if (!rkey) {
      clearRecipeFields(form);
      activeRecipe = null;
      recipeOwnerDID = '';
      setFormField(form, 'recipe_owner_did', '');
      dispatchRecipeState();
      updatePoursVisibility();
      return;
    }

    const cachedRecipe = recipes.find((recipe) => getRKey(recipe) === rkey);
    if (cachedRecipe?.author_did) recipeOwnerDID = cachedRecipe.author_did;

    try {
      const ownerQuery = recipeOwnerDID ? `?owner=${encodeURIComponent(recipeOwnerDID)}` : '';
      const response = await fetch(`/api/recipes/${rkey}${ownerQuery}`, { credentials: 'same-origin' });
      if (!response.ok) return;

      const recipe = await response.json();
      activeRecipe = recipe;
      if (recipe.author_did) recipeOwnerDID = recipe.author_did;
      setFormField(form, 'recipe_owner_did', recipeOwnerDID);
      setFormField(form, 'coffee_amount', recipe.coffee_amount > 0 ? Math.round(recipe.coffee_amount) : '');
      setFormField(form, 'water_amount', recipe.water_amount > 0 ? Math.round(recipe.water_amount) : '');

      const localBrewer = recipe.brewer_rkey
        ? brewers.find((candidate) => getRKey(candidate) === recipe.brewer_rkey)
        : null;
      combo('brewer')?.dispatchEvent(
        new CustomEvent('combo-set', {
          detail: {
            rkey: localBrewer ? recipe.brewer_rkey : '',
            label: localBrewer ? getName(localBrewer) : ''
          },
          bubbles: false
        })
      );

      if (recipe.brewer_rkey) onBrewerChange(recipe.brewer_rkey);
      if (!brewerCategory) {
        const recipeBrewerType = recipe.brewer_type || recipe.brewer_obj?.brewer_type || '';
        if (recipeBrewerType) setBrewerCategory(normalizeBrewerCategory(recipeBrewerType));
      }

      pours = (recipe.pours || []).map((pour: Pour) => ({
        water: pour.water_amount || '',
        time: pour.time_seconds || ''
      }));
      dispatchPoursChange();
      dispatchRecipeState();
      updatePoursVisibility();
    } catch (error) {
      console.error('Failed to apply recipe:', error);
    }
  }

  function handleComboChange(event: Event) {
    const detail = (event as CustomEvent<Record<string, any>>).detail || {};
    if (detail.entityType === 'brewer') {
      const brewerType = detail.entity?.brewer_type || detail.entity?.BrewerType || '';
      setBrewerCategory(normalizeBrewerCategory(brewerType));
      if (brewerCategory === 'pourover') dispatchPoursShow();
    }

    if (detail.entityType === 'recipe') {
      if (detail.suggestion) {
        const parts = (detail.suggestion.source_uri || '').split('/');
        recipeOwnerDID = parts.length >= 3 ? parts[2] : '';
      } else {
        recipeOwnerDID = '';
      }
      void applyRecipe(detail.rkey || '');
    }
  }

  onMount(() => {
    const form = target.querySelector('form');
    const recipeRKey = form?.getAttribute('data-recipe-rkey') || '';
    recipeOwnerDID = form?.getAttribute('data-recipe-owner') || '';

    const poursData = form?.getAttribute('data-pours');
    if (poursData) {
      try {
        pours = JSON.parse(poursData);
      } catch (error) {
        console.error('Failed to parse pours data:', error);
        pours = [];
      }
    }

    const cacheListener = (data: CacheData) => applyCacheData(data);
    const refreshDropdowns = async () => {
      try {
        const data = await window.AppCache?.invalidateAndRefresh?.();
        applyCacheData(data);
      } catch (error) {
        console.warn('brew form: failed to refresh entity cache:', error);
      }
    };

    target.addEventListener('combo-change', handleComboChange);
    document.body.addEventListener('refresh-dropdowns', refreshDropdowns);
    window.AppCache?.addListener?.(cacheListener);

    void loadCache().then(async () => {
      if (recipeRKey) {
        const match = recipes.find((recipe) => getRKey(recipe) === recipeRKey);
        combo('recipe')?.dispatchEvent(
          new CustomEvent('combo-set', {
            detail: { rkey: recipeRKey, label: match ? getName(match) : '' },
            bubbles: false
          })
        );
        await applyRecipe(recipeRKey);
      }
      updatePoursVisibility();
    });

    return () => {
      target.removeEventListener('combo-change', handleComboChange);
      document.body.removeEventListener('refresh-dropdowns', refreshDropdowns);
      window.AppCache?.removeListener?.(cacheListener);
    };
  });
</script>
