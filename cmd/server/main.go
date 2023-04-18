package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/masterjk/webrtc-poc/internal"
	"github.com/pion/webrtc/v3"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const (
	iceCandidateServerReflexiveType = "srflx"

	tickerCheckICECandidates = 25 * time.Millisecond
)

var stunServerURLs = []string{"stun:stun.l.google.com:19302"}

type SdpRequest struct {
	SdpOffer string `json:"sdpOffer"`
}

type SdpResponse struct {
	SessionDescription *webrtc.SessionDescription `json:"sessionDescription"`
}

func main() {

	zerolog.TimeFieldFormat = time.RFC3339Nano

	httpPort := flag.Uint("http-port", 8080, "HTTP server port for /sdp api endpoint to faciliate SDP exchange")
	flag.Parse()

	if *httpPort == 0 {
		flag.Usage()
		os.Exit(-1)
	}

	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	api, err := internal.NewWebRtcAPI()
	if err != nil {
		log.Error().Err(err).Msg("Error creating webrtc API")
		os.Exit(-1)
	}

	gin.SetMode(gin.DebugMode)

	router := gin.Default()
	router.Use(cors.Default())
	router.StaticFS("/web", http.Dir("./web/"))
	router.POST("/sdp", func(c *gin.Context) {

		// Read request body
		requestBody, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Error().Err(err).Msg("Error reading request body")
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Validate JSON
		var sdpRequest *SdpRequest
		if err = json.Unmarshal(requestBody, &sdpRequest); err != nil {
			log.Error().Err(err).Msg("Unable to parse JSON")
			c.Writer.WriteHeader(http.StatusBadRequest)
			return
		}

		peerConnection, err := api.NewPeerConnection(webrtc.Configuration{
			ICEServers: []webrtc.ICEServer{{URLs: stunServerURLs}},
		})
		if err != nil {
			log.Error().Err(err).Msg("Error creating peer connection")
			c.Writer.WriteHeader(http.StatusBadRequest)
			return
		}

		worker := internal.NewWorker()

		// Delegate webrtc data connection establishment to seashore worker
		peerConnection.OnDataChannel(func(dc *webrtc.DataChannel) {
			go func(dc *webrtc.DataChannel) {

				switch dc.Label() {
				case "control":
					dc.OnMessage(func(msg webrtc.DataChannelMessage) {
						worker.OnMessage(msg.Data)
					})
				case "payload":
					worker.SetPayloadDataChannel(dc)
				}
			}(dc)
		})

		peerConnection.OnConnectionStateChange(func(state webrtc.PeerConnectionState) {
			log.Debug().Str("status", state.String()).Msg("WebRTC connection status change")
		})

		offer := webrtc.SessionDescription{}
		offer.Type = webrtc.SDPTypeOffer
		offer.SDP = sdpRequest.SdpOffer

		// Set the remote SessionDescription
		err = peerConnection.SetRemoteDescription(offer)
		if err != nil {
			log.Error().Err(err).Msg("Error setting remote description")
			if err2 := peerConnection.Close(); err2 != nil {
				log.Error().Err(err).Msg("Error closing peer connection")
			}
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Create an answer
		answer, err := peerConnection.CreateAnswer(nil)
		if err != nil {
			log.Error().Err(err).Msg("Error creating SDP answer")
			if err2 := peerConnection.Close(); err2 != nil {
				log.Error().Err(err).Msg("Error closing peer connection")
			}
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Create channel that is blocked until ICE Gathering is complete
		startGatherTime := time.Now()
		gatherComplete := webrtc.GatheringCompletePromise(peerConnection)

		// Sets the LocalDescription, and starts our UDP listeners
		err = peerConnection.SetLocalDescription(answer)
		if err != nil {
			log.Error().Err(err).Msg("Error setting local description")
			if err2 := peerConnection.Close(); err2 != nil {
				log.Error().Err(err).Msg("Error closing peer connection")
			}
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		ticker := time.NewTicker(tickerCheckICECandidates)
		discoveryDone := false
		for !discoveryDone {
			select {
			case <-ticker.C:
				if strings.Contains(peerConnection.LocalDescription().SDP, iceCandidateServerReflexiveType) {
					log.Debug().
						Float64("elapsedTime", time.Since(startGatherTime).Seconds()).
						Msg("Found at least one srlfx ICE candidate")
					discoveryDone = true
				}

			case <-gatherComplete:
				log.Debug().
					Float64("elapsedTime", time.Since(startGatherTime).Seconds()).
					Msg("ICE gathering in completed state")
				discoveryDone = true
			}
		}

		// Check if found any server reflexive IP
		if !strings.Contains(peerConnection.LocalDescription().SDP, iceCandidateServerReflexiveType) {
			if err := peerConnection.Close(); err != nil {
				log.Error().Err(err).Msg("Error closing peer connection")
			}

			log.Error().
				Float64("elapsedTimeWebRtcPeerGatherSecs", time.Since(startGatherTime).Seconds()).
				Msg("SDP answer has no server reflexive IP")

			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Debug().
			Float64("elapsedTimeWebRtcPeerGatherSecs", time.Since(startGatherTime).Seconds()).
			Str("sdpAnswer", peerConnection.LocalDescription().SDP).
			Msg("SDP answer created")

		sdpResponse := SdpResponse{
			SessionDescription: peerConnection.LocalDescription(),
		}
		c.JSON(http.StatusOK, sdpResponse)
	})

	http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", *httpPort), router)
}
