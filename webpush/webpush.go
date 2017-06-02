package webpush

import (
  "encoding/json"
  "log"

  wp "github.com/SherClockHolmes/webpush-go"
)

func SubscriptionFromJSON(data []byte) (*wp.Subscription, error) {
  s := &wp.Subscription{}
  if err := json.Unmarshal(data, s); err != nil {
    return nil, err
  }

  return s, nil
}

func Send(body []byte, sub *wp.Subscription, key string, ttl int) error {
  _, err := wp.SendNotification(body, sub, &wp.Options{
      Subscriber: "mailto:john.maguire@gmail.com",
      TTL: ttl,
      VAPIDPrivateKey: key,
  })
  if err != nil {
    log.Printf("Error sending webpush:\n%s", err)
  }
  return err
}
