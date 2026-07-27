//go:build windows

package docker

import (
	"syscall"
	"unsafe"
)

type MEMORYSTATUSEX struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

func getPlatformMemory() (float64, float64) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	globalMemoryStatusEx := kernel32.NewProc("GlobalMemoryStatusEx")

	var memStatus MEMORYSTATUSEX
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))

	ret, _, _ := globalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret != 0 {
		totalMB := MathRound(float64(memStatus.ullTotalPhys)/1024/1024, 0)
		availMB := float64(memStatus.ullAvailPhys) / 1024 / 1024
		usedMB := MathRound(totalMB-availMB, 0)
		return totalMB, usedMB
	}

	return 4096.0, 512.0
}
