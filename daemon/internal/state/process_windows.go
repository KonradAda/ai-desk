//go:build windows

package state

import "syscall"

// processAlive reports whether a process with this id is running.
func processAlive(pid int) bool {
	const queryLimitedInformation = 0x1000 // PROCESS_QUERY_LIMITED_INFORMATION
	const stillActive = 259                // STILL_ACTIVE exit code
	if pid <= 0 {
		return false
	}
	h, err := syscall.OpenProcess(queryLimitedInformation, false, uint32(pid))
	if err != nil {
		return false
	}
	defer syscall.CloseHandle(h)
	var code uint32
	return syscall.GetExitCodeProcess(h, &code) == nil && code == stillActive
}
