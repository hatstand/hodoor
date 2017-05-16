if (Notification.permission == "granted") {
  sendHello();
} else {
  Notification.requestPermission().then(function(result) {
    if (result === "granted") {
      sendHello();
    }
  });
}

function sendHello() {
  var notification = new Notification("Yo!", {
      body: "Hello there!",
      icon: "/static/hodor.png",
  });
  setTimeout(notification.close.bind(notification), 5000);
}
