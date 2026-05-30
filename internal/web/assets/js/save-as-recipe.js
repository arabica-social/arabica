// @ts-check

/**
 * Petite-vue scope for saving a brew's parameters as a recipe.
 * @param {string | null} brewRKey
 */
function saveAsRecipe(brewRKey) {
  return {
    showForm: false,
    name: "",
    saving: false,
    error: "",
    success: false,
    brewRKey: brewRKey || "",

    async saveRecipe() {
      if (!this.name.trim()) {
        this.error = "Name is required";
        return;
      }

      this.saving = true;
      this.error = "";

      const body = new URLSearchParams({ name: this.name });

      try {
        const response = await fetch(`/api/recipes/from-brew/${this.brewRKey}`, {
          method: "POST",
          headers: { "Content-Type": "application/x-www-form-urlencoded" },
          body,
          credentials: "same-origin",
        });

        if (!response.ok) throw new Error("Failed to save recipe");

        await response.json();
        this.success = true;
      } catch (e) {
        this.error = "Failed to save recipe";
      } finally {
        this.saving = false;
      }
    },
  };
}

window.saveAsRecipe = saveAsRecipe;
