package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	webpush "github.com/SherClockHolmes/webpush-go"
	"io/ioutil"
)

var exampleJSON = `{"endpoint":"https://fcm.googleapis.com/fcm/send/dYhy8CbUL0s:APA91bFChHz26OclH10Q7Kz_FPhbBU1GL9PITj9QLNzPKYKTLmgWu_McVoRMTIaMkQNO5I7L7QsGEy5dhDCRJbUwzvy-upbRTa3olTnp6R21dPRP7xrjxwMJ_gMyvcrQUsdsLQGomeNg","keys":{"p256dh":"BEwpEUgS_KyW3QGa64RMH07Csw9MZ1rN5xhRj3BnPVXJNux9j8vis_JXALpyWmn3UkPEAaYRdiBfmtDDP_9Tuyg=","auth":"Zkk1ganWA0yn6_0WqZ81Pw=="}}`

var key = flag.String("key", "", "Private VAPID key for sending webpush requests")

func main() {
	flag.Parse()

	s := webpush.Subscription{}
	if err := json.NewDecoder(bytes.NewBufferString(exampleJSON)).Decode(&s); err != nil {
		panic(err)
	}

	r, err := webpush.SendNotification([]byte("Test"), &s, &webpush.Options{
		Subscriber:      "mailto:john.maguire@gmail.com",
		TTL:             60,
		VAPIDPrivateKey: *key,
	})
	if err != nil {
		panic(err)
	}

	out, _ := ioutil.ReadAll(r.Body)
	fmt.Printf("%s", out)
}
