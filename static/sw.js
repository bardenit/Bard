// Minimal service worker — required for desktop PWA install prompt.
// No caching: all requests go straight to the network (app runs locally).
self.addEventListener('install', () => self.skipWaiting());
self.addEventListener('activate', () => self.clients.claim());
