package doorbell

import (
	"context"
	"log"
	"time"

	"github.com/gordonklaus/portaudio"
)

// processSamples returns true if a sound above threshold is contained within and
// it has been less than 10 seconds since the last trigger.
func processSamples(samples []int16, threshold int, lastPressed time.Time) bool {
	sum := 0
	for _, sample := range samples {
		if sample > 0 {
			sum += int(sample)
		}
	}
	average := sum / len(samples)
	if average > threshold && time.Now().Sub(lastPressed) >= time.Second*10 {
		return true
	}
	return false
}

func Listen(ctx context.Context, deviceIndex int, threshold int) (<-chan interface{}, error) {
	err := portaudio.Initialize()
	if err != nil {
		return nil, err
	}
	defer portaudio.Terminate()

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}

	sampleCh := make(chan []int16, 10)
	stream, err := portaudio.OpenStream(portaudio.HighLatencyParameters(devices[deviceIndex], nil), func(in []int16) {
		sampleCh <- in
	})
	if err != nil {
		return nil, err
	}

	err = stream.Start()
	if err != nil {
		return nil, err
	}

	go func() {
		defer stream.Close()
		defer close(sampleCh)
		log.Printf("Listening for doorbell...")
		select {
		case <-ctx.Done():
			return
		}
	}()

	outputCh := make(chan interface{})
	go func() {
		defer close(outputCh)
		lastPressed := time.Unix(0, 0)
		for {
			select {
			case a := <-sampleCh:
				if processSamples(a, threshold, lastPressed) {
					lastPressed = time.Now()
					log.Printf("DING-DONG!")
					outputCh <- nil
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return outputCh, nil
}
