package doorbell

import (
  "fmt"
  "log"
  "time"

  "github.com/gordonklaus/portaudio"
)

type Handler interface {
  HandleDoorBell()
}

func Listen(deviceIndex int, threshold int, handler Handler) error {
  err := portaudio.Initialize()
  if err != nil {
    return err
  }
  defer portaudio.Terminate()

  devices, err := portaudio.Devices()
  if err != nil {
    return err
  }

  lastPressed := time.Unix(0, 0)
  stream, err := portaudio.OpenStream(portaudio.HighLatencyParameters(devices[deviceIndex], nil, func(in []int16) {
    sum := 0
    for _, sample : = range(in) {
      if sample > 0 {
        sum += int(sample)
      }
    }
    average := sum / len(int)
    if average > threshold && time.Now().Sub(lastPressed) >= time.Second * 10 {
      log.Printf("DING-DONG!")
      lastPressed = time.Now()
      handler.HandleDoorBell()
    }
  })
  if err != nil {
    return err
  }
  defer stream.Close()

  err = stream.Start()
  if err != nil {
    return err
  }

  log.Printf("Listening for doorbell...")
  select{}
}
