//go:build windows

package process

import (
	"testing"

	"golang.org/x/sys/windows"
)

func TestScanFindsWrittenValues(t *testing.T) {
	p := openSelf(t)
	base := allocRW(t, 64)

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
	base := allocRW(t, 16)

	const val = int32(0x10203040)
	if _, err := p.WriteInt32AndRead(base, val); err != nil {
		t.Fatalf("write int32: %v", err)
	}

	var oldProtect uint32
	if err := windows.VirtualProtect(base, 16, windows.PAGE_READONLY, &oldProtect); err != nil {
		t.Fatalf("VirtualProtect to READONLY: %v", err)
	}

	if addrs, err := p.ScanInt32(val, 0, true); err != nil {
		t.Fatalf("ScanInt32 writableOnly err: %v", err)
	} else if containsAddress(addrs, base) {
		t.Fatalf("expected READONLY region to be skipped when writableOnly")
	}

	if addrs, err := p.ScanInt32(val, 0, false); err != nil {
		t.Fatalf("ScanInt32 err: %v", err)
	} else if !containsAddress(addrs, base) {
		t.Fatalf("expected to find addr %X when not requiring writable", base)
	}
}

func TestIsReadableWritableMasksProtectFlags(t *testing.T) {
	cases := []struct {
		name     string
		protect  uint32
		readable bool
		writable bool
	}{
		{"noaccess", windows.PAGE_NOACCESS, false, false},
		{"readonly", windows.PAGE_READONLY, true, false},
		{"readwrite", windows.PAGE_READWRITE, true, true},
		{"writecopy", windows.PAGE_WRITECOPY, true, true},
		{"execute", windows.PAGE_EXECUTE, false, false},
		{"exec_read", windows.PAGE_EXECUTE_READ, true, false},
		{"exec_readwrite", windows.PAGE_EXECUTE_READWRITE, true, true},
		{"guard_read", windows.PAGE_READONLY | windows.PAGE_GUARD, true, false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := isReadable(tc.protect); got != tc.readable {
				t.Fatalf("isReadable(%#x)=%v want %v", tc.protect, got, tc.readable)
			}
			if got := isWritable(tc.protect); got != tc.writable {
				t.Fatalf("isWritable(%#x)=%v want %v", tc.protect, got, tc.writable)
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
