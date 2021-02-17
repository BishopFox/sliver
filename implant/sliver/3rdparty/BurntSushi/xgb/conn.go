package xgb

/*
conn.go contains a couple of functions that do some real dirty work related
to the initial connection handshake with X.

This code is largely unmodified from the original XGB package that I forked.
*/

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

// connect connects to the X server given in the 'display' string,
// and does all the necessary setup handshaking.
// If 'display' is empty it will be taken from os.Getenv("DISPLAY").
// Note that you should read and understand the "Connection Setup" of the
// X Protocol Reference Manual before changing this function:
// http://goo.gl/4zGQg
func (c *Conn) connect(display string) error {
	err := c.dial(display)
	if err != nil {
		return err
	}

	return c.postConnect()
}

// connect init from to the net.Conn,
func (c *Conn) connectNet(netConn net.Conn) error {
	c.conn = netConn
	return c.postConnect()
}

// do the postConnect action after Conn get it's underly net.Conn
func (c *Conn) postConnect() error {
	// Get authentication data
	authName, authData, err := readAuthority(c.host, c.display)
	noauth := false
	if err != nil {
		Logger.Printf("Could not get authority info: %v", err)
		Logger.Println("Trying connection without authority info...")
		authName = ""
		authData = []byte{}
		noauth = true
	}

	// Assume that the authentication protocol is "MIT-MAGIC-COOKIE-1".
	if !noauth && (authName != "MIT-MAGIC-COOKIE-1" || len(authData) != 16) {
		return errors.New("unsupported auth protocol " + authName)
	}

	buf := make([]byte, 12+Pad(len(authName))+Pad(len(authData)))
	buf[0] = 0x6c
	buf[1] = 0
	Put16(buf[2:], 11)
	Put16(buf[4:], 0)
	Put16(buf[6:], uint16(len(authName)))
	Put16(buf[8:], uint16(len(authData)))
	Put16(buf[10:], 0)
	copy(buf[12:], []byte(authName))
	copy(buf[12+Pad(len(authName)):], authData)
	if _, err = c.conn.Write(buf); err != nil {
		return err
	}

	head := make([]byte, 8)
	if _, err = io.ReadFull(c.conn, head[0:8]); err != nil {
		return err
	}
	code := head[0]
	reasonLen := head[1]
	major := Get16(head[2:])
	minor := Get16(head[4:])
	dataLen := Get16(head[6:])

	if major != 11 || minor != 0 {
		return fmt.Errorf("x protocol version mismatch: %d.%d", major, minor)
	}

	buf = make([]byte, int(dataLen)*4+8, int(dataLen)*4+8)
	copy(buf, head)
	if _, err = io.ReadFull(c.conn, buf[8:]); err != nil {
		return err
	}

	if code == 0 {
		reason := buf[8 : 8+reasonLen]
		return fmt.Errorf("x protocol authentication refused: %s",
			string(reason))
	}

	// Unfortunately, it isn't really feasible to read the setup bytes here,
	// since the code to do so is in a different package.
	// Users must call 'xproto.Setup(X)' to get the setup info.
	c.SetupBytes = buf

	// But also read stuff that we *need* to get started.
	c.setupResourceIdBase = Get32(buf[12:])
	c.setupResourceIdMask = Get32(buf[16:])

	return nil
}

// dial initializes the actual net connection with X.
func (c *Conn) dial(display string) error {
	if len(display) == 0 {
		display = os.Getenv("DISPLAY")
	}

	display0 := display
	if len(display) == 0 {
		return errors.New("empty display string")
	}

	colonIdx := strings.LastIndex(display, ":")
	if colonIdx < 0 {
		return errors.New("bad display string: " + display0)
	}

	var protocol, socket string

	if display[0] == '/' {
		socket = display[0:colonIdx]
	} else {
		slashIdx := strings.LastIndex(display, "/")
		if slashIdx >= 0 {
			protocol = display[0:slashIdx]
			c.host = display[slashIdx+1 : colonIdx]
		} else {
			c.host = display[0:colonIdx]
		}
	}

	display = display[colonIdx+1 : len(display)]
	if len(display) == 0 {
		return errors.New("bad display string: " + display0)
	}

	var scr string
	dotIdx := strings.LastIndex(display, ".")
	if dotIdx < 0 {
		c.display = display[0:]
	} else {
		c.display = display[0:dotIdx]
		scr = display[dotIdx+1:]
	}

	var err error
	c.DisplayNumber, err = strconv.Atoi(c.display)
	if err != nil || c.DisplayNumber < 0 {
		return errors.New("bad display string: " + display0)
	}

	if len(scr) != 0 {
		c.DefaultScreen, err = strconv.Atoi(scr)
		if err != nil {
			return errors.New("bad display string: " + display0)
		}
	}

	// Connect to server
	if len(socket) != 0 {
		c.conn, err = net.Dial("unix", socket+":"+c.display)
	} else if len(c.host) != 0 {
		if protocol == "" {
			protocol = "tcp"
		}
		c.conn, err = net.Dial(protocol,
			c.host+":"+strconv.Itoa(6000+c.DisplayNumber))
	} else {
		c.conn, err = net.Dial("unix", "/tmp/.X11-unix/X"+c.display)
	}

	if err != nil {
		return errors.New("cannot connect to " + display0 + ": " + err.Error())
	}
	return nil
}
