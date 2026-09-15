package crack

import (
	"fmt"
	"strconv"

	"github.com/bishopfox/sliver/client/console"
	"github.com/bishopfox/sliver/protobuf/clientpb"
	"github.com/jedib0t/go-pretty/v6/table"
)

type backendDeviceDetails struct {
	deviceType          string
	vendor              string
	vendorID            int32
	version             string
	runtimeVersion      string
	deviceID            uint32
	deviceAlias         *uint32
	preferredThreadSize *uint32
	memoryUnified       *bool
	localMemory         string
	cacheSize           string
	memoryAllocPerBlock string
	pciAddress          string
}

func hasReportedBackendDevice(cracker *clientpb.Crackstation) bool {
	for _, device := range cracker.CUDA {
		if device != nil {
			return true
		}
	}
	for _, device := range cracker.HIP {
		if device != nil {
			return true
		}
	}
	for _, device := range cracker.Metal {
		if device != nil {
			return true
		}
	}
	for _, device := range cracker.OpenCL {
		if device != nil {
			return true
		}
	}
	return false
}

func appendCrackstationDetailRows(tw table.Writer, cracker *clientpb.Crackstation) {
	if cracker.HostUUID != "" {
		tw.AppendRow(table.Row{console.StyleBold.Render("Host UUID"), safeCrackCell(cracker.HostUUID)})
	}
	if cracker.Version != "" {
		tw.AppendRow(table.Row{console.StyleBold.Render("Crackstation Version"), safeCrackCell(cracker.Version)})
	}
}

func appendCUDABackendRows(tw table.Writer, device *clientpb.CUDABackendInfo, level uint32) {
	if device == nil {
		return
	}
	appendBackendSummaryRows(tw, "CUDA Device", formatCUDADevice(device), device.MemoryFree, device.MemoryTotal, device.Clock, device.Processors)
	if level < 2 {
		return
	}
	appendBackendDeviceDetailRows(tw, backendDeviceDetails{
		deviceType:          device.Type,
		vendor:              device.Vendor,
		vendorID:            device.VendorID,
		version:             device.Version,
		runtimeVersion:      device.CUDAVersion,
		deviceID:            device.BackendDeviceID,
		deviceAlias:         device.BackendDeviceAlias,
		preferredThreadSize: device.PreferredThreadSize,
		memoryUnified:       device.MemoryUnified,
		localMemory:         device.LocalMemory,
		cacheSize:           device.CacheSize,
		pciAddress:          device.PCIAddress,
	})
}

func appendHIPBackendRows(tw table.Writer, device *clientpb.HIPBackendInfo, level uint32) {
	if device == nil {
		return
	}
	appendBackendSummaryRows(tw, "HIP Device", formatHIPDevice(device), device.MemoryFree, device.MemoryTotal, device.Clock, device.Processors)
	if level < 2 {
		return
	}
	appendBackendDeviceDetailRows(tw, backendDeviceDetails{
		deviceType:          device.Type,
		vendor:              device.Vendor,
		vendorID:            device.VendorID,
		version:             device.Version,
		runtimeVersion:      device.HIPVersion,
		deviceID:            device.BackendDeviceID,
		deviceAlias:         device.BackendDeviceAlias,
		preferredThreadSize: device.PreferredThreadSize,
		memoryUnified:       device.MemoryUnified,
		localMemory:         device.LocalMemory,
		cacheSize:           device.CacheSize,
		pciAddress:          device.PCIAddress,
	})
}

func appendMetalBackendRows(tw table.Writer, device *clientpb.MetalBackendInfo, level uint32) {
	if device == nil {
		return
	}
	appendBackendSummaryRows(tw, "Metal Device", formatMetalDevice(device), device.MemoryFree, device.MemoryTotal, device.Clock, device.Processors)
	if level < 2 {
		return
	}
	appendBackendDeviceDetailRows(tw, backendDeviceDetails{
		deviceType:          device.Type,
		vendor:              device.Vendor,
		vendorID:            device.VendorID,
		version:             device.Version,
		runtimeVersion:      device.MetalVersion,
		deviceID:            device.BackendDeviceID,
		deviceAlias:         device.BackendDeviceAlias,
		preferredThreadSize: device.PreferredThreadSize,
		memoryUnified:       device.MemoryUnified,
		localMemory:         device.LocalMemory,
		cacheSize:           device.CacheSize,
		memoryAllocPerBlock: device.MemoryAllocPerBlock,
	})
	appendStringBackendDetailRow(tw, "Physical Location", device.PhysicalLocation)
	appendUint32BackendDetailRow(tw, "Registry ID", device.RegistryID)
	appendStringBackendDetailRow(tw, "Max TX Rate", device.MaxTXRate)
	appendBoolBackendDetailRow(tw, "Headless", device.GPUHeadless)
	appendBoolBackendDetailRow(tw, "Low Power", device.GPULowPower)
	appendBoolBackendDetailRow(tw, "Removable", device.GPURemovable)
}

func appendOpenCLBackendRows(tw table.Writer, device *clientpb.OpenCLBackendInfo, level uint32) {
	if device == nil {
		return
	}
	appendBackendSummaryRows(tw, "OpenCL Device", formatOpenCLDevice(device), device.MemoryFree, device.MemoryTotal, device.Clock, device.Processors)
	if level < 2 {
		return
	}
	appendBackendDeviceDetailRows(tw, backendDeviceDetails{
		deviceType:          device.Type,
		vendor:              device.Vendor,
		vendorID:            device.VendorID,
		version:             device.Version,
		runtimeVersion:      device.OpenCLVersion,
		deviceID:            device.BackendDeviceID,
		deviceAlias:         device.BackendDeviceAlias,
		preferredThreadSize: device.PreferredThreadSize,
		memoryUnified:       device.MemoryUnified,
		localMemory:         device.LocalMemory,
		memoryAllocPerBlock: device.MemoryAllocPerBlock,
		pciAddress:          device.PCIAddress,
	})
	appendStringBackendDetailRow(tw, "OpenCL Driver", device.OpenCLDriverVersion)
	if device.PlatformID != 0 || device.PlatformVendor != "" || device.PlatformName != "" || device.PlatformVersion != "" {
		tw.AppendRow(table.Row{console.StyleBold.Render("OpenCL Platform ID"), strconv.FormatUint(uint64(device.PlatformID), 10)})
	}
	appendStringBackendDetailRow(tw, "OpenCL Platform Vendor", device.PlatformVendor)
	appendStringBackendDetailRow(tw, "OpenCL Platform", device.PlatformName)
	appendStringBackendDetailRow(tw, "OpenCL Platform Version", device.PlatformVersion)
}

func appendBackendSummaryRows(tw table.Writer, label, name, memoryFree, memoryTotal string, clock, processors int32) {
	tw.AppendSeparator()
	tw.AppendRow(table.Row{console.StyleBold.Render(label), console.StyleBoldGreen.Render(safeCrackCell(name))})
	tw.AppendRow(table.Row{console.StyleBold.Render("Memory"), fmt.Sprintf("%s free of %s", valueOrDash(memoryFree), valueOrDash(memoryTotal))})
	tw.AppendRow(table.Row{console.StyleBold.Render("Clock"), strconv.FormatInt(int64(clock), 10)})
	tw.AppendRow(table.Row{console.StyleBold.Render("Processors"), strconv.FormatInt(int64(processors), 10)})
}

func appendBackendDeviceDetailRows(tw table.Writer, details backendDeviceDetails) {
	appendStringBackendDetailRow(tw, "Device Type", details.deviceType)
	appendStringBackendDetailRow(tw, "Vendor", details.vendor)
	if details.vendorID != 0 {
		tw.AppendRow(table.Row{console.StyleBold.Render("Vendor ID"), strconv.FormatInt(int64(details.vendorID), 10)})
	}
	if details.version != "" && details.version != details.runtimeVersion {
		appendStringBackendDetailRow(tw, "Device Version", details.version)
	}
	tw.AppendRow(table.Row{console.StyleBold.Render("Backend Device ID"), strconv.FormatUint(uint64(details.deviceID), 10)})
	appendUint32BackendDetailRow(tw, "Backend Device Alias", details.deviceAlias)
	appendUint32BackendDetailRow(tw, "Preferred Thread Size", details.preferredThreadSize)
	appendBoolBackendDetailRow(tw, "Unified Memory", details.memoryUnified)
	appendStringBackendDetailRow(tw, "Local Memory", details.localMemory)
	appendStringBackendDetailRow(tw, "Cache Size", details.cacheSize)
	appendStringBackendDetailRow(tw, "Memory Allocation per Block", details.memoryAllocPerBlock)
	appendStringBackendDetailRow(tw, "PCI Address", details.pciAddress)
}

func appendStringBackendDetailRow(tw table.Writer, label, value string) {
	if value == "" {
		return
	}
	tw.AppendRow(table.Row{console.StyleBold.Render(label), safeCrackCell(value)})
}

func appendUint32BackendDetailRow(tw table.Writer, label string, value *uint32) {
	if value == nil {
		return
	}
	tw.AppendRow(table.Row{console.StyleBold.Render(label), strconv.FormatUint(uint64(*value), 10)})
}

func appendBoolBackendDetailRow(tw table.Writer, label string, value *bool) {
	if value == nil {
		return
	}
	tw.AppendRow(table.Row{console.StyleBold.Render(label), strconv.FormatBool(*value)})
}
