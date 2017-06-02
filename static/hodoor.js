var swRegistration;

function urlB64ToUint8Array(base64String) {
  const padding = '='.repeat((4 - base64String.length % 4) % 4);
  const base64 = (base64String + padding)
    .replace(/\-/g, '+')
    .replace(/_/g, '/');

  const rawData = window.atob(base64);
  const outputArray = new Uint8Array(rawData.length);

  for (let i = 0; i < rawData.length; ++i) {
    outputArray[i] = rawData.charCodeAt(i);
  }
  return outputArray;
}

function registerServiceWorker() {
  return navigator.serviceWorker.register('/static/worker.js')
      .then(function(registration) {
        swRegistration = registration;
        console.log('Service worker successfully registered');
        registration.pushManager.getSubscription()
        .then(function(subscription) {
          console.log('Subscription: ', subscription);
          updateSubscriptionOnServer(subscription);
        });
      })
      .catch(function(err) {
        console.error('Unable to register service worker: ', err);
      });
}

function askPermission() {
  return new Promise(function(resolve, reject) {
    const permissionResult = Notification.requestPermission(function(result) {
      resolve(result);
    });

    if (permissionResult) {
      permissionResult.then(resolve, reject);
    }
  })
  .then(function(permissionResult) {
    if (permissionResult !== 'granted') {
      throw new Error('Permission not granted :-(');
    }
  });
}

function initialiseUI() {
  var button = document.querySelector('#subscribe');
  button.addEventListener('click', function() {
    button.disabled = true;
    subscribeUser();
  });
}

function subscribeUser() {
  const applicationServerKey = urlB64ToUint8Array(
      'BAwwUxAyu2GYT_du11uv6DEZlYP2xA7N3Nr_vXxhNhw539hiYJNdUIUfJA110oRY432E3DHRW8h9-8Zzuojl_mQ');
  swRegistration.pushManager.subscribe({
    userVisibleOnly: true,
    applicationServerKey: applicationServerKey
  })
  .then(function(subscription) {
    console.log('User is subscribed');
    updateSubscriptionOnServer(subscription);
  })
  .catch(function(err) {
    console.log('Failed to subscribe user: ', err);
  });
}

function updateSubscriptionOnServer(subscription) {
  const subscriptionJson = document.querySelector('.js-subscription-json');
  const subscriptionDetails = document.querySelector('.js-subscription-details');

  if (subscription) {
    var json = JSON.stringify(subscription);
    subscriptionJson.textContent = json;
    fetch('/subscribe', {
      method: 'POST',
      body: json
    });
  }
}

registerServiceWorker();
askPermission();
initialiseUI();
