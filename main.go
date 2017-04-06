package main

import "flag"
import "html/template"
import "log"
import "net/http"
import "strconv"
import "sync"
import "time"

import "github.com/stianeikeland/go-rpio"

const GPIOPin = 18
const DelaySeconds = 5

var port = flag.Int("port", 8080, "Port to start HTTP server on")

type gpioHandler struct {
  lock sync.Mutex
  pin rpio.Pin
}

func GpioHandler(pin rpio.Pin) http.Handler {
  return &gpioHandler{pin:pin}
}

func (f *gpioHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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

func indexHandler(w http.ResponseWriter, r *http.Request) {
  t, err := template.ParseFiles("templates/index.html")
  if err != nil {
    log.Fatal(err)
  }

  t.Execute(w, nil)
}

func main() {
  flag.Parse()

  err := rpio.Open()
  defer rpio.Close()

  if err != nil {
    log.Fatal(err)
  }

  http.Handle("/hodoor", GpioHandler(rpio.Pin(18)))
  http.HandleFunc("/", indexHandler)
  http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
  err = http.ListenAndServe(":" + strconv.Itoa(*port), nil)
  if err != nil {
    log.Print(err)
  }
}
