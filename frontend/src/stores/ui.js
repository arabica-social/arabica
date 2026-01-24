import { writable } from 'svelte/store';

/**
 * UI store - manages global UI state like modals, notifications, etc.
 */
function createUIStore() {
  const { subscribe, update } = writable({
    notifications: [],
  });
  
  return {
    subscribe,
    
    /**
     * Show a notification
     * @param {string} message - Notification message
     * @param {string} type - Type: 'success', 'error', 'info'
     * @param {number} duration - Duration in ms (0 = no auto-dismiss)
     */
    notify(message, type = 'info', duration = 5000) {
      const id = Date.now();
      update(state => ({
        ...state,
        notifications: [...state.notifications, { id, message, type }],
      }));
      
      if (duration > 0) {
        setTimeout(() => {
          this.dismissNotification(id);
        }, duration);
      }
      
      return id;
    },
    
    /**
     * Dismiss a notification by ID
     */
    dismissNotification(id) {
      update(state => ({
        ...state,
        notifications: state.notifications.filter(n => n.id !== id),
      }));
    },
  };
}

export const uiStore = createUIStore();
