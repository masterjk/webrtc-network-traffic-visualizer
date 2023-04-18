package internal

import (
	"github.com/pion/ice/v2"
	"github.com/pion/interceptor"
	"github.com/pion/logging"
	"github.com/pion/webrtc/v3"
)

// NewWebRtcAPI returns a new webrtc API instance
func NewWebRtcAPI() (*webrtc.API, error) {
	mediaEngine := &webrtc.MediaEngine{}
	if err := mediaEngine.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}
	interceptRegistry := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(mediaEngine, interceptRegistry); err != nil {
		return nil, err
	}

	f := logging.NewDefaultLoggerFactory()
	f.DefaultLogLevel.Set(logging.LogLevelError)

	settingsEngine := webrtc.SettingEngine{
		LoggerFactory: f,
	}

	// Disable ICE multicast DNS mode since wave doesn't use it and it leaks UDP sockets
	settingsEngine.SetICEMulticastDNSMode(ice.MulticastDNSModeDisabled)
	api := webrtc.NewAPI(
		webrtc.WithMediaEngine(mediaEngine),
		webrtc.WithInterceptorRegistry(interceptRegistry),
		webrtc.WithSettingEngine(settingsEngine),
	)

	return api, nil
}
