//go:build !windows

package state

import (
	"os"
	"syscall"
)

// processAlive reports whether a process with this id is running.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	p, err := os.FindProcess(pid)
	return err == nil && p.Signal(syscall.Signal(0)) == nil
}
