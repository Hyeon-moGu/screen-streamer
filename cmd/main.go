package main

import (
	"log"
	"rtsp-screen-streamer/internal/capture"
	"rtsp-screen-streamer/internal/encode"
	"rtsp-screen-streamer/internal/stream"
	"rtsp-screen-streamer/pkg/config"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("failed to load configuration: %v", err)
	}

	capturer := capture.NewCapturer(cfg)
	encoder := encode.NewEncoder(cfg)
	stream.RunRTSPServer(cfg, capturer.Chan(), encoder)
}
