package main

/*
	Sliver Implant Framework
	Copyright (C) 2019  Bishop Fox

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
	"crypto/rand"
	"encoding/binary"
	insecureRand "math/rand"
	"time"

	"github.com/bishopfox/sliver/client/cli"
)

// Attempt to seed insecure rand with secure rand, but we really
// don't care that much if it fails since it's insecure anyways
func init() {
	buf := make([]byte, 8)
	_, err := rand.Read(buf)
	if err != nil {
		insecureRand.Seed(int64(time.Now().Unix()))
	} else {
		insecureRand.Seed(int64(binary.LittleEndian.Uint64(buf)))
	}
}

func main() {
	cli.Execute()
}
