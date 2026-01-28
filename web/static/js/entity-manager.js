/**
 * Entity Manager Module
 * Provides reusable CRUD operations for entity management
 * Eliminates duplication between brew-form.js and manage-page.js
 */

/**
 * Creates an entity manager for a specific entity type
 * @param {Object} config - Configuration object
 * @param {string} config.entityType - Entity type name (e.g., 'bean', 'grinder', 'brewer', 'roaster')
 * @param {string} config.apiEndpoint - API endpoint (e.g., '/api/beans')
 * @param {Object} config.defaultFormData - Default form data structure
 * @param {Function} config.validate - Validation function that returns error message or null
 * @param {Function} config.onSuccess - Callback after successful save/delete (optional)
 * @param {boolean} config.reloadOnSuccess - Whether to reload page on success (default: false)
 * @returns {Object} Entity manager with CRUD methods
 */
function createEntityManager(config) {
  const {
    entityType,
    apiEndpoint,
    defaultFormData,
    validate,
    onSuccess,
    reloadOnSuccess = false,
  } = config;

  return {
    // Modal state
    showForm: false,
    editingId: null,

    // Form data (will be initialized with defaultFormData)
    formData: { ...defaultFormData },

    /**
     * Opens the form for creating a new entity
     */
    openNew() {
      this.editingId = null;
      this.formData = { ...defaultFormData };
      this.showForm = true;
    },

    /**
     * Opens the form for editing an existing entity
     * @param {string} rkey - Record key
     * @param {Object} data - Entity data to populate form
     */
    openEdit(rkey, data) {
      this.editingId = rkey;
      this.formData = { ...data };
      this.showForm = true;
    },

    /**
     * Saves the entity (create or update)
     */
    async save() {
      // Validate form data
      const error = validate(this.formData);
      if (error) {
        alert(error);
        return;
      }

      // Determine URL and method
      const url = this.editingId
        ? `${apiEndpoint}/${this.editingId}`
        : apiEndpoint;
      const method = this.editingId ? "PUT" : "POST";

      try {
        const response = await fetch(url, {
          method,
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(this.formData),
        });

        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(errorText);
        }

        const result = await response.json();

        // Invalidate cache
        if (window.ArabicaCache) {
          window.ArabicaCache.invalidateCache();
        }

        // Call success callback if provided
        if (onSuccess) {
          await onSuccess(result, this.editingId);
        }

        // Close modal and reset form
        this.showForm = false;
        this.formData = { ...defaultFormData };
        this.editingId = null;

        // Reload page if configured to do so
        if (reloadOnSuccess) {
          window.location.reload();
        }
      } catch (error) {
        const action = this.editingId ? "update" : "add";
        alert(`Failed to ${action} ${entityType}: ${error.message}`);
      }
    },

    /**
     * Deletes an entity
     * @param {string} rkey - Record key to delete
     */
    async delete(rkey) {
      if (!confirm(`Are you sure you want to delete this ${entityType}?`)) {
        return;
      }

      try {
        const response = await fetch(`${apiEndpoint}/${rkey}`, {
          method: "DELETE",
        });

        if (!response.ok) {
          const errorText = await response.text();
          throw new Error(errorText);
        }

        // Invalidate cache
        if (window.ArabicaCache) {
          window.ArabicaCache.invalidateCache();
        }

        // Reload page if configured to do so
        if (reloadOnSuccess) {
          window.location.reload();
        }
      } catch (error) {
        alert(`Failed to delete ${entityType}: ${error.message}`);
      }
    },

    /**
     * Closes the form modal without saving
     */
    closeForm() {
      this.showForm = false;
      this.formData = { ...defaultFormData };
      this.editingId = null;
    },
  };
}

// Export for use in other modules
window.createEntityManager = createEntityManager;
