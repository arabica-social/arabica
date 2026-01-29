/**
 * Entity Modal and Dropdown Helpers
 * Provides helper functions for managing entity modals and refreshing dropdowns
 */

/**
 * Opens a modal dialog by fetching it from the server via HTMX
 * @param {string} modalUrl - The URL to fetch the modal content from
 */
window.openEntityModal = function(modalUrl) {
  // The HTMX attributes on the button will handle the fetch
  // This function is here for future extensibility if needed
};

/**
 * Refreshes an entity dropdown select element after entity creation/update
 * @param {string} entityType - Type of entity ('beans', 'grinders', 'brewers', 'roasters')
 */
window.refreshEntityDropdown = function(entityType) {
  // Find the select element for this entity type
  const selectId = entityType.replace(/s$/, '') + '_rkey'; // beans -> bean_rkey
  const selectElement = document.getElementById(selectId);

  if (selectElement) {
    // Trigger a refresh by fetching updated data
    // The dropdown manager will handle the actual refresh
    if (window.ArabicaCache && window.ArabicaCache.invalidateAndRefresh) {
      window.ArabicaCache.invalidateAndRefresh();
    }
  }
};

/**
 * Shows a modal dialog element
 * @param {string} dialogId - ID of the dialog element
 */
window.showModal = function(dialogId) {
  const dialog = document.getElementById(dialogId);
  if (dialog && typeof dialog.showModal === 'function') {
    dialog.showModal();
  }
};

/**
 * Closes a modal dialog element
 * @param {string} dialogId - ID of the dialog element
 */
window.closeModal = function(dialogId) {
  const dialog = document.getElementById(dialogId);
  if (dialog && typeof dialog.close === 'function') {
    dialog.close();
  }
};
