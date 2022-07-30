package portscan

import (
	"net"
	"strconv"
	"time"
)

type Probe struct {
	host string
	port int
	open bool
}

func NewProbe(host string, port int) *Probe {
	return &Probe{host: host, port: port, open: false}
}

func (probe *Probe) Probe() {
	addr := probe.host + ":" + strconv.Itoa(probe.port)

	conn, err := net.DialTimeout("tcp", addr, 3*time.Second)
	if err != nil {
		return
	}
	defer conn.Close()

	probe.open = true
}

func (probe *Probe) Report() string {
	return probe.host + ":" + strconv.Itoa(probe.port)
}
