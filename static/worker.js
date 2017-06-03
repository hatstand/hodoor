self.addEventListener('push', function(event) {
  console.log('[Service Worker] Push Received');
  console.log(`[Service Worker] Push had this data: "${event.data.text()}"`);

  const title = 'Hodoor!';
  const options = {
    body: 'Yay it works',
    icon: '/static/hodor.png',
    badge: '/static/hodor.png',
    actions: [{
      action: 'open',
      title: 'Open'
    }, {
      action: 'nope',
      title: 'Deny'
    }]
  };

  event.waitUntil(self.registration.showNotification(title, options));
});

self.addEventListener('notificationclick', function(event) {
  console.log('[Service Worker] Notification click received');
  event.notification.close();

  if (event.action === 'open') {
    fetch('/hodoor', {
      method: 'POST'
    });
  } else {
    console.log('Denied');
  }
});
