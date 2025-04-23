package capture

import (
	"image"
	"rtsp-screen-streamer/pkg/config"
	"time"

	"github.com/kbinani/screenshot"
)

type Capturer struct {
	bounds image.Rectangle
	rate   time.Duration
	ch     chan image.Image
}

func NewCapturer(cfg *config.Config) *Capturer {
	numDisplays := screenshot.NumActiveDisplays()
	if cfg.DisplayIndex < 0 || cfg.DisplayIndex >= numDisplays {
		panic("Invalid displayIndex in config. Check your monitor setup.")
	}

	c := &Capturer{
		bounds: screenshot.GetDisplayBounds(cfg.DisplayIndex),
		rate:   time.Second / time.Duration(cfg.FrameRate),
		ch:     make(chan image.Image, 8),
	}
	go c.run()
	return c
}

func (c *Capturer) run() {
	ticker := time.NewTicker(c.rate)
	defer ticker.Stop()
	for range ticker.C {
		img, err := screenshot.CaptureRect(c.bounds)
		if err == nil {
			select {
			case c.ch <- img:
			default:
				<-c.ch
				c.ch <- img
			}
		}
	}
}

func (c *Capturer) Chan() <-chan image.Image {
	return c.ch
}
