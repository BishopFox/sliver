//go:build !s390x && !ppc64le && !darwin && !windows && (linux || freebsd || openbsd || netbsd)

package screenshot

import (
	"fmt"
	"github.com/gen2brain/shm"
	"github.com/jezek/xgb"
	mshm "github.com/jezek/xgb/shm"
	"github.com/jezek/xgb/xinerama"
	"github.com/jezek/xgb/xproto"
	"image"
	"image/color"
)

func captureXinerama(x, y, width, height int) (img *image.RGBA, e error) {
	defer func() {
		err := recover()
		if err != nil {
			img = nil
			e = fmt.Errorf("%v", err)
		}
	}()
	c, err := xgb.NewConn()
	if err != nil {
		return nil, err
	}
	defer c.Close()

	err = xinerama.Init(c)
	if err != nil {
		return nil, err
	}

	reply, err := xinerama.QueryScreens(c).Reply()
	if err != nil {
		return nil, err
	}

	primary := reply.ScreenInfo[0]
	x0 := int(primary.XOrg)
	y0 := int(primary.YOrg)

	useShm := true
	err = mshm.Init(c)
	if err != nil {
		useShm = false
	}

	screen := xproto.Setup(c).DefaultScreen(c)
	wholeScreenBounds := image.Rect(0, 0, int(screen.WidthInPixels), int(screen.HeightInPixels))
	targetBounds := image.Rect(x+x0, y+y0, x+x0+width, y+y0+height)
	intersect := wholeScreenBounds.Intersect(targetBounds)

	rect := image.Rect(0, 0, width, height)
	img, err = createImage(rect)
	if err != nil {
		return nil, err
	}

	// Paint with opaque black
	index := 0
	for iy := 0; iy < height; iy++ {
		j := index
		for ix := 0; ix < width; ix++ {
			img.Pix[j+3] = 255
			j += 4
		}
		index += img.Stride
	}

	if !intersect.Empty() {
		var data []byte

		if useShm {
			shmSize := intersect.Dx() * intersect.Dy() * 4
			shmId, err := shm.Get(shm.IPC_PRIVATE, shmSize, shm.IPC_CREAT|0777)
			if err != nil {
				return nil, err
			}

			seg, err := mshm.NewSegId(c)
			if err != nil {
				return nil, err
			}

			data, err = shm.At(shmId, 0, 0)
			if err != nil {
				return nil, err
			}

			mshm.Attach(c, seg, uint32(shmId), false)

			defer mshm.Detach(c, seg)
			defer func() {
				_ = shm.Rm(shmId)
			}()
			defer func() {
				_ = shm.Dt(data)
			}()

			_, err = mshm.GetImage(c, xproto.Drawable(screen.Root),
				int16(intersect.Min.X), int16(intersect.Min.Y),
				uint16(intersect.Dx()), uint16(intersect.Dy()), 0xffffffff,
				byte(xproto.ImageFormatZPixmap), seg, 0).Reply()
			if err != nil {
				return nil, err
			}
		} else {
			xImg, err := xproto.GetImage(c, xproto.ImageFormatZPixmap, xproto.Drawable(screen.Root),
				int16(intersect.Min.X), int16(intersect.Min.Y),
				uint16(intersect.Dx()), uint16(intersect.Dy()), 0xffffffff).Reply()
			if err != nil {
				return nil, err
			}

			data = xImg.Data
		}

		// BitBlt by hand
		offset := 0
		for iy := intersect.Min.Y; iy < intersect.Max.Y; iy++ {
			for ix := intersect.Min.X; ix < intersect.Max.X; ix++ {
				r := data[offset+2]
				g := data[offset+1]
				b := data[offset]
				img.SetRGBA(ix-(x+x0), iy-(y+y0), color.RGBA{r, g, b, 255})
				offset += 4
			}
		}
	}

	return img, e
}
