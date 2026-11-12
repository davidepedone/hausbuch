// Turns notifications on and off for this device, from the account page's notifications section.
(() => {
  const box = document.querySelector('[data-notifications]');
  if (!box) {
    return;
  }
  const status = box.querySelector('[data-push-status]');
  const button = box.querySelector('[data-push-button]');
  const error = box.querySelector('[data-push-error]');
  if (!('serviceWorker' in navigator) || !('PushManager' in window) || !('Notification' in window)) {
    box.querySelector('[data-push-unsupported]').hidden = false;
    return;
  }
  const known = JSON.parse(box.dataset.endpoints);
  let subscription = null;

  const isOn = () => subscription !== null && known.includes(subscription.endpoint);
  const show = () => {
    status.textContent = isOn() ? 'Notifications are on for this device.' : 'Notifications are off for this device.';
    button.textContent = isOn() ? 'Turn off notifications on this device' : 'Turn on notifications on this device';
    status.hidden = false;
    button.hidden = false;
  };
  const fail = (message) => {
    error.textContent = message;
    error.hidden = false;
  };
  const post = async (url, body) => {
    const response = await fetch(url, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });
    if (!response.ok) {
      throw new Error(`${url}: ${response.status}`);
    }
  };
  // The VAPID key comes base64url-encoded; subscribe wants its bytes.
  const keyBytes = () => {
    const b64 = box.dataset.key.replace(/-/g, '+').replace(/_/g, '/');
    return Uint8Array.from(atob(b64 + '='.repeat((4 - b64.length % 4) % 4)), (c) => c.charCodeAt(0));
  };

  navigator.serviceWorker.register('/sw.js')
    .then((registration) => registration.pushManager.getSubscription())
    .then((current) => {
      subscription = current;
      show();
    })
    .catch(() => fail("Notifications aren't available in this browser right now."));

  button.addEventListener('click', async () => {
    error.hidden = true;
    button.disabled = true;
    try {
      const registration = await navigator.serviceWorker.ready;
      if (isOn()) {
        await post('/account/notifications/unsubscribe', { endpoint: subscription.endpoint });
        known.splice(known.indexOf(subscription.endpoint), 1);
        await subscription.unsubscribe();
        subscription = null;
      } else if (await Notification.requestPermission() !== 'granted') {
        fail("The browser didn't allow notifications. Allow them for this site in its settings, then try again.");
      } else {
        subscription = await registration.pushManager.subscribe({
          userVisibleOnly: true,
          applicationServerKey: keyBytes(),
        });
        await post('/account/notifications/subscribe', subscription.toJSON());
        known.push(subscription.endpoint);
      }
      show();
    } catch {
      fail("Notifications couldn't be changed. Try again.");
    } finally {
      button.disabled = false;
    }
  });
})();
