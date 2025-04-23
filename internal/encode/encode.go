package encode

import (
	"bytes"
	"image"
	"image/draw"
	"log"
	"runtime"
	"sync"
	"time"

	"rtsp-screen-streamer/pkg/config"

	"github.com/disintegration/imaging"
	"github.com/nfnt/resize"
)

type EncodedFrame struct {
	Data   []byte
	Width  int
	Height int
}

type Encoder struct {
	cfg   *config.Config
	in    <-chan image.Image
	out   chan EncodedFrame
	bufs  sync.Pool
	bytes sync.Pool
	pool  sync.Pool
}

func NewEncoder(cfg *config.Config) *Encoder {
	e := &Encoder{
		cfg: cfg,
		out: make(chan EncodedFrame, 8),
		bufs: sync.Pool{
			New: func() any { return new(bytes.Buffer) },
		},
		bytes: sync.Pool{
			New: func() any { return make([]byte, 0, 256*1024) },
		},
		pool: sync.Pool{
			New: func() any {
				return image.NewRGBA(image.Rect(0, 0, int(cfg.ResizeWidth), int(cfg.ResizeHeight)))
			},
		},
	}
	return e
}

func (e *Encoder) Start(input <-chan image.Image) {
	e.in = input
	numWorkers := runtime.NumCPU()
	for i := 0; i < numWorkers; i++ {
		go e.worker()
	}
}

func (e *Encoder) worker() {
	for img := range e.in {
		start := time.Now()
		resized := resize.Resize(e.cfg.ResizeWidth, e.cfg.ResizeHeight, img, resize.NearestNeighbor)
		temp := e.pool.Get().(*image.RGBA)
		draw.Draw(temp, temp.Bounds(), resized, image.Point{}, draw.Src)

		buf := e.bufs.Get().(*bytes.Buffer)
		buf.Reset()
		err := imaging.Encode(buf, temp, imaging.JPEG, imaging.JPEGQuality(e.cfg.JpegQuality))
		if err != nil {
			log.Printf("JPEG encoding error: %v", err)
			e.bufs.Put(buf)
			e.pool.Put(temp)
			continue
		}

		out := e.bytes.Get().([]byte)[:0]
		out = append(out, buf.Bytes()...)
		e.bufs.Put(buf)
		e.pool.Put(temp)

		select {
		case e.out <- EncodedFrame{Data: out, Width: int(e.cfg.ResizeWidth), Height: int(e.cfg.ResizeHeight)}:
		default:
			<-e.out
			e.out <- EncodedFrame{Data: out, Width: int(e.cfg.ResizeWidth), Height: int(e.cfg.ResizeHeight)}
		}

		if elapsed := time.Since(start); elapsed > (time.Second / time.Duration(e.cfg.FrameRate) * 2) {
			log.Printf("[WARN] encoding delay detected: %v", elapsed)
		}
	}
}

func (e *Encoder) Chan() <-chan EncodedFrame {
	return e.out
}
