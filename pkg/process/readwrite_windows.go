//go:build windows

package process

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func (p *Process) readExact(addr uintptr, buf []byte) error {
	var read uintptr
	if err := windows.ReadProcessMemory(p.Handle, addr, &buf[0], uintptr(len(buf)), &read); err != nil {
		return err
	}
	if read != uintptr(len(buf)) {
		return fmt.Errorf("short read: %d", read)
	}
	return nil
}

func (p *Process) writeExact(addr uintptr, buf []byte) error {
	var written uintptr
	if err := windows.WriteProcessMemory(p.Handle, addr, &buf[0], uintptr(len(buf)), &written); err != nil {
		return err
	}
	if written != uintptr(len(buf)) {
		return fmt.Errorf("short write: %d", written)
	}
	return nil
}
