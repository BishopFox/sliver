/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2023 WireGuard LLC. All Rights Reserved.
 */

package tun

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
)

type NativeTun struct {
	name      string // "/net/ipifc/2"
	ctlFile   *os.File
	dataFile  *os.File
	events    chan Event
	errors    chan error
	closeOnce sync.Once
}

func CreateTUN(_ string, mtu int) (Device, error) {
	ctl, err := os.OpenFile("/net/ipifc/clone", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	nbuf := make([]byte, 5)
	n, err := ctl.Read(nbuf)
	if err != nil {
		ctl.Close()
		return nil, fmt.Errorf("error reading from clone file: %w", err)
	}
	ifn, err := strconv.Atoi(strings.TrimSpace(string(nbuf[:n])))
	if err != nil {
		ctl.Close()
		return nil, fmt.Errorf("error converting clone result %q to int: %w", nbuf[:n], err)
	}

	if _, err := fmt.Fprintf(ctl, "bind pkt\n"); err != nil {
		ctl.Close()
		return nil, fmt.Errorf("error binding to pkt: %w", err)
	}
	if mtu > 0 {
		if _, err := fmt.Fprintf(ctl, "mtu %d\n", mtu); err != nil {
			ctl.Close()
			return nil, fmt.Errorf("error setting MTU: %w", err)
		}
	}

	dataFile, err := os.OpenFile(fmt.Sprintf("/net/ipifc/%d/data", ifn), os.O_RDWR, 0)
	if err != nil {
		ctl.Close()
		return nil, err
	}

	tun := &NativeTun{
		ctlFile:  ctl,
		dataFile: dataFile,
		name:     fmt.Sprintf("/net/ipifc/%d", ifn),
		events:   make(chan Event, 10),
		errors:   make(chan error, 5),
	}
	tun.events <- EventUp

	return tun, nil
}

func (tun *NativeTun) Name() (string, error) {
	return tun.name, nil
}

func (tun *NativeTun) File() *os.File {
	return tun.ctlFile
}

func (tun *NativeTun) Events() <-chan Event {
	return tun.events
}

func (tun *NativeTun) Read(bufs [][]byte, sizes []int, offset int) (int, error) {
	select {
	case err := <-tun.errors:
		return 0, err
	default:
		n, err := tun.dataFile.Read(bufs[0][offset:])
		if n == 1 && bufs[0][offset] == 0 {
			// EOF
			err = io.EOF
			n = 0
		}
		sizes[0] = n
		return 1, err
	}
}

func (tun *NativeTun) Write(bufs [][]byte, offset int) (int, error) {
	for i, buf := range bufs {
		if _, err := tun.dataFile.Write(buf[offset:]); err != nil {
			return i, err
		}
	}
	return len(bufs), nil
}

func (tun *NativeTun) Close() error {
	var err1, err2 error
	tun.closeOnce.Do(func() {
		_, err1 := fmt.Fprintf(tun.ctlFile, "unbind\n")
		if err := tun.ctlFile.Close(); err != nil && err1 == nil {
			err1 = err
		}
		err2 = tun.dataFile.Close()
	})
	if err1 != nil {
		return err1
	}
	return err2
}

func (tun *NativeTun) MTU() (int, error) {
	var buf [100]byte
	f, err := os.Open(tun.name + "/status")
	if err != nil {
		return 0, err
	}
	defer f.Close()
	n, err := f.Read(buf[:])
	_, res, ok := strings.Cut(string(buf[:n]), " maxtu ")
	if ok {
		if mtus, _, ok := strings.Cut(res, " "); ok {
			mtu, err := strconv.Atoi(mtus)
			if err != nil {
				return 0, fmt.Errorf("error converting mtu %q to int: %w", mtus, err)
			}
			return mtu, nil
		}
	}
	return 0, fmt.Errorf("no 'maxtu' field found in %s/status", tun.name)
}

func (tun *NativeTun) BatchSize() int {
	return 1
}
