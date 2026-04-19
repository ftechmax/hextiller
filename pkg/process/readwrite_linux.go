//go:build linux

package process

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func (p *Process) readExact(addr uintptr, buf []byte) error {
	local := []unix.Iovec{{Base: &buf[0]}}
	local[0].SetLen(len(buf))
	remote := []unix.RemoteIovec{{Base: addr, Len: len(buf)}}
	n, err := unix.ProcessVMReadv(int(p.PID), local, remote, 0)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("short read: %d", n)
	}
	return nil
}

func (p *Process) writeExact(addr uintptr, buf []byte) error {
	local := []unix.Iovec{{Base: &buf[0]}}
	local[0].SetLen(len(buf))
	remote := []unix.RemoteIovec{{Base: addr, Len: len(buf)}}
	n, err := unix.ProcessVMWritev(int(p.PID), local, remote, 0)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("short write: %d", n)
	}
	return nil
}
