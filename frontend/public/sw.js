self.addEventListener('push', (event) => {
  let payload = {
    title: 'World Cup Stakes',
    body: 'There is an update.',
    url: '/',
    type: 'update',
  }

  if (event.data) {
    try {
      payload = { ...payload, ...event.data.json() }
    } catch (_) {
      payload.body = event.data.text()
    }
  }

  event.waitUntil(
    self.registration.showNotification(payload.title, {
      body: payload.body,
      data: { url: payload.url || '/', type: payload.type },
    }),
  )
})

self.addEventListener('notificationclick', (event) => {
  event.notification.close()
  const url = new URL(event.notification.data?.url || '/', self.location.origin).href

  event.waitUntil(
    clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      for (const client of clientList) {
        if (client.url === url && 'focus' in client) {
          return client.focus()
        }
      }
      if (clients.openWindow) {
        return clients.openWindow(url)
      }
    }),
  )
})
