// hausbuch's service worker: it shows the daily summary and opens the app when it is chosen. It caches nothing.
self.addEventListener('push', (event) => {
  const summary = event.data ? event.data.json() : { title: 'hausbuch', body: '' };
  event.waitUntil(self.registration.showNotification(summary.title, {
    body: summary.body,
    icon: '/static/icon-192.png',
    tag: 'summary',
  }));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  event.waitUntil(self.clients.matchAll({ type: 'window' }).then((windows) => {
    const open = windows.find((w) => 'focus' in w);
    if (open) {
      return open.navigate('/').then((w) => (w || open).focus());
    }
    return self.clients.openWindow('/');
  }));
});
