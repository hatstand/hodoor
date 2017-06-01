self.addEventListener('push', function(event) {
  console.log('[Service Worker] Push Received');
  console.log(`[Service Worker] Push had this data: "${event.data.text()}"`);

  const title = 'Hodoor!';
  const options = {
    body: 'Yay it works',
    icon: '/static/hodor.png',
    badge: '/static/hodor.png'
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', function(event) {
  console.log('[Service Worker] Notification click received');
  event.notification.close();

  fetch('/hodoor', {
    method: 'POST'
  });
});
