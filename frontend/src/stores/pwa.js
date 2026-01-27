import { writable } from 'svelte/store';

// Track online/offline status
export const isOnline = writable(
  typeof navigator !== 'undefined' ? navigator.onLine : true
);

// Track service worker update availability
export const updateAvailable = writable(false);

// Initialize online/offline detection
if (typeof window !== 'undefined') {
  window.addEventListener('online', () => isOnline.set(true));
  window.addEventListener('offline', () => isOnline.set(false));

  // Listen for service worker updates
  window.addEventListener('sw-update-available', () => {
    console.log('[PWA] Update available');
    updateAvailable.set(true);
  });
}

// Trigger service worker update
export function updateServiceWorker() {
  if ('serviceWorker' in navigator) {
    navigator.serviceWorker.getRegistration().then((registration) => {
      if (registration) {
        registration.unregister().then(() => {
          window.location.reload();
        });
      }
    });
  }
}

// Request for notification permission (future use for push notifications)
export function requestNotificationPermission() {
  if ('Notification' in window && 'serviceWorker' in navigator) {
    if (Notification.permission === 'granted') {
      return Promise.resolve();
    }
    if (Notification.permission !== 'denied') {
      return Notification.requestPermission();
    }
  }
  return Promise.reject('Notifications not supported');
}
