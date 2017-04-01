package main

import "io"
import "log"
import "net/http"
import "time"

import "github.com/stianeikeland/go-rpio"

type gpioHandler struct {
  pin rpio.Pin
}

func GpioHandler(pin rpio.Pin) http.Handler {
  return &gpioHandler{pin}
}

func (f *gpioHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
  io.WriteString(w, "hello, gpio!\n")

  timer := time.NewTimer(time.Second * 5)
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
  log.Fatal(http.ListenAndServe(":8080", nil))
}
