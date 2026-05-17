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
  // On the manage page, the manage partial is HTMX-driven — just trigger a refresh.
  const manageLoader = document.querySelector('[hx-get="/api/manage"]');
  if (manageLoader) {
    document.body.dispatchEvent(new CustomEvent('refreshManage'));
    return;
  }

  // Otherwise (e.g. brew form) refresh the dropdown cache + repopulate.
  // The brew-form petite-vue scope owns its dropdownManager but listens
  // for a `refresh-dropdowns` event to invalidate it; we also do a cache
  // refresh ourselves so any non-scope select stays in sync.
  document.body.dispatchEvent(new CustomEvent('refresh-dropdowns', { bubbles: true }));

  if (window.AppCache) {
    const freshData = await window.AppCache.invalidateAndRefresh();
    if (freshData && window.createDropdownManager) {
      const tempManager = window.createDropdownManager();
      tempManager.applyData(freshData);
      tempManager.populateDropdowns();
    }
  }

  // Auto-select the newly created item if we have its rkey.
  const selectElement = document.querySelector(`select[name="${selectName}"]`);
  if (newRkey && selectElement) {
    setTimeout(() => {
      selectElement.value = newRkey;
      selectElement.dispatchEvent(new Event('change', { bubbles: true }));
    }, 50);
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

  // Listen for HTMX afterSwap events on document.body
  // In HTMX 2.x, afterSwap fires on the source element (the button),
  // not the target, so we listen on body to catch all swaps.
  function initModalHandling() {
    document.body.removeEventListener("htmx:afterSwap", handleModalSwap);
    document.body.addEventListener("htmx:afterSwap", handleModalSwap);
  }

  function handleModalSwap(evt) {
    // Check if the swap target is the modal container
    const modalContainer = document.getElementById("modal-container");
    if (!modalContainer) return;

    // evt.detail.target is the element content was swapped into
    const swapTarget = evt.detail?.target || evt.target;
    if (swapTarget !== modalContainer && !modalContainer.contains(swapTarget)) return;

    // Find the dialog element that was just loaded
    const dialog = modalContainer.querySelector("dialog#entity-modal");
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
