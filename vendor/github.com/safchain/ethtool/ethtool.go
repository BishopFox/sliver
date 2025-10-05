/*
 *
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 */

// The ethtool package aims to provide a library that provides easy access
// to the Linux SIOCETHTOOL ioctl operations. It can be used to retrieve information
// from a network device such as statistics, driver related information or even
// the peer of a VETH interface.
package ethtool

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Maximum size of an interface name
const (
	IFNAMSIZ = 16
)

// ioctl ethtool request
const (
	SIOCETHTOOL = 0x8946
)

// ethtool stats related constants.
const (
	ETH_GSTRING_LEN   = 32
	ETH_SS_STATS      = 1
	ETH_SS_PRIV_FLAGS = 2
	ETH_SS_FEATURES   = 4

	// CMD supported
	ETHTOOL_GSET          = 0x00000001                 /* Get settings. */
	ETHTOOL_SSET          = 0x00000002                 /* Set settings. */
	ETHTOOL_GWOL          = 0x00000005                 /* Get wake-on-lan options. */
	ETHTOOL_SWOL          = 0x00000006                 /* Set wake-on-lan options. */
	ETHTOOL_GDRVINFO      = 0x00000003                 /* Get driver info. */
	ETHTOOL_GMSGLVL       = 0x00000007                 /* Get driver message level */
	ETHTOOL_SMSGLVL       = 0x00000008                 /* Set driver msg level. */
	ETHTOOL_GLINKSETTINGS = unix.ETHTOOL_GLINKSETTINGS // 0x4c
	ETHTOOL_SLINKSETTINGS = unix.ETHTOOL_SLINKSETTINGS // 0x4d

	// Get link status for host, i.e. whether the interface *and* the
	// physical port (if there is one) are up (ethtool_value).
	ETHTOOL_GLINK            = 0x0000000a
	ETHTOOL_GCOALESCE        = 0x0000000e /* Get coalesce config */
	ETHTOOL_SCOALESCE        = 0x0000000f /* Set coalesce config */
	ETHTOOL_GRINGPARAM       = 0x00000010 /* Get ring parameters */
	ETHTOOL_SRINGPARAM       = 0x00000011 /* Set ring parameters. */
	ETHTOOL_GPAUSEPARAM      = 0x00000012 /* Get pause parameters */
	ETHTOOL_SPAUSEPARAM      = 0x00000013 /* Set pause parameters. */
	ETHTOOL_GSTRINGS         = 0x0000001b /* Get specified string set */
	ETHTOOL_PHYS_ID          = 0x0000001c /* Identify the NIC */
	ETHTOOL_GSTATS           = 0x0000001d /* Get NIC-specific statistics */
	ETHTOOL_GPERMADDR        = 0x00000020 /* Get permanent hardware address */
	ETHTOOL_GFLAGS           = 0x00000025 /* Get flags bitmap(ethtool_value) */
	ETHTOOL_GPFLAGS          = 0x00000027 /* Get driver-private flags bitmap */
	ETHTOOL_SPFLAGS          = 0x00000028 /* Set driver-private flags bitmap */
	ETHTOOL_GSSET_INFO       = 0x00000037 /* Get string set info */
	ETHTOOL_GFEATURES        = 0x0000003a /* Get device offload settings */
	ETHTOOL_SFEATURES        = 0x0000003b /* Change device offload settings */
	ETHTOOL_GCHANNELS        = 0x0000003c /* Get no of channels */
	ETHTOOL_SCHANNELS        = 0x0000003d /* Set no of channels */
	ETHTOOL_GET_TS_INFO      = 0x00000041 /* Get time stamping and PHC info */
	ETHTOOL_GMODULEINFO      = 0x00000042 /* Get plug-in module information */
	ETHTOOL_GMODULEEEPROM    = 0x00000043 /* Get plug-in module eeprom */
	ETHTOOL_GRXFHINDIR       = 0x00000038 /* Get RX flow hash indir'n table */
	ETHTOOL_SRXFHINDIR       = 0x00000039 /* Set RX flow hash indir'n table */
	ETH_RXFH_INDIR_NO_CHANGE = 0xFFFFFFFF

	// Speed and Duplex unknowns/constants (Manually defined based on <linux/ethtool.h>)
	SPEED_UNKNOWN  = 0xffffffff // ((__u32)-1) SPEED_UNKNOWN
	DUPLEX_HALF    = 0x00       // DUPLEX_HALF
	DUPLEX_FULL    = 0x01       // DUPLEX_FULL
	DUPLEX_UNKNOWN = 0xff       // DUPLEX_UNKNOWN

	// Port types (Manually defined based on <linux/ethtool.h>)
	PORT_TP    = 0x00 // PORT_TP
	PORT_AUI   = 0x01 // PORT_AUI
	PORT_MII   = 0x02 // PORT_MII
	PORT_FIBRE = 0x03 // PORT_FIBRE
	PORT_BNC   = 0x04 // PORT_BNC
	PORT_DA    = 0x05 // PORT_DA
	PORT_NONE  = 0xef // PORT_NONE
	PORT_OTHER = 0xff // PORT_OTHER

	// Autoneg settings (Manually defined based on <linux/ethtool.h>)
	AUTONEG_DISABLE = 0x00 // AUTONEG_DISABLE
	AUTONEG_ENABLE  = 0x01 // AUTONEG_ENABLE

	// MDIX states (Manually defined based on <linux/ethtool.h>)
	ETH_TP_MDI_INVALID = 0x00 // ETH_TP_MDI_INVALID
	ETH_TP_MDI         = 0x01 // ETH_TP_MDI
	ETH_TP_MDI_X       = 0x02 // ETH_TP_MDI_X
	ETH_TP_MDI_AUTO    = 0x03 // Control value ETH_TP_MDI_AUTO

	// Link mode mask bits count (Manually defined based on ethtool.h)
	ETHTOOL_LINK_MODE_MASK_NBITS = 92 // __ETHTOOL_LINK_MODE_MASK_NBITS

	// Calculate max nwords based on NBITS using the manually defined constant
	MAX_LINK_MODE_MASK_NWORDS = (ETHTOOL_LINK_MODE_MASK_NBITS + 31) / 32 // = 3
)

// MAX_GSTRINGS maximum number of stats entries that ethtool can
// retrieve currently.
const (
	MAX_GSTRINGS       = 32768
	MAX_FEATURE_BLOCKS = (MAX_GSTRINGS + 32 - 1) / 32
	EEPROM_LEN         = 640
	PERMADDR_LEN       = 32
)

// ethtool sset_info related constants
const (
	MAX_SSET_INFO = 64
)

const (
	DEFAULT_BLINK_DURATION = 60 * time.Second
)

var (
	gstringsPool = sync.Pool{
		New: func() interface{} {
			// new() will allocate and zero-initialize the struct.
			// The large data array within ethtoolGStrings will be zeroed.
			return new(EthtoolGStrings)
		},
	}
	statsPool = sync.Pool{
		New: func() interface{} {
			// new() will allocate and zero-initialize the struct.
			// The large data array within ethtoolStats will be zeroed.
			return new(EthtoolStats)
		},
	}
)

type ifreq struct {
	ifr_name [IFNAMSIZ]byte
	ifr_data uintptr
}

// following structures comes from uapi/linux/ethtool.h
type ethtoolSsetInfo struct {
	cmd       uint32
	reserved  uint32
	sset_mask uint64
	data      [MAX_SSET_INFO]uint32
}

type ethtoolGetFeaturesBlock struct {
	available     uint32
	requested     uint32
	active        uint32
	never_changed uint32
}

type ethtoolGfeatures struct {
	cmd    uint32
	size   uint32
	blocks [MAX_FEATURE_BLOCKS]ethtoolGetFeaturesBlock
}

type ethtoolSetFeaturesBlock struct {
	valid     uint32
	requested uint32
}

type ethtoolSfeatures struct {
	cmd    uint32
	size   uint32
	blocks [MAX_FEATURE_BLOCKS]ethtoolSetFeaturesBlock
}

type ethtoolDrvInfo struct {
	cmd          uint32
	driver       [32]byte
	version      [32]byte
	fw_version   [32]byte
	bus_info     [32]byte
	erom_version [32]byte
	reserved2    [12]byte
	n_priv_flags uint32
	n_stats      uint32
	testinfo_len uint32
	eedump_len   uint32
	regdump_len  uint32
}

// DrvInfo contains driver information
// ethtool.h v3.5: struct ethtool_drvinfo
type DrvInfo struct {
	Cmd         uint32
	Driver      string
	Version     string
	FwVersion   string
	BusInfo     string
	EromVersion string
	Reserved2   string
	NPrivFlags  uint32
	NStats      uint32
	TestInfoLen uint32
	EedumpLen   uint32
	RegdumpLen  uint32
}

// Channels contains the number of channels for a given interface.
type Channels struct {
	Cmd           uint32
	MaxRx         uint32
	MaxTx         uint32
	MaxOther      uint32
	MaxCombined   uint32
	RxCount       uint32
	TxCount       uint32
	OtherCount    uint32
	CombinedCount uint32
}

// Coalesce is a coalesce config for an interface
type Coalesce struct {
	Cmd                      uint32
	RxCoalesceUsecs          uint32
	RxMaxCoalescedFrames     uint32
	RxCoalesceUsecsIrq       uint32
	RxMaxCoalescedFramesIrq  uint32
	TxCoalesceUsecs          uint32
	TxMaxCoalescedFrames     uint32
	TxCoalesceUsecsIrq       uint32
	TxMaxCoalescedFramesIrq  uint32
	StatsBlockCoalesceUsecs  uint32
	UseAdaptiveRxCoalesce    uint32
	UseAdaptiveTxCoalesce    uint32
	PktRateLow               uint32
	RxCoalesceUsecsLow       uint32
	RxMaxCoalescedFramesLow  uint32
	TxCoalesceUsecsLow       uint32
	TxMaxCoalescedFramesLow  uint32
	PktRateHigh              uint32
	RxCoalesceUsecsHigh      uint32
	RxMaxCoalescedFramesHigh uint32
	TxCoalesceUsecsHigh      uint32
	TxMaxCoalescedFramesHigh uint32
	RateSampleInterval       uint32
}

// IdentityConf is an identity config for an interface
type IdentityConf struct {
	Cmd      uint32
	Duration uint32
}

// WoL options
const (
	WAKE_PHY         = 1 << 0
	WAKE_UCAST       = 1 << 1
	WAKE_MCAST       = 1 << 2
	WAKE_BCAST       = 1 << 3
	WAKE_ARP         = 1 << 4
	WAKE_MAGIC       = 1 << 5
	WAKE_MAGICSECURE = 1 << 6 // only meaningful if WAKE_MAGIC
)

var WoLMap = map[uint32]string{
	WAKE_PHY:         "p", // Wake on PHY activity
	WAKE_UCAST:       "u", // Wake on unicast messages
	WAKE_MCAST:       "m", // Wake on multicast messages
	WAKE_BCAST:       "b", // Wake on broadcast messages
	WAKE_ARP:         "a", // Wake on ARP
	WAKE_MAGIC:       "g", // Wake on MagicPacket™
	WAKE_MAGICSECURE: "s", // Enable SecureOn™ password for MagicPacket™
	// f Wake on filter(s)
	// d Disable (wake on  nothing). This option clears all previous options.
}

// WakeOnLan contains WoL config for an interface
type WakeOnLan struct {
	Cmd       uint32 // ETHTOOL_GWOL or ETHTOOL_SWOL
	Supported uint32 // r/o bitmask of WAKE_* flags for supported WoL modes
	Opts      uint32 // Bitmask of WAKE_* flags for enabled WoL modes
}

// Timestamping options
// see: https://www.kernel.org/doc/Documentation/networking/timestamping.txt
const (
	SOF_TIMESTAMPING_TX_HARDWARE  = (1 << 0)  /* Request tx timestamps generated by the network adapter. */
	SOF_TIMESTAMPING_TX_SOFTWARE  = (1 << 1)  /* Request tx timestamps when data leaves the kernel. */
	SOF_TIMESTAMPING_RX_HARDWARE  = (1 << 2)  /* Request rx timestamps generated by the network adapter. */
	SOF_TIMESTAMPING_RX_SOFTWARE  = (1 << 3)  /* Request rx timestamps when data enters the kernel. */
	SOF_TIMESTAMPING_SOFTWARE     = (1 << 4)  /* Report any software timestamps when available. */
	SOF_TIMESTAMPING_SYS_HARDWARE = (1 << 5)  /* This option is deprecated and ignored. */
	SOF_TIMESTAMPING_RAW_HARDWARE = (1 << 6)  /* Report hardware timestamps. */
	SOF_TIMESTAMPING_OPT_ID       = (1 << 7)  /* Generate a unique identifier along with each packet. */
	SOF_TIMESTAMPING_TX_SCHED     = (1 << 8)  /* Request tx timestamps prior to entering the packet scheduler. */
	SOF_TIMESTAMPING_TX_ACK       = (1 << 9)  /* Request tx timestamps when all data in the send buffer has been acknowledged. */
	SOF_TIMESTAMPING_OPT_CMSG     = (1 << 10) /* Support recv() cmsg for all timestamped packets. */
	SOF_TIMESTAMPING_OPT_TSONLY   = (1 << 11) /* Applies to transmit timestamps only. */
	SOF_TIMESTAMPING_OPT_STATS    = (1 << 12) /* Optional stats that are obtained along with the transmit timestamps. */
	SOF_TIMESTAMPING_OPT_PKTINFO  = (1 << 13) /* Enable the SCM_TIMESTAMPING_PKTINFO control message for incoming packets with hardware timestamps. */
	SOF_TIMESTAMPING_OPT_TX_SWHW  = (1 << 14) /* Request both hardware and software timestamps for outgoing packets when SOF_TIMESTAMPING_TX_HARDWARE and SOF_TIMESTAMPING_TX_SOFTWARE are enabled at the same time. */
	SOF_TIMESTAMPING_BIND_PHC     = (1 << 15) /* Bind the socket to a specific PTP Hardware Clock. */
)

const (
	/*
	 * No outgoing packet will need hardware time stamping;
	 * should a packet arrive which asks for it, no hardware
	 * time stamping will be done.
	 */
	HWTSTAMP_TX_OFF = iota

	/*
	 * Enables hardware time stamping for outgoing packets;
	 * the sender of the packet decides which are to be
	 * time stamped by setting %SOF_TIMESTAMPING_TX_SOFTWARE
	 * before sending the packet.
	 */
	HWTSTAMP_TX_ON

	/*
	 * Enables time stamping for outgoing packets just as
	 * HWTSTAMP_TX_ON does, but also enables time stamp insertion
	 * directly into Sync packets. In this case, transmitted Sync
	 * packets will not received a time stamp via the socket error
	 * queue.
	 */
	HWTSTAMP_TX_ONESTEP_SYNC

	/*
	 * Same as HWTSTAMP_TX_ONESTEP_SYNC, but also enables time
	 * stamp insertion directly into PDelay_Resp packets. In this
	 * case, neither transmitted Sync nor PDelay_Resp packets will
	 * receive a time stamp via the socket error queue.
	 */
	HWTSTAMP_TX_ONESTEP_P2P
)

const (
	HWTSTAMP_FILTER_NONE                = iota /* time stamp no incoming packet at all */
	HWTSTAMP_FILTER_ALL                        /* time stamp any incoming packet */
	HWTSTAMP_FILTER_SOME                       /* return value: time stamp all packets requested plus some others */
	HWTSTAMP_FILTER_PTP_V1_L4_EVENT            /* PTP v1, UDP, any kind of event packet */
	HWTSTAMP_FILTER_PTP_V1_L4_SYNC             /* PTP v1, UDP, Sync packet */
	HWTSTAMP_FILTER_PTP_V1_L4_DELAY_REQ        /* PTP v1, UDP, Delay_req packet */
	HWTSTAMP_FILTER_PTP_V2_L4_EVENT            /* PTP v2, UDP, any kind of event packet */
	HWTSTAMP_FILTER_PTP_V2_L4_SYNC             /* PTP v2, UDP, Sync packet */
	HWTSTAMP_FILTER_PTP_V2_L4_DELAY_REQ        /* PTP v2, UDP, Delay_req packet */
	HWTSTAMP_FILTER_PTP_V2_L2_EVENT            /* 802.AS1, Ethernet, any kind of event packet */
	HWTSTAMP_FILTER_PTP_V2_L2_SYNC             /* 802.AS1, Ethernet, Sync packet */
	HWTSTAMP_FILTER_PTP_V2_L2_DELAY_REQ        /* 802.AS1, Ethernet, Delay_req packet */
	HWTSTAMP_FILTER_PTP_V2_EVENT               /* PTP v2/802.AS1, any layer, any kind of event packet */
	HWTSTAMP_FILTER_PTP_V2_SYNC                /* PTP v2/802.AS1, any layer, Sync packet */
	HWTSTAMP_FILTER_PTP_V2_DELAY_REQ           /* PTP v2/802.AS1, any layer, Delay_req packet */
	HWTSTAMP_FILTER_NTP_ALL                    /* NTP, UDP, all versions and packet modes */
)

// TimestampingInformation contains PTP timetstapming information
type TimestampingInformation struct {
	Cmd            uint32
	SoTimestamping uint32 /* SOF_TIMESTAMPING_* bitmask */
	PhcIndex       int32
	TxTypes        uint32 /* HWTSTAMP_TX_* */
	txReserved     [3]uint32
	RxFilters      uint32 /* HWTSTAMP_FILTER_ */
	rxReserved     [3]uint32
}

type EthtoolGStrings struct {
	cmd        uint32
	string_set uint32
	len        uint32
	data       [MAX_GSTRINGS * ETH_GSTRING_LEN]byte
}

type EthtoolStats struct {
	cmd     uint32
	n_stats uint32
	data    [MAX_GSTRINGS]uint64
}

type ethtoolEeprom struct {
	cmd    uint32
	magic  uint32
	offset uint32
	len    uint32
	data   [EEPROM_LEN]byte
}

type ethtoolModInfo struct {
	cmd        uint32
	tpe        uint32
	eeprom_len uint32
	reserved   [8]uint32
}

type ethtoolLink struct {
	cmd  uint32
	data uint32
}

type ethtoolPermAddr struct {
	cmd  uint32
	size uint32
	data [PERMADDR_LEN]byte
}

// Ring is a ring config for an interface
type Ring struct {
	Cmd               uint32
	RxMaxPending      uint32
	RxMiniMaxPending  uint32
	RxJumboMaxPending uint32
	TxMaxPending      uint32
	RxPending         uint32
	RxMiniPending     uint32
	RxJumboPending    uint32
	TxPending         uint32
}

// Pause is a pause config for an interface
type Pause struct {
	Cmd     uint32
	Autoneg uint32
	RxPause uint32
	TxPause uint32
}

// Ethtool is a struct that contains the file descriptor for the ethtool
type Ethtool struct {
	fd int
}

// max values for my setup dont know how to make this dynamic
const MAX_INDIR_SIZE = 256
const MAX_CORES = 32

type Indir struct {
	Cmd       uint32
	Size      uint32
	RingIndex [MAX_INDIR_SIZE]uint32 // statically definded otherwise crash

}

type SetIndir struct {
	Equal  uint8    // used to set number of cores
	Weight []uint32 // used to select cores
}

// Convert zero-terminated array of chars (string in C) to a Go string.
func goString(s []byte) string {
	strEnd := bytes.IndexByte(s, 0)
	if strEnd == -1 {
		return string(s)
	}
	return string(s[:strEnd])
}

// DriverName returns the driver name of the given interface name.
func (e *Ethtool) DriverName(intf string) (string, error) {
	info, err := e.getDriverInfo(intf)
	if err != nil {
		return "", err
	}
	return goString(info.driver[:]), nil
}

// BusInfo returns the bus information of the given interface name.
func (e *Ethtool) BusInfo(intf string) (string, error) {
	info, err := e.getDriverInfo(intf)
	if err != nil {
		return "", err
	}
	return goString(info.bus_info[:]), nil
}

// ModuleEeprom returns Eeprom information of the given interface name.
func (e *Ethtool) ModuleEeprom(intf string) ([]byte, error) {
	eeprom, _, err := e.getModuleEeprom(intf)
	if err != nil {
		return nil, err
	}

	return eeprom.data[:eeprom.len], nil
}

// ModuleEepromHex returns Eeprom information as hexadecimal string
func (e *Ethtool) ModuleEepromHex(intf string) (string, error) {
	eeprom, _, err := e.getModuleEeprom(intf)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(eeprom.data[:eeprom.len]), nil
}

// DriverInfo returns driver information of the given interface name.
func (e *Ethtool) DriverInfo(intf string) (DrvInfo, error) {
	i, err := e.getDriverInfo(intf)
	if err != nil {
		return DrvInfo{}, err
	}

	drvInfo := DrvInfo{
		Cmd:         i.cmd,
		Driver:      goString(i.driver[:]),
		Version:     goString(i.version[:]),
		FwVersion:   goString(i.fw_version[:]),
		BusInfo:     goString(i.bus_info[:]),
		EromVersion: goString(i.erom_version[:]),
		Reserved2:   goString(i.reserved2[:]),
		NPrivFlags:  i.n_priv_flags,
		NStats:      i.n_stats,
		TestInfoLen: i.testinfo_len,
		EedumpLen:   i.eedump_len,
		RegdumpLen:  i.regdump_len,
	}

	return drvInfo, nil
}

// GetIndir retrieves the indirection table of the given interface name.
func (e *Ethtool) GetIndir(intf string) (Indir, error) {
	indir, err := e.getIndir(intf)
	if err != nil {
		return Indir{}, err
	}

	return indir, nil
}

// SetIndir sets the indirection table of the given interface from the SetIndir struct
func (e *Ethtool) SetIndir(intf string, setIndir SetIndir) (Indir, error) {

	if setIndir.Equal != 0 && setIndir.Weight != nil {
		return Indir{}, fmt.Errorf("equal and weight options are mutually exclusive")
	}

	indir, err := e.GetIndir(intf)
	if err != nil {
		return Indir{}, err
	}

	newindir, err := e.setIndir(intf, indir, setIndir)
	if err != nil {
		return Indir{}, err
	}

	return newindir, nil
}

// GetChannels returns the number of channels for the given interface name.
func (e *Ethtool) GetChannels(intf string) (Channels, error) {
	channels, err := e.getChannels(intf)
	if err != nil {
		return Channels{}, err
	}

	return channels, nil
}

// SetChannels sets the number of channels for the given interface name and
// returns the new number of channels.
func (e *Ethtool) SetChannels(intf string, channels Channels) (Channels, error) {
	channels, err := e.setChannels(intf, channels)
	if err != nil {
		return Channels{}, err
	}

	return channels, nil
}

// GetCoalesce returns the coalesce config for the given interface name.
func (e *Ethtool) GetCoalesce(intf string) (Coalesce, error) {
	coalesce, err := e.getCoalesce(intf)
	if err != nil {
		return Coalesce{}, err
	}
	return coalesce, nil
}

// SetCoalesce sets the coalesce config for the given interface name.
func (e *Ethtool) SetCoalesce(intf string, coalesce Coalesce) (Coalesce, error) {
	coalesce, err := e.setCoalesce(intf, coalesce)
	if err != nil {
		return Coalesce{}, err
	}
	return coalesce, nil
}

// GetTimestampingInformation returns the PTP timestamping information for the given interface name.
func (e *Ethtool) GetTimestampingInformation(intf string) (TimestampingInformation, error) {
	ts, err := e.getTimestampingInformation(intf)
	if err != nil {
		return TimestampingInformation{}, err
	}
	return ts, nil
}

// PermAddr returns permanent address of the given interface name.
func (e *Ethtool) PermAddr(intf string) (string, error) {
	permAddr, err := e.getPermAddr(intf)
	if err != nil {
		return "", err
	}

	if permAddr.data[0] == 0 && permAddr.data[1] == 0 &&
		permAddr.data[2] == 0 && permAddr.data[3] == 0 &&
		permAddr.data[4] == 0 && permAddr.data[5] == 0 {
		return "", nil
	}

	return fmt.Sprintf("%x:%x:%x:%x:%x:%x",
		permAddr.data[0:1],
		permAddr.data[1:2],
		permAddr.data[2:3],
		permAddr.data[3:4],
		permAddr.data[4:5],
		permAddr.data[5:6],
	), nil
}

// GetWakeOnLan returns the WoL config for the given interface name.
func (e *Ethtool) GetWakeOnLan(intf string) (WakeOnLan, error) {
	wol := WakeOnLan{
		Cmd: ETHTOOL_GWOL,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&wol))); err != nil {
		return WakeOnLan{}, err
	}

	return wol, nil
}

// SetWakeOnLan sets the WoL config for the given interface name and
// returns the new WoL config.
func (e *Ethtool) SetWakeOnLan(intf string, wol WakeOnLan) (WakeOnLan, error) {
	wol.Cmd = ETHTOOL_SWOL

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&wol))); err != nil {
		return WakeOnLan{}, err
	}

	return wol, nil
}

func (e *Ethtool) ioctl(intf string, data uintptr) error {
	var name [IFNAMSIZ]byte
	copy(name[:], []byte(intf))

	ifr := ifreq{
		ifr_name: name,
		ifr_data: data,
	}

	_, _, ep := unix.Syscall(unix.SYS_IOCTL, uintptr(e.fd), SIOCETHTOOL, uintptr(unsafe.Pointer(&ifr)))
	if ep != 0 {
		return ep
	}

	return nil
}

func (e *Ethtool) getDriverInfo(intf string) (ethtoolDrvInfo, error) {
	drvinfo := ethtoolDrvInfo{
		cmd: ETHTOOL_GDRVINFO,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&drvinfo))); err != nil {
		return ethtoolDrvInfo{}, err
	}

	return drvinfo, nil
}

// parsing of do_grxfhindir from ethtool.c
func (e *Ethtool) getIndir(intf string) (Indir, error) {
	indir_head := Indir{
		Cmd:  ETHTOOL_GRXFHINDIR,
		Size: 0,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&indir_head))); err != nil {
		return Indir{}, err
	}

	indir := Indir{
		Cmd:  ETHTOOL_GRXFHINDIR,
		Size: indir_head.Size,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&indir))); err != nil {
		return Indir{}, err
	}

	return indir, nil
}

// parsing of do_srxfhindir from ethtool.c
func (e *Ethtool) setIndir(intf string, indir Indir, setIndir SetIndir) (Indir, error) {

	err := fillIndirTable(&indir.Size, indir.RingIndex[:], 0, 0, int(setIndir.Equal), setIndir.Weight, uint32(len(setIndir.Weight)))
	if err != nil {
		return Indir{}, err
	}

	if indir.Size == ETH_RXFH_INDIR_NO_CHANGE {
		indir.Size = MAX_INDIR_SIZE
		return indir, nil
	}

	indir.Cmd = ETHTOOL_SRXFHINDIR
	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&indir))); err != nil {
		return Indir{}, err
	}

	return indir, nil
}

func fillIndirTable(indirSize *uint32, indir []uint32, rxfhindirDefault int,
	rxfhindirStart int, rxfhindirEqual int, rxfhindirWeight []uint32,
	numWeights uint32) error {

	switch {
	case rxfhindirEqual != 0:
		for i := uint32(0); i < *indirSize; i++ {
			indir[i] = uint32(rxfhindirStart) + (i % uint32(rxfhindirEqual))
		}
	case rxfhindirWeight != nil:
		var sum, partial uint32 = 0, 0
		var j, weight uint32
		for j = range numWeights {
			weight = rxfhindirWeight[j]
			sum += weight
		}

		if sum == 0 {
			return fmt.Errorf("at least one weight must be non-zero")
		}

		if sum > *indirSize {
			return fmt.Errorf("total weight exceeds the size of the indirection table")
		}

		j = ^uint32(0) // equivalent to -1 for unsigned
		for i := uint32(0); i < *indirSize; i++ {
			for i >= (*indirSize*partial)/sum {
				j++
				weight = rxfhindirWeight[j]
				partial += weight
			}
			indir[i] = uint32(rxfhindirStart) + j
		}
	case rxfhindirDefault != 0:
		*indirSize = 0
	default:
		*indirSize = ETH_RXFH_INDIR_NO_CHANGE
	}
	return nil
}

func (e *Ethtool) getChannels(intf string) (Channels, error) {
	channels := Channels{
		Cmd: ETHTOOL_GCHANNELS,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&channels))); err != nil {
		return Channels{}, err
	}

	return channels, nil
}

func (e *Ethtool) setChannels(intf string, channels Channels) (Channels, error) {
	channels.Cmd = ETHTOOL_SCHANNELS

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&channels))); err != nil {
		return Channels{}, err
	}

	return channels, nil
}

func (e *Ethtool) getCoalesce(intf string) (Coalesce, error) {
	coalesce := Coalesce{
		Cmd: ETHTOOL_GCOALESCE,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&coalesce))); err != nil {
		return Coalesce{}, err
	}

	return coalesce, nil
}

func (e *Ethtool) setCoalesce(intf string, coalesce Coalesce) (Coalesce, error) {
	coalesce.Cmd = ETHTOOL_SCOALESCE

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&coalesce))); err != nil {
		return Coalesce{}, err
	}

	return coalesce, nil
}

func (e *Ethtool) getTimestampingInformation(intf string) (TimestampingInformation, error) {
	ts := TimestampingInformation{
		Cmd: ETHTOOL_GET_TS_INFO,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&ts))); err != nil {
		return TimestampingInformation{}, err
	}

	return ts, nil
}

func (e *Ethtool) getPermAddr(intf string) (ethtoolPermAddr, error) {
	permAddr := ethtoolPermAddr{
		cmd:  ETHTOOL_GPERMADDR,
		size: PERMADDR_LEN,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&permAddr))); err != nil {
		return ethtoolPermAddr{}, err
	}

	return permAddr, nil
}

func (e *Ethtool) getModuleEeprom(intf string) (ethtoolEeprom, ethtoolModInfo, error) {
	modInfo := ethtoolModInfo{
		cmd: ETHTOOL_GMODULEINFO,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&modInfo))); err != nil {
		return ethtoolEeprom{}, ethtoolModInfo{}, err
	}

	eeprom := ethtoolEeprom{
		cmd:    ETHTOOL_GMODULEEEPROM,
		len:    modInfo.eeprom_len,
		offset: 0,
	}

	if modInfo.eeprom_len > EEPROM_LEN {
		return ethtoolEeprom{}, ethtoolModInfo{}, fmt.Errorf("eeprom size: %d is larger than buffer size: %d", modInfo.eeprom_len, EEPROM_LEN)
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&eeprom))); err != nil {
		return ethtoolEeprom{}, ethtoolModInfo{}, err
	}

	return eeprom, modInfo, nil
}

// GetRing retrieves ring parameters of the given interface name.
func (e *Ethtool) GetRing(intf string) (Ring, error) {
	ring := Ring{
		Cmd: ETHTOOL_GRINGPARAM,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&ring))); err != nil {
		return Ring{}, err
	}

	return ring, nil
}

// SetRing sets ring parameters of the given interface name.
func (e *Ethtool) SetRing(intf string, ring Ring) (Ring, error) {
	ring.Cmd = ETHTOOL_SRINGPARAM

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&ring))); err != nil {
		return Ring{}, err
	}

	return ring, nil
}

// GetPause retrieves pause parameters of the given interface name.
func (e *Ethtool) GetPause(intf string) (Pause, error) {
	pause := Pause{
		Cmd: ETHTOOL_GPAUSEPARAM,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&pause))); err != nil {
		return Pause{}, err
	}

	return pause, nil
}

// SetPause sets pause parameters of the given interface name.
func (e *Ethtool) SetPause(intf string, pause Pause) (Pause, error) {
	pause.Cmd = ETHTOOL_SPAUSEPARAM

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&pause))); err != nil {
		return Pause{}, err
	}

	return pause, nil
}

func isFeatureBitSet(blocks [MAX_FEATURE_BLOCKS]ethtoolGetFeaturesBlock, index uint) bool {
	return (blocks)[index/32].active&(1<<(index%32)) != 0
}

// FeatureState contains the state of a feature.
type FeatureState struct {
	Available    bool
	Requested    bool
	Active       bool
	NeverChanged bool
}

func getFeatureStateBits(blocks [MAX_FEATURE_BLOCKS]ethtoolGetFeaturesBlock, index uint) FeatureState {
	return FeatureState{
		Available:    (blocks)[index/32].available&(1<<(index%32)) != 0,
		Requested:    (blocks)[index/32].requested&(1<<(index%32)) != 0,
		Active:       (blocks)[index/32].active&(1<<(index%32)) != 0,
		NeverChanged: (blocks)[index/32].never_changed&(1<<(index%32)) != 0,
	}
}

func setFeatureBit(blocks *[MAX_FEATURE_BLOCKS]ethtoolSetFeaturesBlock, index uint, value bool) {
	blockIndex, bitIndex := index/32, index%32

	blocks[blockIndex].valid |= 1 << bitIndex

	if value {
		blocks[blockIndex].requested |= 1 << bitIndex
	} else {
		blocks[blockIndex].requested &= ^(1 << bitIndex)
	}
}

func (e *Ethtool) getNames(intf string, mask int) (map[string]uint, error) {
	ssetInfo := ethtoolSsetInfo{
		cmd:       ETHTOOL_GSSET_INFO,
		sset_mask: 1 << mask,
		data:      [MAX_SSET_INFO]uint32{},
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&ssetInfo))); err != nil {
		return nil, err
	}

	/* we only read data on first index because single bit was set in sset_mask(0x10) */
	length := ssetInfo.data[0]
	if length == 0 {
		return map[string]uint{}, nil
	} else if length > MAX_GSTRINGS {
		return nil, fmt.Errorf("ethtool currently doesn't support more than %d entries, received %d", MAX_GSTRINGS, length)
	}

	gstrings := EthtoolGStrings{
		cmd:        ETHTOOL_GSTRINGS,
		string_set: uint32(mask),
		len:        length,
		data:       [MAX_GSTRINGS * ETH_GSTRING_LEN]byte{},
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&gstrings))); err != nil {
		return nil, err
	}

	result := make(map[string]uint)
	for i := 0; i != int(length); i++ {
		b := gstrings.data[i*ETH_GSTRING_LEN : i*ETH_GSTRING_LEN+ETH_GSTRING_LEN]
		key := goString(b)
		if key != "" {
			result[key] = uint(i)
		}
	}

	return result, nil
}

// FeatureNames shows supported features by their name.
func (e *Ethtool) FeatureNames(intf string) (map[string]uint, error) {
	return e.getNames(intf, ETH_SS_FEATURES)
}

// Features retrieves features of the given interface name.
func (e *Ethtool) Features(intf string) (map[string]bool, error) {
	names, err := e.FeatureNames(intf)
	if err != nil {
		return nil, err
	}

	length := uint32(len(names))
	if length == 0 {
		return map[string]bool{}, nil
	}

	features := ethtoolGfeatures{
		cmd:  ETHTOOL_GFEATURES,
		size: (length + 32 - 1) / 32,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&features))); err != nil {
		return nil, err
	}

	result := make(map[string]bool, length)
	for key, index := range names {
		result[key] = isFeatureBitSet(features.blocks, index)
	}

	return result, nil
}

// FeaturesWithState retrieves features of the given interface name,
// with extra flags to explain if they can be enabled
func (e *Ethtool) FeaturesWithState(intf string) (map[string]FeatureState, error) {
	names, err := e.FeatureNames(intf)
	if err != nil {
		return nil, err
	}

	length := uint32(len(names))
	if length == 0 {
		return map[string]FeatureState{}, nil
	}

	features := ethtoolGfeatures{
		cmd:  ETHTOOL_GFEATURES,
		size: (length + 32 - 1) / 32,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&features))); err != nil {
		return nil, err
	}

	var result = make(map[string]FeatureState, length)
	for key, index := range names {
		result[key] = getFeatureStateBits(features.blocks, index)
	}

	return result, nil
}

// Change requests a change in the given device's features.
func (e *Ethtool) Change(intf string, config map[string]bool) error {
	names, err := e.FeatureNames(intf)
	if err != nil {
		return err
	}

	length := uint32(len(names))

	features := ethtoolSfeatures{
		cmd:  ETHTOOL_SFEATURES,
		size: (length + 32 - 1) / 32,
	}

	for key, value := range config {
		if index, ok := names[key]; ok {
			setFeatureBit(&features.blocks, index, value)
		} else {
			return fmt.Errorf("unsupported feature %q", key)
		}
	}

	return e.ioctl(intf, uintptr(unsafe.Pointer(&features)))
}

// PrivFlagsNames shows supported private flags by their name.
func (e *Ethtool) PrivFlagsNames(intf string) (map[string]uint, error) {
	return e.getNames(intf, ETH_SS_PRIV_FLAGS)
}

// PrivFlags retrieves private flags of the given interface name.
func (e *Ethtool) PrivFlags(intf string) (map[string]bool, error) {
	names, err := e.PrivFlagsNames(intf)
	if err != nil {
		return nil, err
	}

	length := uint32(len(names))
	if length == 0 {
		return map[string]bool{}, nil
	}

	var val ethtoolLink
	val.cmd = ETHTOOL_GPFLAGS
	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&val))); err != nil {
		return nil, err
	}

	result := make(map[string]bool, length)
	for name, mask := range names {
		result[name] = val.data&(1<<mask) != 0
	}

	return result, nil
}

// UpdatePrivFlags requests a change in the given device's private flags.
func (e *Ethtool) UpdatePrivFlags(intf string, config map[string]bool) error {
	names, err := e.PrivFlagsNames(intf)
	if err != nil {
		return err
	}

	var curr ethtoolLink
	curr.cmd = ETHTOOL_GPFLAGS
	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&curr))); err != nil {
		return err
	}

	var update ethtoolLink
	update.cmd = ETHTOOL_SPFLAGS
	update.data = curr.data
	for name, value := range config {
		if index, ok := names[name]; ok {
			if value {
				update.data |= 1 << index
			} else {
				update.data &= ^(1 << index)
			}
		} else {
			return fmt.Errorf("unsupported priv flag %q", name)
		}
	}

	return e.ioctl(intf, uintptr(unsafe.Pointer(&update)))
}

// LinkState get the state of a link.
func (e *Ethtool) LinkState(intf string) (uint32, error) {
	x := ethtoolLink{
		cmd: ETHTOOL_GLINK,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&x))); err != nil {
		return 0, err
	}

	return x.data, nil
}

// Stats retrieves stats of the given interface name.
// This maintains backward compatibility with existing code.
func (e *Ethtool) Stats(intf string) (map[string]uint64, error) {
	// Create temporary buffers and delegate to StatsWithBuffer
	gstrings := gstringsPool.Get().(*EthtoolGStrings)
	stats := statsPool.Get().(*EthtoolStats)
	defer func() {
		gstringsPool.Put(gstrings)
		statsPool.Put(stats)
	}()

	return e.StatsWithBuffer(intf, gstrings, stats)
}

// StatsWithBuffer retrieves stats of the given interface name using pre-allocated buffers.
// This allows the caller to control where the large structures are allocated,
// which can be useful to avoid heap allocations in Go 1.24+.
func (e *Ethtool) StatsWithBuffer(intf string, gstringsPtr *EthtoolGStrings, statsPtr *EthtoolStats) (map[string]uint64, error) {
	drvinfo := ethtoolDrvInfo{
		cmd: ETHTOOL_GDRVINFO,
	}

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(&drvinfo))); err != nil {
		return nil, err
	}

	if drvinfo.n_stats > MAX_GSTRINGS {
		return nil, fmt.Errorf("ethtool currently doesn't support more than %d entries, received %d", MAX_GSTRINGS, drvinfo.n_stats)
	}

	gstringsPtr.cmd = ETHTOOL_GSTRINGS
	gstringsPtr.string_set = ETH_SS_STATS
	gstringsPtr.len = drvinfo.n_stats

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(gstringsPtr))); err != nil {
		return nil, err
	}

	statsPtr.cmd = ETHTOOL_GSTATS
	statsPtr.n_stats = drvinfo.n_stats

	if err := e.ioctl(intf, uintptr(unsafe.Pointer(statsPtr))); err != nil {
		return nil, err
	}

	result := make(map[string]uint64, drvinfo.n_stats)
	for i := 0; i != int(drvinfo.n_stats); i++ {
		b := gstringsPtr.data[i*ETH_GSTRING_LEN : (i+1)*ETH_GSTRING_LEN]

		strEnd := bytes.IndexByte(b, 0)
		if strEnd == -1 {
			strEnd = ETH_GSTRING_LEN
		}
		key := string(b[:strEnd])

		if len(key) != 0 {
			result[key] = statsPtr.data[i]
		}
	}

	return result, nil
}

// Close closes the ethool handler
func (e *Ethtool) Close() {
	unix.Close(e.fd)
}

// Identity the nic with blink duration, if not specify blink for 60 seconds
func (e *Ethtool) Identity(intf string, duration *time.Duration) error {
	dur := uint32(DEFAULT_BLINK_DURATION.Seconds())
	if duration != nil {
		dur = uint32(duration.Seconds())
	}
	return e.identity(intf, IdentityConf{Duration: dur})
}

func (e *Ethtool) identity(intf string, identity IdentityConf) error {
	identity.Cmd = ETHTOOL_PHYS_ID
	return e.ioctl(intf, uintptr(unsafe.Pointer(&identity)))
}

// NewEthtool returns a new ethtool handler
func NewEthtool() (*Ethtool, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM|unix.SOCK_CLOEXEC, unix.IPPROTO_IP)
	if err != nil {
		return nil, err
	}

	return &Ethtool{
		fd: int(fd),
	}, nil
}

// BusInfo returns bus information of the given interface name.
func BusInfo(intf string) (string, error) {
	e, err := NewEthtool()
	if err != nil {
		return "", err
	}
	defer e.Close()
	return e.BusInfo(intf)
}

// DriverName returns the driver name of the given interface name.
func DriverName(intf string) (string, error) {
	e, err := NewEthtool()
	if err != nil {
		return "", err
	}
	defer e.Close()
	return e.DriverName(intf)
}

// Stats retrieves stats of the given interface name.
func Stats(intf string) (map[string]uint64, error) {
	e, err := NewEthtool()
	if err != nil {
		return nil, err
	}
	defer e.Close()
	return e.Stats(intf)
}

// PermAddr returns permanent address of the given interface name.
func PermAddr(intf string) (string, error) {
	e, err := NewEthtool()
	if err != nil {
		return "", err
	}
	defer e.Close()
	return e.PermAddr(intf)
}

// Identity the nic with blink duration, if not specify blink infinity
func Identity(intf string, duration *time.Duration) error {
	e, err := NewEthtool()
	if err != nil {
		return err
	}
	defer e.Close()
	return e.Identity(intf, duration)
}

func supportedSpeeds(mask uint64) (ret []struct {
	name  string
	mask  uint64
	speed uint64
}) {
	for _, mode := range supportedCapabilities {
		if mode.speed > 0 && ((1<<mode.mask)&mask) != 0 {
			ret = append(ret, mode)
		}
	}
	return ret
}

// SupportedLinkModes returns the names of the link modes supported by the interface.
func SupportedLinkModes(mask uint64) []string {
	var ret []string
	for _, mode := range supportedSpeeds(mask) {
		ret = append(ret, mode.name)
	}
	return ret
}

// SupportedSpeed returns the maximum capacity of this interface.
func SupportedSpeed(mask uint64) uint64 {
	var ret uint64
	for _, mode := range supportedSpeeds(mask) {
		if mode.speed > ret {
			ret = mode.speed
		}
	}
	return ret
}
