# mjpeg

[![Build Status](https://travis-ci.org/icza/mjpeg.svg?branch=master)](https://travis-ci.org/icza/mjpeg)
[![GoDoc](https://godoc.org/github.com/icza/mjpeg?status.svg)](https://godoc.org/github.com/icza/mjpeg)
[![Go Report Card](https://goreportcard.com/badge/github.com/icza/mjpeg)](https://goreportcard.com/report/github.com/icza/mjpeg)

MJPEG video writer implementation in Go.

## Examples

Let's see an example how to turn the JPEG files `1.jpg`, `2.jpg`, ..., `10.jpg` into a movie file:

    checkErr := func(err error) {
        if err != nil {
            panic(err)
        }
    }

    // Video size: 200x100 pixels, FPS: 2
    aw, err := mjpeg.New("test.avi", 200, 100, 2)
    checkErr(err)

    // Create a movie from images: 1.jpg, 2.jpg, ..., 10.jpg
    for i := 1; i <= 10; i++ {
        data, err := ioutil.ReadFile(fmt.Sprintf("%d.jpg", i))
        checkErr(err)
        checkErr(aw.AddFrame(data))
    }

    checkErr(aw.Close())

Example to add an `image.Image` as a frame to the video:

    aw, err := mjpeg.New("test.avi", 200, 100, 2)
    checkErr(err)

    var img image.Image
    // Acquire / initialize image, e.g.:
    // img = image.NewRGBA(image.Rect(0, 0, 200, 100))

    buf := &bytes.Buffer{}
    checkErr(jpeg.Encode(buf, img, nil))
    checkErr(aw.AddFrame(buf.Bytes()))

    checkErr(aw.Close())
