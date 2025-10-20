//go:build !s390x && !ppc64le && !darwin && !windows && !freebsd && (linux || openbsd || netbsd)

package screenshot

import (
	"fmt"
	"github.com/godbus/dbus/v5"
	"image"
	"image/draw"
	"image/png"
	"net/url"
	"os"
	"sync/atomic"
)

var gTokenCounter uint64 = 0

func captureDbus(x, y, width, height int) (img *image.RGBA, e error) {
	c, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, fmt.Errorf("dbus.SessionBus() failed: %v", err)
	}
	defer func(c *dbus.Conn) {
		err := c.Close()
		if err != nil {
			e = err
		}
	}(c)
	token := atomic.AddUint64(&gTokenCounter, 1)
	options := map[string]dbus.Variant{
		"modal":        dbus.MakeVariant(false),
		"interactive":  dbus.MakeVariant(false),
		"handle_token": dbus.MakeVariant(token),
	}
	obj := c.Object("org.freedesktop.portal.Desktop", dbus.ObjectPath("/org/freedesktop/portal/desktop"))
	call := obj.Call("org.freedesktop.portal.Screenshot.Screenshot", 0, "", options)
	var path dbus.ObjectPath
	err = call.Store(&path)
	if err != nil {
		return nil, fmt.Errorf("dbus.Store() failed: %v", err)
	}
	ch := make(chan *dbus.Message)
	c.Eavesdrop(ch)
	for msg := range ch {
		o, ok := msg.Headers[dbus.FieldPath]
		if !ok {
			continue
		}
		s, ok := o.Value().(dbus.ObjectPath)
		if !ok {
			return nil, fmt.Errorf("dbus.FieldPath value does't have ObjectPath type")
		}
		if s != path {
			continue
		}
		for _, body := range msg.Body {
			v, ok := body.(map[string]dbus.Variant)
			if !ok {
				continue
			}
			uri, ok := v["uri"]
			if !ok {
				continue
			}
			path, ok := uri.Value().(string)
			if !ok {
				return nil, fmt.Errorf("uri is not a string")
			}
			fpath, err := url.Parse(path)
			if err != nil {
				return nil, fmt.Errorf("url.Parse(%v) failed: %v", path, err)
			}
			if fpath.Scheme != "file" {
				return nil, fmt.Errorf("uri is not a file path")
			}
			file, err := os.Open(fpath.Path)
			if err != nil {
				return nil, fmt.Errorf("os.Open(%s) failed: %v", path, err)
			}
			defer func(file *os.File) {
				_ = file.Close()
				_ = os.Remove(fpath.Path)
			}(file)
			img, err := png.Decode(file)
			if err != nil {
				return nil, fmt.Errorf("png.Decode(%s) failed: %v", path, err)
			}
			canvas, err := createImage(image.Rect(0, 0, width, height))
			if err != nil {
				return nil, fmt.Errorf("createImage(%v) failed: %v", path, err)
			}
			draw.Draw(canvas, image.Rect(0, 0, width, height), img, image.Point{x, y}, draw.Src)
			return canvas, e
		}
	}
	return nil, fmt.Errorf("dbus.Message doesn't contain uri")
}
