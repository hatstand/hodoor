package main

import "flag"
import "fmt"
import "html/template"
import "io/ioutil"
import "log"
import "net/http"
import "runtime"
import "strconv"
import "sync"
import "time"

import "github.com/hatstand/hodoor/dash"
import "github.com/hatstand/hodoor/doorbell"
import "github.com/hatstand/hodoor/webpush"
import "github.com/stianeikeland/go-rpio"
import wp "github.com/SherClockHolmes/webpush-go"

const GPIOPin = 18
const DelaySeconds = 5

var port = flag.Int("port", 8080, "Port to start HTTP server on")
var deviceIndex = flag.Int("device", 2, "Audio device to listen with")
var threshold = flag.Int("threshold", 3000, "Arbitrary threshold for doorbell activation")
var webpushKey = flag.String("key", "", "Private key for sending webpush requests")

type gpioHandler struct {
  lock sync.Mutex
  pin rpio.Pin
  subscriptions []*wp.Subscription
}

func GpioHandler(pin rpio.Pin) *gpioHandler {
  return &gpioHandler{pin:pin}
}

func (f *gpioHandler) HandleButtonPress() {
  log.Printf("Dash button pressed!")
  f.openDoor()
}

func (f *gpioHandler) openDoor() {
  timer := time.NewTimer(time.Second * DelaySeconds)
  go func() {
    f.lock.Lock()
    defer f.lock.Unlock()
    log.Printf("Toggling door on pin %d for %d seconds", f.pin, DelaySeconds)
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
    Pin rpio.Pin
    Delay int
  }
  output := &TemplateOutput{f.pin, DelaySeconds}

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
  f.subscriptions = append(f.subscriptions, sub)
}

func (f *gpioHandler) handlePing(w http.ResponseWriter, r *http.Request) {
  for _, sub := range(f.subscriptions) {
    go func(sub *wp.Subscription) {
      log.Printf("Sending webpush to endpoint: %v", sub.Endpoint)
      err := webpush.Send([]byte("Yay! Web Push!"), sub, *webpushKey, 60)
      if err != nil {
        log.Printf("Failed to send webpush: %v", err)
      } else {
        log.Printf("Sent webpush successfully")
      }
    }(sub)
  }
  fmt.Fprintf(w, "Pinging %d subscribers", len(f.subscriptions))
  runtime.Gosched()
}

type doorbellHandler struct{}

func (h *doorbellHandler) HandleDoorBell() {
  log.Println("Doorbell handled")
}

func main() {
  flag.Parse()
  runtime.GOMAXPROCS(6)

  err := rpio.Open()
  defer rpio.Close()

  if err != nil {
    log.Fatal(err)
  }

  handler := GpioHandler(rpio.Pin(18))

  go func() {
    err := dash.Listen(handler)
    if err != nil {
      log.Fatal(err)
    }
  }()

  go func() {
    err := doorbell.Listen(*deviceIndex, *threshold, &doorbellHandler{})
    if err != nil {
      log.Fatal(err)
    }
  }()

  http.Handle("/hodoor", handler)
  http.Handle("/", handler)
  http.Handle("/subscribe", handler)
  http.Handle("/ping", handler)
  http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
  log.Printf("Starting HTTP Server on port: %d", *port)
  go http.ListenAndServe(":" + strconv.Itoa(*port), nil)
  select{}
}
