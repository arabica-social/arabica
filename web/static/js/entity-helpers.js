/**
 * Entity Modal and Dropdown Helpers
 * Provides helper functions for managing entity modals and refreshing dropdowns
 */

/**
 * Opens a modal dialog by fetching it from the server via HTMX
 * @param {string} modalUrl - The URL to fetch the modal content from
 */
window.openEntityModal = function (modalUrl) {
  // The HTMX attributes on the button will handle the fetch
  // This function is here for future extensibility if needed
};

/**
 * Refreshes an entity dropdown select element after entity creation/update
 * Also handles reloading the manage page if we're on it
 * @param {string} entityType - Type of entity ('beans', 'grinders', 'brewers', 'roasters')
 * @param {string} selectName - Name of the select element (e.g., 'bean_rkey')
 * @param {string} newRkey - RKey of the newly created entity to auto-select
 */
window.refreshEntityDropdown = async function (entityType, selectName, newRkey) {
  // Check if we're on the manage page by looking for the manage content loader
  const manageLoader = document.querySelector('[hx-get="/api/manage"]');

  if (manageLoader) {
    // We're on the manage page - reload the manage partial
    if (typeof htmx !== 'undefined') {
      htmx.trigger(manageLoader, 'load');
    }
  } else {
    // We're on another page (like brew form) - refresh dropdowns
    const selectElement = document.querySelector(`select[name="${selectName}"]`);

    if (selectElement) {
      // Get the Alpine component instance if it exists
      const formElement = selectElement.closest('[x-data]');
      let dropdownManager = null;

      if (formElement && formElement._x_dataStack) {
        // Access Alpine data stack to get the dropdown manager
        const alpineData = formElement._x_dataStack[0];
        if (alpineData && alpineData.dropdownManager) {
          dropdownManager = alpineData.dropdownManager;
        }
      }

      // Refresh data through the dropdown manager if available
      if (dropdownManager && dropdownManager.invalidateAndRefresh) {
        await dropdownManager.invalidateAndRefresh();
      } else if (window.ArabicaCache && window.ArabicaCache.invalidateAndRefresh) {
        // Fallback to global cache refresh
        await window.ArabicaCache.invalidateAndRefresh();
      }

      // Auto-select the newly created item if we have its rkey
      if (newRkey && selectElement) {
        selectElement.value = newRkey;
        // Trigger change event in case there are listeners
        selectElement.dispatchEvent(new Event('change', { bubbles: true }));
      }
    }
  }
};

/**
 * Shows a modal dialog element with fade-in animation
 * @param {string} dialogId - ID of the dialog element
 */
window.showModal = function (dialogId) {
  const dialog = document.getElementById(dialogId);
  if (dialog && typeof dialog.showModal === "function") {
    dialog.showModal();
  }
};

/**
 * Closes a modal dialog element with fade-out animation
 * @param {string} dialogId - ID of the dialog element
 */
window.closeModal = function (dialogId) {
  const dialog = document.getElementById(dialogId);
  if (dialog && typeof dialog.close === "function") {
    // Add closing class for fade-out animation
    dialog.style.opacity = "0";
    dialog.style.transform = "scale(0.95)";

    // Wait for animation to complete before actually closing
    setTimeout(() => {
      dialog.close();
      // Reset styles for next open
      dialog.style.opacity = "";
      dialog.style.transform = "";
    }, 200);
  }
};

/**
 * Initialize modal handling after HTMX loads modal content
 */
(function () {
  "use strict";

  // Listen for HTMX afterSwap events on the modal container
  function initModalHandling() {
    const modalContainer = document.getElementById("modal-container");
    if (!modalContainer) return;

    // Remove any existing listener to prevent duplicates
    modalContainer.removeEventListener("htmx:afterSwap", handleModalSwap);
    modalContainer.addEventListener("htmx:afterSwap", handleModalSwap);
  }

  function handleModalSwap(evt) {
    // Find the dialog element that was just loaded
    const dialog = evt.target.querySelector("dialog#entity-modal");
    if (dialog && typeof dialog.showModal === "function") {
      // Small delay to ensure DOM is fully settled
      setTimeout(() => {
        dialog.showModal();
      }, 10);
    }
  }

  // Initialize on DOM load
  function initFormHandling() {
    // Handle successful form submissions in entity modals
    document.body.addEventListener('htmx:afterRequest', function(evt) {
      // Check if this was a successful request from a form inside entity-modal
      if (evt.detail.successful && evt.target.tagName === 'FORM') {
        const dialog = evt.target.closest('dialog#entity-modal');
        if (dialog) {
          // Determine entity type from the form's action URL
          const actionUrl = evt.target.getAttribute('hx-post') || evt.target.getAttribute('hx-put');
          let entityType = '';
          let selectName = '';

          if (actionUrl) {
            if (actionUrl.includes('/api/beans')) {
              entityType = 'beans';
              selectName = 'bean_rkey';
            } else if (actionUrl.includes('/api/grinders')) {
              entityType = 'grinders';
              selectName = 'grinder_rkey';
            } else if (actionUrl.includes('/api/brewers')) {
              entityType = 'brewers';
              selectName = 'brewer_rkey';
            } else if (actionUrl.includes('/api/roasters')) {
              entityType = 'roasters';
              selectName = 'roaster_rkey';
            }
          }

          // Parse response to get the new entity's rkey
          let newRkey = null;
          try {
            const responseText = evt.detail.xhr.responseText;
            if (responseText) {
              const responseData = JSON.parse(responseText);
              newRkey = responseData.rkey || responseData.RKey;
            }
          } catch (e) {
            console.warn('Failed to parse entity response:', e);
          }

          // Refresh the appropriate dropdown and auto-select the new item
          if (entityType) {
            window.refreshEntityDropdown(entityType, selectName, newRkey);
          }

          // Close the modal
          window.closeModal('entity-modal');
        }
      }
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", function() {
      initModalHandling();
      initFormHandling();
    });
  } else {
    initModalHandling();
    initFormHandling();
  }
})();
