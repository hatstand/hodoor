package main

import "html/template"
import "log"
import "net/http"
import "sync"
import "time"

import "github.com/stianeikeland/go-rpio"

const GPIOPin = 18
const DelaySeconds = 5

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

func main() {
  err := rpio.Open()
  defer rpio.Close()

  if err != nil {
    log.Fatal(err)
  }

  http.Handle("/hodoor", GpioHandler(rpio.Pin(18)))
  http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
  log.Fatal(http.ListenAndServe(":8080", nil))
}
