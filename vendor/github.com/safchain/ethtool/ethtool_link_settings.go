package ethtool

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

type LinkSettingSource string

// Constants defining the source of the LinkSettings data
const (
	SourceGLinkSettings LinkSettingSource = "GLINKSETTINGS"
	SourceGSet          LinkSettingSource = "GSET"
)

// EthtoolLinkSettingsFixed corresponds to struct ethtool_link_settings fixed part
type EthtoolLinkSettingsFixed struct {
	Cmd                 uint32
	Speed               uint32
	Duplex              uint8
	Port                uint8
	PhyAddress          uint8
	Autoneg             uint8
	MdixSupport         uint8 // Renamed from mdio_support
	EthTpMdix           uint8
	EthTpMdixCtrl       uint8
	LinkModeMasksNwords int8
	Transceiver         uint8
	MasterSlaveCfg      uint8
	MasterSlaveState    uint8
	Reserved1           [1]byte
	Reserved            [7]uint32
	// Flexible array member link_mode_masks[0] starts here implicitly
}

// ethtoolLinkSettingsRequest includes space for the flexible array members
type ethtoolLinkSettingsRequest struct {
	Settings EthtoolLinkSettingsFixed
	Masks    [3 * MAX_LINK_MODE_MASK_NWORDS]uint32 // Uses MAX_LINK_MODE_MASK_NWORDS constant from ethtool.go
}

// LinkSettings is the user-friendly representation returned by GetLinkSettings
type LinkSettings struct {
	Speed                uint32
	Duplex               uint8
	Port                 uint8
	PhyAddress           uint8
	Autoneg              uint8
	MdixSupport          uint8
	EthTpMdix            uint8
	EthTpMdixCtrl        uint8
	Transceiver          uint8
	MasterSlaveCfg       uint8
	MasterSlaveState     uint8
	SupportedLinkModes   []string
	AdvertisingLinkModes []string
	LpAdvertisingModes   []string
	Source               LinkSettingSource // "GSET" or "GLINKSETTINGS"
}

// GetLinkSettings retrieves link settings, preferring ETHTOOL_GLINKSETTINGS and falling back to ETHTOOL_GSET.
// Uses a single ioctl call with the maximum expected buffer size.
func (e *Ethtool) GetLinkSettings(intf string) (*LinkSettings, error) {
	// 1. Attempt ETHTOOL_GLINKSETTINGS with max buffer size
	var req ethtoolLinkSettingsRequest
	req.Settings.Cmd = ETHTOOL_GLINKSETTINGS
	// Provide the maximum expected nwords based on our constant
	req.Settings.LinkModeMasksNwords = int8(MAX_LINK_MODE_MASK_NWORDS)

	err := e.ioctl(intf, uintptr(unsafe.Pointer(&req)))
	fallbackReason := ""

	var errno syscall.Errno
	switch {
	case errors.As(err, &errno) && errors.Is(errno, unix.EOPNOTSUPP):
		// Condition 1: ioctl returned EOPNOTSUPP
		fallbackReason = "EOPNOTSUPP"
	case err == nil:
		// Condition 2: ioctl succeeded, but nwords might be invalid or buffer too small
		nwords := int(req.Settings.LinkModeMasksNwords)
		switch {
		case nwords <= 0 || nwords > MAX_LINK_MODE_MASK_NWORDS:
			// Sub-case 2a: Invalid nwords -> fallback
			fmt.Printf("Warning: GLINKSETTINGS succeeded but returned invalid nwords (%d), attempting fallback to GSET\n", nwords)
			fallbackReason = "invalid nwords from GLINKSETTINGS"
		case 3*nwords > len(req.Masks):
			// Sub-case 2b: Buffer too small -> error
			return nil, fmt.Errorf("kernel requires %d words for GLINKSETTINGS, buffer only has space for %d (max %d)", nwords, len(req.Masks)/3, MAX_LINK_MODE_MASK_NWORDS)
		default:
			// Sub-case 2c: Success (nwords valid and buffer sufficient)
			results := &LinkSettings{
				Speed:                req.Settings.Speed,
				Duplex:               req.Settings.Duplex,
				Port:                 req.Settings.Port,
				PhyAddress:           req.Settings.PhyAddress,
				Autoneg:              req.Settings.Autoneg,
				MdixSupport:          req.Settings.MdixSupport,
				EthTpMdix:            req.Settings.EthTpMdix,
				EthTpMdixCtrl:        req.Settings.EthTpMdixCtrl,
				Transceiver:          req.Settings.Transceiver,
				MasterSlaveCfg:       req.Settings.MasterSlaveCfg,
				MasterSlaveState:     req.Settings.MasterSlaveState,
				SupportedLinkModes:   parseLinkModeMasks(req.Masks[0*nwords : 1*nwords]),
				AdvertisingLinkModes: parseLinkModeMasks(req.Masks[1*nwords : 2*nwords]),
				LpAdvertisingModes:   parseLinkModeMasks(req.Masks[2*nwords : 3*nwords]),
				Source:               SourceGLinkSettings,
			}
			return results, nil
		}
	default:
		// Condition 3: ioctl failed with an error other than EOPNOTSUPP
		// No fallback in this case.
		return nil, fmt.Errorf("ETHTOOL_GLINKSETTINGS ioctl failed: %w", err)
	}

	// Fallback to ETHTOOL_GSET using e.CmdGet
	var cmd EthtoolCmd
	_, errGet := e.CmdGet(&cmd, intf)
	if errGet != nil {
		return nil, fmt.Errorf("ETHTOOL_GLINKSETTINGS failed (%s), fallback ETHTOOL_GSET (CmdGet) also failed: %w", fallbackReason, errGet)
	}
	results := convertCmdToLinkSettings(&cmd)
	results.Source = SourceGSet
	return results, nil
}

// SetLinkSettings applies link settings, determining whether to use ETHTOOL_SLINKSETTINGS or ETHTOOL_SSET.
func (e *Ethtool) SetLinkSettings(intf string, settings *LinkSettings) error {
	var checkReq ethtoolLinkSettingsRequest
	checkReq.Settings.Cmd = ETHTOOL_GLINKSETTINGS
	checkReq.Settings.LinkModeMasksNwords = int8(MAX_LINK_MODE_MASK_NWORDS)

	errGLinkSettings := e.ioctl(intf, uintptr(unsafe.Pointer(&checkReq)))
	canUseGLinkSettings := false
	nwords := 0

	if errGLinkSettings == nil {
		nwords = int(checkReq.Settings.LinkModeMasksNwords)
		if nwords <= 0 || nwords > MAX_LINK_MODE_MASK_NWORDS {
			return fmt.Errorf("ETHTOOL_GLINKSETTINGS check succeeded but returned invalid nwords: %d", nwords)
		}
		canUseGLinkSettings = true
	} else {
		var errno syscall.Errno
		if !errors.As(errGLinkSettings, &errno) || !errors.Is(errno, unix.EOPNOTSUPP) {
			return fmt.Errorf("checking support via ETHTOOL_GLINKSETTINGS failed: %w", errGLinkSettings)
		}
	}

	if canUseGLinkSettings {
		var setReq ethtoolLinkSettingsRequest
		if 3*nwords > len(setReq.Masks) {
			return fmt.Errorf("internal error: required nwords (%d) exceeds allocated buffer (%d)", nwords, MAX_LINK_MODE_MASK_NWORDS)
		}
		setReq.Settings.Cmd = ETHTOOL_SLINKSETTINGS
		setReq.Settings.Speed = settings.Speed
		setReq.Settings.Duplex = settings.Duplex
		setReq.Settings.Port = settings.Port
		setReq.Settings.PhyAddress = settings.PhyAddress
		setReq.Settings.Autoneg = settings.Autoneg
		setReq.Settings.EthTpMdixCtrl = settings.EthTpMdixCtrl
		setReq.Settings.MasterSlaveCfg = settings.MasterSlaveCfg
		setReq.Settings.LinkModeMasksNwords = int8(nwords)

		advertisingMask := buildLinkModeMask(settings.AdvertisingLinkModes, nwords)
		if len(advertisingMask) != nwords {
			return fmt.Errorf("failed to build advertising mask with correct size (%d != %d)", len(advertisingMask), nwords)
		}
		copy(setReq.Masks[nwords:2*nwords], advertisingMask)
		zeroMaskSupported := make([]uint32, nwords)
		zeroMaskLp := make([]uint32, nwords)
		copy(setReq.Masks[0*nwords:1*nwords], zeroMaskSupported)
		copy(setReq.Masks[2*nwords:3*nwords], zeroMaskLp)

		if err := e.ioctl(intf, uintptr(unsafe.Pointer(&setReq))); err != nil {
			return fmt.Errorf("ETHTOOL_SLINKSETTINGS ioctl failed: %w", err)
		}
		return nil

	}
	// Check if trying to set high bits when only SSET is available
	advertisingMaskCheck := buildLinkModeMask(settings.AdvertisingLinkModes, MAX_LINK_MODE_MASK_NWORDS)
	for i := 1; i < len(advertisingMaskCheck); i++ {
		if advertisingMaskCheck[i] != 0 {
			return fmt.Errorf("cannot set link modes beyond 32 bits using legacy ETHTOOL_SSET; device does not support ETHTOOL_SLINKSETTINGS")
		}
	}

	// Fallback to SSET
	cmd := convertLinkSettingsToCmd(settings)
	_, errSet := e.CmdSet(cmd, intf)
	if errSet != nil {
		return fmt.Errorf("ETHTOOL_SLINKSETTINGS not supported, fallback ETHTOOL_SSET (CmdSet) failed: %w", errSet)
	}
	return nil
}

// parseLinkModeMasks converts a slice of uint32 bitmasks to a list of mode names.
// It filters out non-speed/duplex modes (like TP, Autoneg, Pause).
func parseLinkModeMasks(mask []uint32) []string {
	modes := make([]string, 0, 8)
	for _, capability := range supportedCapabilities {
		// Only include capabilities that represent a speed/duplex mode
		if capability.speed > 0 {
			bitIndex := int(capability.mask)
			wordIndex := bitIndex / 32
			bitInWord := uint(bitIndex % 32)
			if wordIndex < len(mask) && (mask[wordIndex]>>(bitInWord))&1 != 0 {
				modes = append(modes, capability.name)
			}
		}
	}
	return modes
}

// buildLinkModeMask converts a list of mode names back into a uint32 bitmask slice.
// It filters out non-speed/duplex modes.
func buildLinkModeMask(modes []string, nwords int) []uint32 {
	if nwords <= 0 || nwords > MAX_LINK_MODE_MASK_NWORDS {
		return make([]uint32, 0)
	}
	mask := make([]uint32, nwords)
	modeMap := make(map[string]struct {
		bitIndex int
		speed    uint64
	})
	for _, capability := range supportedCapabilities {
		// Only consider capabilities that represent a speed/duplex mode
		if capability.speed > 0 {
			modeMap[capability.name] = struct {
				bitIndex int
				speed    uint64
			}{bitIndex: int(capability.mask), speed: capability.speed}
		}
	}
	for _, modeName := range modes {
		if info, ok := modeMap[strings.TrimSpace(modeName)]; ok {
			wordIndex := info.bitIndex / 32
			bitInWord := uint(info.bitIndex % 32)
			if wordIndex < nwords {
				mask[wordIndex] |= 1 << bitInWord
			} else {
				fmt.Printf("Warning: Link mode '%s' (bit %d) exceeds device's mask size (%d words)\n", modeName, info.bitIndex, nwords)
			}
		} else {
			// Check if the user provided a non-speed mode name - ignore it for the mask, maybe warn?
			isKnownNonSpeed := false
			for _, capability := range supportedCapabilities {
				if capability.speed == 0 && capability.name == strings.TrimSpace(modeName) {
					isKnownNonSpeed = true
					break
				}
			}
			if !isKnownNonSpeed {
				fmt.Printf("Warning: Unknown link mode '%s' specified for mask building\n", modeName)
			} // Silently ignore known non-speed modes like Autoneg, TP, Pause for the mask
		}
	}
	return mask
}

// convertCmdToLinkSettings converts data from the legacy EthtoolCmd to the new LinkSettings format.
func convertCmdToLinkSettings(cmd *EthtoolCmd) *LinkSettings {
	ls := &LinkSettings{
		Speed:                (uint32(cmd.Speed_hi) << 16) | uint32(cmd.Speed),
		Duplex:               cmd.Duplex,
		Port:                 cmd.Port,
		PhyAddress:           cmd.Phy_address,
		Autoneg:              cmd.Autoneg,
		MdixSupport:          cmd.Mdio_support,
		EthTpMdix:            cmd.Eth_tp_mdix,
		EthTpMdixCtrl:        ETH_TP_MDI_INVALID,
		Transceiver:          cmd.Transceiver,
		MasterSlaveCfg:       0, // No equivalent in EthtoolCmd
		MasterSlaveState:     0, // No equivalent in EthtoolCmd
		SupportedLinkModes:   parseLegacyLinkModeMask(cmd.Supported),
		AdvertisingLinkModes: parseLegacyLinkModeMask(cmd.Advertising),
		LpAdvertisingModes:   parseLegacyLinkModeMask(cmd.Lp_advertising),
	}
	if cmd.Speed == math.MaxUint16 && cmd.Speed_hi == math.MaxUint16 {
		ls.Speed = SPEED_UNKNOWN // GSET uses 0xFFFF/0xFFFF for unknown/auto
	}
	return ls
}

// parseLegacyLinkModeMask helper for converting single uint32 mask.
func parseLegacyLinkModeMask(mask uint32) []string {
	return parseLinkModeMasks([]uint32{mask})
}

// convertLinkSettingsToCmd converts new LinkSettings data back to the legacy EthtoolCmd format for SSET fallback.
func convertLinkSettingsToCmd(ls *LinkSettings) *EthtoolCmd {
	cmd := &EthtoolCmd{}
	if ls.Speed == 0 || ls.Speed == SPEED_UNKNOWN {
		cmd.Speed = math.MaxUint16
		cmd.Speed_hi = math.MaxUint16
	} else {
		cmd.Speed = uint16(ls.Speed & 0xFFFF)
		cmd.Speed_hi = uint16((ls.Speed >> 16) & 0xFFFF)
	}
	cmd.Duplex = ls.Duplex
	cmd.Port = ls.Port
	cmd.Phy_address = ls.PhyAddress
	cmd.Autoneg = ls.Autoneg
	// Cannot set EthTpMdixCtrl via EthtoolCmd
	cmd.Transceiver = ls.Transceiver
	cmd.Advertising = buildLegacyLinkModeMask(ls.AdvertisingLinkModes)
	return cmd
}

// buildLegacyLinkModeMask helper for building single uint32 mask from names.
func buildLegacyLinkModeMask(modes []string) uint32 {
	maskSlice := buildLinkModeMask(modes, 1)
	if len(maskSlice) > 0 {
		return maskSlice[0]
	}
	return 0
}
