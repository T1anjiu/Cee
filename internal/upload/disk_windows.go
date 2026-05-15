//go:build windows

package upload

import (
	"syscall"
	"unsafe"
)

func getFreeDiskSyscall(path string) int64 {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpaceEx := kernel32.NewProc("GetDiskFreeSpaceExW")

	var freeBytesAvailableToCaller uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64

	pathPtr, _ := syscall.UTF16PtrFromString(path)
	_, _, _ = getDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailableToCaller)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)

	return int64(freeBytesAvailableToCaller)
}
