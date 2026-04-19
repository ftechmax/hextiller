//go:build linux

package process

import (
	"os"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func openSelf(t *testing.T) *Process {
	t.Helper()
	p, err := Open(uint32(os.Getpid()))
	if err != nil {
		t.Fatalf("open self: %v", err)
	}
	t.Cleanup(func() { _ = p.Close() })
	return p
}

func allocRW(t *testing.T, size uintptr) (uintptr, []byte) {
	t.Helper()
	data, err := unix.Mmap(-1, 0, int(size), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANON|unix.MAP_PRIVATE)
	if err != nil {
		t.Fatalf("Mmap: %v", err)
	}
	t.Cleanup(func() { _ = unix.Munmap(data) })
	return uintptr(unsafe.Pointer(&data[0])), data
}

func containsAddress(addrs []uintptr, target uintptr) bool {
	for _, a := range addrs {
		if a == target {
			return true
		}
	}
	return false
}
