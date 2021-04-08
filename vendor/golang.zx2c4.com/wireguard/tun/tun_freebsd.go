/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2017-2021 WireGuard LLC. All Rights Reserved.
 */

package tun

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/net/ipv6"
	"golang.org/x/sys/unix"
)

// _TUNSIFHEAD, value derived from sys/net/{if_tun,ioccom}.h
// const _TUNSIFHEAD = ((0x80000000) | (((4) & ((1 << 13) - 1) ) << 16) | (uint32(byte('t')) << 8) | (96))
const (
	_TUNSIFHEAD = 0x80047460
	_TUNSIFMODE = 0x8004745e
	_TUNSIFPID  = 0x2000745f
)

// TODO: move into x/sys/unix
const (
	SIOCGIFINFO_IN6        = 0xc048696c
	SIOCSIFINFO_IN6        = 0xc048696d
	ND6_IFF_AUTO_LINKLOCAL = 0x20
	ND6_IFF_NO_DAD         = 0x100
)

// Iface status string max len
const _IFSTATMAX = 800

const SIZEOF_UINTPTR = 4 << (^uintptr(0) >> 32 & 1)

// structure for iface requests with a pointer
type ifreq_ptr struct {
	Name [unix.IFNAMSIZ]byte
	Data uintptr
	Pad0 [16 - SIZEOF_UINTPTR]byte
}

// Structure for iface mtu get/set ioctls
type ifreq_mtu struct {
	Name [unix.IFNAMSIZ]byte
	MTU  uint32
	Pad0 [12]byte
}

// Structure for interface status request ioctl
type ifstat struct {
	IfsName [unix.IFNAMSIZ]byte
	Ascii   [_IFSTATMAX]byte
}

// Structures for nd6 flag manipulation
type in6_ndireq struct {
	Name          [unix.IFNAMSIZ]byte
	Linkmtu       uint32
	Maxmtu        uint32
	Basereachable uint32
	Reachable     uint32
	Retrans       uint32
	Flags         uint32
	Recalctm      int
	Chlim         uint8
	Initialized   uint8
	Randomseed0   [8]byte
	Randomseed1   [8]byte
	Randomid      [8]byte
}

type NativeTun struct {
	name        string
	tunFile     *os.File
	events      chan Event
	errors      chan error
	routeSocket int
	closeOnce   sync.Once
}

func (tun *NativeTun) routineRouteListener(tunIfindex int) {
	var (
		statusUp  bool
		statusMTU int
	)

	defer close(tun.events)

	data := make([]byte, os.Getpagesize())
	for {
	retry:
		n, err := unix.Read(tun.routeSocket, data)
		if err != nil {
			if errno, ok := err.(syscall.Errno); ok && errno == syscall.EINTR {
				goto retry
			}
			tun.errors <- err
			return
		}

		if n < 14 {
			continue
		}

		if data[3 /* type */] != unix.RTM_IFINFO {
			continue
		}
		ifindex := int(*(*uint16)(unsafe.Pointer(&data[12 /* ifindex */])))
		if ifindex != tunIfindex {
			continue
		}

		iface, err := net.InterfaceByIndex(ifindex)
		if err != nil {
			tun.errors <- err
			return
		}

		// Up / Down event
		up := (iface.Flags & net.FlagUp) != 0
		if up != statusUp && up {
			tun.events <- EventUp
		}
		if up != statusUp && !up {
			tun.events <- EventDown
		}
		statusUp = up

		// MTU changes
		if iface.MTU != statusMTU {
			tun.events <- EventMTUUpdate
		}
		statusMTU = iface.MTU
	}
}

func tunName(fd uintptr) (string, error) {
	//Terrible hack to make up for freebsd not having a TUNGIFNAME

	//First, make sure the tun pid matches this proc's pid
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(_TUNSIFPID),
		uintptr(0),
	)

	if errno != 0 {
		return "", fmt.Errorf("failed to set tun device PID: %s", errno.Error())
	}

	// Open iface control socket

	confd, err := unix.Socket(
		unix.AF_INET,
		unix.SOCK_DGRAM,
		0,
	)

	if err != nil {
		return "", err
	}

	defer unix.Close(confd)

	procPid := os.Getpid()

	//Try to find interface with matching PID
	for i := 1; ; i++ {
		iface, _ := net.InterfaceByIndex(i)
		if err != nil || iface == nil {
			break
		}

		// Structs for getting data in and out of SIOCGIFSTATUS ioctl
		var ifstatus ifstat
		copy(ifstatus.IfsName[:], iface.Name)

		// Make the syscall to get the status string
		_, _, errno := unix.Syscall(
			unix.SYS_IOCTL,
			uintptr(confd),
			uintptr(unix.SIOCGIFSTATUS),
			uintptr(unsafe.Pointer(&ifstatus)),
		)

		if errno != 0 {
			continue
		}

		nullStr := ifstatus.Ascii[:]
		i := bytes.IndexByte(nullStr, 0)
		if i < 1 {
			continue
		}
		statStr := string(nullStr[:i])
		var pidNum int = 0

		// Finally get the owning PID
		// Format string taken from sys/net/if_tun.c
		_, err := fmt.Sscanf(statStr, "\tOpened by PID %d\n", &pidNum)
		if err != nil {
			continue
		}

		if pidNum == procPid {
			return iface.Name, nil
		}
	}

	return "", nil
}

// Destroy a named system interface
func tunDestroy(name string) error {
	// Open control socket.
	var fd int
	fd, err := unix.Socket(
		unix.AF_INET,
		unix.SOCK_DGRAM,
		0,
	)
	if err != nil {
		return err
	}
	defer unix.Close(fd)

	var ifr [32]byte
	copy(ifr[:], name)
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.SIOCIFDESTROY),
		uintptr(unsafe.Pointer(&ifr[0])),
	)
	if errno != 0 {
		return fmt.Errorf("failed to destroy interface %s: %s", name, errno.Error())
	}

	return nil
}

func CreateTUN(name string, mtu int) (Device, error) {
	if len(name) > unix.IFNAMSIZ-1 {
		return nil, errors.New("interface name too long")
	}

	// See if interface already exists
	iface, _ := net.InterfaceByName(name)
	if iface != nil {
		return nil, fmt.Errorf("interface %s already exists", name)
	}

	tunFile, err := os.OpenFile("/dev/tun", unix.O_RDWR, 0)
	if err != nil {
		return nil, err
	}

	tun := NativeTun{tunFile: tunFile}
	var assignedName string
	tun.operateOnFd(func(fd uintptr) {
		assignedName, err = tunName(fd)
	})
	if err != nil {
		tunFile.Close()
		return nil, err
	}

	// Enable ifhead mode, otherwise tun will complain if it gets a non-AF_INET packet
	ifheadmode := 1
	var errno syscall.Errno
	tun.operateOnFd(func(fd uintptr) {
		_, _, errno = unix.Syscall(
			unix.SYS_IOCTL,
			fd,
			uintptr(_TUNSIFHEAD),
			uintptr(unsafe.Pointer(&ifheadmode)),
		)
	})

	if errno != 0 {
		tunFile.Close()
		tunDestroy(assignedName)
		return nil, fmt.Errorf("Unable to put into IFHEAD mode: %w", errno)
	}

	// Open control sockets
	confd, err := unix.Socket(
		unix.AF_INET,
		unix.SOCK_DGRAM,
		0,
	)
	if err != nil {
		tunFile.Close()
		tunDestroy(assignedName)
		return nil, err
	}
	defer unix.Close(confd)
	confd6, err := unix.Socket(
		unix.AF_INET6,
		unix.SOCK_DGRAM,
		0,
	)
	if err != nil {
		tunFile.Close()
		tunDestroy(assignedName)
		return nil, err
	}
	defer unix.Close(confd6)

	// Disable link-local v6, not just because WireGuard doesn't do that anyway, but
	// also because there are serious races with attaching and detaching LLv6 addresses
	// in relation to interface lifetime within the FreeBSD kernel.
	var ndireq in6_ndireq
	copy(ndireq.Name[:], assignedName)
	_, _, errno = unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(confd6),
		uintptr(SIOCGIFINFO_IN6),
		uintptr(unsafe.Pointer(&ndireq)),
	)
	if errno != 0 {
		tunFile.Close()
		tunDestroy(assignedName)
		return nil, fmt.Errorf("Unable to get nd6 flags for %s: %w", assignedName, errno)
	}
	ndireq.Flags = ndireq.Flags &^ ND6_IFF_AUTO_LINKLOCAL
	ndireq.Flags = ndireq.Flags | ND6_IFF_NO_DAD
	_, _, errno = unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(confd6),
		uintptr(SIOCSIFINFO_IN6),
		uintptr(unsafe.Pointer(&ndireq)),
	)
	if errno != 0 {
		tunFile.Close()
		tunDestroy(assignedName)
		return nil, fmt.Errorf("Unable to set nd6 flags for %s: %w", assignedName, errno)
	}

	if name != "" {
		// Rename the interface
		var newnp [unix.IFNAMSIZ]byte
		copy(newnp[:], name)
		var ifr ifreq_ptr
		copy(ifr.Name[:], assignedName)
		ifr.Data = uintptr(unsafe.Pointer(&newnp[0]))
		_, _, errno = unix.Syscall(
			unix.SYS_IOCTL,
			uintptr(confd),
			uintptr(unix.SIOCSIFNAME),
			uintptr(unsafe.Pointer(&ifr)),
		)
		if errno != 0 {
			tunFile.Close()
			tunDestroy(assignedName)
			return nil, fmt.Errorf("Failed to rename %s to %s: %w", assignedName, name, errno)
		}
	}

	return CreateTUNFromFile(tunFile, mtu)
}

func CreateTUNFromFile(file *os.File, mtu int) (Device, error) {

	tun := &NativeTun{
		tunFile: file,
		events:  make(chan Event, 10),
		errors:  make(chan error, 1),
	}

	name, err := tun.Name()
	if err != nil {
		tun.tunFile.Close()
		return nil, err
	}

	tunIfindex, err := func() (int, error) {
		iface, err := net.InterfaceByName(name)
		if err != nil {
			return -1, err
		}
		return iface.Index, nil
	}()
	if err != nil {
		tun.tunFile.Close()
		return nil, err
	}

	tun.routeSocket, err = unix.Socket(unix.AF_ROUTE, unix.SOCK_RAW, unix.AF_UNSPEC)
	if err != nil {
		tun.tunFile.Close()
		return nil, err
	}

	go tun.routineRouteListener(tunIfindex)

	err = tun.setMTU(mtu)
	if err != nil {
		tun.Close()
		return nil, err
	}

	return tun, nil
}

func (tun *NativeTun) Name() (string, error) {
	var name string
	var err error
	tun.operateOnFd(func(fd uintptr) {
		name, err = tunName(fd)
	})
	if err != nil {
		return "", err
	}
	tun.name = name
	return name, nil
}

func (tun *NativeTun) File() *os.File {
	return tun.tunFile
}

func (tun *NativeTun) Events() chan Event {
	return tun.events
}

func (tun *NativeTun) Read(buff []byte, offset int) (int, error) {
	select {
	case err := <-tun.errors:
		return 0, err
	default:
		buff := buff[offset-4:]
		n, err := tun.tunFile.Read(buff[:])
		if n < 4 {
			return 0, err
		}
		return n - 4, err
	}
}

func (tun *NativeTun) Write(buff []byte, offset int) (int, error) {

	// reserve space for header

	buff = buff[offset-4:]

	// add packet information header

	buff[0] = 0x00
	buff[1] = 0x00
	buff[2] = 0x00

	if buff[4]>>4 == ipv6.Version {
		buff[3] = unix.AF_INET6
	} else {
		buff[3] = unix.AF_INET
	}

	// write

	return tun.tunFile.Write(buff)
}

func (tun *NativeTun) Flush() error {
	// TODO: can flushing be implemented by buffering and using sendmmsg?
	return nil
}

func (tun *NativeTun) Close() error {
	var err1, err2, err3 error
	tun.closeOnce.Do(func() {
		err1 = tun.tunFile.Close()
		err2 = tunDestroy(tun.name)
		if tun.routeSocket != -1 {
			unix.Shutdown(tun.routeSocket, unix.SHUT_RDWR)
			err3 = unix.Close(tun.routeSocket)
			tun.routeSocket = -1
		} else if tun.events != nil {
			close(tun.events)
		}
	})
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return err3
}

func (tun *NativeTun) setMTU(n int) error {
	// open datagram socket

	var fd int

	fd, err := unix.Socket(
		unix.AF_INET,
		unix.SOCK_DGRAM,
		0,
	)

	if err != nil {
		return err
	}

	defer unix.Close(fd)

	// do ioctl call

	var ifr ifreq_mtu
	copy(ifr.Name[:], tun.name)
	ifr.MTU = uint32(n)

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.SIOCSIFMTU),
		uintptr(unsafe.Pointer(&ifr)),
	)

	if errno != 0 {
		return fmt.Errorf("failed to set MTU on %s", tun.name)
	}

	return nil
}

func (tun *NativeTun) MTU() (int, error) {
	// open datagram socket

	fd, err := unix.Socket(
		unix.AF_INET,
		unix.SOCK_DGRAM,
		0,
	)

	if err != nil {
		return 0, err
	}

	defer unix.Close(fd)

	// do ioctl call
	var ifr ifreq_mtu
	copy(ifr.Name[:], tun.name)

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.SIOCGIFMTU),
		uintptr(unsafe.Pointer(&ifr)),
	)
	if errno != 0 {
		return 0, fmt.Errorf("failed to get MTU on %s", tun.name)
	}

	return int(*(*int32)(unsafe.Pointer(&ifr.MTU))), nil
}
