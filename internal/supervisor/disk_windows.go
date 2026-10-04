//go:build windows

package supervisor

import (
	"math"
	"syscall"
	"unsafe"
)

var (
	kernel32            = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceExW = kernel32.NewProc("GetDiskFreeSpaceExW")
)

func getDiskUsage(path string) (total uint64, free uint64, used uint64, percent float64, err error) {
	ptr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, 0, 0, err
	}

	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	r1, _, errSys := getDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)
	if r1 == 0 {
		return 0, 0, 0, 0, errSys
	}

	total = totalNumberOfBytes
	free = freeBytesAvailable
	if total > free {
		used = total - free
	}
	if total > 0 {
		percent = math.Round((float64(used)/float64(total)*100)*10) / 10
	}
	return total, free, used, percent, nil
}
