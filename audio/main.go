package main

import (
  "flag"
  "fmt"
  "log"
  "time"

  "github.com/gordonklaus/portaudio"
)

var threshold = flag.Int("threshold", 3000, "Arbitrary threshold for doorbell activation")
var deviceIndex = flag.Int("device", 2, "Index of device to record with")
var help = flag.Bool("help", false, "Help!")

func main() {
  flag.Parse()

  err := portaudio.Initialize()
  if err != nil {
    panic(err)
  }
  defer portaudio.Terminate()
  devices, err := portaudio.Devices()
  if err != nil {
    panic(err)
  }

  if *help {
    fmt.Println("Available devices:")
    for i, device := range devices {
      fmt.Printf("%d: %v\n", i, device.Name)
    }
    return
  }

  lastPressed := time.Unix(0, 0)
  stream, err := portaudio.OpenStream(portaudio.HighLatencyParameters(devices[2], nil), func(in []int16, timeInfo portaudio.StreamCallbackTimeInfo) {
    sum := 0
    for _, sample := range(in) {
      if sample > 0 {
        sum += int(sample)
      }
    }
    average := sum / len(in)
    if average > *threshold && time.Now().Sub(lastPressed) >= time.Second * 10 {
      log.Printf("DING-DONG! Doorbell pressed (%v)\n", average)
      lastPressed = time.Now()
    }
  })
  defer stream.Close()
  if err != nil {
    panic(err)
  }

  err = stream.Start()
  if err != nil {
    panic(err)
  }

  log.Printf("Listening...")
  select{}
}
