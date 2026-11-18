/* Domus Web Push handlers loaded into the generated Workbox service worker. */
self.addEventListener('push', (event) => {
  let payload = {};
  try {
    payload = event.data ? event.data.json() : {};
  } catch {
    payload = { body: event.data ? event.data.text() : '' };
  }

  const title = typeof payload.title === 'string' && payload.title.trim()
    ? payload.title
    : 'Domus';
  const target = typeof payload.url === 'string' && payload.url.startsWith('/') && !payload.url.startsWith('//')
    ? payload.url
    : '/';
  const options = {
    body: typeof payload.body === 'string' ? payload.body : '',
    icon: '/icons/icon-192x192.png',
    badge: '/icons/icon-96x96.png',
    tag: typeof payload.tag === 'string' ? payload.tag : undefined,
    renotify: false,
    data: { url: target, category: payload.category || 'general', payload: payload.data || {} }
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', (event) => {
  event.notification.close();
  const requestedPath = event.notification && event.notification.data
    ? event.notification.data.url
    : '/';
  const safePath = typeof requestedPath === 'string' && requestedPath.startsWith('/') && !requestedPath.startsWith('//')
    ? requestedPath
    : '/';

  event.waitUntil((async () => {
    const target = new URL(safePath, self.location.origin);
    if (target.origin !== self.location.origin) return;
    const clientsList = await self.clients.matchAll({ type: 'window', includeUncontrolled: true });
    for (const client of clientsList) {
      if (client.url.startsWith(self.location.origin) && 'focus' in client) {
        if ('navigate' in client && client.url !== target.href) {
          await client.navigate(target.href);
        }
        return client.focus();
      }
    }
    if (self.clients.openWindow) return self.clients.openWindow(target.href);
  })());
});
