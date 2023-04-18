package internal

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	pion "github.com/pion/webrtc/v3"
	"github.com/rs/zerolog/log"
	rate "golang.org/x/time/rate"
)

type Command struct {
	Action          string `json:"action"`
	RateLimit       int    `json:"limit"`
	RateLimitBurst  int    `json:"burst"`
	BytesPerMessage int    `json:"bytesPerMessage"`
}

type Payload struct {
	Counter int    `json:"counter"`
	Data    []byte `json:"data"`
}

type Worker struct {
	dataChannelPaylod *pion.DataChannel
	running           atomic.Bool
	rateLimiter       *rate.Limiter

	counter int
}

func NewWorker() *Worker {
	return &Worker{
		dataChannelPaylod: nil,
		running:           atomic.Bool{},
		rateLimiter:       nil,
	}
}

func (w *Worker) Start(bytesPerMessage int) {

	w.counter = 0

	data := make([]byte, bytesPerMessage)
	for i := 0; i < bytesPerMessage; i++ {
		data[i] = 0
	}

	log.Info().Int("bytesPerMessage", bytesPerMessage).Msg("Starting worker")

	for w.running.Load() {
		payload := &Payload{
			Counter: w.counter,
			Data:    data,
		}
		w.counter++

		if buf, err := json.Marshal(payload); err == nil {
			if err2 := w.dataChannelPaylod.Send(buf); err2 != nil {
				log.Error().Err(err2).Msg("Error sending payload, stopping worker")
				w.running.Store(false)
			}
		} else {
			log.Error().Err(err).Msg("Error marshalling payload")
		}

		start := time.Now()
		if err := w.rateLimiter.Wait(context.Background()); err != nil {
			log.Error().Err(err).Msg("Error sleeping")
		}
		log.Trace().Dur("elapsed", time.Since(start)).Msg("Finished sleeping")
	}
}

func (w *Worker) SetPayloadDataChannel(dataChannel *pion.DataChannel) {
	w.dataChannelPaylod = dataChannel
}

func (w *Worker) OnMessage(buf []byte) {
	var command *Command
	if err := json.Unmarshal(buf, &command); err == nil {
		switch command.Action {
		case "start":
			if w.dataChannelPaylod != nil {
				w.rateLimiter = rate.NewLimiter(rate.Limit(command.RateLimit), command.RateLimitBurst)
				w.running.Store(true)
				go w.Start(command.BytesPerMessage)
			} else {
				log.Error().Msg("Data channel payload not set")
			}
		}
	} else {
		log.Error().Err(err).Str("buf", string(buf)).Msg("Error unmarshalling command")
	}
}
