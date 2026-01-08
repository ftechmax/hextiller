//go:build windows

package process

import (
	"os"
	"testing"

	"golang.org/x/sys/windows"
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

func allocRW(t *testing.T, size uintptr) uintptr {
	t.Helper()
	addr, err := windows.VirtualAlloc(0, size, windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil || addr == 0 {
		t.Fatalf("VirtualAlloc: %v", err)
	}
	t.Cleanup(func() { _ = windows.VirtualFree(addr, 0, windows.MEM_RELEASE) })
	return addr
}

func containsAddress(addrs []uintptr, target uintptr) bool {
	for _, a := range addrs {
		if a == target {
			return true
		}
	}
	return false
}
