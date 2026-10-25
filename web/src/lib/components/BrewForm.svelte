<script lang="ts">
  import { onMount, tick } from "svelte";
  import {
    initialBrewFormMode,
    normalizeBrewerCategory,
    preparationLayout,
    type BrewFormMode,
  } from "../utils/brewFormPolicy";
  import EntityCombo from "./EntityCombo.svelte";
  import Field from "./BrewFormField.svelte";
  import PoursEditor from "./PoursEditor.svelte";
  import FormWorkspace from "./FormWorkspace.svelte";
  import LedgerHeader from "./LedgerHeader.svelte";
  import FormSection from "./FormSection.svelte";
  import { appCache } from "../stores/appCache";
  import {
    loadUserPrefs,
    setBrewFormMode,
    userPrefs,
  } from "../stores/userPrefs";
  import { session, warnIfSessionExpired } from "../stores/session";
  import { pushToast } from "../stores/toasts";
  import { beforeNavigate, goto } from "$app/navigation";
  import { createBrew, updateBrew, type BrewInput } from "../api/entities";
  import {
    APIError,
    notifySessionExpired,
    notifySessionExpiredForResponse,
  } from "../api/client";
  import {
    computeAutofill,
    type AutofillField,
    type AutofillSuggestion,
  } from "../utils/brewAutofill";
  import type { Brew, Brewer, Recipe } from "../types/entity_view";

  type Pour = { water: string; time: string };

  type Props = {
    brew: Brew | null;
    recipeRKey?: string;
    recipeOwnerDID?: string;
    isEdit: boolean;
  };

  let { brew, recipeRKey = "", recipeOwnerDID = "", isEdit }: Props = $props();

  let cachedData = $state<Record<string, unknown>>({});
  let activeRecipe = $state<Recipe | null>(null);
  let recipeExpanded = $state(false);
  let recipeRatio = $state("");
  let recipePours = $state<Pour[]>([]);
  let brewerCategory = $state("");

  // svelte-ignore state_referenced_locally
  let recipeRKeyValue = $state(recipeRKey);
  let recipeLabel = $state("");
  // svelte-ignore state_referenced_locally
  let recipeOwner = $state(recipeOwnerDID);
  let beanRKey = $state("");
  let beanLabel = $state("");
  let grinderRKey = $state("");
  let grinderLabel = $state("");
  let brewerRKey = $state("");
  let brewerLabel = $state("");
  let coffeeAmount = $state("");
  let waterAmount = $state("");
  let grindSize = $state("");
  let temperature = $state("");
  let timeSeconds = $state("");
  let timeUnit = $state("1");
  let tastingNotes = $state("");
  let rating = $state("");
  let method = $state("");
  let pours = $state<Pour[]>([]);
  let espressoYieldWeight = $state("");
  let espressoPressure = $state("");
  let espressoPreInfusionSeconds = $state("");
  let pouroverBloomWater = $state("");
  let pouroverBloomSeconds = $state("");
  let pouroverDrawdownSeconds = $state("");
  let pouroverBypassWater = $state("");
  let pouroverFilter = $state("");
  let submitting = $state(false);
  let historyBrews = $state<Brew[]>([]);
  let autofilled = $state<Partial<Record<AutofillField, AutofillSuggestion>>>(
    {},
  );

  // Mode is presentation only: both modes serialize this same draft.
  // svelte-ignore state_referenced_locally
  let mode = $state<BrewFormMode>(
    initialBrewFormMode(isEdit, brew, $userPrefs.brew_form_mode),
  );
  let modeChosen = false;
  let moreExpanded = $state(false);
  let dataState = $state<"loading" | "ready" | "error" | "auth-expired">(
    "loading",
  );
  let dataLoad: Promise<void> | undefined;
  let formElement = $state<HTMLFormElement>();
  let validationAttempted = $state(false);
  let brewerChoiceVersion = 0;
  let recipeLoading = $state(false);
  let recipeError = $state("");
  let recipeNotice = $state("");
  let formError = $state("");
  let dirty = $state(false);
  let saved = false;
  let recipeGen = 0;
  let acceptedGrinder = { rkey: "", label: "" };
  let acceptedBrewer = { rkey: "", label: "" };
  // svelte-ignore state_referenced_locally
  let acceptedRecipe = { rkey: recipeRKey, label: "", owner: recipeOwnerDID };
  let grinderRevision = $state(0);
  let brewerRevision = $state(0);

  function focusCombo(name: string) {
    void tick().then(() =>
      document
        .querySelector<HTMLInputElement>(`input[name="${name}"]`)
        ?.closest(".combo-select")
        ?.querySelector<HTMLInputElement>('[role="combobox"]')
        ?.focus(),
    );
  }
  // EntityCombo exposes no validation props yet. Apply ARIA to its actual
  // editable control, not the hidden reference input or the wrapper.
  $effect(() => {
    const form = formElement;
    const invalidBean = validationAttempted && !beanRKey;
    const invalidBrewer = validationAttempted && !isEdit && !brewerRKey;
    brewerRevision;
    void tick().then(() => {
      for (const [name, required, invalid] of [
        ["bean_rkey", true, invalidBean],
        ["brewer_rkey", !isEdit, invalidBrewer],
      ] as const) {
        const control = form
          ?.querySelector(`input[name="${name}"]`)
          ?.closest(".combo-select")
          ?.querySelector('[role="combobox"]');
        control?.setAttribute("aria-required", String(required));
        control?.setAttribute("aria-invalid", String(invalid));
        if (invalid) control?.setAttribute("aria-describedby", `${name}-error`);
        else control?.removeAttribute("aria-describedby");
      }
    });
  });

  let layout = $derived(preparationLayout(brewerCategory));
  let hasEspresso = $derived(
    !!(espressoYieldWeight || espressoPressure || espressoPreInfusionSeconds),
  );
  let hasPourover = $derived(
    !!(
      pouroverBloomWater ||
      pouroverBloomSeconds ||
      pouroverDrawdownSeconds ||
      pouroverBypassWater ||
      pouroverFilter
    ),
  );
  let hasDetails = $derived(
    !!(
      coffeeAmount ||
      waterAmount ||
      temperature ||
      timeSeconds ||
      grindSize ||
      grinderRKey ||
      method ||
      hasEspresso ||
      hasPourover ||
      pours.length
    ),
  );
  let showRecipe = $derived(
    !!recipeRKeyValue ||
      (dataState === "ready" &&
        ((cachedData.recipes as Recipe[] | undefined)?.length ?? 0) > 0),
  );
  let missingBean = $derived(
    !isEdit &&
      dataState === "ready" &&
      Array.isArray(cachedData.beans) &&
      cachedData.beans.length === 0 &&
      !beanRKey,
  );
  let missingBrewer = $derived(
    !isEdit &&
      dataState === "ready" &&
      Array.isArray(cachedData.brewers) &&
      cachedData.brewers.length === 0 &&
      !brewerRKey,
  );

  function chooseMode(next: BrewFormMode) {
    mode = next;
    modeChosen = true;
    if (!isEdit)
      void setBrewFormMode(next).catch(() => {
        /* Preference failure must not block the draft. */
      });
    if (next === "detailed") applyAutofill();
  }

  async function openDetails() {
    // Opening a summary is not an explicit remembered-mode choice.
    mode = "detailed";
    modeChosen = true;
    applyAutofill();
    await tick();
    formElement
      ?.querySelector<HTMLInputElement>(
        recipeExpanded && activeRecipe
          ? "#recipe-adjustment-coffee"
          : ".preparation-fields input",
      )
      ?.focus();
  }

  beforeNavigate((navigation) => {
    // Full-document exits use the browser's beforeunload prompt below.
    if (navigation.willUnload) return;
    if (
      dirty &&
      !saved &&
      !window.confirm("Leave this brew? Your unsaved changes will be lost.")
    )
      navigation.cancel();
  });

  function smartAutofillOn(): boolean {
    return (
      !isEdit &&
      mode === "detailed" &&
      !recipeLoading &&
      !$userPrefs.loading &&
      $userPrefs.smartAutofillEnabled
    );
  }

  function transitionMethod(nextCategory: string): boolean {
    const next = normalizeBrewerCategory(nextCategory);
    if (next === normalizeBrewerCategory(brewerCategory)) return true;
    const dropEspresso = next !== "espresso" && hasEspresso;
    const dropPourover =
      next !== "pourover" && (hasPourover || pours.length > 0);
    // Known legacy method labels describe the old category. Keep unknown
    // free text, but do not carry an Espresso label into a pour-over draft.
    const methodCategory = normalizeBrewerCategory(method);
    const dropMethod =
      !!method && methodCategory !== "other" && methodCategory !== next;
    if (
      (dropEspresso || dropPourover || (dropMethod && !autofilled.method)) &&
      !window.confirm(
        "Changing brewer will clear incompatible brewing details. Continue?",
      )
    )
      return false;
    if (dropEspresso) {
      for (const field of [
        "espresso_yield_weight",
        "espresso_pressure",
        "espresso_pre_infusion_seconds",
      ] as AutofillField[]) {
        clearFieldValue(field);
        clearAutofillField(field);
      }
    }
    if (dropPourover) {
      for (const field of [
        "pourover_bloom_water",
        "pourover_bloom_seconds",
        "pourover_drawdown_seconds",
        "pourover_bypass_water",
        "pourover_filter",
      ] as AutofillField[]) {
        clearFieldValue(field);
        clearAutofillField(field);
      }
      pours = [];
    }
    if (dropMethod) {
      method = "";
      clearAutofillField("method");
    }
    brewerCategory = next;
    return true;
  }

  function transitionGrinder(rkey: string, label: string): boolean {
    if (rkey !== acceptedGrinder.rkey && grindSize) {
      if (
        !autofilled.grind_size &&
        !window.confirm(
          acceptedGrinder.rkey
            ? "Changing grinder will clear its grind setting. Continue?"
            : "This saved grind setting has no grinder. Clear it before associating this grinder? Cancel to keep the original setting.",
        )
      ) {
        grinderRKey = acceptedGrinder.rkey;
        grinderLabel = acceptedGrinder.label;
        ++grinderRevision;
        focusCombo("grinder_rkey");
        return false;
      }
      grindSize = "";
      clearAutofillField("grind_size");
    }
    grinderRKey = rkey;
    grinderLabel = label;
    acceptedGrinder = { rkey, label };
    clearAutofillField("grinder_rkey");
    return true;
  }

  function mustBePositive(value: string): boolean {
    if (value === null || value === "") return false;
    const parsed = Number(value);
    return !Number.isFinite(parsed) || parsed <= 0;
  }

  let coffeeAmountError = $derived(mustBePositive(coffeeAmount));
  let waterAmountError = $derived(mustBePositive(waterAmount));
  let temperatureError = $derived(mustBePositive(temperature));
  let timeSecondsError = $derived(mustBePositive(timeSeconds));

  function autofillBadge(field: AutofillField): string {
    const s = autofilled[field];
    if (!s) return "";
    const pct = s.total > 0 ? Math.round((s.matches / s.total) * 100) : 0;
    return `Autofilled · in ${s.matches} of ${s.total} brews (${pct}%)`;
  }

  function autofillInputClass(field: AutofillField): string {
    return `w-full form-input-lg${autofilled[field] ? " autofilled-input" : ""}`;
  }

  function clearFieldValue(field: AutofillField) {
    switch (field) {
      case "grinder_rkey":
        grinderRKey = "";
        grinderLabel = "";
        acceptedGrinder = { rkey: "", label: "" };
        break;
      case "brewer_rkey":
        brewerRKey = "";
        brewerLabel = "";
        acceptedBrewer = { rkey: "", label: "" };
        if (!activeRecipe) brewerCategory = "";
        break;
      case "method":
        method = "";
        break;
      case "grind_size":
        grindSize = "";
        break;
      case "temperature":
        temperature = "";
        break;
      case "time_seconds":
        timeSeconds = "";
        break;
      case "water_amount":
        waterAmount = "";
        break;
      case "coffee_amount":
        coffeeAmount = "";
        break;
      case "pourover_filter":
        pouroverFilter = "";
        break;
      case "pourover_bloom_water":
        pouroverBloomWater = "";
        break;
      case "pourover_bloom_seconds":
        pouroverBloomSeconds = "";
        break;
      case "pourover_drawdown_seconds":
        pouroverDrawdownSeconds = "";
        break;
      case "pourover_bypass_water":
        pouroverBypassWater = "";
        break;
      case "espresso_yield_weight":
        espressoYieldWeight = "";
        break;
      case "espresso_pressure":
        espressoPressure = "";
        break;
      case "espresso_pre_infusion_seconds":
        espressoPreInfusionSeconds = "";
        break;
    }
  }

  // Remove one field from autofill tracking because the user has edited it.
  // The value itself is left alone — the user's edit now owns it.
  function clearAutofillField(field: AutofillField) {
    if (!(field in autofilled)) return;
    const next = { ...autofilled };
    delete next[field];
    autofilled = next;
  }

  // Clear every still-tracked (not user-edited) autofilled field and drop the
  // badges. Used by the form-level Clear autofill action and when the
  // selection context changes before re-applying new suggestions.
  function clearAutofill(preserveBrewer = false) {
    for (const field of Object.keys(autofilled) as AutofillField[]) {
      if (preserveBrewer && field === "brewer_rkey") continue;
      // A setting edited after autofill now depends on this grinder. Keep
      // that association rather than creating an unassociated user value.
      if (field === "grinder_rkey" && grindSize && !autofilled.grind_size)
        continue;
      if (
        field === "brewer_rkey" &&
        ((hasEspresso &&
          [
            "espresso_yield_weight",
            "espresso_pressure",
            "espresso_pre_infusion_seconds",
          ].some(
            (f) =>
              measurements[f as Measurement].value &&
              !autofilled[f as AutofillField],
          )) ||
          (hasPourover &&
            [
              "pourover_bloom_water",
              "pourover_bloom_seconds",
              "pourover_drawdown_seconds",
              "pourover_bypass_water",
              "pourover_filter",
            ].some(
              (f) =>
                measurements[f as Measurement].value &&
                !autofilled[f as AutofillField],
            )) ||
          pours.length > 0)
      )
        continue;
      clearFieldValue(field);
    }
    autofilled = {};
  }

  // Resolve the hydrated brewer type for an autofilled brewer rkey so the
  // Espresso/Pour-over sections become visible. Prefers the cached brewers
  // list; falls back to a brewer object from the user's brew history.
  function brewerTypeForRkey(rkey: string): string {
    if (!rkey) return "";
    const brewers = (cachedData.brewers as Brewer[] | undefined) ?? [];
    const cached = brewers.find((b) => b.rkey === rkey);
    if (cached?.brewer_type) return cached.brewer_type;
    for (const hb of historyBrews) {
      if (hb.brewer_rkey === rkey && hb.brewer_obj?.brewer_type) {
        return hb.brewer_obj.brewer_type;
      }
    }
    return "";
  }

  function applySuggestion(
    field: AutofillField,
    suggestion: AutofillSuggestion,
  ) {
    if (field.startsWith("espresso_") && layout.category !== "espresso") return;
    if (field.startsWith("pourover_") && layout.category !== "pourover") return;
    switch (field) {
      case "grinder_rkey":
        if (grinderRKey) return;
        grinderRKey = suggestion.rkey ?? suggestion.value;
        grinderLabel = suggestion.label ?? "";
        acceptedGrinder = { rkey: grinderRKey, label: grinderLabel };
        autofilled = { ...autofilled, grinder_rkey: suggestion };
        return;
      case "brewer_rkey":
        if (brewerRKey) return;
        if (
          !transitionMethod(
            brewerTypeForRkey(suggestion.rkey ?? suggestion.value),
          )
        )
          return;
        brewerRKey = suggestion.rkey ?? suggestion.value;
        brewerLabel = suggestion.label ?? "";
        brewerCategory = normalizeBrewerCategory(brewerTypeForRkey(brewerRKey));
        acceptedBrewer = { rkey: brewerRKey, label: brewerLabel };
        autofilled = { ...autofilled, brewer_rkey: suggestion };
        return;
      case "method":
        if (method) return;
        method = suggestion.value;
        autofilled = { ...autofilled, method: suggestion };
        return;
      case "grind_size":
        if (grindSize || !grinderRKey) return;
        grindSize = suggestion.value;
        autofilled = { ...autofilled, grind_size: suggestion };
        return;
      case "temperature":
        if (temperature) return;
        temperature = suggestion.value;
        autofilled = { ...autofilled, temperature: suggestion };
        return;
      case "time_seconds":
        if (timeSeconds) return;
        timeSeconds = suggestion.value;
        autofilled = { ...autofilled, time_seconds: suggestion };
        return;
      case "water_amount":
        if (waterAmount) return;
        waterAmount = suggestion.value;
        autofilled = { ...autofilled, water_amount: suggestion };
        return;
      case "coffee_amount":
        if (coffeeAmount) return;
        coffeeAmount = suggestion.value;
        autofilled = { ...autofilled, coffee_amount: suggestion };
        return;
      case "pourover_filter":
        if (pouroverFilter) return;
        pouroverFilter = suggestion.value;
        autofilled = { ...autofilled, pourover_filter: suggestion };
        return;
      case "pourover_bloom_water":
        if (pouroverBloomWater) return;
        pouroverBloomWater = suggestion.value;
        autofilled = { ...autofilled, pourover_bloom_water: suggestion };
        return;
      case "pourover_bloom_seconds":
        if (pouroverBloomSeconds) return;
        pouroverBloomSeconds = suggestion.value;
        autofilled = { ...autofilled, pourover_bloom_seconds: suggestion };
        return;
      case "pourover_drawdown_seconds":
        if (pouroverDrawdownSeconds) return;
        pouroverDrawdownSeconds = suggestion.value;
        autofilled = { ...autofilled, pourover_drawdown_seconds: suggestion };
        return;
      case "pourover_bypass_water":
        if (pouroverBypassWater) return;
        pouroverBypassWater = suggestion.value;
        autofilled = { ...autofilled, pourover_bypass_water: suggestion };
        return;
      case "espresso_yield_weight":
        if (espressoYieldWeight) return;
        espressoYieldWeight = suggestion.value;
        autofilled = { ...autofilled, espresso_yield_weight: suggestion };
        return;
      case "espresso_pressure":
        if (espressoPressure) return;
        espressoPressure = suggestion.value;
        autofilled = { ...autofilled, espresso_pressure: suggestion };
        return;
      case "espresso_pre_infusion_seconds":
        if (espressoPreInfusionSeconds) return;
        espressoPreInfusionSeconds = suggestion.value;
        autofilled = {
          ...autofilled,
          espresso_pre_infusion_seconds: suggestion,
        };
        return;
    }
  }

  // Recompute suggestions for the current bean + recipe selection and apply
  // them to still-empty fields. This must run only for new brews, only when
  // the preference is enabled, only with a bean selected, and only after a
  // recipe (if any) has populated its fields.
  function applyAutofill() {
    if (!smartAutofillOn()) return;
    if (!beanRKey) return;
    const result = computeAutofill(historyBrews, {
      beanRKey,
      recipeRKey: recipeRKeyValue || undefined,
      recipeOwnerDID: recipeOwner || undefined,
      brewerRKey: brewerRKey || undefined,
      brewerCategory: brewerRKey
        ? normalizeBrewerCategory(brewerCategory)
        : undefined,
      grinderRKey: grinderRKey || undefined,
    });
    if (!result) return;
    for (const [field, suggestion] of Object.entries(result.fields) as [
      AutofillField,
      AutofillSuggestion,
    ][]) {
      applySuggestion(field, suggestion);
    }
  }

  let autofillCount = $derived(Object.keys(autofilled).length);

  $effect(() => {
    const enabled = $userPrefs.smartAutofillEnabled;
    const loading = $userPrefs.loading;
    // A mode switch never clears already-applied suggestions or provenance.
    if (!loading && !enabled && autofillCount > 0) clearAutofill();
  });

  type ComboType = "recipe" | "bean" | "grinder" | "brewer";

  function handleComboChange(type: ComboType, detail: Record<string, unknown>) {
    dirty = true;
    const rkey = (detail.rkey as string) ?? "";
    const entity = detail.entity as Record<string, unknown> | undefined;
    if (!rkey) {
      clearCombo(type);
      return;
    }
    if (type === "recipe") {
      recipeRKeyValue = rkey;
      recipeOwner =
        (detail.owner as string) ||
        (entity?.author_did as string) ||
        $session.did;
      void applyRecipe(rkey, recipeOwner);
      return;
    }
    if (type === "bean") {
      beanRKey = rkey;
      clearAutofill();
      applyAutofill();
      return;
    }
    if (type === "grinder") {
      if (transitionGrinder(rkey, grinderLabel)) applyAutofill();
      return;
    }
    if (type === "brewer") {
      if (!transitionMethod((entity?.brewer_type as string) ?? "")) {
        brewerRKey = acceptedBrewer.rkey;
        brewerLabel = acceptedBrewer.label;
        ++brewerRevision;
        focusCombo("brewer_rkey");
        return;
      }
      ++brewerChoiceVersion;
      brewerRKey = rkey;
      acceptedBrewer = { rkey, label: brewerLabel };
      clearAutofillField("brewer_rkey");
      applyAutofill();
    }
  }

  function clearCombo(type: ComboType) {
    if (type === "recipe") {
      ++recipeGen;
      recipeLoading = false;
      recipeError = "";
      recipeNotice = "";
      recipeRKeyValue = "";
      recipeLabel = "";
      activeRecipe = null;
      recipeOwner = "";
      acceptedRecipe = { rkey: "", label: "", owner: "" };
      recipeExpanded = false;
      recipeRatio = "";
      recipePours = [];
      applyAutofill();
    }
    if (type === "bean") {
      beanRKey = "";
      beanLabel = "";
      clearAutofill();
    }
    if (type === "grinder") transitionGrinder("", "");
    if (type === "brewer") {
      if (!transitionMethod("")) {
        brewerRKey = acceptedBrewer.rkey;
        brewerLabel = acceptedBrewer.label;
        ++brewerRevision;
        focusCombo("brewer_rkey");
        return;
      }
      ++brewerChoiceVersion;
      brewerRKey = "";
      brewerLabel = "";
      acceptedBrewer = { rkey: "", label: "" };
      clearAutofillField("brewer_rkey");
    }
  }

  function ownRecipeBrewer(recipe: Recipe, owner: string): Brewer | undefined {
    const brewers = (cachedData.brewers as Brewer[] | undefined) ?? [];
    if (!owner || owner === $session.did) {
      const own = brewers.find((b) => b.rkey === recipe.brewer_rkey);
      if (own) return own;
    }
    // The foreign-recipe endpoint attempts a name/type match, but leaves the
    // source rkey intact on failure and keeps brewer_obj as the source object.
    // Never treat an rkey collision alone as proof of ownership.
    const source = recipe.brewer_obj;
    if (!source) return undefined;
    const sourceURI = `at://${owner}/social.arabica.alpha.brewer/${source.rkey}`;
    const linked = brewers.find((b) => b.source_ref === sourceURI);
    if (linked) return linked;
    const named =
      source.name &&
      brewers.find((b) => b.name.toLowerCase() === source.name.toLowerCase());
    if (named) return named;
    // Same policy as internal/matching: only an unambiguous type fallback.
    const typed = brewers.filter(
      (b) =>
        source.brewer_type &&
        b.brewer_type?.toLowerCase() === source.brewer_type.toLowerCase(),
    );
    return typed.length === 1 ? typed[0] : undefined;
  }

  async function applyRecipe(selectedRKey: string, selectedOwner = "") {
    if (!selectedRKey) {
      activeRecipe = null;
      recipeOwner = "";
      recipeExpanded = false;
      return;
    }
    const gen = ++recipeGen;
    const brewerVersion = brewerChoiceVersion;
    recipeLoading = true;
    recipeError = "";
    recipeNotice = "";
    // The endpoint defaults to the authenticated user's recipe when no
    // owner query is supplied. Still retain the DID for owner-safe matching.
    recipeOwner = selectedOwner || $session.did;
    const ownerQuery = selectedOwner
      ? `?owner=${encodeURIComponent(selectedOwner)}`
      : "";
    try {
      const res = await fetch(`/api/recipes/${selectedRKey}${ownerQuery}`, {
        credentials: "same-origin",
      });
      if (gen !== recipeGen) return;
      notifySessionExpiredForResponse(res);
      if (!res.ok) throw new Error("Recipe could not be loaded");
      const recipe = (await res.json()) as Recipe;
      // URL recipes can resolve before the user's own collection arrives.
      await dataLoad;
      if (gen !== recipeGen) return;
      const recipeBrewer =
        brewerVersion === brewerChoiceVersion
          ? ownRecipeBrewer(recipe, selectedOwner || $session.did)
          : undefined;
      const recipeType =
        recipe.brewer_obj?.brewer_type ||
        recipe.brewer_type ||
        recipeBrewer?.brewer_type ||
        "";
      const recipeCategory = recipeType
        ? normalizeBrewerCategory(recipeType)
        : "";
      // A newer explicit brewer choice wins over this pending recipe. Only a
      // real, own saved brewer can change the draft's method.
      if (recipeBrewer && !transitionMethod(recipeBrewer.brewer_type)) {
        recipeRKeyValue = acceptedRecipe.rkey;
        recipeLabel = acceptedRecipe.label;
        recipeOwner = acceptedRecipe.owner;
        return;
      }
      clearAutofill(true);
      if (recipeBrewer) {
        brewerCategory = normalizeBrewerCategory(recipeBrewer.brewer_type);
        brewerRKey = recipeBrewer.rkey;
        brewerLabel = recipeBrewer.name;
        acceptedBrewer = { rkey: brewerRKey, label: brewerLabel };
      }
      activeRecipe = recipe;
      recipeLabel = recipe.name;
      acceptedRecipe = {
        rkey: selectedRKey,
        label: recipeLabel,
        owner: recipeOwner,
      };
      recipeExpanded = false;
      coffeeAmount =
        recipe.coffee_amount > 0
          ? String(Math.round(recipe.coffee_amount))
          : "";
      waterAmount =
        recipe.water_amount > 0 ? String(Math.round(recipe.water_amount)) : "";
      recipeRatio = formatRatio(recipe.coffee_amount, recipe.water_amount);
      recipePours = (recipe.pours ?? []).map((p) => ({
        water: String(p.water_amount ?? ""),
        time: String(p.time_seconds ?? ""),
      }));
      const copyPours =
        brewerCategory === "pourover" &&
        (!recipeCategory || recipeCategory === "pourover");
      if (copyPours) pours = recipePours.map((pour) => ({ ...pour }));
      else if (recipePours.length) {
        recipePours = [];
        recipeNotice =
          "Recipe pours were not copied because they do not match the selected brewer. Choose a compatible saved brewer and reselect the recipe to copy them.";
      }
      if (!brewerRKey)
        recipeNotice =
          "Choose one of your saved brewers before saving. This recipe has no matching brewer in your collection.";
      // For pour-over recipes, the bloom mirrors the first pour: its water
      // becomes the bloom amount and its time becomes the bloom duration.
      if (copyPours && pours.length > 0) {
        const firstPourWater = positiveNumber(pours[0].water);
        if (firstPourWater !== null)
          pouroverBloomWater = String(firstPourWater);
        const firstPourTime = positiveNumber(pours[0].time);
        if (firstPourTime !== null)
          pouroverBloomSeconds = String(firstPourTime);
      }
    } catch {
      if (gen !== recipeGen) return;
      recipeError =
        "Could not load that recipe. Your brew details have been kept. Try again or choose another recipe.";
      recipeRKeyValue = acceptedRecipe.rkey;
      recipeLabel = acceptedRecipe.label;
      recipeOwner = acceptedRecipe.owner;
    } finally {
      if (gen === recipeGen) {
        recipeLoading = false;
        applyAutofill();
      }
    }
  }

  function formatRatio(coffee: number, water: number): string {
    if (coffee <= 0 || water <= 0) return "";
    return String(Number((water / coffee).toFixed(2)));
  }

  function positiveNumber(value: string): number | null {
    const parsed = Number(value);
    return Number.isFinite(parsed) && parsed > 0 ? parsed : null;
  }

  function roundedAmount(value: number): string {
    return String(Math.round(value));
  }

  function scalePoursToWater(targetWater: number) {
    const pourTotal = recipePours.reduce(
      (total, pour) => total + (positiveNumber(pour.water) ?? 0),
      0,
    );
    if (pourTotal <= 0 || brewerCategory !== "pourover") return;

    const lastPourIndex = recipePours.reduce(
      (lastIndex, pour, index) =>
        positiveNumber(pour.water) === null ? lastIndex : index,
      -1,
    );
    if (lastPourIndex === -1) return;

    let scaledTotal = 0;
    const scaled = recipePours.map((pour) => {
      const water = positiveNumber(pour.water);
      if (water === null) return pour;
      const scaledWater = Math.round((water / pourTotal) * targetWater);
      scaledTotal += scaledWater;
      return { ...pour, water: String(scaledWater) };
    });
    const lastPour = scaled[lastPourIndex];
    scaled[lastPourIndex] = {
      ...lastPour,
      water: String(
        Math.max(0, Number(lastPour.water) + targetWater - scaledTotal),
      ),
    };
    pours = scaled;
    // Keep the pour-over bloom water in lockstep with the first pour while a
    // recipe is driving the pours. Bloom only tracks the first pour; later
    // manual edits to bloom water survive until the next scaling pass.
    if (activeRecipe && brewerCategory === "pourover" && scaled.length > 0) {
      const firstPourWater = positiveNumber(scaled[0].water);
      if (firstPourWater !== null) pouroverBloomWater = String(firstPourWater);
    }
  }

  function setRecipeMeasurements(nextCoffee: number, nextWater: number) {
    coffeeAmount = roundedAmount(nextCoffee);
    waterAmount = roundedAmount(nextWater);
    scalePoursToWater(Math.round(nextWater));
  }

  function adjustRecipeCoffee(value: string) {
    coffeeAmount = value;
    const coffee = positiveNumber(value);
    const ratio = positiveNumber(recipeRatio);
    if (coffee === null || ratio === null) return;
    setRecipeMeasurements(coffee, coffee * ratio);
  }

  function adjustRecipeWater(value: string) {
    waterAmount = value;
    const water = positiveNumber(value);
    const ratio = positiveNumber(recipeRatio);
    if (water === null || ratio === null) return;
    setRecipeMeasurements(water / ratio, water);
  }

  function adjustRecipeRatio(value: string) {
    recipeRatio = value;
    const ratio = positiveNumber(value);
    const coffee = positiveNumber(coffeeAmount);
    if (ratio === null || coffee === null) return;
    setRecipeMeasurements(coffee, coffee * ratio);
  }

  function recipeSummary(): string {
    if (!activeRecipe) return "";
    const parts: string[] = [];
    if (coffeeAmount) parts.push(`${coffeeAmount}g coffee`);
    if (waterAmount) parts.push(`${waterAmount}g water`);
    if (pours.length) parts.push(`${pours.length} pours`);
    return parts.join(" · ");
  }

  function initializeFromBrew() {
    if (!brew) return;
    beanRKey = brew.bean_rkey ?? "";
    beanLabel = brew.bean
      ? `${brew.bean.name || brew.bean.origin} (${brew.bean.origin} - ${brew.bean.roast_level})`
      : "";
    grinderRKey = brew.grinder_rkey ?? "";
    grinderLabel = brew.grinder_obj?.name ?? "";
    brewerRKey = brew.brewer_rkey ?? "";
    brewerLabel = brew.brewer_obj?.name ?? "";
    recipeRKeyValue = brew.recipe_rkey ?? "";
    recipeLabel = brew.recipe_obj?.name ?? "";
    coffeeAmount = brew.coffee_amount > 0 ? String(brew.coffee_amount) : "";
    waterAmount = brew.water_amount > 0 ? String(brew.water_amount) : "";
    grindSize = brew.grind_size ?? "";
    temperature = brew.temperature > 0 ? String(brew.temperature) : "";
    timeSeconds = brew.time_seconds > 0 ? String(brew.time_seconds) : "";
    tastingNotes = brew.tasting_notes ?? "";
    rating = brew.rating > 0 ? String(brew.rating) : "";
    recipeOwner =
      brew.recipe_owner_did ||
      brew.recipe_obj?.author_did ||
      (recipeRKeyValue ? $session.did : "");
    method = brew.method ?? "";
    pours = (brew.pours ?? []).map((p) => ({
      water: String(p.water_amount ?? ""),
      time: String(p.time_seconds ?? ""),
    }));
    if (brew.espresso_params) {
      espressoYieldWeight =
        brew.espresso_params.yield_weight > 0
          ? String(brew.espresso_params.yield_weight)
          : "";
      espressoPressure =
        brew.espresso_params.pressure > 0
          ? String(brew.espresso_params.pressure)
          : "";
      espressoPreInfusionSeconds =
        brew.espresso_params.pre_infusion_seconds > 0
          ? String(brew.espresso_params.pre_infusion_seconds)
          : "";
    }
    if (brew.pourover_params) {
      pouroverBloomWater =
        brew.pourover_params.bloom_water > 0
          ? String(brew.pourover_params.bloom_water)
          : "";
      pouroverBloomSeconds =
        brew.pourover_params.bloom_seconds > 0
          ? String(brew.pourover_params.bloom_seconds)
          : "";
      pouroverDrawdownSeconds =
        brew.pourover_params.drawdown_seconds > 0
          ? String(brew.pourover_params.drawdown_seconds)
          : "";
      pouroverBypassWater =
        brew.pourover_params.bypass_water > 0
          ? String(brew.pourover_params.bypass_water)
          : "";
      pouroverFilter = brew.pourover_params.filter ?? "";
    }
    brewerCategory = normalizeBrewerCategory(
      brew.brewer_obj?.brewer_type ?? "",
    );
    acceptedGrinder = { rkey: grinderRKey, label: grinderLabel };
    acceptedBrewer = { rkey: brewerRKey, label: brewerLabel };
    acceptedRecipe = {
      rkey: recipeRKeyValue,
      label: recipeLabel,
      owner: recipeOwner,
    };
  }

  initializeFromBrew();

  async function loadData() {
    dataState = "loading";
    try {
      // getData() falls back to stale cached data on refresh failures, which
      // would misreport an expired session (or stale empty list) as success.
      const data = await appCache.refreshCache(true);
      if (!data) throw new Error("No data loaded");
      cachedData = data;
      historyBrews = (data.brews as Brew[]) ?? [];
      dataState = "ready";
      applyAutofill();
    } catch (error) {
      // appCache currently throws a plain Error with this message, not an
      // APIError. Handle both until the shared cache exposes typed failures.
      const unauthorized =
        error instanceof APIError
          ? error.status === 401
          : error instanceof Error &&
            error.message === "Failed to fetch data: 401";
      dataState = unauthorized ? "auth-expired" : "error";
      if (unauthorized)
        notifySessionExpired(
          new APIError("Session expired", { status: 401, kind: "http" }),
        );
    }
  }

  onMount(() => {
    dataLoad = loadData();
    if (!isEdit)
      void loadUserPrefs().then(() => {
        if (!modeChosen && !dirty)
          mode = initialBrewFormMode(false, null, $userPrefs.brew_form_mode);
        applyAutofill();
      });
    const beforeUnload = (event: BeforeUnloadEvent) => {
      if (dirty && !saved) {
        event.preventDefault();
        event.returnValue = "";
      }
    };
    window.addEventListener("beforeunload", beforeUnload);
    // Auto-apply recipe from URL param on new brew.
    if (recipeRKeyValue && !brew) {
      void applyRecipe(recipeRKeyValue, recipeOwner);
    }
    // Proactively check whether the OAuth session is still resumable so we
    // can prompt re-authentication before a failed save, rather than after
    // the user has filled in the whole form.
    void warnIfSessionExpired();
    return () => {
      ++recipeGen;
      window.removeEventListener("beforeunload", beforeUnload);
    };
  });

  async function submitForm(e: SubmitEvent) {
    e.preventDefault();
    if (submitting || recipeLoading) return;
    formError = "";
    validationAttempted = true;
    if (!beanRKey || (!isEdit && !brewerRKey)) {
      formError = !beanRKey
        ? "Choose a coffee bean before saving."
        : "Choose a brewer before saving.";
      const name = !beanRKey ? "bean_rkey" : "brewer_rkey";
      focusCombo(name);
      return;
    }
    if (
      coffeeAmountError ||
      waterAmountError ||
      temperatureError ||
      timeSecondsError
    ) {
      formError =
        "Check the brewing measurements. Values must be greater than 0.";
      mode = "detailed";
      moreExpanded = true;
      await tick();
      formElement
        ?.querySelector<HTMLInputElement>(
          '.preparation-fields input[aria-invalid="true"]',
        )
        ?.focus();
      return;
    }
    submitting = true;

    const input: BrewInput = { bean_rkey: beanRKey };
    if (recipeRKeyValue) {
      input.recipe_rkey = recipeRKeyValue;
      if (recipeOwner) input.recipe_owner_did = recipeOwner;
    }
    if (method) input.method = method;
    if (coffeeAmount) input.coffee_amount = num(coffeeAmount);
    if (waterAmount) input.water_amount = num(waterAmount);
    if (grindSize) input.grind_size = grindSize;
    if (temperature) input.temperature = num(temperature);
    if (timeSeconds) input.time_seconds = num(timeSeconds);
    if (tastingNotes) input.tasting_notes = tastingNotes;
    if (rating) input.rating = num(rating);
    if (grinderRKey) input.grinder_rkey = grinderRKey;
    if (brewerRKey) input.brewer_rkey = brewerRKey;
    const poursInput = pours
      .filter((pour) => pour.water || pour.time)
      .map((pour) => ({
        water_amount: num(pour.water),
        time_seconds: num(pour.time),
      }));
    if (poursInput.length > 0) input.pours = poursInput;
    if (espressoYieldWeight || espressoPressure || espressoPreInfusionSeconds) {
      input.espresso_params = {
        yield_weight: num(espressoYieldWeight),
        pressure: num(espressoPressure),
        pre_infusion_seconds: num(espressoPreInfusionSeconds),
      };
    }
    if (
      pouroverBloomWater ||
      pouroverBloomSeconds ||
      pouroverDrawdownSeconds ||
      pouroverBypassWater ||
      pouroverFilter
    ) {
      input.pourover_params = {
        bloom_water: num(pouroverBloomWater),
        bloom_seconds: num(pouroverBloomSeconds),
        drawdown_seconds: num(pouroverDrawdownSeconds),
        bypass_water: num(pouroverBypassWater),
        filter: pouroverFilter,
      };
    }

    try {
      const data = isEdit
        ? await updateBrew(fetch, brew?.rkey ?? "", input)
        : await createBrew(fetch, input);
      saved = true;
      const savedBrew = data.brew;
      pushToast(isEdit ? "Brew updated!" : "Brew saved!");
      // The JSON envelope carries author_did at the top level (the Brew
      // record model has no author field); fall back to the session DID
      // for edits where the envelope may omit it.
      const actor = data.author_did ?? $session.did ?? "";
      const rkey = savedBrew.rkey ?? brew?.rkey ?? "";
      if (actor && rkey) {
        goto(`/brews/${actor}/${rkey}`);
      } else {
        goto("/my-coffee");
      }
    } catch (error) {
      console.error("Brew save failed:", error);
      formError =
        "Failed to save brew. Your draft is still here; please try again.";
      pushToast("Failed to save brew");
    } finally {
      submitting = false;
    }
  }

  let submitLabel = $derived(isEdit ? "Update brew" : "Save brew");

  function num(value: string): number {
    const n = Number(value);
    return Number.isFinite(n) ? n : 0;
  }
  let coffeeValue = $derived(num(coffeeAmount));
  let waterValue = $derived(num(waterAmount));
  let ratio = $derived(
    coffeeValue > 0 && waterValue > 0 ? waterValue / coffeeValue : null,
  );

  // Each control reads and writes the shared draft, regardless of its section.
  let measurements = $derived({
    coffee_amount: {
      label: "Coffee amount (g)",
      value: coffeeAmount,
      set: (v: string) => (coffeeAmount = v),
      placeholder: "e.g. 18",
      step: "1",
    },
    water_amount: {
      label: "Water amount (g)",
      value: waterAmount,
      set: (v: string) => (waterAmount = v),
      placeholder: "e.g. 250",
      step: "1",
    },
    temperature: {
      label: "Temperature (°F/°C)",
      value: temperature,
      set: (v: string) => (temperature = v),
      placeholder: "e.g. 93.5",
      step: "0.1",
    },
    time_seconds: {
      label: layout.timeLabel.replace(
        "(s)",
        timeUnit === "60" ? "(min)" : timeUnit === "3600" ? "(h)" : "(s)",
      ),
      value: timeSeconds ? String(Number(timeSeconds) / Number(timeUnit)) : "",
      set: (v: string) =>
        (timeSeconds = v
          ? String(Math.round(Number(v) * Number(timeUnit)))
          : ""),
      placeholder: "e.g. 180",
      step: timeUnit === "1" ? "1" : "any",
    },
    grind_size: {
      label: "Grind setting",
      value: grindSize,
      set: (v: string) => (grindSize = v),
      placeholder: "e.g. 18, Medium, 3.5, Fine",
      step: "",
    },
    espresso_yield_weight: {
      label: "Yield weight (g)",
      value: espressoYieldWeight,
      set: (v: string) => (espressoYieldWeight = v),
      placeholder: "e.g. 36",
      step: "0.1",
    },
    espresso_pressure: {
      label: "Pressure (bar)",
      value: espressoPressure,
      set: (v: string) => (espressoPressure = v),
      placeholder: "e.g. 9",
      step: "0.1",
    },
    espresso_pre_infusion_seconds: {
      label: "Pre-infusion time (s)",
      value: espressoPreInfusionSeconds,
      set: (v: string) => (espressoPreInfusionSeconds = v),
      placeholder: "e.g. 5",
      step: "1",
    },
    pourover_bloom_water: {
      label: "Bloom water (g)",
      value: pouroverBloomWater,
      set: (v: string) => (pouroverBloomWater = v),
      placeholder: "e.g. 50",
      step: "1",
    },
    pourover_bloom_seconds: {
      label: "Bloom time (s)",
      value: pouroverBloomSeconds,
      set: (v: string) => (pouroverBloomSeconds = v),
      placeholder: "e.g. 45",
      step: "1",
    },
    pourover_drawdown_seconds: {
      label: "Drawdown time (s)",
      value: pouroverDrawdownSeconds,
      set: (v: string) => (pouroverDrawdownSeconds = v),
      placeholder: "e.g. 30",
      step: "1",
    },
    pourover_bypass_water: {
      label: "Bypass water (g)",
      value: pouroverBypassWater,
      set: (v: string) => (pouroverBypassWater = v),
      placeholder: "e.g. 100",
      step: "1",
    },
    pourover_filter: {
      label: "Filter",
      value: pouroverFilter,
      set: (v: string) => (pouroverFilter = v),
      placeholder: "e.g. paper, metal, cloth",
      step: "",
    },
  });
  type Measurement = keyof typeof measurements;
  let detailsSummary = $derived(
    [
      coffeeAmount && `${coffeeAmount}g coffee`,
      waterAmount && `${waterAmount}g water`,
      timeSeconds && `${timeSeconds}s`,
      temperature && `${temperature}° (recorded unit)`,
      grinderRKey && (grinderLabel || "Grinder selected"),
      grindSize && `Grind setting: ${grindSize}`,
      hasEspresso && "Espresso details",
      hasPourover && "Pour-over details",
      pours.length && `${pours.length} pours`,
      method,
    ]
      .filter(Boolean)
      .join(" · "),
  );
</script>

{#snippet measurement(field: Measurement)}
  {@const spec = measurements[field]}
  {@const invalid =
    ["coffee_amount", "water_amount", "temperature", "time_seconds"].includes(
      field,
    ) && mustBePositive(spec.value)}
  <Field label={spec.label} badge={autofillBadge(field)}>
    <input
      type={spec.step ? "number" : "text"}
      value={spec.value}
      oninput={(event) => {
        spec.set(event.currentTarget.value);
        clearAutofillField(field);
      }}
      placeholder={spec.placeholder}
      step={spec.step || undefined}
      min={spec.step ? "0" : undefined}
      class={autofillInputClass(field)}
      id={`brew-${field}`}
      name={field}
      aria-invalid={invalid}
      aria-describedby={invalid ? `${field}-error` : undefined}
    />
    {#if invalid}<p id={`${field}-error`} class="text-xs text-red-600 mt-1">
        {spec.label} must be greater than 0.
      </p>{/if}
  </Field>
{/snippet}

{#snippet grinderFields()}
  <div class="combo-select">
    <span class="form-label">Grinder (optional)</span>
    {#key grinderRevision}
      <EntityCombo
        entityType="grinder"
        inputName="grinder_rkey"
        inputClass={autofillInputClass("grinder_rkey")}
        apiEndpoint="/api/grinders"
        suggestEndpoint="/api/suggestions/grinders"
        placeholder="Search grinders..."
        sectionLabel="Your grinders"
        allowCreate={false}
        bind:rkey={grinderRKey}
        bind:label={grinderLabel}
        ariaLabel="Search grinders"
        onChange={(detail) => handleComboChange("grinder", detail)}
      />
    {/key}
    {#if autofillBadge("grinder_rkey")}<span class="autofill-badge"
        >{autofillBadge("grinder_rkey")}</span
      >{/if}
  </div>
  {#if grinderRKey || grindSize}
    {#if !grinderRKey}<p class="text-helper">
        This saved setting has no associated grinder. You can keep or edit it as
        it is.
      </p>{/if}
    {@render measurement("grind_size")}
  {/if}
{/snippet}

<FormWorkspace>
  <LedgerHeader title={isEdit ? "Edit Brew" : "New Brew"} showBack={true} />

  <form
    class="brew-form-sheet"
    bind:this={formElement}
    novalidate
    onsubmit={submitForm}
    oninput={() => (dirty = true)}
  >
    <div class="mode-switch" role="group" aria-label="Brew logging mode">
      <button
        type="button"
        class="btn-secondary"
        aria-pressed={mode === "quick"}
        onclick={() => chooseMode("quick")}>Quick</button
      >
      <button
        type="button"
        class="btn-secondary"
        aria-pressed={mode === "detailed"}
        onclick={() => chooseMode("detailed")}>Detailed</button
      >
    </div>
    {#if formError}<p id="brew-form-error" role="alert" class="text-red-600">
        {formError}
      </p>{/if}
    {#if dataState === "error" || dataState === "auth-expired"}
      <p role="alert">
        {dataState === "auth-expired"
          ? "Your session expired. Sign in again to load your collection; your draft is still here."
          : "Could not load your beans, brewers, and recipes. This does not mean your collection is empty."}
      </p>
      <button
        type="button"
        class="btn-secondary"
        onclick={() => void loadData()}>Retry loading</button
      >
    {/if}
    {#if missingBean || missingBrewer}
      <div class="section-box" aria-label="Before your first brew">
        <p>
          {missingBean && missingBrewer
            ? "Add a bean and brewer before logging your brew."
            : missingBean
              ? "Add a bean before logging your brew."
              : "Add a brewer before logging your brew."}
        </p>
        <p class="text-helper">
          Creation opens in a new tab. Return here to keep your draft and
          recipe, then refresh your collection.
        </p>
        {#if missingBean}<a
            href="/beans/new"
            target="_blank"
            rel="noopener"
            class="btn-secondary">Add a bean</a
          >{/if}
        {#if missingBrewer}<a
            href="/brewers/new"
            target="_blank"
            rel="noopener"
            class="btn-secondary">Add a brewer</a
          >{/if}
        <button
          type="button"
          class="btn-secondary"
          onclick={() => void loadData()}>Refresh collection</button
        >
      </div>
    {/if}
    {#if autofillCount > 0}
      <div class="autofill-summary">
        <p class="text-helper">
          Some empty fields were filled from your brew history.
        </p>
        <button
          type="button"
          onclick={() => clearAutofill()}
          class="btn-secondary text-sm">Clear autofill</button
        >
      </div>
    {/if}
    {#if showRecipe || recipeError}
      <FormSection
        title="Recipe (optional)"
        description="Copies brewing details into this brew."
      >
        {#if recipeError}<p role="alert">{recipeError}</p>{/if}
        {#if recipeNotice}<p class="text-helper">{recipeNotice}</p>{/if}
        {#if recipeLoading}<p role="status">Loading recipe…</p>{/if}
        <div class="combo-select">
          <EntityCombo
            entityType="recipe"
            inputName="recipe_rkey"
            apiEndpoint="/api/recipes"
            suggestEndpoint="/api/suggestions/recipes"
            placeholder="Search recipes..."
            sectionLabel="Your recipes"
            passthrough={true}
            allowCreate={false}
            bind:rkey={recipeRKeyValue}
            bind:label={recipeLabel}
            ariaLabel="Search recipes"
            onChange={(detail) => handleComboChange("recipe", detail)}
          />
        </div>

        {#if activeRecipe || (isEdit && recipeRKeyValue)}
          <div class="section-box">
            <p class="text-helper">Changes here do not change the recipe.</p>
            <div class="flex items-center justify-between gap-2">
              <p class="text-sm text-emphasis flex-1">
                {recipeSummary() || recipeLabel || "Attached recipe"}
              </p>
              <button
                type="button"
                onclick={() => {
                  recipeExpanded = !recipeExpanded;
                  void openDetails();
                }}
                class="text-sm btn-secondary"
                aria-expanded={recipeExpanded}
              >
                {recipeExpanded ? "Done adjusting" : "Edit brewing details"}
              </button>
            </div>
            {#if recipeExpanded && mode === "detailed" && activeRecipe}
              <div class="recipe-adjustment" aria-label="Recipe adjustment">
                <div class="recipe-adjustment-heading">
                  <p class="text-helper">
                    Set a dose, water, or ratio. The other measurement follows,
                    and recipe pours scale with the water.
                  </p>
                </div>
                <div class="recipe-adjustment-grid">
                  <label for="recipe-adjustment-coffee">
                    <span class="form-label">Coffee (g)</span>
                    <input
                      id="recipe-adjustment-coffee"
                      aria-invalid={coffeeAmountError}
                      aria-describedby={coffeeAmountError
                        ? "coffee_amount-error"
                        : undefined}
                      type="number"
                      value={coffeeAmount}
                      oninput={(event) =>
                        adjustRecipeCoffee(event.currentTarget.value)}
                      step="1"
                      min="1"
                      class="w-full form-input-lg"
                    />
                  </label>
                  <label for="recipe-adjustment-ratio">
                    <span class="form-label">Ratio (1:X)</span>
                    <input
                      id="recipe-adjustment-ratio"
                      type="number"
                      value={recipeRatio}
                      oninput={(event) =>
                        adjustRecipeRatio(event.currentTarget.value)}
                      step="0.1"
                      min="0.1"
                      class="w-full form-input-lg"
                    />
                  </label>
                  <label for="recipe-adjustment-water">
                    <span class="form-label">Water (g)</span>
                    <input
                      id="recipe-adjustment-water"
                      aria-invalid={waterAmountError}
                      aria-describedby={waterAmountError
                        ? "water_amount-error"
                        : undefined}
                      type="number"
                      value={waterAmount}
                      oninput={(event) =>
                        adjustRecipeWater(event.currentTarget.value)}
                      step="1"
                      min="1"
                      class="w-full form-input-lg"
                    />
                  </label>
                </div>
              </div>
            {/if}
          </div>
        {/if}
      </FormSection>
    {/if}

    <FormSection title="Coffee">
      <div class="combo-select">
        <span class="form-label"
          >Coffee bean <span class="text-red-500" aria-hidden="true">*</span
          ></span
        >
        <EntityCombo
          entityType="bean"
          inputName="bean_rkey"
          apiEndpoint="/api/beans"
          suggestEndpoint="/api/suggestions/beans"
          placeholder="Search beans..."
          sectionLabel="Your beans"
          required={true}
          allowCreate={false}
          bind:rkey={beanRKey}
          bind:label={beanLabel}
          ariaLabel="Search coffee beans"
          onChange={(detail) => handleComboChange("bean", detail)}
        />
        {#if validationAttempted && !beanRKey}<p
            id="bean_rkey-error"
            class="text-red-600"
          >
            Choose a coffee bean before saving.
          </p>{/if}
      </div>

      <div class="combo-select">
        <span class="form-label">Brewer{isEdit ? "" : " (required)"}</span>
        {#key brewerRevision}
          <EntityCombo
            entityType="brewer"
            inputName="brewer_rkey"
            inputClass={autofillInputClass("brewer_rkey")}
            apiEndpoint="/api/brewers"
            suggestEndpoint="/api/suggestions/brewers"
            placeholder="Search brewers..."
            sectionLabel="Your brewers"
            required={!isEdit}
            allowCreate={false}
            bind:rkey={brewerRKey}
            bind:label={brewerLabel}
            ariaLabel="Search brew methods"
            onChange={(detail) => handleComboChange("brewer", detail)}
          />
        {/key}
        {#if validationAttempted && !isEdit && !brewerRKey}<p
            id="brewer_rkey-error"
            class="text-red-600"
          >
            Choose a brewer before saving.
          </p>{/if}
        <div class="autofill-status" aria-live="polite">
          {#if autofillBadge("brewer_rkey")}<span class="autofill-badge"
              >{autofillBadge("brewer_rkey")}</span
            >{/if}
        </div>
      </div>
    </FormSection>

    {#if mode === "quick" && hasDetails}
      <div class="section-box" aria-label="Brewing details added">
        <p class="form-label">Brewing details added</p>
        <p class="text-helper">{detailsSummary}</p>
        <button type="button" class="btn-secondary" onclick={openDetails}
          >Edit brewing details</button
        >
      </div>
    {/if}

    {#if mode === "detailed"}
      <div class="preparation-fields">
        <FormSection title="Brewing (optional)">
          {@render measurement("coffee_amount")}
          {#if layout.mainWater}
            <div>
              {@render measurement("water_amount")}
              {#if ratio !== null}<p class="text-helper brew-ratio">
                  Ratio 1:{ratio.toFixed(1)}
                </p>{/if}
            </div>
          {/if}
          {#if layout.category === "espresso"}{@render measurement(
              "espresso_yield_weight",
            )}{/if}
          {#if layout.mainTime}
            {#if ["immersion", "coldbrew", "cupping"].includes(layout.category)}
              <Field label="Time unit" helper="Steep time is saved in seconds."
                ><select class="w-full form-input-lg" bind:value={timeUnit}
                  ><option value="1">Seconds</option><option value="60"
                    >Minutes</option
                  ><option value="3600">Hours</option></select
                ></Field
              >
            {/if}
            {@render measurement("time_seconds")}
          {/if}
          {#if layout.mainGrinder}{@render grinderFields()}{/if}
        </FormSection>
        <details class="more-details" bind:open={moreExpanded}>
          <summary>More brewing details</summary>
          <div class="more-details-fields">
            {#if !layout.mainGrinder}{@render grinderFields()}{/if}
            {@render measurement("temperature")}
            {#if !layout.mainTime}{@render measurement("time_seconds")}{/if}
            {#if !layout.mainWater && waterAmount}{@render measurement(
                "water_amount",
              )}{/if}
            {#if method}<Field label="Saved method"
                ><input
                  class="w-full form-input-lg"
                  bind:value={method}
                /></Field
              >{/if}
            {#if layout.category === "pourover" || pours.length > 0}<PoursEditor
                bind:pours
                expectedWater={waterAmount}
              />{/if}

            {#if layout.category === "espresso" || hasEspresso}
              <FormSection title="Espresso">
                {#if layout.category !== "espresso"}{@render measurement(
                    "espresso_yield_weight",
                  )}{/if}
                {@render measurement("espresso_pressure")}
                {@render measurement("espresso_pre_infusion_seconds")}
              </FormSection>
            {/if}

            {#if layout.category === "pourover" || hasPourover}
              <FormSection title="Pour-over details">
                {@render measurement("pourover_bloom_water")}
                {@render measurement("pourover_bloom_seconds")}
                {@render measurement("pourover_drawdown_seconds")}
                {@render measurement("pourover_bypass_water")}
                {@render measurement("pourover_filter")}
              </FormSection>
            {/if}
          </div>
        </details>
      </div>
    {/if}

    <FormSection title="Results">
      <Field label="Tasting notes">
        <textarea
          bind:value={tastingNotes}
          placeholder="Describe the flavors, aroma, and your thoughts..."
          rows="4"
          class="w-full form-input-lg"
        ></textarea>
      </Field>
      <div class="space-y-2">
        <div class="flex items-center justify-between gap-3">
          {#if rating}<label class="form-label mb-0" for="brew-rating"
              >Rating</label
            >{/if}
          <button
            type="button"
            class="btn-secondary text-sm"
            aria-expanded={!!rating}
            aria-controls="brew-rating-control"
            onclick={() => {
              rating = rating ? "" : "5";
              dirty = true;
              if (rating)
                void tick().then(() =>
                  document.getElementById("brew-rating")?.focus(),
                );
            }}>{rating ? "Remove rating" : "Add rating"}</button
          >
        </div>
        <div id="brew-rating-control">
          {#if rating}
            <div class="flex items-center gap-3">
              <input
                id="brew-rating"
                type="range"
                min="1"
                max="10"
                step="1"
                value={rating}
                oninput={(event) => (rating = event.currentTarget.value)}
                aria-valuetext={`${rating} out of 10`}
                class="w-full accent-brown-700"
              />
              <span class="text-sm font-medium text-emphasis min-w-[2.5rem]"
                >{rating}/10</span
              >
            </div>
          {/if}
        </div>
      </div>
    </FormSection>

    <div class="flex flex-col-reverse sm:flex-row sm:justify-end gap-3 pt-2">
      <a
        href={isEdit && brew
          ? `/brews/${encodeURIComponent($session.did || $session.handle)}/${encodeURIComponent(brew.rkey)}`
          : "/my-coffee"}
        class="btn-secondary text-center">Cancel</a
      >
      <button
        type="submit"
        class="btn-primary"
        disabled={submitting || recipeLoading || missingBean || missingBrewer}
      >
        {submitting ? "Saving..." : submitLabel}
      </button>
    </div>
  </form>
</FormWorkspace>

<style>
  .brew-form-sheet {
    padding-top: 1.5rem;
  }
  .mode-switch {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.25rem;
  }
  .mode-switch button {
    min-height: 44px;
  }
  .mode-switch [aria-pressed="true"] {
    background: var(--text-primary);
    color: var(--surface-bg);
  }
  .more-details {
    border-top: 1px solid var(--card-border);
  }
  .more-details summary {
    box-sizing: border-box;
    cursor: pointer;
    min-height: 44px;
    padding-block: 0.625rem;
    font-weight: 600;
  }
  .more-details-fields {
    padding-block: 0.5rem 1.25rem;
  }
  .brew-form-sheet :global(.form-section) {
    margin: 0;
    padding-block: 1.25rem;
    border-top: 1px solid var(--card-border);
  }
  .brew-form-sheet :global(.form-section > header) {
    display: flex;
    flex-wrap: wrap;
    align-items: baseline;
    gap: 0.5rem 1.5rem;
    margin-bottom: 1rem;
  }
  .brew-form-sheet :global(.form-section__body),
  .more-details-fields {
    display: grid;
    gap: 1.25rem;
  }
  .brew-ratio {
    margin-top: 0.5rem;
  }
  .brew-form-sheet .autofill-summary {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    margin-bottom: 1.25rem;
    padding: 0.65rem 0.9rem;
    border: 1px dashed var(--card-border);
    border-radius: 0.5rem;
    background: color-mix(in oklch, var(--surface-bg) 74%, transparent);
  }
  .brew-form-sheet :global(.autofilled-input) {
    color: var(--autofill-text);
  }
  .brew-form-sheet :global(.autofill-status) {
    display: flex;
    align-items: center;
  }
  .brew-form-sheet :global(.autofill-status:not(:empty)) {
    margin-top: 0.5rem;
  }
  .brew-form-sheet :global(.autofill-badge) {
    display: inline-flex;
    align-items: center;
    min-height: 1.1rem;
    padding: 0.15rem 0.5rem;
    border: 1px dashed var(--autofill-text);
    border-radius: 999px;
    background: color-mix(in oklch, var(--surface-bg) 74%, transparent);
    color: var(--autofill-text);
    font-size: 0.68rem;
    font-weight: 600;
    letter-spacing: 0.02em;
    line-height: 1.1;
  }
  .recipe-adjustment {
    margin-top: 0.75rem;
    padding: 1rem;
    border: 1px solid
      color-mix(in oklch, var(--type-recipe) 35%, var(--surface-border));
    border-radius: 0.5rem;
    background: color-mix(
      in oklch,
      var(--type-recipe-tint) 40%,
      var(--surface-bg)
    );
  }
  .recipe-adjustment-heading {
    margin-bottom: 0.75rem;
  }
  .recipe-adjustment-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 0.75rem;
  }
  @media (max-width: 38rem) {
    .recipe-adjustment-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
