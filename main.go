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
import "github.com/hatstand/hodoor/model"
import "github.com/hatstand/hodoor/webpush"
import "github.com/stianeikeland/go-rpio"
import wp "github.com/SherClockHolmes/webpush-go"

var port = flag.Int("port", 8080, "Port to start HTTP server on")
var deviceIndex = flag.Int("device", 2, "Audio device to listen with")
var threshold = flag.Int("threshold", 3000, "Arbitrary threshold for doorbell activation")
var webpushKey = flag.String("key", "", "Private key for sending webpush requests")
var GPIOPin = flag.Int("pin", 18, "GPIO pin to toggle to open door")
var delaySeconds = flag.Int("delay", 5, "Time in seconds to hold door open")

type gpioHandler struct {
  lock sync.Mutex
  pin rpio.Pin
  db *model.Database
}

func GpioHandler(pin rpio.Pin) *gpioHandler {
  db, err := model.OpenDatabase("db")
  if err != nil {
    log.Fatal("Failed to open database: ", err)
  }
  return &gpioHandler{pin:pin, db:db}
}

func (f *gpioHandler) HandleButtonPress() {
  log.Printf("Dash button pressed!")
  f.openDoor()
}

func (f *gpioHandler) openDoor() {
  timer := time.NewTimer(*delaySeconds * time.Second)
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
  f.db.Subscribe(sub)
}

func (f *gpioHandler) handlePing(w http.ResponseWriter, r *http.Request) {
  fmt.Fprintf(w, "Pinging subscribers")
  f.notifySubscribers("Ping!")
}

func (f *gpioHandler) notifySubscribers(message string) error {
  subs, err := f.db.GetSubscriptions()
  if err != nil {
    log.Printf("Failed to fetch subscribers: ", err)
    return err
  }
  for _, sub := range(subs) {
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

func (f *gpioHandler) HandleDoorBell() {
  log.Println("Doorbell handled")
  f.notifySubscribers("DING DONG")
}

func main() {
  flag.Parse()
  runtime.GOMAXPROCS(6)

  err := rpio.Open()
  defer rpio.Close()

  if err != nil {
    log.Fatal(err)
  }

  handler := GpioHandler(rpio.Pin(*GPIOPin))

  go func() {
    err := dash.Listen(handler)
    if err != nil {
      log.Fatal(err)
    }
  }()

  go func() {
    err := doorbell.Listen(*deviceIndex, *threshold, handler)
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
