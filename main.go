package main

import "html/template"
import "log"
import "net/http"
import "time"

import "github.com/stianeikeland/go-rpio"

const GPIOPin = 18
const DelaySeconds = 5

type gpioHandler struct {
  pin rpio.Pin
}

func GpioHandler(pin rpio.Pin) http.Handler {
  return &gpioHandler{pin}
}

func (f *gpioHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  t, err := template.ParseFiles("templates/hodoor.html")
  if err != nil {
    log.Fatal(err)
  }

  type TemplateOutput struct {
    Pin int
    Delay int
  }
  output := &TemplateOutput{GPIOPin, DelaySeconds}

  t.Execute(w, output)

  timer := time.NewTimer(time.Second * DelaySeconds)
  go func() {
    f.pin.Output()
    f.pin.High()
    <-timer.C
    f.pin.Low()
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
