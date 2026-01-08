//go:build windows

package process

import (
	"encoding/binary"
	"fmt"
	"math"

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

func (p *Process) ReadInt32(addr uintptr) (int32, error) {
	buf, err := p.read32(addr)
	if err != nil {
		return 0, err
	}
	return int32(binary.LittleEndian.Uint32(buf[:])), nil
}

func (p *Process) read32(addr uintptr) ([4]byte, error) {
	var buf [4]byte
	if err := p.readExact(addr, buf[:]); err != nil {
		return buf, err
	}
	return buf, nil
}

func (p *Process) write32(addr uintptr, buf [4]byte) error {
	return p.writeExact(addr, buf[:])
}

func (p *Process) read64(addr uintptr) ([8]byte, error) {
	var buf [8]byte
	if err := p.readExact(addr, buf[:]); err != nil {
		return buf, err
	}
	return buf, nil
}

func (p *Process) write64(addr uintptr, buf [8]byte) error {
	return p.writeExact(addr, buf[:])
}

func (p *Process) ReadUint32(addr uintptr) (uint32, error) {
	buf, err := p.read32(addr)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(buf[:]), nil
}

func (p *Process) WriteInt32(addr uintptr, value int32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], uint32(value))
	return p.write32(addr, buf)
}

func (p *Process) WriteUint32(addr uintptr, value uint32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], value)
	return p.write32(addr, buf)
}

func (p *Process) WriteInt32AndRead(addr uintptr, value int32) (int32, error) {
	if err := p.WriteInt32(addr, value); err != nil {
		return 0, err
	}
	return p.ReadInt32(addr)
}

func (p *Process) WriteUint32AndRead(addr uintptr, value uint32) (uint32, error) {
	if err := p.WriteUint32(addr, value); err != nil {
		return 0, err
	}
	return p.ReadUint32(addr)
}

func (p *Process) ReadInt64(addr uintptr) (int64, error) {
	buf, err := p.read64(addr)
	if err != nil {
		return 0, err
	}
	return int64(binary.LittleEndian.Uint64(buf[:])), nil
}

func (p *Process) ReadUint64(addr uintptr) (uint64, error) {
	buf, err := p.read64(addr)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buf[:]), nil
}

func (p *Process) WriteInt64(addr uintptr, value int64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(value))
	return p.write64(addr, buf)
}

func (p *Process) WriteUint64(addr uintptr, value uint64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], value)
	return p.write64(addr, buf)
}

func (p *Process) WriteInt64AndRead(addr uintptr, value int64) (int64, error) {
	if err := p.WriteInt64(addr, value); err != nil {
		return 0, err
	}
	return p.ReadInt64(addr)
}

func (p *Process) WriteUint64AndRead(addr uintptr, value uint64) (uint64, error) {
	if err := p.WriteUint64(addr, value); err != nil {
		return 0, err
	}
	return p.ReadUint64(addr)
}

func (p *Process) ReadFloat32(addr uintptr) (float32, error) {
	buf, err := p.read32(addr)
	if err != nil {
		return 0, err
	}
	return math.Float32frombits(binary.LittleEndian.Uint32(buf[:])), nil
}

func (p *Process) WriteFloat32(addr uintptr, value float32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], math.Float32bits(value))
	return p.write32(addr, buf)
}

func (p *Process) WriteFloat32AndRead(addr uintptr, value float32) (float32, error) {
	if err := p.WriteFloat32(addr, value); err != nil {
		return 0, err
	}
	return p.ReadFloat32(addr)
}

func (p *Process) ReadFloat64(addr uintptr) (float64, error) {
	buf, err := p.read64(addr)
	if err != nil {
		return 0, err
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(buf[:])), nil
}

func (p *Process) WriteFloat64(addr uintptr, value float64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], math.Float64bits(value))
	return p.write64(addr, buf)
}

func (p *Process) WriteFloat64AndRead(addr uintptr, value float64) (float64, error) {
	if err := p.WriteFloat64(addr, value); err != nil {
		return 0, err
	}
	return p.ReadFloat64(addr)
}
