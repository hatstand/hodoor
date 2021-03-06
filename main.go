package main

import "context"
import "encoding/json"
import "flag"
import "html/template"
import "io/ioutil"
import "log"
import "net"
import "net/http"
import "os"
import "os/signal"
import "runtime"
import "strconv"
import "sync"
import "time"

import "github.com/hatstand/hodoor/dash"
import "github.com/hatstand/hodoor/doorbell"
import "github.com/hatstand/hodoor/model"
import "github.com/hatstand/hodoor/webpush"
import "github.com/stianeikeland/go-rpio"
import wp "github.com/SherClockHolmes/webpush-go"

var port = flag.Int("port", 8080, "Port to start HTTP server on")
var deviceIndex = flag.Int("device", 2, "Audio device to listen with")
var threshold = flag.Int("threshold", 3000, "Arbitrary threshold for doorbell activation")
var webpushKey = flag.String("key", "", "Private key for sending webpush requests")
var GPIOPin = flag.Int("pin", 18, "GPIO pin to toggle to open door")
var delaySeconds = flag.Duration("delay", 5*time.Second, "Time in seconds to hold door open")
var dashMAC = flag.String("dash", "", "MAC address of dash button")

func mustParseMAC(s string) net.HardwareAddr {
	mac, err := net.ParseMAC(s)
	if err != nil {
		panic(err)
	}
	return mac
}

type AssistantResponse struct {
	Speech      string `json:"speech"`
	DisplayText string `json:"displayText"`
}

type gpioHandler struct {
	lock sync.Mutex
	pin  rpio.Pin
	db   *model.Database
}

func GpioHandler(pin rpio.Pin) *gpioHandler {
	db, err := model.OpenDatabase("db")
	if err != nil {
		log.Fatal("Failed to open database: ", err)
	}
	return &gpioHandler{pin: pin, db: db}
}

func (f *gpioHandler) HandleButtonPress() {
	log.Printf("Dash button pressed!")
	f.openDoor()
}

func (f *gpioHandler) openDoor() {
	timer := time.NewTimer(*delaySeconds)
	go func() {
		f.lock.Lock()
		defer f.lock.Unlock()
		log.Printf("Toggling door on pin %d for %d seconds", f.pin, *delaySeconds)
		f.pin.Output()
		f.pin.High()
		defer f.pin.Low()
		<-timer.C
	}()
}

func (f *gpioHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch path := r.URL.Path; path {
	case "/":
		f.handleRoot(w, r)
	case "/hodoor":
		f.handleOpenDoor(w, r)
	case "/subscribe":
		f.handleSubscribe(w, r)
	case "/ping":
		f.handlePing(w, r)
	default:
		f.handleRoot(w, r)
	}
}

func (f *gpioHandler) handleRoot(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/index.html")
	if err != nil {
		log.Fatal(err)
	}

	t.Execute(w, nil)
}

func (f *gpioHandler) handleOpenDoor(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/hodoor.html")
	if err != nil {
		log.Fatal(err)
	}

	type TemplateOutput struct {
		Pin   rpio.Pin
		Delay int
	}
	output := &TemplateOutput{f.pin, int(delaySeconds.Seconds())}

	t.Execute(w, output)
	f.openDoor()
}

func (f *gpioHandler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	body, _ := ioutil.ReadAll(r.Body)
	sub, err := webpush.SubscriptionFromJSON(body)
	if err != nil {
		log.Printf("Failed to parse subscription: %v", err)
		http.Error(w, "Failed to parse subscription", 400)
		return
	}
	defer r.Body.Close()
	log.Printf("Subscribing user: %v", sub)
	f.db.Subscribe(sub)
}

func (f *gpioHandler) handlePing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp := AssistantResponse{"Opening door", "Opening door"}
	j, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "Failed to serialise JSON", 500)
		return
	}

	err = f.notifySubscribers("Ping!")
	if err != nil {
		http.Error(w, "Failed to notify subscribers", 500)
		return
	}
	w.Write(j)
}

func (f *gpioHandler) notifySubscribers(message string) error {
	subs, err := f.db.GetSubscriptions()
	if err != nil {
		log.Printf("Failed to fetch subscribers: %v", err)
		return err
	}
	for _, sub := range subs {
		go func(sub *wp.Subscription) {
			log.Printf("Sending webpush to endpoint: %v", sub.Endpoint)
			err := webpush.Send([]byte(message), sub, *webpushKey, 60)
			if err != nil {
				log.Printf("Failed to send webpush: %v", err)
			} else {
				log.Printf("Sent webpush successfully")
			}
		}(sub)
	}
	runtime.Gosched()
	return nil
}

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(6)

	err := rpio.Open()
	defer rpio.Close()

	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := GpioHandler(rpio.Pin(*GPIOPin))

	ringCh, err := doorbell.Listen(ctx, *deviceIndex, *threshold)
	if err != nil {
		log.Fatal(err)
	}

	buttonCh, err := dash.Listen(ctx, mustParseMAC(*dashMAC))
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		for {
			select {
			case <-ringCh:
				handler.notifySubscribers("DING DONG!")
			case <-buttonCh:
				handler.openDoor()
			case <-ctx.Done():
				return
			}
		}
	}()

	http.Handle("/hodoor", handler)
	http.Handle("/", handler)
	http.Handle("/subscribe", handler)
	http.Handle("/ping", handler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	srv := &http.Server{Addr: ":" + strconv.Itoa(*port)}
	go func() {
		log.Printf("Starting HTTP Server on port: %d", *port)
		err := srv.ListenAndServe()
		if err != nil {
			// Probably happens during shutdown.
			log.Printf("HTTP Server error: %v", err)
		}
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt)
	signal.Notify(signalCh, os.Kill)

	select {
	case <-ctx.Done():
		timeout, httpCancel := context.WithDeadline(ctx, time.Now().Add(time.Second*5))
		defer httpCancel()
		srv.Shutdown(timeout)
		return
	case <-signalCh:
		cancel()
	}
}
