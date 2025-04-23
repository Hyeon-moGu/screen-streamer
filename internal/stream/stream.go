package stream

import (
	"fmt"
	"image"
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"rtsp-screen-streamer/internal/encode"
	"rtsp-screen-streamer/pkg/config"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/bluenviron/gortsplib/v4/pkg/format"
	"github.com/pion/rtp"
)

type myHandler struct {
	stream        *gortsplib.ServerStream
	media         *description.Media
	activeClients *int32
	path          string
}

func (h *myHandler) OnConnOpen(ctx *gortsplib.ServerHandlerOnConnOpenCtx) {
	log.Println("[RTSP] client connected from", ctx.Conn.NetConn().RemoteAddr())
}

func (h *myHandler) OnConnClose(ctx *gortsplib.ServerHandlerOnConnCloseCtx) {
	atomic.AddInt32(h.activeClients, -1)
	log.Println("[RTSP] client disconnected, remaining:", atomic.LoadInt32(h.activeClients))
}

func (h *myHandler) OnDescribe(ctx *gortsplib.ServerHandlerOnDescribeCtx) (*base.Response, *gortsplib.ServerStream, error) {
	log.Println("[DEBUG] OnDescribe Path:", ctx.Path)
	if strings.TrimPrefix(ctx.Path, "/") != h.path {
		log.Println("[RTSP] invalid path in Describe:", ctx.Path)
		return &base.Response{StatusCode: base.StatusNotFound}, nil, nil
	}
	return &base.Response{StatusCode: base.StatusOK}, h.stream, nil
}

func (h *myHandler) OnSetup(ctx *gortsplib.ServerHandlerOnSetupCtx) (*base.Response, *gortsplib.ServerStream, error) {
	if strings.TrimPrefix(ctx.Path, "/") != h.path {
		log.Println("[RTSP] OnSetup rejected: invalid path", ctx.Path)
		return &base.Response{StatusCode: base.StatusNotFound}, nil, nil
	}
	return &base.Response{StatusCode: base.StatusOK}, h.stream, nil
}

func (h *myHandler) OnPlay(ctx *gortsplib.ServerHandlerOnPlayCtx) (*base.Response, error) {
	if strings.TrimPrefix(ctx.Path, "/") != h.path {
		log.Println("[RTSP] OnPlay rejected: invalid path", ctx.Path)
		return &base.Response{StatusCode: base.StatusNotFound}, nil
	}
	atomic.AddInt32(h.activeClients, 1)
	log.Println("[RTSP] play requested, active clients:", atomic.LoadInt32(h.activeClients))
	return &base.Response{StatusCode: base.StatusOK}, nil
}

func buildJPEGHeader(offset int, width, height int) []byte {
	h := make([]byte, 8)
	h[0] = 0x00
	h[1] = byte(offset >> 16)
	h[2] = byte(offset >> 8)
	h[3] = byte(offset)
	h[4] = 1
	h[5] = 0x01
	h[6] = byte(width / 8)
	h[7] = byte(height / 8)
	return h
}

func logDebug(debugYn string, format string, v ...any) {
	if debugYn == "Y" {
		log.Printf(format, v...)
	}
}

func RunRTSPServer(cfg *config.Config, capturer <-chan image.Image, encoder *encode.Encoder) {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", cfg.RtspPort), 500*time.Millisecond)
	if err == nil {
		conn.Close()
		log.Fatalf("[ERROR] port %d already in use", cfg.RtspPort)
	}

	mjpeg := &format.MJPEG{}
	media := &description.Media{
		Type:    description.MediaTypeVideo,
		Control: "trackID=0",
		Formats: []format.Format{mjpeg},
	}

	var activeClients int32
	h := &myHandler{
		media:         media,
		activeClients: &activeClients,
		path:          cfg.RtspPath,
	}

	server := &gortsplib.Server{
		RTSPAddress:   fmt.Sprintf(":%d", cfg.RtspPort),
		Handler:       h,
		MaxPacketSize: 1472,
	}
	go func() {
		log.Println("[RTSP] server starting")
		if err := server.Start(); err != nil {
			log.Panicln("[RTSP] failed to start server:", err)
		}
	}()
	time.Sleep(time.Second)

	stream := &gortsplib.ServerStream{
		Server: server,
		Desc: &description.Session{
			Medias: []*description.Media{media},
			Title:  "RTSP Desktop Streamer",
		},
	}
	if err := stream.Initialize(); err != nil {
		log.Panicln("[RTSP] failed to initialize stream:", err)
	}
	h.stream = stream
	url := fmt.Sprintf("rtsp://localhost:%d/%s", cfg.RtspPort, cfg.RtspPath)
	log.Println("[RTSP] ready on", url)

	encoder.Start(capturer)

	go func() {
		var seq uint16
		var ts uint32
		var wasActive bool
		ticker := time.NewTicker(time.Second / time.Duration(cfg.FrameRate))
		defer ticker.Stop()
		for range ticker.C {
			if atomic.LoadInt32(&activeClients) == 0 {
				if wasActive {
					log.Println("[RTSP] no clients, streaming paused")
					wasActive = false
				}
				continue
			}
			if !wasActive {
				log.Println("[RTSP] client detected, streaming started")
				wasActive = true
			}
			select {
			case frame := <-encoder.Chan():
				offset := 0
				for offset < len(frame.Data) {
					remain := len(frame.Data) - offset
					payloadSize := cfg.RtpPayloadMaxSize - 8
					if remain < payloadSize {
						payloadSize = remain
					}
					fragment := frame.Data[offset : offset+payloadSize]
					rtpHeader := buildJPEGHeader(offset, frame.Width, frame.Height)
					packet := &rtp.Packet{
						Header: rtp.Header{
							Version:        2,
							PayloadType:    mjpeg.PayloadType(),
							SequenceNumber: seq,
							Timestamp:      ts,
							SSRC:           12345678,
							Marker:         offset+payloadSize >= len(frame.Data),
						},
						Payload: append(rtpHeader, fragment...),
					}
					if err := stream.WritePacketRTP(media, packet); err != nil {
						log.Println("[RTSP] failed to send packet:", err)
					} else {
						logDebug(cfg.DebugYn, "[RTSP] sent: Seq=%d TS=%d Bytes=%d", seq, ts, payloadSize)
					}
					offset += payloadSize
					seq++
				}
				ts += 90000 / uint32(cfg.FrameRate)
			default:
			}
		}
	}()

	select {}
}
