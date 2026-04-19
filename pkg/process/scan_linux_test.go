//go:build linux

package process

import (
	"encoding/binary"
	"testing"

	"golang.org/x/sys/unix"
)

func TestScanFindsWrittenValues(t *testing.T) {
	p := openSelf(t)
	base, _ := allocRW(t, 64)

	addrI32 := base
	addrU32 := base + 8
	addrI64 := base + 16
	addrU64 := base + 32
	addrF32 := base + 48
	addrF64 := base + 56

	const (
		valI32 = int32(0x12AB34CD)
		valU32 = uint32(0x89ABCDEF)
		valI64 = int64(0x1234567890ABCDEF)
		valU64 = uint64(0x0FEDCBA987654321)
		valF32 = float32(1234.25)
		valF64 = float64(98765.5)
	)

	if _, err := p.WriteInt32AndRead(addrI32, valI32); err != nil {
		t.Fatalf("write int32: %v", err)
	}
	if _, err := p.WriteUint32AndRead(addrU32, valU32); err != nil {
		t.Fatalf("write uint32: %v", err)
	}
	if _, err := p.WriteInt64AndRead(addrI64, valI64); err != nil {
		t.Fatalf("write int64: %v", err)
	}
	if _, err := p.WriteUint64AndRead(addrU64, valU64); err != nil {
		t.Fatalf("write uint64: %v", err)
	}
	if _, err := p.WriteFloat32AndRead(addrF32, valF32); err != nil {
		t.Fatalf("write float32: %v", err)
	}
	if _, err := p.WriteFloat64AndRead(addrF64, valF64); err != nil {
		t.Fatalf("write float64: %v", err)
	}

	if addrs, err := p.ScanInt32(valI32, 0, false); err != nil || !containsAddress(addrs, addrI32) {
		t.Fatalf("ScanInt32 missing addr %X err %v", addrI32, err)
	}
	if addrs, err := p.ScanUint32(valU32, 0, false); err != nil || !containsAddress(addrs, addrU32) {
		t.Fatalf("ScanUint32 missing addr %X err %v", addrU32, err)
	}
	if addrs, err := p.ScanInt64(valI64, 0, false); err != nil || !containsAddress(addrs, addrI64) {
		t.Fatalf("ScanInt64 missing addr %X err %v", addrI64, err)
	}
	if addrs, err := p.ScanUint64(valU64, 0, false); err != nil || !containsAddress(addrs, addrU64) {
		t.Fatalf("ScanUint64 missing addr %X err %v", addrU64, err)
	}
	if addrs, err := p.ScanFloat32Approx(valF32, 1e-4, 0, false); err != nil || !containsAddress(addrs, addrF32) {
		t.Fatalf("ScanFloat32Approx missing addr %X err %v", addrF32, err)
	}
	if addrs, err := p.ScanFloat64Approx(valF64, 1e-6, 0, false); err != nil || !containsAddress(addrs, addrF64) {
		t.Fatalf("ScanFloat64Approx missing addr %X err %v", addrF64, err)
	}
}

func TestWritableOnlyScanSkipsReadOnly(t *testing.T) {
	p := openSelf(t)
	base, data := allocRW(t, 4096)

	const val = int32(0x10203040)
	// write via the mmap'd slice so the region can later be marked read-only
	binary.LittleEndian.PutUint32(data[:4], uint32(val))

	if err := unix.Mprotect(data, unix.PROT_READ); err != nil {
		t.Fatalf("Mprotect READ: %v", err)
	}
	t.Cleanup(func() { _ = unix.Mprotect(data, unix.PROT_READ|unix.PROT_WRITE) })

	if addrs, err := p.ScanInt32(val, 0, true); err != nil {
		t.Fatalf("ScanInt32 writableOnly err: %v", err)
	} else if containsAddress(addrs, base) {
		t.Fatalf("expected read-only region to be skipped when writableOnly")
	}

	if addrs, err := p.ScanInt32(val, 0, false); err != nil {
		t.Fatalf("ScanInt32 err: %v", err)
	} else if !containsAddress(addrs, base) {
		t.Fatalf("expected to find addr %X when not requiring writable", base)
	}
}

func TestParseMapsLine(t *testing.T) {
	cases := []struct {
		name     string
		line     string
		wantOK   bool
		start    uintptr
		end      uintptr
		readable bool
		writable bool
	}{
		{
			name:     "rw-p anonymous",
			line:     "7f0000000000-7f0000001000 rw-p 00000000 00:00 0",
			wantOK:   true,
			start:    0x7f0000000000,
			end:      0x7f0000001000,
			readable: true,
			writable: true,
		},
		{
			name:     "r--p file-backed",
			line:     "55a4f5c6b000-55a4f5c6c000 r--p 00000000 fd:01 12345                       /usr/bin/foo",
			wantOK:   true,
			start:    0x55a4f5c6b000,
			end:      0x55a4f5c6c000,
			readable: true,
			writable: false,
		},
		{
			name:   "vsyscall skipped",
			line:   "ffffffffff600000-ffffffffff601000 --xp 00000000 00:00 0                  [vsyscall]",
			wantOK: false,
		},
		{
			name:   "malformed",
			line:   "not a maps line",
			wantOK: false,
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r, ok := parseMapsLine(tc.line)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if r.start != tc.start || r.end != tc.end || r.readable != tc.readable || r.writable != tc.writable {
				t.Fatalf("got %+v want start=%x end=%x r=%v w=%v", r, tc.start, tc.end, tc.readable, tc.writable)
			}
		})
	}
}

func TestFloat32Abs(t *testing.T) {
	if float32Abs(-1.5) != 1.5 {
		t.Fatalf("float32Abs negative failed")
	}
	if float32Abs(0) != 0 {
		t.Fatalf("float32Abs zero failed")
	}
}
