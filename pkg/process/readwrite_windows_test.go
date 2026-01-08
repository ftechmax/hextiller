//go:build windows

package process

import (
	"testing"
	"unsafe"
)

func TestReadWriteRoundTrip(t *testing.T) {
	p := openSelf(t)

	buf := make([]byte, 64)
	base := uintptr(unsafe.Pointer(&buf[0]))

	if v, err := p.WriteInt32AndRead(base, int32(0x12345678)); err != nil || v != int32(0x12345678) {
		t.Fatalf("int32 roundtrip got %v err %v", v, err)
	}
	if v, err := p.WriteUint32AndRead(base+4, uint32(0x89ABCDEF)); err != nil || v != uint32(0x89ABCDEF) {
		t.Fatalf("uint32 roundtrip got %v err %v", v, err)
	}
	if v, err := p.WriteInt64AndRead(base+8, int64(0x123456789ABCDEF0)); err != nil || v != int64(0x123456789ABCDEF0) {
		t.Fatalf("int64 roundtrip got %v err %v", v, err)
	}
	if v, err := p.WriteUint64AndRead(base+16, uint64(0x0FEDCBA987654321)); err != nil || v != uint64(0x0FEDCBA987654321) {
		t.Fatalf("uint64 roundtrip got %v err %v", v, err)
	}
	if v, err := p.WriteFloat32AndRead(base+24, float32(12345.125)); err != nil || v != float32(12345.125) {
		t.Fatalf("float32 roundtrip got %v err %v", v, err)
	}
	if v, err := p.WriteFloat64AndRead(base+32, float64(98765.875)); err != nil || v != float64(98765.875) {
		t.Fatalf("float64 roundtrip got %v err %v", v, err)
	}
}
